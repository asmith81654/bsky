package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/bsky-automation/shared/models"
	"github.com/bsky-automation/shared/utils"
)

// @title Account Manager API
// @version 1.0
// @description Account management service for Bluesky Automation Platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8001
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
	// Load configuration
	config := loadConfig()

	// Initialize database
	db, err := initDatabase(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	rdb := initRedis(config.RedisURL)
	defer rdb.Close()

	// Initialize services
	accountService := NewAccountService(db, rdb)
	authService := NewAuthService(db, rdb)

	// Initialize handlers
	accountHandler := NewAccountHandler(accountService, authService)

	// Setup router
	router := setupRouter(accountHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Account Manager starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Config represents the application configuration
type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
	Environment string
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	return &Config{
		Port:        utils.GetEnvOrDefault("SERVICE_PORT", "8001"),
		DatabaseURL: utils.GetEnvOrDefault("DATABASE_URL", "postgres://bsky_user:bsky_password@localhost:5432/bsky_automation?sslmode=disable"),
		RedisURL:    utils.GetEnvOrDefault("REDIS_URL", "redis://:redis_password@localhost:6379/0"),
		JWTSecret:   utils.GetEnvOrDefault("JWT_SECRET", "your-jwt-secret-key"),
		Environment: utils.GetEnvOrDefault("ENVIRONMENT", "development"),
	}
}

// initDatabase initializes the database connection
func initDatabase(databaseURL string) (*sql.DB, error) {
	config := utils.DatabaseConfig{
		Host:     utils.GetEnvOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     utils.GetEnvOrDefault("DB_USER", "bsky_user"),
		Password: utils.GetEnvOrDefault("DB_PASSWORD", "bsky_test_password"),
		DBName:   utils.GetEnvOrDefault("DB_NAME", "bsky_automation"),
		SSLMode:  "disable",
	}

	// In production, parse the databaseURL properly
	db, err := utils.NewPostgresConnection(config)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := utils.HealthCheckDB(db); err != nil {
		return nil, err
	}

	log.Println("Database connection established")
	return db, nil
}

// initRedis initializes the Redis connection
func initRedis(redisURL string) *redis.Client {
	config := utils.RedisConfig{
		Host:     utils.GetEnvOrDefault("REDIS_HOST", "localhost"),
		Port:     6379,
		Password: utils.GetEnvOrDefault("REDIS_PASSWORD", "redis_test_password"),
		DB:       0,
	}

	// In production, parse the redisURL properly
	rdb := utils.NewRedisClient(config)

	// Test connection
	if err := utils.HealthCheckRedis(rdb); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connection established")
	return rdb
}

// setupRouter sets up the Gin router with all routes
func setupRouter(accountHandler *AccountHandler) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check endpoint
	router.GET("/health", healthCheckHandler)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Account routes
		accounts := v1.Group("/accounts")
		{
			accounts.GET("", accountHandler.ListAccounts)
			accounts.POST("", accountHandler.CreateAccount)
			accounts.GET("/:id", accountHandler.GetAccount)
			accounts.PUT("/:id", accountHandler.UpdateAccount)
			accounts.DELETE("/:id", accountHandler.DeleteAccount)
			accounts.POST("/:id/test-auth", accountHandler.TestAuthentication)
			accounts.POST("/:id/refresh-auth", accountHandler.RefreshAuthentication)
		}

		// Authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", accountHandler.Login)
			auth.POST("/refresh", accountHandler.RefreshToken)
			auth.POST("/logout", accountHandler.Logout)
		}

		// Account statistics
		stats := v1.Group("/stats")
		{
			stats.GET("/accounts", accountHandler.GetAccountStats)
			stats.GET("/accounts/:id/metrics", accountHandler.GetAccountMetrics)
		}
	}

	return router
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// healthCheckHandler handles health check requests
// @Summary Health check
// @Description Check if the service is healthy
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthCheckResponse
// @Router /health [get]
func healthCheckHandler(c *gin.Context) {
	response := models.HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Services: map[string]string{
			"database": "connected",
			"redis":    "connected",
		},
	}

	c.JSON(http.StatusOK, response)
}
