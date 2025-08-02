package bluesky

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/bsky-automation/shared/models"
)

// Client represents a Bluesky client with proxy support
type Client struct {
	xrpcc   *xrpc.Client
	account *models.Account
	proxy   *models.Proxy
}

// ClientConfig represents configuration for creating a client
type ClientConfig struct {
	Account *models.Account
	Proxy   *models.Proxy
	Timeout time.Duration
}

// NewClient creates a new Bluesky client with optional proxy support
func NewClient(config ClientConfig) (*Client, error) {
	if config.Account == nil {
		return nil, fmt.Errorf("account is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	client := &Client{
		account: config.Account,
		proxy:   config.Proxy,
	}

	// Create HTTP client with optional proxy
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Configure proxy if provided
	if config.Proxy != nil {
		proxyURL, err := buildProxyURL(config.Proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to build proxy URL: %w", err)
		}

		transport := &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
		httpClient.Transport = transport
	}

	// Create XRPC client
	client.xrpcc = &xrpc.Client{
		Client: httpClient,
		Host:   config.Account.Host,
		Auth:   &xrpc.AuthInfo{Handle: config.Account.Handle},
	}

	return client, nil
}

// buildProxyURL constructs a proxy URL from proxy configuration
func buildProxyURL(proxy *models.Proxy) (*url.URL, error) {
	var scheme string
	switch proxy.Type {
	case models.ProxyTypeHTTP:
		scheme = "http"
	case models.ProxyTypeSOCKS5:
		scheme = "socks5"
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", proxy.Type)
	}

	proxyURL := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", proxy.Host, proxy.Port),
	}

	if proxy.Username != nil && proxy.Password != nil {
		proxyURL.User = url.UserPassword(*proxy.Username, *proxy.Password)
	}

	return proxyURL, nil
}

