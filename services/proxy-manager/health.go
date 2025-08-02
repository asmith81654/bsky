package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/bsky-automation/shared/models"
	"github.com/bsky-automation/shared/utils"
)

// HealthService handles proxy health checking
type HealthService struct {
	db  *sql.DB
	rdb *redis.Client
	proxyService *ProxyService
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewHealthService creates a new health service
func NewHealthService(db *sql.DB, rdb *redis.Client) *HealthService {
	return &HealthService{
		db:  db,
		rdb: rdb,
		proxyService: NewProxyService(db, rdb),
		stopChan: make(chan struct{}),
	}
}

// StartHealthCheckScheduler starts the health check scheduler
func (h *HealthService) StartHealthCheckScheduler(ctx context.Context) {
	log.Println("Starting proxy health check scheduler...")

	// Get health check interval from settings or use default
	interval := utils.GetEnvAsInt("PROXY_HEALTH_CHECK_INTERVAL", 300) // 5 minutes default
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Run initial health check
	h.runHealthCheckCycle(ctx)

	for {
		select {
		case <-ticker.C:
			h.runHealthCheckCycle(ctx)
		case <-h.stopChan:
			log.Println("Health check scheduler stopped")
			return
		case <-ctx.Done():
			log.Println("Health check scheduler context cancelled")
			return
		}
	}
}

// StopHealthCheckScheduler stops the health check scheduler
func (h *HealthService) StopHealthCheckScheduler() {
	close(h.stopChan)
	h.wg.Wait()
}

// runHealthCheckCycle runs a complete health check cycle for all active proxies
func (h *HealthService) runHealthCheckCycle(ctx context.Context) {
	log.Println("Starting health check cycle...")

	// Get all active proxies
	proxies, err := h.getActiveProxies(ctx)
	if err != nil {
		log.Printf("Failed to get active proxies: %v", err)
		return
	}

	if len(proxies) == 0 {
		log.Println("No active proxies to check")
		return
	}

	log.Printf("Checking health of %d proxies", len(proxies))

	// Create a semaphore to limit concurrent health checks
	maxConcurrent := utils.GetEnvAsInt("MAX_CONCURRENT_HEALTH_CHECKS", 10)
	semaphore := make(chan struct{}, maxConcurrent)

	// Check each proxy concurrently
	for _, proxy := range proxies {
		h.wg.Add(1)
		go func(p models.Proxy) {
			defer h.wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			h.checkProxyHealth(ctx, &p)
		}(proxy)
	}

	// Wait for all health checks to complete
	h.wg.Wait()
	log.Println("Health check cycle completed")
}

// checkProxyHealth checks the health of a single proxy
func (h *HealthService) checkProxyHealth(ctx context.Context, proxy *models.Proxy) {
	log.Printf("Checking health of proxy %s (%s:%d)", proxy.Name, proxy.Host, proxy.Port)

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	success := true
	var errorMsg string

	// Test proxy connection
	err := h.proxyService.testProxyConnection(checkCtx, proxy)
	duration := time.Since(start)

	if err != nil {
		success = false
		errorMsg = err.Error()
		log.Printf("Proxy %s health check failed: %v", proxy.Name, err)
	} else {
		log.Printf("Proxy %s health check passed (response time: %v)", proxy.Name, duration)
	}

	// Update proxy health status
	err = h.updateProxyHealthStatus(ctx, proxy.ID, success, int(duration.Milliseconds()), errorMsg)
	if err != nil {
		log.Printf("Failed to update health status for proxy %s: %v", proxy.Name, err)
	}

	// Update proxy status based on consecutive failures
	if !success {
		h.handleProxyFailure(ctx, proxy)
	} else {
		h.handleProxySuccess(ctx, proxy)
	}
}

// getActiveProxies retrieves all active proxies that need health checking
func (h *HealthService) getActiveProxies(ctx context.Context) ([]models.Proxy, error) {
	query := `
		SELECT id, uuid, name, type, host, port, username, password, status,
		       health_check_url, last_health_check, health_check_success,
		       response_time_ms, created_at, updated_at
		FROM proxies
		WHERE status = 'active'
		ORDER BY last_health_check ASC NULLS FIRST
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active proxies: %w", err)
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var proxy models.Proxy
		err := rows.Scan(
			&proxy.ID, &proxy.UUID, &proxy.Name, &proxy.Type, &proxy.Host,
			&proxy.Port, &proxy.Username, &proxy.Password, &proxy.Status,
			&proxy.HealthCheckURL, &proxy.LastHealthCheck, &proxy.HealthCheckSuccess,
			&proxy.ResponseTimeMs, &proxy.CreatedAt, &proxy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy: %w", err)
		}
		proxies = append(proxies, proxy)
	}

	return proxies, nil
}

// updateProxyHealthStatus updates the health status of a proxy
func (h *HealthService) updateProxyHealthStatus(ctx context.Context, proxyID int, success bool, responseTimeMs int, errorMsg string) error {
	query := `
		UPDATE proxies 
		SET health_check_success = $1, 
		    response_time_ms = $2, 
		    last_health_check = NOW(), 
		    updated_at = NOW()
		WHERE id = $3
	`

	_, err := h.db.ExecContext(ctx, query, success, responseTimeMs, proxyID)
	if err != nil {
		return fmt.Errorf("failed to update proxy health status: %w", err)
	}

	// Store health check result in Redis for metrics
	healthKey := fmt.Sprintf("proxy_health:%d", proxyID)
	healthData := map[string]interface{}{
		"success":         success,
		"response_time":   responseTimeMs,
		"timestamp":       time.Now().Unix(),
		"error":          errorMsg,
	}

	err = h.rdb.HMSet(ctx, healthKey, healthData).Err()
	if err != nil {
		log.Printf("Failed to store health check result in Redis: %v", err)
	}

	// Set expiration for health data (keep for 24 hours)
	h.rdb.Expire(ctx, healthKey, 24*time.Hour)

	return nil
}

// handleProxyFailure handles consecutive proxy failures
func (h *HealthService) handleProxyFailure(ctx context.Context, proxy *models.Proxy) {
	// Get consecutive failure count from Redis
	failureKey := fmt.Sprintf("proxy_failures:%d", proxy.ID)
	failures, err := h.rdb.Incr(ctx, failureKey).Result()
	if err != nil {
		log.Printf("Failed to increment failure count for proxy %s: %v", proxy.Name, err)
		return
	}

	// Set expiration for failure counter (reset after 1 hour of no failures)
	h.rdb.Expire(ctx, failureKey, time.Hour)

	maxFailures := utils.GetEnvAsInt("MAX_PROXY_FAILURES", 3)
	if failures >= int64(maxFailures) {
		log.Printf("Proxy %s has %d consecutive failures, marking as error", proxy.Name, failures)
		
		// Update proxy status to error
		err = h.updateProxyStatus(ctx, proxy.ID, models.ProxyStatusError)
		if err != nil {
			log.Printf("Failed to update proxy status to error: %v", err)
		}

		// Reset failure counter
		h.rdb.Del(ctx, failureKey)

		// Notify about proxy failure (could send to monitoring system)
		h.notifyProxyFailure(ctx, proxy, int(failures))
	}
}

// handleProxySuccess handles successful proxy health check
func (h *HealthService) handleProxySuccess(ctx context.Context, proxy *models.Proxy) {
	// Reset failure counter
	failureKey := fmt.Sprintf("proxy_failures:%d", proxy.ID)
	h.rdb.Del(ctx, failureKey)

	// If proxy was in error state, restore it to active
	if proxy.Status == models.ProxyStatusError {
		log.Printf("Proxy %s recovered, marking as active", proxy.Name)
		err := h.updateProxyStatus(ctx, proxy.ID, models.ProxyStatusActive)
		if err != nil {
			log.Printf("Failed to update proxy status to active: %v", err)
		}
	}
}

// updateProxyStatus updates the status of a proxy
func (h *HealthService) updateProxyStatus(ctx context.Context, proxyID int, status models.ProxyStatus) error {
	query := "UPDATE proxies SET status = $1, updated_at = NOW() WHERE id = $2"
	_, err := h.db.ExecContext(ctx, query, status, proxyID)
	return err
}

// notifyProxyFailure sends notification about proxy failure
func (h *HealthService) notifyProxyFailure(ctx context.Context, proxy *models.Proxy, failures int) {
	// This could send notifications to Slack, email, or monitoring systems
	log.Printf("ALERT: Proxy %s (%s:%d) has failed %d times and is now marked as error", 
		proxy.Name, proxy.Host, proxy.Port, failures)

	// Store alert in Redis for dashboard
	alertKey := fmt.Sprintf("proxy_alert:%d:%d", proxy.ID, time.Now().Unix())
	alertData := map[string]interface{}{
		"proxy_id":     proxy.ID,
		"proxy_name":   proxy.Name,
		"proxy_host":   proxy.Host,
		"proxy_port":   proxy.Port,
		"failure_count": failures,
		"timestamp":    time.Now().Unix(),
		"type":         "proxy_failure",
	}

	h.rdb.HMSet(ctx, alertKey, alertData)
	h.rdb.Expire(ctx, alertKey, 7*24*time.Hour) // Keep alerts for 7 days
}

// GetHealthMetrics returns health metrics for monitoring
func (h *HealthService) GetHealthMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Get overall health statistics
	healthQuery := `
		SELECT 
			COUNT(*) as total_proxies,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_proxies,
			COUNT(CASE WHEN status = 'active' AND health_check_success = true THEN 1 END) as healthy_proxies,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as error_proxies,
			AVG(CASE WHEN status = 'active' THEN response_time_ms END) as avg_response_time
		FROM proxies
	`

	var totalProxies, activeProxies, healthyProxies, errorProxies int
	var avgResponseTime sql.NullFloat64

	err := h.db.QueryRowContext(ctx, healthQuery).Scan(
		&totalProxies, &activeProxies, &healthyProxies, &errorProxies, &avgResponseTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get health metrics: %w", err)
	}

	metrics["total_proxies"] = totalProxies
	metrics["active_proxies"] = activeProxies
	metrics["healthy_proxies"] = healthyProxies
	metrics["error_proxies"] = errorProxies
	metrics["health_rate"] = 0.0

	if activeProxies > 0 {
		metrics["health_rate"] = float64(healthyProxies) / float64(activeProxies) * 100
	}

	if avgResponseTime.Valid {
		metrics["avg_response_time_ms"] = avgResponseTime.Float64
	} else {
		metrics["avg_response_time_ms"] = 0.0
	}

	return metrics, nil
}
