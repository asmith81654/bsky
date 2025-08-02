package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/bsky-automation/shared/models"
	"github.com/bsky-automation/shared/utils"
)

// ProxyService handles proxy-related business logic
type ProxyService struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewProxyService creates a new proxy service
func NewProxyService(db *sql.DB, rdb *redis.Client) *ProxyService {
	return &ProxyService{
		db:  db,
		rdb: rdb,
	}
}

// CreateProxy creates a new proxy
func (s *ProxyService) CreateProxy(ctx context.Context, req *models.CreateProxyRequest) (*models.Proxy, error) {
	// Validate proxy URL format
	proxyURL := fmt.Sprintf("%s://%s:%d", req.Type, req.Host, req.Port)
	if err := utils.ValidateProxyURL(proxyURL); err != nil {
		return nil, fmt.Errorf("invalid proxy configuration: %w", err)
	}

	// Check if proxy already exists
	exists, err := s.proxyExists(ctx, req.Host, req.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to check proxy existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("proxy with host %s and port %d already exists", req.Host, req.Port)
	}

	// Create proxy
	proxy := &models.Proxy{
		UUID:               utils.GenerateUUID(),
		Name:               req.Name,
		Type:               req.Type,
		Host:               req.Host,
		Port:               req.Port,
		Username:           req.Username,
		Password:           req.Password,
		Status:             models.ProxyStatusActive,
		HealthCheckURL:     req.HealthCheckURL,
		HealthCheckSuccess: true,
		ResponseTimeMs:     0,
	}

	// Insert into database
	query := `
		INSERT INTO proxies (uuid, name, type, host, port, username, password, status, health_check_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		proxy.UUID, proxy.Name, proxy.Type, proxy.Host, proxy.Port,
		proxy.Username, proxy.Password, proxy.Status, proxy.HealthCheckURL,
	).Scan(&proxy.ID, &proxy.CreatedAt, &proxy.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create proxy: %w", err)
	}

	// Test proxy connection
	if err := s.testProxyConnection(ctx, proxy); err != nil {
		// Log the error but don't fail the creation
		// Update proxy status to error
		proxy.Status = models.ProxyStatusError
		s.updateProxyStatus(ctx, proxy.ID, proxy.Status)
	}

	return proxy, nil
}

// GetProxy retrieves a proxy by ID
func (s *ProxyService) GetProxy(ctx context.Context, id int) (*models.Proxy, error) {
	query := `
		SELECT id, uuid, name, type, host, port, username, password, status,
		       health_check_url, last_health_check, health_check_success,
		       response_time_ms, created_at, updated_at
		FROM proxies
		WHERE id = $1
	`

	proxy := &models.Proxy{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&proxy.ID, &proxy.UUID, &proxy.Name, &proxy.Type, &proxy.Host,
		&proxy.Port, &proxy.Username, &proxy.Password, &proxy.Status,
		&proxy.HealthCheckURL, &proxy.LastHealthCheck, &proxy.HealthCheckSuccess,
		&proxy.ResponseTimeMs, &proxy.CreatedAt, &proxy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proxy not found")
		}
		return nil, fmt.Errorf("failed to get proxy: %w", err)
	}

	return proxy, nil
}

// ListProxies retrieves a paginated list of proxies
func (s *ProxyService) ListProxies(ctx context.Context, page, pageSize int, status *models.ProxyStatus, proxyType *models.ProxyType) (*models.ListResponse, error) {
	// Calculate pagination
	offset, limit, _ := utils.Paginate(page, pageSize, 0)

	// Build query
	baseQuery := `
		SELECT id, uuid, name, type, host, port, status, health_check_success,
		       response_time_ms, last_health_check, created_at
		FROM proxies
	`

	var args []interface{}
	var conditions []string

	if status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)+1))
		args = append(args, *status)
	}

	if proxyType != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)+1))
		args = append(args, *proxyType)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("%s", conditions[0])
		for i := 1; i < len(conditions); i++ {
			whereClause += " AND " + conditions[i]
		}
	}

	query := fmt.Sprintf("%s %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		baseQuery, whereClause, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxies: %w", err)
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var proxy models.Proxy
		err := rows.Scan(
			&proxy.ID, &proxy.UUID, &proxy.Name, &proxy.Type, &proxy.Host,
			&proxy.Port, &proxy.Status, &proxy.HealthCheckSuccess,
			&proxy.ResponseTimeMs, &proxy.LastHealthCheck, &proxy.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy: %w", err)
		}
		proxies = append(proxies, proxy)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM proxies"
	if whereClause != "" {
		countQuery += " " + whereClause
	}

	var totalItems int64
	countArgs := args[:len(args)-2] // Remove limit and offset
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalItems)
	if err != nil {
		return nil, fmt.Errorf("failed to count proxies: %w", err)
	}

	_, _, totalPages := utils.Paginate(page, pageSize, totalItems)

	return &models.ListResponse{
		Data: proxies,
		Pagination: models.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateProxy updates an existing proxy
func (s *ProxyService) UpdateProxy(ctx context.Context, id int, req *UpdateProxyRequest) (*models.Proxy, error) {
	// Get existing proxy
	proxy, err := s.GetProxy(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Host != nil {
		updates["host"] = *req.Host
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Password != nil {
		updates["password"] = *req.Password
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.HealthCheckURL != nil {
		updates["health_check_url"] = *req.HealthCheckURL
	}

	if len(updates) == 0 {
		return proxy, nil // No updates
	}

	updates["updated_at"] = time.Now()

	setClause, args := utils.BuildUpdateClause(updates)
	query := fmt.Sprintf("UPDATE proxies %s WHERE id = $%d", setClause, len(args)+1)
	args = append(args, id)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update proxy: %w", err)
	}

	// Return updated proxy
	return s.GetProxy(ctx, id)
}

// DeleteProxy deletes a proxy
func (s *ProxyService) DeleteProxy(ctx context.Context, id int) error {
	// Check if proxy exists
	_, err := s.GetProxy(ctx, id)
	if err != nil {
		return err
	}

	// Check if proxy is in use
	inUse, err := s.isProxyInUse(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check proxy usage: %w", err)
	}
	if inUse {
		return fmt.Errorf("cannot delete proxy: it is currently in use by accounts")
	}

	// Delete proxy
	query := "DELETE FROM proxies WHERE id = $1"
	_, err = s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete proxy: %w", err)
	}

	return nil
}

// TestProxy tests proxy connection
func (s *ProxyService) TestProxy(ctx context.Context, id int) (*ProxyTestResult, error) {
	proxy, err := s.GetProxy(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &ProxyTestResult{
		ProxyID:   id,
		Success:   false,
		Timestamp: time.Now(),
	}

	// Test proxy connection
	start := time.Now()
	err = s.testProxyConnection(ctx, proxy)
	duration := time.Since(start)

	result.ResponseTime = duration
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	// Update proxy health status
	s.updateProxyHealth(ctx, id, result.Success, int(duration.Milliseconds()))

	return result, nil
}

// GetAvailableProxies returns available proxies for assignment
func (s *ProxyService) GetAvailableProxies(ctx context.Context, proxyType *models.ProxyType) ([]models.Proxy, error) {
	query := `
		SELECT id, uuid, name, type, host, port, status, health_check_success,
		       response_time_ms, created_at
		FROM proxies
		WHERE status = 'active' AND health_check_success = true
	`

	var args []interface{}
	if proxyType != nil {
		query += " AND type = $1"
		args = append(args, *proxyType)
	}

	query += " ORDER BY response_time_ms ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get available proxies: %w", err)
	}
	defer rows.Close()

	var proxies []models.Proxy
	for rows.Next() {
		var proxy models.Proxy
		err := rows.Scan(
			&proxy.ID, &proxy.UUID, &proxy.Name, &proxy.Type, &proxy.Host,
			&proxy.Port, &proxy.Status, &proxy.HealthCheckSuccess,
			&proxy.ResponseTimeMs, &proxy.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy: %w", err)
		}
		proxies = append(proxies, proxy)
	}

	return proxies, nil
}

// Helper methods

func (s *ProxyService) proxyExists(ctx context.Context, host string, port int) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM proxies WHERE host = $1 AND port = $2)"
	var exists bool
	err := s.db.QueryRowContext(ctx, query, host, port).Scan(&exists)
	return exists, err
}

func (s *ProxyService) isProxyInUse(ctx context.Context, proxyID int) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM accounts WHERE proxy_id = $1)"
	var inUse bool
	err := s.db.QueryRowContext(ctx, query, proxyID).Scan(&inUse)
	return inUse, err
}

func (s *ProxyService) testProxyConnection(ctx context.Context, proxy *models.Proxy) error {
	// Create HTTP client with proxy
	proxyURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", proxy.Type, proxy.Host, proxy.Port))
	if err != nil {
		return fmt.Errorf("invalid proxy URL: %w", err)
	}

	if proxy.Username != nil && proxy.Password != nil {
		proxyURL.User = url.UserPassword(*proxy.Username, *proxy.Password)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Test URL - use health check URL if provided, otherwise use a default
	testURL := "https://httpbin.org/ip"
	if proxy.HealthCheckURL != nil {
		testURL = *proxy.HealthCheckURL
	}

	// Make test request
	resp, err := client.Get(testURL)
	if err != nil {
		return fmt.Errorf("proxy connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy returned status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *ProxyService) updateProxyStatus(ctx context.Context, id int, status models.ProxyStatus) error {
	query := "UPDATE proxies SET status = $1, updated_at = NOW() WHERE id = $2"
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

func (s *ProxyService) updateProxyHealth(ctx context.Context, id int, success bool, responseTimeMs int) error {
	query := `
		UPDATE proxies
		SET health_check_success = $1, response_time_ms = $2,
		    last_health_check = NOW(), updated_at = NOW()
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, success, responseTimeMs, id)
	return err
}

