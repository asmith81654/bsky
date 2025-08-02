package main

import (
	"time"

	"github.com/bsky-automation/shared/models"
)

// UpdateProxyRequest represents a request to update a proxy
type UpdateProxyRequest struct {
	Name           *string              `json:"name,omitempty"`
	Host           *string              `json:"host,omitempty"`
	Port           *int                 `json:"port,omitempty"`
	Username       *string              `json:"username,omitempty"`
	Password       *string              `json:"password,omitempty"`
	Status         *models.ProxyStatus  `json:"status,omitempty"`
	HealthCheckURL *string              `json:"health_check_url,omitempty"`
}

// ProxyTestResult represents the result of testing a proxy
type ProxyTestResult struct {
	ProxyID      int           `json:"proxy_id"`
	Success      bool          `json:"success"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// ProxyAssignmentRequest represents a request to assign a proxy
type ProxyAssignmentRequest struct {
	AccountID int                `json:"account_id" validate:"required"`
	ProxyID   *int               `json:"proxy_id,omitempty"`
	ProxyType *models.ProxyType  `json:"proxy_type,omitempty"`
	Strategy  string             `json:"strategy,omitempty"` // auto, manual, round_robin, least_used
}

// ProxyAssignmentResponse represents the result of proxy assignment
type ProxyAssignmentResponse struct {
	AccountID int           `json:"account_id"`
	ProxyID   int           `json:"proxy_id"`
	ProxyName string        `json:"proxy_name"`
	ProxyHost string        `json:"proxy_host"`
	ProxyPort int           `json:"proxy_port"`
	AssignedAt time.Time    `json:"assigned_at"`
}

// ProxyReleaseRequest represents a request to release a proxy
type ProxyReleaseRequest struct {
	AccountID int `json:"account_id" validate:"required"`
}

// ProxyUsageResponse represents proxy usage statistics
type ProxyUsageResponse struct {
	TotalProxies     int                    `json:"total_proxies"`
	ActiveProxies    int                    `json:"active_proxies"`
	AssignedProxies  int                    `json:"assigned_proxies"`
	AvailableProxies int                    `json:"available_proxies"`
	UsageByProxy     []ProxyUsageDetail     `json:"usage_by_proxy"`
	UsageByType      map[models.ProxyType]int `json:"usage_by_type"`
}

// ProxyUsageDetail represents usage details for a specific proxy
type ProxyUsageDetail struct {
	ProxyID      int       `json:"proxy_id"`
	ProxyName    string    `json:"proxy_name"`
	ProxyHost    string    `json:"proxy_host"`
	ProxyPort    int       `json:"proxy_port"`
	ProxyType    string    `json:"proxy_type"`
	AccountCount int       `json:"account_count"`
	LastUsed     *time.Time `json:"last_used,omitempty"`
}

// ProxyStatsResponse represents overall proxy statistics
type ProxyStatsResponse struct {
	TotalProxies      int                        `json:"total_proxies"`
	ActiveProxies     int                        `json:"active_proxies"`
	InactiveProxies   int                        `json:"inactive_proxies"`
	ErrorProxies      int                        `json:"error_proxies"`
	StatusBreakdown   map[models.ProxyStatus]int `json:"status_breakdown"`
	TypeBreakdown     map[models.ProxyType]int   `json:"type_breakdown"`
	HealthyProxies    int                        `json:"healthy_proxies"`
	UnhealthyProxies  int                        `json:"unhealthy_proxies"`
	AverageResponseTime float64                  `json:"average_response_time_ms"`
	LastHealthCheck   *time.Time                 `json:"last_health_check"`
}

// ProxyHealthStatsResponse represents proxy health statistics
type ProxyHealthStatsResponse struct {
	TotalProxies       int                    `json:"total_proxies"`
	HealthyProxies     int                    `json:"healthy_proxies"`
	UnhealthyProxies   int                    `json:"unhealthy_proxies"`
	HealthChecksPassed int                    `json:"health_checks_passed"`
	HealthChecksFailed int                    `json:"health_checks_failed"`
	HealthRate         float64                `json:"health_rate"`
	ProxyHealthDetails []ProxyHealthDetail    `json:"proxy_health_details"`
	HealthByType       map[models.ProxyType]ProxyTypeHealth `json:"health_by_type"`
}

// ProxyHealthDetail represents health details for a specific proxy
type ProxyHealthDetail struct {
	ProxyID            int       `json:"proxy_id"`
	ProxyName          string    `json:"proxy_name"`
	ProxyHost          string    `json:"proxy_host"`
	ProxyPort          int       `json:"proxy_port"`
	ProxyType          string    `json:"proxy_type"`
	IsHealthy          bool      `json:"is_healthy"`
	LastHealthCheck    *time.Time `json:"last_health_check"`
	ResponseTimeMs     int       `json:"response_time_ms"`
	ConsecutiveFailures int      `json:"consecutive_failures"`
}

// ProxyTypeHealth represents health statistics for a proxy type
type ProxyTypeHealth struct {
	TotalProxies     int     `json:"total_proxies"`
	HealthyProxies   int     `json:"healthy_proxies"`
	UnhealthyProxies int     `json:"unhealthy_proxies"`
	HealthRate       float64 `json:"health_rate"`
	AvgResponseTime  float64 `json:"avg_response_time_ms"`
}

// ProxyPerformanceStatsResponse represents proxy performance statistics
type ProxyPerformanceStatsResponse struct {
	TimeRange              string                     `json:"time_range"`
	TotalRequests          int                        `json:"total_requests"`
	SuccessfulRequests     int                        `json:"successful_requests"`
	FailedRequests         int                        `json:"failed_requests"`
	SuccessRate            float64                    `json:"success_rate"`
	AverageResponseTime    float64                    `json:"average_response_time_ms"`
	MedianResponseTime     float64                    `json:"median_response_time_ms"`
	P95ResponseTime        float64                    `json:"p95_response_time_ms"`
	P99ResponseTime        float64                    `json:"p99_response_time_ms"`
	ProxyPerformanceDetails []ProxyPerformanceDetail  `json:"proxy_performance_details"`
	DailyStats             []DailyPerformanceStats    `json:"daily_stats"`
}

// ProxyPerformanceDetail represents performance details for a specific proxy
type ProxyPerformanceDetail struct {
	ProxyID             int     `json:"proxy_id"`
	ProxyName           string  `json:"proxy_name"`
	ProxyHost           string  `json:"proxy_host"`
	ProxyPort           int     `json:"proxy_port"`
	ProxyType           string  `json:"proxy_type"`
	TotalRequests       int     `json:"total_requests"`
	SuccessfulRequests  int     `json:"successful_requests"`
	FailedRequests      int     `json:"failed_requests"`
	SuccessRate         float64 `json:"success_rate"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	MinResponseTime     float64 `json:"min_response_time_ms"`
	MaxResponseTime     float64 `json:"max_response_time_ms"`
}

