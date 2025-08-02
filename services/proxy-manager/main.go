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

// @title Proxy Manager API
// @version 1.0
// @description Proxy management service for Bluesky Automation Platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8002
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
	proxyService := NewProxyService(db, rdb)
	healthService := NewHealthService(db, rdb)

	// Initialize handlers
	proxyHandler := NewProxyHandler(proxyService)

	// Setup router
	router := setupRouter(proxyHandler)

	// Start health check scheduler
	go healthService.StartHealthCheckScheduler(context.Background())

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Proxy Manager starting on port %s", config.Port)
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
	Environment string
}

// loadConfig loads configuration from environment variables
func loadConfig() *Config {
	return &Config{
		Port:        utils.GetEnvOrDefault("SERVICE_PORT", "8002"),
		DatabaseURL: utils.GetEnvOrDefault("DATABASE_URL", "postgres://bsky_user:bsky_password@localhost:5432/bsky_automation?sslmode=disable"),
		RedisURL:    utils.GetEnvOrDefault("REDIS_URL", "redis://:redis_password@localhost:6379/1"),
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
		DB:       1, // Use DB 1 for proxy manager
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
func setupRouter(proxyHandler *ProxyHandler) *gin.Engine {
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
		// Proxy routes
		proxies := v1.Group("/proxies")
		{
			proxies.GET("", proxyHandler.ListProxies)
			proxies.POST("", proxyHandler.CreateProxy)
			proxies.GET("/:id", proxyHandler.GetProxy)
			proxies.PUT("/:id", proxyHandler.UpdateProxy)
			proxies.DELETE("/:id", proxyHandler.DeleteProxy)
			proxies.POST("/:id/test", proxyHandler.TestProxy)
			proxies.POST("/:id/health-check", proxyHandler.RunHealthCheck)
		}

		// Proxy assignment routes
		assignment := v1.Group("/assignment")
		{
			assignment.GET("/available", proxyHandler.GetAvailableProxies)
			assignment.POST("/assign", proxyHandler.AssignProxy)
			assignment.POST("/release", proxyHandler.ReleaseProxy)
			assignment.GET("/usage", proxyHandler.GetProxyUsage)
		}

		// Proxy statistics
		stats := v1.Group("/stats")
		{
			stats.GET("/proxies", proxyHandler.GetProxyStats)
			stats.GET("/health", proxyHandler.GetHealthStats)
			stats.GET("/performance", proxyHandler.GetPerformanceStats)
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
