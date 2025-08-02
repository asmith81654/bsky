package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"golang.org/x/net/context"
)

func main() {
	fmt.Println("🚀 Testing Bluesky Automation Platform Services...")

	// Test PostgreSQL connection
	fmt.Println("\n📊 Testing PostgreSQL connection...")
	db, err := sql.Open("postgres", "postgres://bsky_user:bsky_test_password@localhost:5432/bsky_test?sslmode=disable")
	if err != nil {
		log.Printf("❌ Failed to connect to PostgreSQL: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Printf("❌ Failed to ping PostgreSQL: %v", err)
		os.Exit(1)
	}
	fmt.Println("✅ PostgreSQL connection successful!")

	// Test Redis connection
	fmt.Println("\n🔴 Testing Redis connection...")
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "redis_test_password",
		DB:       0,
	})
	defer rdb.Close()

	ctx := context.Background()
	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("❌ Failed to connect to Redis: %v", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Redis connection successful! Response: %s\n", pong)

	// Test basic database operations
	fmt.Println("\n🗄️ Testing database operations...")
	
	// Create a test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("❌ Failed to create test table: %v", err)
		os.Exit(1)
	}

	// Insert test data
	_, err = db.Exec("INSERT INTO test_table (name) VALUES ($1)", "Test Service")
	if err != nil {
		log.Printf("❌ Failed to insert test data: %v", err)
		os.Exit(1)
	}

	// Query test data
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_table").Scan(&count)
	if err != nil {
		log.Printf("❌ Failed to query test data: %v", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Database operations successful! Records: %d\n", count)

	// Test Redis operations
	fmt.Println("\n🔄 Testing Redis operations...")
	
	// Set a test key
	err = rdb.Set(ctx, "test_key", "test_value", 0).Err()
	if err != nil {
		log.Printf("❌ Failed to set Redis key: %v", err)
		os.Exit(1)
	}

	// Get the test key
	val, err := rdb.Get(ctx, "test_key").Result()
	if err != nil {
		log.Printf("❌ Failed to get Redis key: %v", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Redis operations successful! Value: %s\n", val)

	// Clean up
	fmt.Println("\n🧹 Cleaning up test data...")
	_, err = db.Exec("DROP TABLE IF EXISTS test_table")
	if err != nil {
		log.Printf("⚠️ Warning: Failed to clean up test table: %v", err)
	}

	err = rdb.Del(ctx, "test_key").Err()
	if err != nil {
		log.Printf("⚠️ Warning: Failed to clean up Redis key: %v", err)
	}

	fmt.Println("\n🎉 All service tests passed successfully!")
	fmt.Println("✅ PostgreSQL: Connected and operational")
	fmt.Println("✅ Redis: Connected and operational")
	fmt.Println("✅ Database operations: Working correctly")
	fmt.Println("✅ Cache operations: Working correctly")
	fmt.Println("\n🏆 Bluesky Automation Platform infrastructure is ready!")
}
