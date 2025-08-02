package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/bsky-automation/shared/models"
)

// AccountHandler handles HTTP requests for account management
type AccountHandler struct {
	accountService *AccountService
	authService    *AuthService
	validator      *validator.Validate
}

// NewAccountHandler creates a new account handler
func NewAccountHandler(accountService *AccountService, authService *AuthService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		authService:    authService,
		validator:      validator.New(),
	}
}

// CreateAccount creates a new account
// @Summary Create a new account
// @Description Create a new Bluesky account in the system
// @Tags accounts
// @Accept json
// @Produce json
// @Param account body models.CreateAccountRequest true "Account data"
// @Success 201 {object} models.Account
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req models.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Validation failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	account, err := h.accountService.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create account",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, account)
}

// GetAccount retrieves an account by ID
// @Summary Get account by ID
// @Description Get a specific account by its ID
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {object} models.Account
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	account, err := h.accountService.GetAccount(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Account not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get account",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

// ListAccounts retrieves a paginated list of accounts
// @Summary List accounts
// @Description Get a paginated list of accounts
// @Tags accounts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param status query string false "Filter by status" Enums(active,inactive,suspended,error)
// @Success 200 {object} models.ListResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/accounts [get]
func (h *AccountHandler) ListAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	
	var status *models.AccountStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := models.AccountStatus(statusStr)
		status = &s
	}

	result, err := h.accountService.ListAccounts(c.Request.Context(), page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to list accounts",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateAccount updates an existing account
// @Summary Update account
// @Description Update an existing account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Param account body models.UpdateAccountRequest true "Account update data"
// @Success 200 {object} models.Account
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/accounts/{id} [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req models.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	account, err := h.accountService.UpdateAccount(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Account not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to update account",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

// DeleteAccount deletes an account
// @Summary Delete account
// @Description Delete an account and all related data
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/accounts/{id} [delete]
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err = h.accountService.DeleteAccount(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Account not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to delete account",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestAuthentication tests account authentication
// @Summary Test account authentication
// @Description Test if an account can authenticate with Bluesky
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/accounts/{id}/test-auth [post]
func (h *AccountHandler) TestAuthentication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err = h.accountService.TestAuthentication(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Authentication test failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Authentication test passed",
	})
}

// RefreshAuthentication refreshes account authentication
// @Summary Refresh account authentication
// @Description Refresh authentication tokens for an account
// @Tags accounts
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Success 200 {object} models.Account
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/accounts/{id}/refresh-auth [post]
func (h *AccountHandler) RefreshAuthentication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	account, err := h.accountService.RefreshAuthentication(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to refresh authentication",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

// Login handles user login
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AccountHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	response, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "Login failed",
			Message: err.Error(),
			Code:    http.StatusUnauthorized,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// RefreshToken refreshes JWT token
// @Summary Refresh JWT token
// @Description Refresh an expired JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *AccountHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	response, err := h.authService.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "Token refresh failed",
			Message: err.Error(),
			Code:    http.StatusUnauthorized,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Logout handles user logout
// @Summary User logout
// @Description Logout user and invalidate token
// @Tags auth
// @Accept json
// @Produce json
// @Param token body LogoutRequest true "Logout token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *AccountHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.authService.Logout(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Logout failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Logged out successfully",
	})
}

// GetAccountStats returns account statistics
// @Summary Get account statistics
// @Description Get overall account statistics
// @Tags stats
// @Accept json
// @Produce json
// @Success 200 {object} AccountStatsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/stats/accounts [get]
func (h *AccountHandler) GetAccountStats(c *gin.Context) {
	stats, err := h.accountService.GetAccountStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get account stats",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetAccountMetrics returns metrics for a specific account
// @Summary Get account metrics
// @Description Get metrics and performance data for a specific account
// @Tags stats
// @Accept json
// @Produce json
// @Param id path int true "Account ID"
// @Param days query int false "Number of days to include" default(7)
// @Success 200 {object} AccountMetricsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/stats/accounts/{id}/metrics [get]
func (h *AccountHandler) GetAccountMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid account ID",
			Message: "Account ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	metrics, err := h.accountService.GetAccountMetrics(c.Request.Context(), id, days)
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Account not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get account metrics",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}
