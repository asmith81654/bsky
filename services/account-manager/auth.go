package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/bsky-automation/shared/models"
	"github.com/bsky-automation/shared/utils"
)

// AuthService handles authentication and authorization
type AuthService struct {
	db        *sql.DB
	rdb       *redis.Client
	jwtSecret []byte
}

// NewAuthService creates a new auth service
func NewAuthService(db *sql.DB, rdb *redis.Client) *AuthService {
	jwtSecret := utils.GetEnvOrDefault("JWT_SECRET", "your-jwt-secret-key")
	return &AuthService{
		db:        db,
		rdb:       rdb,
		jwtSecret: []byte(jwtSecret),
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
	User         UserInfo  `json:"user"`
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	AccessToken string `json:"access_token" validate:"required"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// AccountStatsResponse represents account statistics
type AccountStatsResponse struct {
	TotalAccounts    int                            `json:"total_accounts"`
	ActiveAccounts   int                            `json:"active_accounts"`
	InactiveAccounts int                            `json:"inactive_accounts"`
	ErrorAccounts    int                            `json:"error_accounts"`
	StatusBreakdown  map[models.AccountStatus]int   `json:"status_breakdown"`
	ProxyUsage       map[string]int                 `json:"proxy_usage"`
	RecentActivity   []AccountActivitySummary       `json:"recent_activity"`
}

// AccountActivitySummary represents recent account activity
type AccountActivitySummary struct {
	AccountID    int       `json:"account_id"`
	Handle       string    `json:"handle"`
	LastActivity time.Time `json:"last_activity"`
	Status       string    `json:"status"`
}

// AccountMetricsResponse represents account metrics
type AccountMetricsResponse struct {
	AccountID        int                    `json:"account_id"`
	Handle           string                 `json:"handle"`
	TotalTasks       int                    `json:"total_tasks"`
	CompletedTasks   int                    `json:"completed_tasks"`
	FailedTasks      int                    `json:"failed_tasks"`
	SuccessRate      float64                `json:"success_rate"`
	LastActivity     *time.Time             `json:"last_activity"`
	DailyMetrics     []DailyMetric          `json:"daily_metrics"`
	StrategyMetrics  []StrategyMetric       `json:"strategy_metrics"`
}

// DailyMetric represents daily performance metrics
type DailyMetric struct {
	Date           string  `json:"date"`
	TasksCompleted int     `json:"tasks_completed"`
	TasksFailed    int     `json:"tasks_failed"`
	SuccessRate    float64 `json:"success_rate"`
}

// StrategyMetric represents strategy-specific metrics
type StrategyMetric struct {
	StrategyName   string  `json:"strategy_name"`
	StrategyType   string  `json:"strategy_type"`
	TasksCompleted int     `json:"tasks_completed"`
	TasksFailed    int     `json:"tasks_failed"`
	SuccessRate    float64 `json:"success_rate"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// For now, use a simple admin user check
	// In production, this would check against a users table
	if req.Username != "admin" || req.Password != "admin123" {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	accessToken, refreshToken, expiresAt, err := s.generateTokens(1, req.Username, "admin")
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store refresh token in Redis
	err = s.storeRefreshToken(ctx, refreshToken, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
		User: UserInfo{
			ID:       1,
			Username: req.Username,
			Role:     "admin",
		},
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, req *RefreshTokenRequest) (*LoginResponse, error) {
	// Validate refresh token
	userID, err := s.validateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new tokens
	accessToken, refreshToken, expiresAt, err := s.generateTokens(userID, "admin", "admin")
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store new refresh token and invalidate old one
	err = s.storeRefreshToken(ctx, refreshToken, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	err = s.invalidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to invalidate old refresh token: %v\n", err)
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
		User: UserInfo{
			ID:       userID,
			Username: "admin",
			Role:     "admin",
		},
	}, nil
}

// Logout invalidates tokens
func (s *AuthService) Logout(ctx context.Context, req *LogoutRequest) error {
	// Parse access token to get refresh token info
	claims, err := s.parseToken(req.AccessToken)
	if err != nil {
		return fmt.Errorf("invalid access token: %w", err)
	}

	// Add access token to blacklist
	err = s.blacklistToken(ctx, req.AccessToken, claims.ExpiresAt.Time)
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// ValidateToken validates a JWT token
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	return s.parseToken(tokenString)
}

// Helper methods

func (s *AuthService) generateTokens(userID int, username, role string) (string, string, time.Time, error) {
	// Access token (15 minutes)
	accessExpiresAt := time.Now().Add(15 * time.Minute)
	accessClaims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bsky-automation",
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Refresh token (7 days)
	refreshToken, err := s.generateRandomToken()
	if err != nil {
		return "", "", time.Time{}, err
	}

	return accessTokenString, refreshToken, accessExpiresAt, nil
}

func (s *AuthService) generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *AuthService) parseToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (s *AuthService) storeRefreshToken(ctx context.Context, token string, userID int) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	expiration := 7 * 24 * time.Hour // 7 days
	return s.rdb.Set(ctx, key, userID, expiration).Err()
}

func (s *AuthService) validateRefreshToken(ctx context.Context, token string) (int, error) {
	key := fmt.Sprintf("refresh_token:%s", token)
	result, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	userID := 0
	if _, err := fmt.Sscanf(result, "%d", &userID); err != nil {
		return 0, err
	}

	return userID, nil
}

func (s *AuthService) invalidateRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("refresh_token:%s", token)
	return s.rdb.Del(ctx, key).Err()
}

func (s *AuthService) blacklistToken(ctx context.Context, token string, expiresAt time.Time) error {
	key := fmt.Sprintf("blacklist:%s", token)
	expiration := time.Until(expiresAt)
	if expiration <= 0 {
		return nil // Token already expired
	}
	return s.rdb.Set(ctx, key, "1", expiration).Err()
}

func (s *AuthService) isTokenBlacklisted(ctx context.Context, token string) bool {
	key := fmt.Sprintf("blacklist:%s", token)
	_, err := s.rdb.Get(ctx, key).Result()
	return err == nil
}