// AssignProxy assigns a proxy to an account
func (s *ProxyService) AssignProxy(ctx context.Context, req *ProxyAssignmentRequest) (*ProxyAssignmentResponse, error) {
	var proxyID int
	var proxy *models.Proxy
	var err error

	if req.ProxyID != nil {
		// Manual assignment
		proxy, err = s.GetProxy(ctx, *req.ProxyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get specified proxy: %w", err)
		}
		proxyID = *req.ProxyID
	} else {
		// Auto assignment based on strategy
		strategy := req.Strategy
		if strategy == "" {
			strategy = "auto"
		}

		proxyID, err = s.selectProxyByStrategy(ctx, strategy, req.ProxyType)
		if err != nil {
			return nil, fmt.Errorf("failed to select proxy: %w", err)
		}

		proxy, err = s.GetProxy(ctx, proxyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get selected proxy: %w", err)
		}
	}

	// Update account with proxy assignment
	updateQuery := "UPDATE accounts SET proxy_id = $1, updated_at = NOW() WHERE id = $2"
	_, err = s.db.ExecContext(ctx, updateQuery, proxyID, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign proxy to account: %w", err)
	}

	return &ProxyAssignmentResponse{
		AccountID:  req.AccountID,
		ProxyID:    proxyID,
		ProxyName:  proxy.Name,
		ProxyHost:  proxy.Host,
		ProxyPort:  proxy.Port,
		AssignedAt: time.Now(),
	}, nil
}

