package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"

	"github.com/bsky-automation/shared/models"
)

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.GET("/health", healthCheckHandler)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.HealthCheckResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response.Status)
}

func TestCreateProxyValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Mock handler for testing validation
	handler := &ProxyHandler{
		validator: validator.New(),
	}

	router := gin.New()
	router.POST("/proxies", handler.CreateProxy)

	// Test invalid request body
	invalidReq := map[string]interface{}{
		"name": "", // Empty name should fail validation
		"type": "http",
		"host": "proxy.example.com",
		"port": 8080,
	}
	
	body, _ := json.Marshal(invalidReq)
	req, _ := http.NewRequest("POST", "/proxies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Validation failed", response.Error)
}

func TestProxyTypeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := &ProxyHandler{
		validator: validator.New(),
	}

	router := gin.New()
	router.POST("/proxies", handler.CreateProxy)

	// Test invalid proxy type
	invalidReq := models.CreateProxyRequest{
		Name: "Test Proxy",
		Type: "invalid_type", // Invalid type
		Host: "proxy.example.com",
		Port: 8080,
	}
	
	body, _ := json.Marshal(invalidReq)
	req, _ := http.NewRequest("POST", "/proxies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProxyPortValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := &ProxyHandler{
		validator: validator.New(),
	}

	router := gin.New()
	router.POST("/proxies", handler.CreateProxy)

	// Test invalid port
	invalidReq := models.CreateProxyRequest{
		Name: "Test Proxy",
		Type: models.ProxyTypeHTTP,
		Host: "proxy.example.com",
		Port: 70000, // Invalid port (too high)
	}
	
	body, _ := json.Marshal(invalidReq)
	req, _ := http.NewRequest("POST", "/proxies", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
}

func TestOPTIONSRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestProxyAssignmentRequest(t *testing.T) {
	// Test proxy assignment request structure
	req := ProxyAssignmentRequest{
		AccountID: 1,
		ProxyID:   nil,
		ProxyType: &models.ProxyTypeHTTP,
		Strategy:  "auto",
	}

	assert.Equal(t, 1, req.AccountID)
	assert.Nil(t, req.ProxyID)
	assert.Equal(t, models.ProxyTypeHTTP, *req.ProxyType)
	assert.Equal(t, "auto", req.Strategy)
}

func TestProxyUsageResponse(t *testing.T) {
	// Test proxy usage response structure
	usage := ProxyUsageResponse{
		TotalProxies:     10,
		ActiveProxies:    8,
		AssignedProxies:  5,
		AvailableProxies: 3,
		UsageByType:      make(map[models.ProxyType]int),
	}

	usage.UsageByType[models.ProxyTypeHTTP] = 6
	usage.UsageByType[models.ProxyTypeSOCKS5] = 4

	assert.Equal(t, 10, usage.TotalProxies)
	assert.Equal(t, 8, usage.ActiveProxies)
	assert.Equal(t, 5, usage.AssignedProxies)
	assert.Equal(t, 3, usage.AvailableProxies)
	assert.Equal(t, 6, usage.UsageByType[models.ProxyTypeHTTP])
	assert.Equal(t, 4, usage.UsageByType[models.ProxyTypeSOCKS5])
}
