package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GenerateUUID generates a new UUID
func GenerateUUID() uuid.UUID {
	return uuid.New()
}

// RandomInt generates a random integer between min and max (inclusive)
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	
	diff := max - min + 1
	n, err := rand.Int(rand.Reader, big.NewInt(int64(diff)))
	if err != nil {
		// Fallback to time-based pseudo-random
		return min + int(time.Now().UnixNano())%diff
	}
	
	return min + int(n.Int64())
}

// RandomDelay generates a random delay between min and max seconds
func RandomDelay(minSeconds, maxSeconds int) time.Duration {
	seconds := RandomInt(minSeconds, maxSeconds)
	return time.Duration(seconds) * time.Second
}

// ValidateHandle validates a Bluesky handle format
func ValidateHandle(handle string) bool {
	// Basic validation for Bluesky handles
	// Should be in format: username.domain.tld
	handleRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$`)
	return handleRegex.MatchString(handle) && len(handle) <= 253
}

// ValidateProxyURL validates a proxy URL format
func ValidateProxyURL(proxyURL string) error {
	if proxyURL == "" {
		return nil // Empty is valid (no proxy)
	}
	
	u, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL format: %w", err)
	}
	
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" {
		return fmt.Errorf("unsupported proxy scheme: %s", u.Scheme)
	}
	
	if u.Host == "" {
		return fmt.Errorf("proxy URL must include host")
	}
	
	return nil
}

// ParseCronExpression validates a cron expression format
func ParseCronExpression(cronExpr string) error {
	if cronExpr == "" {
		return nil // Empty is valid (no schedule)
	}
	
	// Basic validation for cron expression (5 or 6 fields)
	fields := strings.Fields(cronExpr)
	if len(fields) != 5 && len(fields) != 6 {
		return fmt.Errorf("cron expression must have 5 or 6 fields, got %d", len(fields))
	}
	
	// More detailed validation could be added here
	return nil
}

// SanitizeString removes potentially harmful characters from strings
func SanitizeString(input string) string {
	// Remove null bytes and control characters
	sanitized := strings.ReplaceAll(input, "\x00", "")
	sanitized = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(sanitized, "")
	
	// Trim whitespace
	sanitized = strings.TrimSpace(sanitized)
	
	return sanitized
}

// TruncateString truncates a string to a maximum length
func TruncateString(input string, maxLength int) string {
	if len(input) <= maxLength {
		return input
	}
	
	if maxLength <= 3 {
		return input[:maxLength]
	}
	
	return input[:maxLength-3] + "..."
}

// ContainsKeyword checks if text contains any of the specified keywords (case-insensitive)
func ContainsKeyword(text string, keywords []string) bool {
	textLower := strings.ToLower(text)
	
	for _, keyword := range keywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			return true
		}
	}
	
	return false
}

// ExtractHashtags extracts hashtags from text
func ExtractHashtags(text string) []string {
	hashtagRegex := regexp.MustCompile(`#[a-zA-Z0-9_\u4e00-\u9fff]+`)
	matches := hashtagRegex.FindAllString(text, -1)
	
	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	
	for _, match := range matches {
		if !seen[match] {
			seen[match] = true
			unique = append(unique, match)
		}
	}
	
	return unique
}

// ExtractMentions extracts mentions from text
func ExtractMentions(text string) []string {
	mentionRegex := regexp.MustCompile(`@[a-zA-Z0-9._-]+`)
	matches := mentionRegex.FindAllString(text, -1)
	
	// Remove duplicates and @ symbol
	seen := make(map[string]bool)
	var unique []string
	
	for _, match := range matches {
		handle := strings.TrimPrefix(match, "@")
		if !seen[handle] && ValidateHandle(handle) {
			seen[handle] = true
			unique = append(unique, handle)
		}
	}
	
	return unique
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}

// ParseDuration parses a duration string with support for various units
func ParseDuration(s string) (time.Duration, error) {
	// Try standard Go duration parsing first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	
	// Try parsing as seconds if it's just a number
	if seconds, err := strconv.Atoi(s); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}
	
	return 0, fmt.Errorf("invalid duration format: %s", s)
}

// MergeJSON merges two JSON objects, with the second overriding the first
func MergeJSON(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy base
	for k, v := range base {
		result[k] = v
	}
	
	// Override with new values
	for k, v := range override {
		result[k] = v
	}
	
	return result
}

// JSONToString converts a JSON object to a string
func JSONToString(obj interface{}) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// StringToJSON converts a JSON string to an object
func StringToJSON(s string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(s), &result)
	return result, err
}

// GetEnvOrDefault gets an environment variable or returns a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvAsInt gets an environment variable as an integer or returns a default value
func GetEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvAsBool gets an environment variable as a boolean or returns a default value
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// CalculateSuccessRate calculates success rate as a percentage
func CalculateSuccessRate(successful, total int) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(successful) / float64(total) * 100.0
}

// IsValidEmail validates an email address format
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidURL validates a URL format
func IsValidURL(urlStr string) bool {
	u, err := url.Parse(urlStr)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// SliceContains checks if a slice contains a specific item
func SliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// ChunkSlice splits a slice into chunks of specified size
func ChunkSlice(slice []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		return [][]string{slice}
	}
	
	var chunks [][]string
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	
	return chunks
}