// ReleaseProxy releases a proxy from an account
func (s *ProxyService) ReleaseProxy(ctx context.Context, req *ProxyReleaseRequest) error {
	updateQuery := "UPDATE accounts SET proxy_id = NULL, updated_at = NOW() WHERE id = $1"
	_, err := s.db.ExecContext(ctx, updateQuery, req.AccountID)
	if err != nil {
		return fmt.Errorf("failed to release proxy from account: %w", err)
	}
	return nil
}

// GetProxyUsage returns proxy usage statistics
func (s *ProxyService) GetProxyUsage(ctx context.Context) (*ProxyUsageResponse, error) {
	usage := &ProxyUsageResponse{
		UsageByType: make(map[models.ProxyType]int),
	}

	// Get total proxy counts
	totalQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN id IN (SELECT DISTINCT proxy_id FROM accounts WHERE proxy_id IS NOT NULL) THEN 1 END) as assigned
		FROM proxies
	`
	err := s.db.QueryRowContext(ctx, totalQuery).Scan(&usage.TotalProxies, &usage.ActiveProxies, &usage.AssignedProxies)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy counts: %w", err)
	}

	usage.AvailableProxies = usage.ActiveProxies - usage.AssignedProxies

	// Get usage by proxy
	usageQuery := `
		SELECT
			p.id, p.name, p.host, p.port, p.type,
			COUNT(a.id) as account_count,
			MAX(a.last_activity) as last_used
		FROM proxies p
		LEFT JOIN accounts a ON p.id = a.proxy_id
		GROUP BY p.id, p.name, p.host, p.port, p.type
		ORDER BY account_count DESC
	`
	rows, err := s.db.QueryContext(ctx, usageQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy usage details: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var detail ProxyUsageDetail
		var lastUsed sql.NullTime
		err := rows.Scan(&detail.ProxyID, &detail.ProxyName, &detail.ProxyHost,
			&detail.ProxyPort, &detail.ProxyType, &detail.AccountCount, &lastUsed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy usage detail: %w", err)
		}
		if lastUsed.Valid {
			detail.LastUsed = &lastUsed.Time
		}
		usage.UsageByProxy = append(usage.UsageByProxy, detail)
	}

	// Get usage by type
	typeQuery := `
		SELECT p.type, COUNT(a.id) as account_count
		FROM proxies p
		LEFT JOIN accounts a ON p.id = a.proxy_id
		GROUP BY p.type
	`
	rows, err = s.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var proxyType models.ProxyType
		var count int
		err := rows.Scan(&proxyType, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan type usage: %w", err)
		}
		usage.UsageByType[proxyType] = count
	}

	return usage, nil
}

// GetProxyStats returns overall proxy statistics
func (s *ProxyService) GetProxyStats(ctx context.Context) (*ProxyStatsResponse, error) {
	stats := &ProxyStatsResponse{
		StatusBreakdown: make(map[models.ProxyStatus]int),
		TypeBreakdown:   make(map[models.ProxyType]int),
	}

	// Get status breakdown
	statusQuery := `
		SELECT status, COUNT(*)
		FROM proxies
		GROUP BY status
	`
	rows, err := s.db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status models.ProxyStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		stats.StatusBreakdown[status] = count
		stats.TotalProxies += count

		switch status {
		case models.ProxyStatusActive:
			stats.ActiveProxies = count
		case models.ProxyStatusInactive:
			stats.InactiveProxies = count
		case models.ProxyStatusError:
			stats.ErrorProxies = count
		}
	}

	// Get type breakdown
	typeQuery := `
		SELECT type, COUNT(*)
		FROM proxies
		GROUP BY type
	`
	rows, err = s.db.QueryContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get type breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var proxyType models.ProxyType
		var count int
		if err := rows.Scan(&proxyType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type row: %w", err)
		}
		stats.TypeBreakdown[proxyType] = count
	}

	// Get health statistics
	healthQuery := `
		SELECT
			COUNT(CASE WHEN health_check_success = true THEN 1 END) as healthy,
			COUNT(CASE WHEN health_check_success = false THEN 1 END) as unhealthy,
			AVG(response_time_ms) as avg_response_time,
			MAX(last_health_check) as last_check
		FROM proxies
		WHERE status = 'active'
	`
	var lastCheck sql.NullTime
	err = s.db.QueryRowContext(ctx, healthQuery).Scan(
		&stats.HealthyProxies, &stats.UnhealthyProxies,
		&stats.AverageResponseTime, &lastCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to get health statistics: %w", err)
	}

	if lastCheck.Valid {
		stats.LastHealthCheck = &lastCheck.Time
	}

	return stats, nil
}

// GetHealthStats returns proxy health statistics
func (s *ProxyService) GetHealthStats(ctx context.Context) (*ProxyHealthStatsResponse, error) {
	stats := &ProxyHealthStatsResponse{
		HealthByType: make(map[models.ProxyType]ProxyTypeHealth),
	}

	// Get overall health statistics
	overallQuery := `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN health_check_success = true THEN 1 END) as healthy,
			COUNT(CASE WHEN health_check_success = false THEN 1 END) as unhealthy
		FROM proxies
		WHERE status = 'active'
	`
	err := s.db.QueryRowContext(ctx, overallQuery).Scan(
		&stats.TotalProxies, &stats.HealthyProxies, &stats.UnhealthyProxies)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall health stats: %w", err)
	}

	if stats.TotalProxies > 0 {
		stats.HealthRate = float64(stats.HealthyProxies) / float64(stats.TotalProxies) * 100
	}

	// Get proxy health details
	detailQuery := `
		SELECT id, name, host, port, type, health_check_success,
		       last_health_check, response_time_ms
		FROM proxies
		WHERE status = 'active'
		ORDER BY health_check_success DESC, response_time_ms ASC
	`
	rows, err := s.db.QueryContext(ctx, detailQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy health details: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var detail ProxyHealthDetail
		var lastCheck sql.NullTime
		err := rows.Scan(&detail.ProxyID, &detail.ProxyName, &detail.ProxyHost,
			&detail.ProxyPort, &detail.ProxyType, &detail.IsHealthy,
			&lastCheck, &detail.ResponseTimeMs)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health detail: %w", err)
		}
		if lastCheck.Valid {
			detail.LastHealthCheck = &lastCheck.Time
		}
		stats.ProxyHealthDetails = append(stats.ProxyHealthDetails, detail)
	}

	// Get health by type
	typeHealthQuery := `
		SELECT
			type,
			COUNT(*) as total,
			COUNT(CASE WHEN health_check_success = true THEN 1 END) as healthy,
			COUNT(CASE WHEN health_check_success = false THEN 1 END) as unhealthy,
			AVG(response_time_ms) as avg_response_time
		FROM proxies
		WHERE status = 'active'
		GROUP BY type
	`
	rows, err = s.db.QueryContext(ctx, typeHealthQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get health by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var proxyType models.ProxyType
		var typeHealth ProxyTypeHealth
		err := rows.Scan(&proxyType, &typeHealth.TotalProxies,
			&typeHealth.HealthyProxies, &typeHealth.UnhealthyProxies,
			&typeHealth.AvgResponseTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan type health: %w", err)
		}
		if typeHealth.TotalProxies > 0 {
			typeHealth.HealthRate = float64(typeHealth.HealthyProxies) / float64(typeHealth.TotalProxies) * 100
		}
		stats.HealthByType[proxyType] = typeHealth
	}

	return stats, nil
}

// GetPerformanceStats returns proxy performance statistics
func (s *ProxyService) GetPerformanceStats(ctx context.Context, days int) (*ProxyPerformanceStatsResponse, error) {
	stats := &ProxyPerformanceStatsResponse{
		TimeRange: fmt.Sprintf("Last %d days", days),
	}

	// For now, return mock data since we don't have detailed request logs
	// In a real implementation, you would query request/response logs
	stats.TotalRequests = 1000
	stats.SuccessfulRequests = 950
	stats.FailedRequests = 50
	stats.SuccessRate = utils.CalculateSuccessRate(stats.SuccessfulRequests, stats.TotalRequests)
	stats.AverageResponseTime = 250.5
	stats.MedianResponseTime = 200.0
	stats.P95ResponseTime = 500.0
	stats.P99ResponseTime = 800.0

	// Get proxy performance details from current health data
	detailQuery := `
		SELECT id, name, host, port, type, response_time_ms, health_check_success
		FROM proxies
		WHERE status = 'active'
		ORDER BY response_time_ms ASC
	`
	rows, err := s.db.QueryContext(ctx, detailQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy performance details: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var detail ProxyPerformanceDetail
		var isHealthy bool
		err := rows.Scan(&detail.ProxyID, &detail.ProxyName, &detail.ProxyHost,
			&detail.ProxyPort, &detail.ProxyType, &detail.AverageResponseTime, &isHealthy)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performance detail: %w", err)
		}

		// Mock data for demonstration
		detail.TotalRequests = utils.RandomInt(50, 200)
		if isHealthy {
			detail.SuccessfulRequests = int(float64(detail.TotalRequests) * 0.95)
		} else {
			detail.SuccessfulRequests = int(float64(detail.TotalRequests) * 0.7)
		}
		detail.FailedRequests = detail.TotalRequests - detail.SuccessfulRequests
		detail.SuccessRate = utils.CalculateSuccessRate(detail.SuccessfulRequests, detail.TotalRequests)
		detail.MinResponseTime = detail.AverageResponseTime * 0.5
		detail.MaxResponseTime = detail.AverageResponseTime * 2.0

		stats.ProxyPerformanceDetails = append(stats.ProxyPerformanceDetails, detail)
	}

	// Generate daily stats for the past week
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		dailyStat := DailyPerformanceStats{
			Date:                date,
			TotalRequests:       utils.RandomInt(100, 200),
			SuccessfulRequests:  0,
			FailedRequests:      0,
			AverageResponseTime: float64(utils.RandomInt(200, 400)),
		}
		dailyStat.SuccessfulRequests = int(float64(dailyStat.TotalRequests) * 0.9)
		dailyStat.FailedRequests = dailyStat.TotalRequests - dailyStat.SuccessfulRequests
		dailyStat.SuccessRate = utils.CalculateSuccessRate(dailyStat.SuccessfulRequests, dailyStat.TotalRequests)

		stats.DailyStats = append(stats.DailyStats, dailyStat)
	}

	return stats, nil
}

// selectProxyByStrategy selects a proxy based on the given strategy
func (s *ProxyService) selectProxyByStrategy(ctx context.Context, strategy string, proxyType *models.ProxyType) (int, error) {
	switch strategy {
	case "least_used":
		return s.selectLeastUsedProxy(ctx, proxyType)
	case "fastest":
		return s.selectFastestProxy(ctx, proxyType)
	case "round_robin":
		return s.selectRoundRobinProxy(ctx, proxyType)
	default: // "auto"
		return s.selectBestProxy(ctx, proxyType)
	}
}

// selectLeastUsedProxy selects the proxy with the least number of assigned accounts
func (s *ProxyService) selectLeastUsedProxy(ctx context.Context, proxyType *models.ProxyType) (int, error) {
	query := `
		SELECT p.id
		FROM proxies p
		LEFT JOIN accounts a ON p.id = a.proxy_id
		WHERE p.status = 'active' AND p.health_check_success = true
	`

	var args []interface{}
	if proxyType != nil {
		query += " AND p.type = $1"
		args = append(args, *proxyType)
	}

	query += `
		GROUP BY p.id
		ORDER BY COUNT(a.id) ASC, p.response_time_ms ASC
		LIMIT 1
	`

	var proxyID int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&proxyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no available proxies found")
		}
		return 0, fmt.Errorf("failed to select least used proxy: %w", err)
	}

	return proxyID, nil
}

// selectFastestProxy selects the proxy with the best response time
func (s *ProxyService) selectFastestProxy(ctx context.Context, proxyType *models.ProxyType) (int, error) {
	query := `
		SELECT id
		FROM proxies
		WHERE status = 'active' AND health_check_success = true
	`

	var args []interface{}
	if proxyType != nil {
		query += " AND type = $1"
		args = append(args, *proxyType)
	}

	query += " ORDER BY response_time_ms ASC LIMIT 1"

	var proxyID int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&proxyID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no available proxies found")
		}
		return 0, fmt.Errorf("failed to select fastest proxy: %w", err)
	}

	return proxyID, nil
}

// selectRoundRobinProxy selects proxy using round-robin algorithm
func (s *ProxyService) selectRoundRobinProxy(ctx context.Context, proxyType *models.ProxyType) (int, error) {
	// For simplicity, use Redis to store round-robin state
	key := "proxy_round_robin"
	if proxyType != nil {
		key += ":" + string(*proxyType)
	}

	// Get current index
	currentIndex, err := s.rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to get round-robin index: %w", err)
	}

	// Get available proxies
	proxies, err := s.GetAvailableProxies(ctx, proxyType)
	if err != nil {
		return 0, err
	}

	if len(proxies) == 0 {
		return 0, fmt.Errorf("no available proxies found")
	}

	// Select proxy by index
	selectedProxy := proxies[currentIndex%len(proxies)]

	// Update index for next selection
	nextIndex := (currentIndex + 1) % len(proxies)
	s.rdb.Set(ctx, key, nextIndex, 0)

	return selectedProxy.ID, nil
}

// selectBestProxy selects the best proxy based on multiple factors
func (s *ProxyService) selectBestProxy(ctx context.Context, proxyType *models.ProxyType) (int, error) {
	// Combine least used and fastest strategies
	query := `
		SELECT p.id, COUNT(a.id) as usage_count, p.response_time_ms
		FROM proxies p
		LEFT JOIN accounts a ON p.id = a.proxy_id
		WHERE p.status = 'active' AND p.health_check_success = true
	`

	var args []interface{}
	if proxyType != nil {
		query += " AND p.type = $1"
		args = append(args, *proxyType)
	}

	query += `
		GROUP BY p.id, p.response_time_ms
		ORDER BY (COUNT(a.id) * 100 + p.response_time_ms) ASC
		LIMIT 1
	`

	var proxyID int
	var usageCount int
	var responseTime int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&proxyID, &usageCount, &responseTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no available proxies found")
		}
		return 0, fmt.Errorf("failed to select best proxy: %w", err)
	}

	return proxyID, nil
}
