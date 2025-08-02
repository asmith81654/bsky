package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JSONB represents a PostgreSQL JSONB field
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}

	return json.Unmarshal(bytes, j)
}

// Account status enumeration
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusInactive  AccountStatus = "inactive"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusError     AccountStatus = "error"
)

// Proxy type enumeration
type ProxyType string

const (
	ProxyTypeHTTP   ProxyType = "http"
	ProxyTypeSOCKS5 ProxyType = "socks5"
)

// Proxy status enumeration
type ProxyStatus string

const (
	ProxyStatusActive   ProxyStatus = "active"
	ProxyStatusInactive ProxyStatus = "inactive"
	ProxyStatusError    ProxyStatus = "error"
)

// Strategy type enumeration
type StrategyType string

const (
	StrategyTypePost    StrategyType = "post"
	StrategyTypeFollow  StrategyType = "follow"
	StrategyTypeLike    StrategyType = "like"
	StrategyTypeRepost  StrategyType = "repost"
	StrategyTypeMonitor StrategyType = "monitor"
	StrategyTypeGrowth  StrategyType = "growth"
)

// Strategy status enumeration
type StrategyStatus string

const (
	StrategyStatusActive   StrategyStatus = "active"
	StrategyStatusInactive StrategyStatus = "inactive"
	StrategyStatusPaused   StrategyStatus = "paused"
)

// Task status enumeration
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Proxy represents a proxy server configuration
type Proxy struct {
	ID                   int         `json:"id" db:"id"`
	UUID                 uuid.UUID   `json:"uuid" db:"uuid"`
	Name                 string      `json:"name" db:"name"`
	Type                 ProxyType   `json:"type" db:"type"`
	Host                 string      `json:"host" db:"host"`
	Port                 int         `json:"port" db:"port"`
	Username             *string     `json:"username,omitempty" db:"username"`
	Password             *string     `json:"password,omitempty" db:"password"`
	Status               ProxyStatus `json:"status" db:"status"`
	HealthCheckURL       *string     `json:"health_check_url,omitempty" db:"health_check_url"`
	LastHealthCheck      *time.Time  `json:"last_health_check,omitempty" db:"last_health_check"`
	HealthCheckSuccess   bool        `json:"health_check_success" db:"health_check_success"`
	ResponseTimeMs       int         `json:"response_time_ms" db:"response_time_ms"`
	CreatedAt            time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at" db:"updated_at"`
}