// Authenticate authenticates the client with Bluesky
func (c *Client) Authenticate(ctx context.Context) error {
	// Try to load existing auth from cache first
	if c.account.AccessJWT != nil && c.account.RefreshJWT != nil {
		c.xrpcc.Auth.AccessJwt = *c.account.AccessJWT
		c.xrpcc.Auth.RefreshJwt = *c.account.RefreshJWT
		c.xrpcc.Auth.Did = *c.account.DID

		// Try to refresh the session
		refresh, err := comatproto.ServerRefreshSession(ctx, c.xrpcc)
		if err == nil {
			c.xrpcc.Auth.Did = refresh.Did
			c.xrpcc.Auth.AccessJwt = refresh.AccessJwt
			c.xrpcc.Auth.RefreshJwt = refresh.RefreshJwt

			// Update account with new tokens
			c.account.DID = &refresh.Did
			c.account.AccessJWT = &refresh.AccessJwt
			c.account.RefreshJWT = &refresh.RefreshJwt
			now := time.Now()
			c.account.LastLogin = &now

			return nil
		}
	}

	// Create new session
	auth, err := comatproto.ServerCreateSession(ctx, c.xrpcc, &comatproto.ServerCreateSession_Input{
		Identifier: c.account.Handle,
		Password:   c.account.Password,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	c.xrpcc.Auth.Did = auth.Did
	c.xrpcc.Auth.AccessJwt = auth.AccessJwt
	c.xrpcc.Auth.RefreshJwt = auth.RefreshJwt

	// Update account with new tokens
	c.account.DID = &auth.Did
	c.account.AccessJWT = &auth.AccessJwt
	c.account.RefreshJWT = &auth.RefreshJwt
	now := time.Now()
	c.account.LastLogin = &now

	return nil
}

// Post creates a new post
func (c *Client) Post(ctx context.Context, text string, options *PostOptions) (*PostResult, error) {
	if options == nil {
		options = &PostOptions{}
	}

	post := &bsky.FeedPost{
		Text:      text,
		CreatedAt: time.Now().Local().Format(time.RFC3339),
	}

	// Handle reply
	if options.ReplyTo != "" {
		reply, err := c.buildReply(ctx, options.ReplyTo)
		if err != nil {
			return nil, fmt.Errorf("failed to build reply: %w", err)
		}
		post.Reply = reply
	}

	// Handle quote
	if options.QuoteTo != "" {
		embed, err := c.buildQuote(ctx, options.QuoteTo)
		if err != nil {
			return nil, fmt.Errorf("failed to build quote: %w", err)
		}
		post.Embed = embed
	}

	// Handle images
	if len(options.Images) > 0 {
		embed, err := c.buildImageEmbed(ctx, options.Images)
		if err != nil {
			return nil, fmt.Errorf("failed to build image embed: %w", err)
		}
		if post.Embed == nil {
			post.Embed = &bsky.FeedPost_Embed{}
		}
		post.Embed.EmbedImages = embed
	}

	// Create the post
	resp, err := comatproto.RepoCreateRecord(ctx, c.xrpcc, &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       c.xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: post,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return &PostResult{
		URI: resp.Uri,
		CID: resp.Cid,
	}, nil
}

// Follow follows a user
func (c *Client) Follow(ctx context.Context, handle string) (*FollowResult, error) {
	profile, err := bsky.ActorGetProfile(ctx, c.xrpcc, handle)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	follow := bsky.GraphFollow{
		LexiconTypeID: "app.bsky.graph.follow",
		CreatedAt:     time.Now().Local().Format(time.RFC3339),
		Subject:       profile.Did,
	}

	resp, err := comatproto.RepoCreateRecord(ctx, c.xrpcc, &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.graph.follow",
		Repo:       c.xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: &follow,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create follow: %w", err)
	}

	return &FollowResult{
		URI:       resp.Uri,
		CID:       resp.Cid,
		TargetDID: profile.Did,
	}, nil
}

// Like likes a post
func (c *Client) Like(ctx context.Context, postURI string) (*LikeResult, error) {
	// Get the post to like
	parts := parseATURI(postURI)
	if parts == nil {
		return nil, fmt.Errorf("invalid post URI: %s", postURI)
	}

	resp, err := comatproto.RepoGetRecord(ctx, c.xrpcc, "", parts.Collection, parts.DID, parts.RKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	like := &bsky.FeedLike{
		CreatedAt: time.Now().Format("2006-01-02T15:04:05.000Z"),
		Subject:   &comatproto.RepoStrongRef{Uri: resp.Uri, Cid: *resp.Cid},
	}

	likeResp, err := comatproto.RepoCreateRecord(ctx, c.xrpcc, &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.like",
		Repo:       c.xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: like,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create like: %w", err)
	}

	return &LikeResult{
		URI:     likeResp.Uri,
		CID:     likeResp.Cid,
		PostURI: postURI,
	}, nil
}

// Repost reposts a post
func (c *Client) Repost(ctx context.Context, postURI string) (*RepostResult, error) {
	// Get the post to repost
	parts := parseATURI(postURI)
	if parts == nil {
		return nil, fmt.Errorf("invalid post URI: %s", postURI)
	}

	resp, err := comatproto.RepoGetRecord(ctx, c.xrpcc, "", parts.Collection, parts.DID, parts.RKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}

	repost := &bsky.FeedRepost{
		CreatedAt: time.Now().Local().Format(time.RFC3339),
		Subject: &comatproto.RepoStrongRef{
			Uri: resp.Uri,
			Cid: *resp.Cid,
		},
	}

	repostResp, err := comatproto.RepoCreateRecord(ctx, c.xrpcc, &comatproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.repost",
		Repo:       c.xrpcc.Auth.Did,
		Record: &lexutil.LexiconTypeDecoder{
			Val: repost,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create repost: %w", err)
	}

	return &RepostResult{
		URI:     repostResp.Uri,
		CID:     repostResp.Cid,
		PostURI: postURI,
	}, nil
}

// GetTimeline gets the user's timeline
func (c *Client) GetTimeline(ctx context.Context, options *TimelineOptions) (*TimelineResult, error) {
	if options == nil {
		options = &TimelineOptions{Limit: 30}
	}

	resp, err := bsky.FeedGetTimeline(ctx, c.xrpcc, "reverse-chronological", options.Cursor, int64(options.Limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline: %w", err)
	}

	result := &TimelineResult{
		Feed: resp.Feed,
	}
	if resp.Cursor != nil {
		result.Cursor = *resp.Cursor
	}

	return result, nil
}

// GetProfile gets a user's profile
func (c *Client) GetProfile(ctx context.Context, handle string) (*ProfileResult, error) {
	profile, err := bsky.ActorGetProfile(ctx, c.xrpcc, handle)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return &ProfileResult{
		Profile: profile,
	}, nil
}

// Search searches for posts
func (c *Client) Search(ctx context.Context, query string, options *SearchOptions) (*SearchResult, error) {
	if options == nil {
		options = &SearchOptions{Limit: 50}
	}

	// Note: This is a simplified implementation
	// The actual API may require different parameters
	resp, err := bsky.FeedSearchPosts(ctx, c.xrpcc, "", "", "", "", int64(options.Limit), "", "", "", "", []string{}, "", query)
	if err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}

	result := &SearchResult{
		Posts: resp.Posts,
	}
	if resp.Cursor != nil {
		result.Cursor = *resp.Cursor
	}

	return result, nil
}

// GetAccount returns the associated account
func (c *Client) GetAccount() *models.Account {
	return c.account
}

// GetProxy returns the associated proxy
func (c *Client) GetProxy() *models.Proxy {
	return c.proxy
}

// UpdateLastActivity updates the account's last activity timestamp
func (c *Client) UpdateLastActivity() {
	now := time.Now()
	c.account.LastActivity = &now
}
