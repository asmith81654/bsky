package utils

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewPostgresConnection creates a new PostgreSQL connection
func NewPostgresConnection(config DatabaseConfig) (*sql.DB, error) {
	if config.SSLMode == "" {
		config.SSLMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// NewRedisClient creates a new Redis client
func NewRedisClient(config RedisConfig) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	return rdb
}

// ParseDatabaseURL parses a database URL into DatabaseConfig
func ParseDatabaseURL(databaseURL string) (DatabaseConfig, error) {
	// Example: postgres://user:password@host:port/dbname?sslmode=disable
	// This is a simplified parser - you might want to use a proper URL parser
	
	config := DatabaseConfig{
		Port:    5432,
		SSLMode: "disable",
	}

	// Basic parsing - in production, use a proper URL parser
	if databaseURL == "" {
		return config, fmt.Errorf("database URL is empty")
	}

	// For now, return a basic config
	// In a real implementation, you'd parse the URL properly
	return config, nil
}

// ParseRedisURL parses a Redis URL into RedisConfig
func ParseRedisURL(redisURL string) (RedisConfig, error) {
	// Example: redis://:password@host:port/db
	
	config := RedisConfig{
		Port: 6379,
		DB:   0,
	}

	// Basic parsing - in production, use a proper URL parser
	if redisURL == "" {
		return config, fmt.Errorf("Redis URL is empty")
	}

	// For now, return a basic config
	// In a real implementation, you'd parse the URL properly
	return config, nil
}

// HealthCheckDB checks if the database connection is healthy
func HealthCheckDB(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// HealthCheckRedis checks if the Redis connection is healthy
func HealthCheckRedis(rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}

	return nil
}

// Transaction executes a function within a database transaction
func Transaction(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// Paginate calculates pagination parameters
func Paginate(page, pageSize int, totalItems int64) (offset int, limit int, totalPages int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset = (page - 1) * pageSize
	limit = pageSize
	totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))

	return offset, limit, totalPages
}

// BuildWhereClause builds a WHERE clause with parameters
func BuildWhereClause(conditions map[string]interface{}) (string, []interface{}) {
	if len(conditions) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}
	argIndex := 1

	for column, value := range conditions {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, argIndex))
		args = append(args, value)
		argIndex++
	}

	whereClause := "WHERE " + strings.Join(clauses, " AND ")
	return whereClause, args
}

// BuildUpdateClause builds an UPDATE SET clause with parameters
func BuildUpdateClause(updates map[string]interface{}) (string, []interface{}) {
	if len(updates) == 0 {
		return "", nil
	}

	var clauses []string
	var args []interface{}
	argIndex := 1

	for column, value := range updates {
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, argIndex))
		args = append(args, value)
		argIndex++
	}

	setClause := "SET " + strings.Join(clauses, ", ")
	return setClause, args
}

// ScanRow scans a database row into a struct using reflection
// This is a simplified version - in production, consider using a library like sqlx
func ScanRow(rows *sql.Rows, dest interface{}) error {
	// This would require reflection to map columns to struct fields
	// For now, return an error indicating it's not implemented
	return fmt.Errorf("ScanRow not implemented - use manual scanning or sqlx library")
}

// GetTableExists checks if a table exists in the database
func GetTableExists(db *sql.DB, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		);
	`
	
	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	
	return exists, nil
}

// GetTableRowCount gets the number of rows in a table
func GetTableRowCount(db *sql.DB, tableName string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	
	var count int64
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get row count: %w", err)
	}
	
	return count, nil
}