// Account represents a Bluesky account
type Account struct {
	ID           int           `json:"id" db:"id"`
	UUID         uuid.UUID     `json:"uuid" db:"uuid"`
	Handle       string        `json:"handle" db:"handle"`
	Password     string        `json:"password" db:"password"`
	Host         string        `json:"host" db:"host"`
	BGS          string        `json:"bgs" db:"bgs"`
	Status       AccountStatus `json:"status" db:"status"`
	ProxyID      *int          `json:"proxy_id,omitempty" db:"proxy_id"`
	DID          *string       `json:"did,omitempty" db:"did"`
	AccessJWT    *string       `json:"access_jwt,omitempty" db:"access_jwt"`
	RefreshJWT   *string       `json:"refresh_jwt,omitempty" db:"refresh_jwt"`
	LastLogin    *time.Time    `json:"last_login,omitempty" db:"last_login"`
	LastActivity *time.Time    `json:"last_activity,omitempty" db:"last_activity"`
	ErrorCount   int           `json:"error_count" db:"error_count"`
	ErrorMessage *string       `json:"error_message,omitempty" db:"error_message"`
	Metadata     JSONB         `json:"metadata" db:"metadata"`
	CreatedAt    time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`

	// Joined fields
	Proxy *Proxy `json:"proxy,omitempty"`
}

// Strategy represents an automation strategy
type Strategy struct {
	ID                  int            `json:"id" db:"id"`
	UUID                uuid.UUID      `json:"uuid" db:"uuid"`
	Name                string         `json:"name" db:"name"`
	Description         *string        `json:"description,omitempty" db:"description"`
	Type                StrategyType   `json:"type" db:"type"`
	Config              JSONB          `json:"config" db:"config"`
	Schedule            *string        `json:"schedule,omitempty" db:"schedule"`
	Status              StrategyStatus `json:"status" db:"status"`
	Priority            int            `json:"priority" db:"priority"`
	MaxConcurrentTasks  int            `json:"max_concurrent_tasks" db:"max_concurrent_tasks"`
	RetryCount          int            `json:"retry_count" db:"retry_count"`
	TimeoutSeconds      int            `json:"timeout_seconds" db:"timeout_seconds"`
	CreatedBy           *string        `json:"created_by,omitempty" db:"created_by"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// AccountStrategy represents the association between an account and a strategy
type AccountStrategy struct {
	ID             int            `json:"id" db:"id"`
	UUID           uuid.UUID      `json:"uuid" db:"uuid"`
	AccountID      int            `json:"account_id" db:"account_id"`
	StrategyID     int            `json:"strategy_id" db:"strategy_id"`
	Config         JSONB          `json:"config" db:"config"`
	Status         StrategyStatus `json:"status" db:"status"`
	LastExecuted   *time.Time     `json:"last_executed,omitempty" db:"last_executed"`
	NextExecution  *time.Time     `json:"next_execution,omitempty" db:"next_execution"`
	ExecutionCount int            `json:"execution_count" db:"execution_count"`
	SuccessCount   int            `json:"success_count" db:"success_count"`
	ErrorCount     int            `json:"error_count" db:"error_count"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`

	// Joined fields
	Account  *Account  `json:"account,omitempty"`
	Strategy *Strategy `json:"strategy,omitempty"`
}

// Task represents a task to be executed
type Task struct {
	ID                 int        `json:"id" db:"id"`
	UUID               uuid.UUID  `json:"uuid" db:"uuid"`
	AccountID          int        `json:"account_id" db:"account_id"`
	StrategyID         int        `json:"strategy_id" db:"strategy_id"`
	AccountStrategyID  int        `json:"account_strategy_id" db:"account_strategy_id"`
	Type               string     `json:"type" db:"type"`
	Payload            JSONB      `json:"payload" db:"payload"`
	Status             TaskStatus `json:"status" db:"status"`
	Priority           int        `json:"priority" db:"priority"`
	RetryCount         int        `json:"retry_count" db:"retry_count"`
	MaxRetries         int        `json:"max_retries" db:"max_retries"`
	TimeoutSeconds     int        `json:"timeout_seconds" db:"timeout_seconds"`
	ScheduledAt        time.Time  `json:"scheduled_at" db:"scheduled_at"`
	StartedAt          *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	WorkerID           *string    `json:"worker_id,omitempty" db:"worker_id"`
	ErrorMessage       *string    `json:"error_message,omitempty" db:"error_message"`
	Result             JSONB      `json:"result" db:"result"`
	ExecutionTimeMs    *int       `json:"execution_time_ms,omitempty" db:"execution_time_ms"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`

	// Joined fields
	Account  *Account  `json:"account,omitempty"`
	Strategy *Strategy `json:"strategy,omitempty"`
}

// TaskDependency represents dependencies between tasks
type TaskDependency struct {
	ID              int `json:"id" db:"id"`
	TaskID          int `json:"task_id" db:"task_id"`
	DependsOnTaskID int `json:"depends_on_task_id" db:"depends_on_task_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Metric represents a performance metric
type Metric struct {
	ID         int       `json:"id" db:"id"`
	UUID       uuid.UUID `json:"uuid" db:"uuid"`
	AccountID  *int      `json:"account_id,omitempty" db:"account_id"`
	StrategyID *int      `json:"strategy_id,omitempty" db:"strategy_id"`
	MetricType string    `json:"metric_type" db:"metric_type"`
	MetricName string    `json:"metric_name" db:"metric_name"`
	MetricValue *float64 `json:"metric_value,omitempty" db:"metric_value"`
	MetricData JSONB     `json:"metric_data" db:"metric_data"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         int       `json:"id" db:"id"`
	UUID       uuid.UUID `json:"uuid" db:"uuid"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   int       `json:"entity_id" db:"entity_id"`
	Action     string    `json:"action" db:"action"`
	OldValues  JSONB     `json:"old_values,omitempty" db:"old_values"`
	NewValues  JSONB     `json:"new_values,omitempty" db:"new_values"`
	UserID     *string   `json:"user_id,omitempty" db:"user_id"`
	IPAddress  *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  *string   `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// SystemSetting represents a system configuration setting
type SystemSetting struct {
	ID          int       `json:"id" db:"id"`
	Key         string    `json:"key" db:"key"`
	Value       *string   `json:"value,omitempty" db:"value"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// API Request/Response models

// CreateAccountRequest represents a request to create an account
type CreateAccountRequest struct {
	Handle   string `json:"handle" validate:"required"`
	Password string `json:"password" validate:"required"`
	Host     string `json:"host,omitempty"`
	BGS      string `json:"bgs,omitempty"`
	ProxyID  *int   `json:"proxy_id,omitempty"`
}

// UpdateAccountRequest represents a request to update an account
type UpdateAccountRequest struct {
	Password *string       `json:"password,omitempty"`
	Host     *string       `json:"host,omitempty"`
	BGS      *string       `json:"bgs,omitempty"`
	Status   *AccountStatus `json:"status,omitempty"`
	ProxyID  *int          `json:"proxy_id,omitempty"`
}

// CreateProxyRequest represents a request to create a proxy
type CreateProxyRequest struct {
	Name           string     `json:"name" validate:"required"`
	Type           ProxyType  `json:"type" validate:"required"`
	Host           string     `json:"host" validate:"required"`
	Port           int        `json:"port" validate:"required,min=1,max=65535"`
	Username       *string    `json:"username,omitempty"`
	Password       *string    `json:"password,omitempty"`
	HealthCheckURL *string    `json:"health_check_url,omitempty"`
}

// CreateStrategyRequest represents a request to create a strategy
type CreateStrategyRequest struct {
	Name               string       `json:"name" validate:"required"`
	Description        *string      `json:"description,omitempty"`
	Type               StrategyType `json:"type" validate:"required"`
	Config             JSONB        `json:"config" validate:"required"`
	Schedule           *string      `json:"schedule,omitempty"`
	Priority           *int         `json:"priority,omitempty"`
	MaxConcurrentTasks *int         `json:"max_concurrent_tasks,omitempty"`
	RetryCount         *int         `json:"retry_count,omitempty"`
	TimeoutSeconds     *int         `json:"timeout_seconds,omitempty"`
}

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	AccountID      int    `json:"account_id" validate:"required"`
	StrategyID     int    `json:"strategy_id" validate:"required"`
	Type           string `json:"type" validate:"required"`
	Payload        JSONB  `json:"payload" validate:"required"`
	Priority       *int   `json:"priority,omitempty"`
	TimeoutSeconds *int   `json:"timeout_seconds,omitempty"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services,omitempty"`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int `json:"page" query:"page"`
	PageSize int `json:"page_size" query:"page_size"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Data       interface{}        `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
