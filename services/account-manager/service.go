package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/bsky-automation/shared/models"
	"github.com/bsky-automation/shared/utils"
	bluesky "github.com/bsky-automation/shared/bluesky-client"
)

// AccountService handles account-related business logic
type AccountService struct {
	db  *sql.DB
	rdb *redis.Client
}

// NewAccountService creates a new account service
func NewAccountService(db *sql.DB, rdb *redis.Client) *AccountService {
	return &AccountService{
		db:  db,
		rdb: rdb,
	}
}

// CreateAccount creates a new account
func (s *AccountService) CreateAccount(ctx context.Context, req *models.CreateAccountRequest) (*models.Account, error) {
	// Validate input
	if !utils.ValidateHandle(req.Handle) {
		return nil, fmt.Errorf("invalid handle format")
	}

	// Set defaults
	if req.Host == "" {
		req.Host = "https://bsky.social"
	}
	if req.BGS == "" {
		req.BGS = "https://bsky.network"
	}

	// Check if account already exists
	exists, err := s.accountExists(ctx, req.Handle)
	if err != nil {
		return nil, fmt.Errorf("failed to check account existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("account with handle %s already exists", req.Handle)
	}

	// Create account
	account := &models.Account{
		UUID:     utils.GenerateUUID(),
		Handle:   req.Handle,
		Password: req.Password,
		Host:     req.Host,
		BGS:      req.BGS,
		Status:   models.AccountStatusActive,
		ProxyID:  req.ProxyID,
		Metadata: make(models.JSONB),
	}

	// Insert into database
	query := `
		INSERT INTO accounts (uuid, handle, password, host, bgs, status, proxy_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		account.UUID, account.Handle, account.Password, account.Host,
		account.BGS, account.Status, account.ProxyID, account.Metadata,
	).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	// Test authentication if requested
	if err := s.testAccountAuthentication(ctx, account); err != nil {
		// Log the error but don't fail the creation
		// Update account status to error
		account.Status = models.AccountStatusError
		errMsg := err.Error()
		account.ErrorMessage = &errMsg
		s.updateAccountStatus(ctx, account.ID, account.Status, account.ErrorMessage)
	}

	return account, nil
}

// GetAccount retrieves an account by ID
func (s *AccountService) GetAccount(ctx context.Context, id int) (*models.Account, error) {
	query := `
		SELECT a.id, a.uuid, a.handle, a.password, a.host, a.bgs, a.status,
		       a.proxy_id, a.did, a.access_jwt, a.refresh_jwt, a.last_login,
		       a.last_activity, a.error_count, a.error_message, a.metadata,
		       a.created_at, a.updated_at,
		       p.id, p.uuid, p.name, p.type, p.host, p.port, p.status
		FROM accounts a
		LEFT JOIN proxies p ON a.proxy_id = p.id
		WHERE a.id = $1
	`

	account := &models.Account{}
	var proxy models.Proxy
	var proxyID sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.UUID, &account.Handle, &account.Password,
		&account.Host, &account.BGS, &account.Status, &account.ProxyID,
		&account.DID, &account.AccessJWT, &account.RefreshJWT,
		&account.LastLogin, &account.LastActivity, &account.ErrorCount,
		&account.ErrorMessage, &account.Metadata, &account.CreatedAt,
		&account.UpdatedAt,
		&proxyID, &proxy.UUID, &proxy.Name, &proxy.Type,
		&proxy.Host, &proxy.Port, &proxy.Status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Set proxy if exists
	if proxyID.Valid {
		proxy.ID = int(proxyID.Int64)
		account.Proxy = &proxy
	}

	return account, nil
}

// ListAccounts retrieves a paginated list of accounts
func (s *AccountService) ListAccounts(ctx context.Context, page, pageSize int, status *models.AccountStatus) (*models.ListResponse, error) {
	// Calculate pagination
	offset, limit, _ := utils.Paginate(page, pageSize, 0)

	// Build query
	baseQuery := `
		SELECT a.id, a.uuid, a.handle, a.host, a.status, a.proxy_id,
		       a.last_login, a.last_activity, a.error_count, a.created_at,
		       p.name as proxy_name
		FROM accounts a
		LEFT JOIN proxies p ON a.proxy_id = p.id
	`

	var args []interface{}
	whereClause := ""

	if status != nil {
		whereClause = "WHERE a.status = $1"
		args = append(args, *status)
	}

	query := fmt.Sprintf("%s %s ORDER BY a.created_at DESC LIMIT $%d OFFSET $%d",
		baseQuery, whereClause, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		var proxyName sql.NullString

		err := rows.Scan(
			&account.ID, &account.UUID, &account.Handle, &account.Host,
			&account.Status, &account.ProxyID, &account.LastLogin,
			&account.LastActivity, &account.ErrorCount, &account.CreatedAt,
			&proxyName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		accounts = append(accounts, account)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM accounts"
	if whereClause != "" {
		countQuery += " " + whereClause
		// Use only the status parameter for count
		if status != nil {
			var totalItems int64
			err = s.db.QueryRowContext(ctx, countQuery, *status).Scan(&totalItems)
		}
	} else {
		var totalItems int64
		err = s.db.QueryRowContext(ctx, countQuery).Scan(&totalItems)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to count accounts: %w", err)
	}

	var totalItems int64
	if status != nil {
		err = s.db.QueryRowContext(ctx, countQuery, *status).Scan(&totalItems)
	} else {
		err = s.db.QueryRowContext(ctx, countQuery).Scan(&totalItems)
	}

	_, _, totalPages := utils.Paginate(page, pageSize, totalItems)

	return &models.ListResponse{
		Data: accounts,
		Pagination: models.PaginationResponse{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

// UpdateAccount updates an existing account
func (s *AccountService) UpdateAccount(ctx context.Context, id int, req *models.UpdateAccountRequest) (*models.Account, error) {
	// Get existing account
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build update query dynamically
	updates := make(map[string]interface{})
	if req.Password != nil {
		updates["password"] = *req.Password
	}
	if req.Host != nil {
		updates["host"] = *req.Host
	}
	if req.BGS != nil {
		updates["bgs"] = *req.BGS
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.ProxyID != nil {
		updates["proxy_id"] = *req.ProxyID
	}

	if len(updates) == 0 {
		return account, nil // No updates
	}

	updates["updated_at"] = time.Now()

	setClause, args := utils.BuildUpdateClause(updates)
	query := fmt.Sprintf("UPDATE accounts %s WHERE id = $%d", setClause, len(args)+1)
	args = append(args, id)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	// Return updated account
	return s.GetAccount(ctx, id)
}

// DeleteAccount deletes an account
func (s *AccountService) DeleteAccount(ctx context.Context, id int) error {
	// Check if account exists
	_, err := s.GetAccount(ctx, id)
	if err != nil {
		return err
	}

	// Delete account (this will cascade to related records)
	query := "DELETE FROM accounts WHERE id = $1"
	_, err = s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// TestAuthentication tests account authentication with Bluesky
func (s *AccountService) TestAuthentication(ctx context.Context, id int) error {
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return err
	}

	return s.testAccountAuthentication(ctx, account)
}

// RefreshAuthentication refreshes account authentication tokens
func (s *AccountService) RefreshAuthentication(ctx context.Context, id int) (*models.Account, error) {
	account, err := s.GetAccount(ctx, id)
	if err != nil {
		return nil, err
	}

	// Create Bluesky client
	client, err := bluesky.NewClient(bluesky.ClientConfig{
		Account: account,
		Proxy:   account.Proxy,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Bluesky client: %w", err)
	}

	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		// Update account status to error
		account.Status = models.AccountStatusError
		errMsg := err.Error()
		account.ErrorMessage = &errMsg
		account.ErrorCount++
		s.updateAccountStatus(ctx, account.ID, account.Status, account.ErrorMessage)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Update account with new tokens
	updatedAccount := client.GetAccount()
	query := `
		UPDATE accounts 
		SET did = $1, access_jwt = $2, refresh_jwt = $3, last_login = $4,
		    status = $5, error_count = 0, error_message = NULL, updated_at = NOW()
		WHERE id = $6
	`

	_, err = s.db.ExecContext(ctx, query,
		updatedAccount.DID, updatedAccount.AccessJWT, updatedAccount.RefreshJWT,
		updatedAccount.LastLogin, models.AccountStatusActive, account.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update account tokens: %w", err)
	}

	return s.GetAccount(ctx, id)
}

// Helper methods

func (s *AccountService) accountExists(ctx context.Context, handle string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM accounts WHERE handle = $1)"
	var exists bool
	err := s.db.QueryRowContext(ctx, query, handle).Scan(&exists)
	return exists, err
}

func (s *AccountService) testAccountAuthentication(ctx context.Context, account *models.Account) error {
	client, err := bluesky.NewClient(bluesky.ClientConfig{
		Account: account,
		Proxy:   account.Proxy,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create Bluesky client: %w", err)
	}

	return client.Authenticate(ctx)
}

func (s *AccountService) updateAccountStatus(ctx context.Context, id int, status models.AccountStatus, errorMessage *string) error {
	query := "UPDATE accounts SET status = $1, error_message = $2, updated_at = NOW() WHERE id = $3"
	_, err := s.db.ExecContext(ctx, query, status, errorMessage, id)
	return err
}

// GetAccountStats returns overall account statistics
func (s *AccountService) GetAccountStats(ctx context.Context) (*AccountStatsResponse, error) {
	stats := &AccountStatsResponse{
		StatusBreakdown: make(map[models.AccountStatus]int),
		ProxyUsage:      make(map[string]int),
	}

	// Get total counts by status
	statusQuery := `
		SELECT status, COUNT(*)
		FROM accounts
		GROUP BY status
	`
	rows, err := s.db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get status breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status models.AccountStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		stats.StatusBreakdown[status] = count
		stats.TotalAccounts += count

		switch status {
		case models.AccountStatusActive:
			stats.ActiveAccounts = count
		case models.AccountStatusInactive:
			stats.InactiveAccounts = count
		case models.AccountStatusError:
			stats.ErrorAccounts = count
		}
	}

	// Get proxy usage
	proxyQuery := `
		SELECT COALESCE(p.name, 'No Proxy') as proxy_name, COUNT(*)
		FROM accounts a
		LEFT JOIN proxies p ON a.proxy_id = p.id
		GROUP BY p.name
	`
	rows, err = s.db.QueryContext(ctx, proxyQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var proxyName string
		var count int
		if err := rows.Scan(&proxyName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan proxy row: %w", err)
		}
		stats.ProxyUsage[proxyName] = count
	}

	// Get recent activity
	activityQuery := `
		SELECT id, handle, last_activity, status
		FROM accounts
		WHERE last_activity IS NOT NULL
		ORDER BY last_activity DESC
		LIMIT 10
	`
	rows, err = s.db.QueryContext(ctx, activityQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var activity AccountActivitySummary
		if err := rows.Scan(&activity.AccountID, &activity.Handle, &activity.LastActivity, &activity.Status); err != nil {
			return nil, fmt.Errorf("failed to scan activity row: %w", err)
		}
		stats.RecentActivity = append(stats.RecentActivity, activity)
	}

	return stats, nil
}

// GetAccountMetrics returns metrics for a specific account
func (s *AccountService) GetAccountMetrics(ctx context.Context, accountID int, days int) (*AccountMetricsResponse, error) {
	// Get account info
	account, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}

	metrics := &AccountMetricsResponse{
		AccountID: accountID,
		Handle:    account.Handle,
		LastActivity: account.LastActivity,
	}

	// Get task statistics
	taskStatsQuery := `
		SELECT
			COUNT(*) as total_tasks,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_tasks,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_tasks
		FROM tasks
		WHERE account_id = $1 AND created_at >= NOW() - INTERVAL '%d days'
	`
	query := fmt.Sprintf(taskStatsQuery, days)
	err = s.db.QueryRowContext(ctx, query, accountID).Scan(
		&metrics.TotalTasks, &metrics.CompletedTasks, &metrics.FailedTasks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get task statistics: %w", err)
	}

	// Calculate success rate
	if metrics.TotalTasks > 0 {
		metrics.SuccessRate = utils.CalculateSuccessRate(metrics.CompletedTasks, metrics.TotalTasks)
	}

	// Get daily metrics
	dailyQuery := `
		SELECT
			DATE(created_at) as date,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM tasks
		WHERE account_id = $1 AND created_at >= NOW() - INTERVAL '%d days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`
	query = fmt.Sprintf(dailyQuery, days)
	rows, err := s.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily metrics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var daily DailyMetric
		var completed, failed int
		if err := rows.Scan(&daily.Date, &completed, &failed); err != nil {
			return nil, fmt.Errorf("failed to scan daily metric: %w", err)
		}
		daily.TasksCompleted = completed
		daily.TasksFailed = failed
		total := completed + failed
		if total > 0 {
			daily.SuccessRate = utils.CalculateSuccessRate(completed, total)
		}
		metrics.DailyMetrics = append(metrics.DailyMetrics, daily)
	}

	// Get strategy metrics
	strategyQuery := `
		SELECT
			s.name as strategy_name,
			s.type as strategy_type,
			COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed,
			COUNT(CASE WHEN t.status = 'failed' THEN 1 END) as failed
		FROM tasks t
		JOIN strategies s ON t.strategy_id = s.id
		WHERE t.account_id = $1 AND t.created_at >= NOW() - INTERVAL '%d days'
		GROUP BY s.id, s.name, s.type
		ORDER BY completed DESC
	`
	query = fmt.Sprintf(strategyQuery, days)
	rows, err = s.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy metrics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var strategy StrategyMetric
		var completed, failed int
		if err := rows.Scan(&strategy.StrategyName, &strategy.StrategyType, &completed, &failed); err != nil {
			return nil, fmt.Errorf("failed to scan strategy metric: %w", err)
		}
		strategy.TasksCompleted = completed
		strategy.TasksFailed = failed
		total := completed + failed
		if total > 0 {
			strategy.SuccessRate = utils.CalculateSuccessRate(completed, total)
		}
		metrics.StrategyMetrics = append(metrics.StrategyMetrics, strategy)
	}

	return metrics, nil
}
