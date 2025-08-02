package bluesky

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	comatproto "github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
)

// PostOptions represents options for creating a post
type PostOptions struct {
	ReplyTo string   `json:"reply_to,omitempty"`
	QuoteTo string   `json:"quote_to,omitempty"`
	Images  []string `json:"images,omitempty"`
}

// PostResult represents the result of creating a post
type PostResult struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// FollowResult represents the result of following a user
type FollowResult struct {
	URI       string `json:"uri"`
	CID       string `json:"cid"`
	TargetDID string `json:"target_did"`
}

// LikeResult represents the result of liking a post
type LikeResult struct {
	URI     string `json:"uri"`
	CID     string `json:"cid"`
	PostURI string `json:"post_uri"`
}

// RepostResult represents the result of reposting a post
type RepostResult struct {
	URI     string `json:"uri"`
	CID     string `json:"cid"`
	PostURI string `json:"post_uri"`
}

// TimelineOptions represents options for getting timeline
type TimelineOptions struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// TimelineResult represents the result of getting timeline
type TimelineResult struct {
	Feed   []*bsky.FeedDefs_FeedViewPost `json:"feed"`
	Cursor string                        `json:"cursor,omitempty"`
}

// ProfileResult represents the result of getting a profile
type ProfileResult struct {
	Profile *bsky.ActorDefs_ProfileViewDetailed `json:"profile"`
}

// SearchOptions represents options for searching
type SearchOptions struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// SearchResult represents the result of searching
type SearchResult struct {
	Posts  []*bsky.FeedDefs_PostView `json:"posts"`
	Cursor string                    `json:"cursor,omitempty"`
}

// ATURIParts represents parsed AT URI components
type ATURIParts struct {
	DID        string
	Collection string
	RKey       string
}

// parseATURI parses an AT URI into its components
func parseATURI(uri string) *ATURIParts {
	// Expected format: at://did:plc:xxx/app.bsky.feed.post/yyy
	if !strings.HasPrefix(uri, "at://") {
		return nil
	}

	parts := strings.Split(uri[5:], "/") // Remove "at://" prefix
	if len(parts) < 3 {
		return nil
	}

	return &ATURIParts{
		DID:        parts[0],
		Collection: parts[1],
		RKey:       parts[2],
	}
}

// buildReply builds a reply structure for a post
func (c *Client) buildReply(ctx context.Context, replyToURI string) (*bsky.FeedPost_ReplyRef, error) {
	parts := parseATURI(replyToURI)
	if parts == nil {
		return nil, fmt.Errorf("invalid reply URI: %s", replyToURI)
	}

	resp, err := comatproto.RepoGetRecord(ctx, c.xrpcc, "", parts.Collection, parts.DID, parts.RKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get reply target: %w", err)
	}

	reply := &bsky.FeedPost_ReplyRef{
		Parent: &comatproto.RepoStrongRef{Cid: *resp.Cid, Uri: resp.Uri},
	}

	// Check if the target post is also a reply to set the root
	if orig, ok := resp.Value.Val.(*bsky.FeedPost); ok && orig.Reply != nil && orig.Reply.Root != nil {
		reply.Root = &comatproto.RepoStrongRef{Cid: orig.Reply.Root.Cid, Uri: orig.Reply.Root.Uri}
	} else {
		reply.Root = &comatproto.RepoStrongRef{Cid: *resp.Cid, Uri: resp.Uri}
	}

	return reply, nil
}

// buildQuote builds a quote embed for a post
func (c *Client) buildQuote(ctx context.Context, quoteURI string) (*bsky.FeedPost_Embed, error) {
	parts := parseATURI(quoteURI)
	if parts == nil {
		return nil, fmt.Errorf("invalid quote URI: %s", quoteURI)
	}

	resp, err := comatproto.RepoGetRecord(ctx, c.xrpcc, "", parts.Collection, parts.DID, parts.RKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote target: %w", err)
	}

	embed := &bsky.FeedPost_Embed{
		EmbedRecord: &bsky.EmbedRecord{
			Record: &comatproto.RepoStrongRef{Cid: *resp.Cid, Uri: resp.Uri},
		},
	}

	return embed, nil
}

