package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/bsky-automation/shared/models"
)

// ProxyHandler handles HTTP requests for proxy management
type ProxyHandler struct {
	proxyService *ProxyService
	validator    *validator.Validate
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(proxyService *ProxyService) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		validator:    validator.New(),
	}
}

// CreateProxy creates a new proxy
// @Summary Create a new proxy
// @Description Create a new proxy server configuration
// @Tags proxies
// @Accept json
// @Produce json
// @Param proxy body models.CreateProxyRequest true "Proxy data"
// @Success 201 {object} models.Proxy
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/proxies [post]
func (h *ProxyHandler) CreateProxy(c *gin.Context) {
	var req models.CreateProxyRequest
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

	proxy, err := h.proxyService.CreateProxy(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to create proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, proxy)
}

// GetProxy retrieves a proxy by ID
// @Summary Get proxy by ID
// @Description Get a specific proxy by its ID
// @Tags proxies
// @Accept json
// @Produce json
// @Param id path int true "Proxy ID"
// @Success 200 {object} models.Proxy
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/proxies/{id} [get]
func (h *ProxyHandler) GetProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid proxy ID",
			Message: "Proxy ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	proxy, err := h.proxyService.GetProxy(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "proxy not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Proxy not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, proxy)
}

// ListProxies retrieves a paginated list of proxies
// @Summary List proxies
// @Description Get a paginated list of proxies
// @Tags proxies
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param status query string false "Filter by status" Enums(active,inactive,error)
// @Param type query string false "Filter by type" Enums(http,socks5)
// @Success 200 {object} models.ListResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/proxies [get]
func (h *ProxyHandler) ListProxies(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	
	var status *models.ProxyStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := models.ProxyStatus(statusStr)
		status = &s
	}

	var proxyType *models.ProxyType
	if typeStr := c.Query("type"); typeStr != "" {
		t := models.ProxyType(typeStr)
		proxyType = &t
	}

	result, err := h.proxyService.ListProxies(c.Request.Context(), page, pageSize, status, proxyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to list proxies",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateProxy updates an existing proxy
// @Summary Update proxy
// @Description Update an existing proxy
// @Tags proxies
// @Accept json
// @Produce json
// @Param id path int true "Proxy ID"
// @Param proxy body UpdateProxyRequest true "Proxy update data"
// @Success 200 {object} models.Proxy
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/proxies/{id} [put]
func (h *ProxyHandler) UpdateProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid proxy ID",
			Message: "Proxy ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	proxy, err := h.proxyService.UpdateProxy(c.Request.Context(), id, &req)
	if err != nil {
		if err.Error() == "proxy not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Proxy not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to update proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, proxy)
}

// DeleteProxy deletes a proxy
// @Summary Delete proxy
// @Description Delete a proxy and all related data
// @Tags proxies
// @Accept json
// @Produce json
// @Param id path int true "Proxy ID"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/proxies/{id} [delete]
func (h *ProxyHandler) DeleteProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid proxy ID",
			Message: "Proxy ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	err = h.proxyService.DeleteProxy(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "proxy not found" {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "Proxy not found",
				Message: err.Error(),
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to delete proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// TestProxy tests proxy connection
// @Summary Test proxy connection
// @Description Test if a proxy server is working correctly
// @Tags proxies
// @Accept json
// @Produce json
// @Param id path int true "Proxy ID"
// @Success 200 {object} ProxyTestResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/proxies/{id}/test [post]
func (h *ProxyHandler) TestProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid proxy ID",
			Message: "Proxy ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	result, err := h.proxyService.TestProxy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to test proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RunHealthCheck runs health check for a proxy
// @Summary Run health check
// @Description Run health check for a specific proxy
// @Tags proxies
// @Accept json
// @Produce json
// @Param id path int true "Proxy ID"
// @Success 200 {object} ProxyTestResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/proxies/{id}/health-check [post]
func (h *ProxyHandler) RunHealthCheck(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid proxy ID",
			Message: "Proxy ID must be a valid integer",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Health check is the same as test for now
	result, err := h.proxyService.TestProxy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to run health check",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAvailableProxies returns available proxies for assignment
// @Summary Get available proxies
// @Description Get list of available proxies for assignment
// @Tags assignment
// @Accept json
// @Produce json
// @Param type query string false "Filter by proxy type" Enums(http,socks5)
// @Success 200 {array} models.Proxy
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/assignment/available [get]
func (h *ProxyHandler) GetAvailableProxies(c *gin.Context) {
	var proxyType *models.ProxyType
	if typeStr := c.Query("type"); typeStr != "" {
		t := models.ProxyType(typeStr)
		proxyType = &t
	}

	proxies, err := h.proxyService.GetAvailableProxies(c.Request.Context(), proxyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get available proxies",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, proxies)
}

// AssignProxy assigns a proxy to an account
// @Summary Assign proxy to account
// @Description Assign a proxy to a specific account
// @Tags assignment
// @Accept json
// @Produce json
// @Param assignment body ProxyAssignmentRequest true "Assignment data"
// @Success 200 {object} ProxyAssignmentResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/assignment/assign [post]
func (h *ProxyHandler) AssignProxy(c *gin.Context) {
	var req ProxyAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	result, err := h.proxyService.AssignProxy(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to assign proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ReleaseProxy releases a proxy from an account
// @Summary Release proxy from account
// @Description Release a proxy from a specific account
// @Tags assignment
// @Accept json
// @Produce json
// @Param release body ProxyReleaseRequest true "Release data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/assignment/release [post]
func (h *ProxyHandler) ReleaseProxy(c *gin.Context) {
	var req ProxyReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	err := h.proxyService.ReleaseProxy(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to release proxy",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Proxy released successfully",
	})
}

// GetProxyUsage returns proxy usage statistics
// @Summary Get proxy usage
// @Description Get proxy usage statistics and assignments
// @Tags assignment
// @Accept json
// @Produce json
// @Success 200 {object} ProxyUsageResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/assignment/usage [get]
func (h *ProxyHandler) GetProxyUsage(c *gin.Context) {
	usage, err := h.proxyService.GetProxyUsage(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get proxy usage",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetProxyStats returns proxy statistics
// @Summary Get proxy statistics
// @Description Get overall proxy statistics
// @Tags stats
// @Accept json
// @Produce json
// @Success 200 {object} ProxyStatsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/stats/proxies [get]
func (h *ProxyHandler) GetProxyStats(c *gin.Context) {
	stats, err := h.proxyService.GetProxyStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get proxy stats",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetHealthStats returns proxy health statistics
// @Summary Get proxy health statistics
// @Description Get proxy health and availability statistics
// @Tags stats
// @Accept json
// @Produce json
// @Success 200 {object} ProxyHealthStatsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/stats/health [get]
func (h *ProxyHandler) GetHealthStats(c *gin.Context) {
	stats, err := h.proxyService.GetHealthStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get health stats",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetPerformanceStats returns proxy performance statistics
// @Summary Get proxy performance statistics
// @Description Get proxy performance and response time statistics
// @Tags stats
// @Accept json
// @Produce json
// @Param days query int false "Number of days to include" default(7)
// @Success 200 {object} ProxyPerformanceStatsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/stats/performance [get]
func (h *ProxyHandler) GetPerformanceStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	stats, err := h.proxyService.GetPerformanceStats(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Failed to get performance stats",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