// DailyPerformanceStats represents daily performance statistics
type DailyPerformanceStats struct {
	Date                string  `json:"date"`
	TotalRequests       int     `json:"total_requests"`
	SuccessfulRequests  int     `json:"successful_requests"`
	FailedRequests      int     `json:"failed_requests"`
	SuccessRate         float64 `json:"success_rate"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
}

// ProxyAssignmentStrategy represents different proxy assignment strategies
type ProxyAssignmentStrategy string

const (
	AssignmentStrategyAuto       ProxyAssignmentStrategy = "auto"
	AssignmentStrategyManual     ProxyAssignmentStrategy = "manual"
	AssignmentStrategyRoundRobin ProxyAssignmentStrategy = "round_robin"
	AssignmentStrategyLeastUsed  ProxyAssignmentStrategy = "least_used"
	AssignmentStrategyFastest    ProxyAssignmentStrategy = "fastest"
)

// ProxyMetric represents a proxy performance metric
type ProxyMetric struct {
	ProxyID     int       `json:"proxy_id"`
	MetricType  string    `json:"metric_type"`
	MetricValue float64   `json:"metric_value"`
	Timestamp   time.Time `json:"timestamp"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled         bool          `json:"enabled"`
	Interval        time.Duration `json:"interval"`
	Timeout         time.Duration `json:"timeout"`
	MaxFailures     int           `json:"max_failures"`
	TestURL         string        `json:"test_url"`
	ExpectedStatus  int           `json:"expected_status"`
	FollowRedirects bool          `json:"follow_redirects"`
}

// ProxyPool represents a pool of proxies for load balancing
type ProxyPool struct {
	ID          int                     `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Strategy    ProxyAssignmentStrategy `json:"strategy"`
	ProxyIDs    []int                   `json:"proxy_ids"`
	IsActive    bool                    `json:"is_active"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}