// buildImageEmbed builds an image embed for a post
func (c *Client) buildImageEmbed(ctx context.Context, imagePaths []string) (*bsky.EmbedImages, error) {
	if len(imagePaths) == 0 {
		return nil, fmt.Errorf("no images provided")
	}

	if len(imagePaths) > 4 {
		return nil, fmt.Errorf("maximum 4 images allowed")
	}

	var images []*bsky.EmbedImages_Image
	for _, imagePath := range imagePaths {
		// Read image file
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read image %s: %w", imagePath, err)
		}

		// Upload blob
		resp, err := comatproto.RepoUploadBlob(ctx, c.xrpcc, strings.NewReader(string(imageData)))
		if err != nil {
			return nil, fmt.Errorf("failed to upload image %s: %w", imagePath, err)
		}

		// Detect content type
		contentType := http.DetectContentType(imageData)

		image := &bsky.EmbedImages_Image{
			Image: &lexutil.LexBlob{
				Ref:      resp.Blob.Ref,
				MimeType: contentType,
				Size:     resp.Blob.Size,
			},
			Alt: "", // Could be enhanced to support alt text
		}

		images = append(images, image)
	}

	return &bsky.EmbedImages{
		Images: images,
	}, nil
}

// ImageUploadOptions represents options for uploading images
type ImageUploadOptions struct {
	AltText string `json:"alt_text,omitempty"`
}

// UploadImage uploads an image and returns the blob reference
func (c *Client) UploadImage(ctx context.Context, imagePath string, options *ImageUploadOptions) (*lexutil.LexBlob, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	resp, err := comatproto.RepoUploadBlob(ctx, c.xrpcc, strings.NewReader(string(imageData)))
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	contentType := http.DetectContentType(imageData)

	return &lexutil.LexBlob{
		Ref:      resp.Blob.Ref,
		MimeType: contentType,
		Size:     resp.Blob.Size,
	}, nil
}

// NotificationOptions represents options for getting notifications
type NotificationOptions struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Seen   *bool  `json:"seen,omitempty"`
}

// NotificationResult represents the result of getting notifications
type NotificationResult struct {
	Notifications []*bsky.NotificationListNotifications_Notification `json:"notifications"`
	Cursor        string                                              `json:"cursor,omitempty"`
}

// GetNotifications gets the user's notifications
func (c *Client) GetNotifications(ctx context.Context, options *NotificationOptions) (*NotificationResult, error) {
	if options == nil {
		options = &NotificationOptions{Limit: 50}
	}

	// Note: Simplified implementation - API signature may have changed
	seenVal := false
	if options.Seen != nil {
		seenVal = *options.Seen
	}
	resp, err := bsky.NotificationListNotifications(ctx, c.xrpcc, options.Cursor, int64(options.Limit), seenVal, []string{}, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}

	result := &NotificationResult{
		Notifications: resp.Notifications,
	}
	if resp.Cursor != nil {
		result.Cursor = *resp.Cursor
	}

	return result, nil
}

// MarkNotificationsRead marks notifications as read
func (c *Client) MarkNotificationsRead(ctx context.Context, seenAt *time.Time) error {
	if seenAt == nil {
		now := time.Now()
		seenAt = &now
	}

	err := bsky.NotificationUpdateSeen(ctx, c.xrpcc, &bsky.NotificationUpdateSeen_Input{
		SeenAt: seenAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("failed to mark notifications as read: %w", err)
	}

	return nil
}

// DeletePost deletes a post
func (c *Client) DeletePost(ctx context.Context, postURI string) error {
	parts := parseATURI(postURI)
	if parts == nil {
		return fmt.Errorf("invalid post URI: %s", postURI)
	}

	_, err := comatproto.RepoDeleteRecord(ctx, c.xrpcc, &comatproto.RepoDeleteRecord_Input{
		Collection: parts.Collection,
		Repo:       c.xrpcc.Auth.Did,
		Rkey:       parts.RKey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	return nil
}

// Block blocks a user
func (c *Client) Block(ctx context.Context, handle string) (*BlockResult, error) {
	profile, err := bsky.ActorGetProfile(ctx, c.xrpcc, handle)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	block := bsky.GraphBlock{
		LexiconTypeID: "app.bsky.graph.block",
		CreatedAt:     time.Now().Local().Format(time.RFC3339),
		Subject:       profile.Did,
	}

	resp, err := comatproto.RepoCreateRecord(ctx, c.xrpcc, &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.graph.block",
		Repo:       c.xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: &block,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	return &BlockResult{
		URI:       resp.Uri,
		CID:       resp.Cid,
		TargetDID: profile.Did,
	}, nil
}

// BlockResult represents the result of blocking a user
type BlockResult struct {
	URI       string `json:"uri"`
	CID       string `json:"cid"`
	TargetDID string `json:"target_did"`
}
