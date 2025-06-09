package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models/notification"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UnifiedResponse represents the standard API response format
type UnifiedResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Details string `json:"details"`
}

// MetaInfo represents response metadata
type MetaInfo struct {
	RequestID     string `json:"request_id"`
	Timestamp     string `json:"timestamp"`
	ExecutionTime string `json:"execution_time"`
	Method        string `json:"method"`
	Path          string `json:"path"`
}

// responseWriter wraps gin.ResponseWriter to capture response
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// UnifiedResponseMiddleware transforms all responses to unified format
func UnifiedResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Generate request ID (always)
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Skip unified response for Swagger documentation paths or Swagger UI requests
		if shouldSkipUnifiedResponse(c) {
			// Still run audit logging for swagger requests
			defer func() {
				executionTime := time.Since(startTime)
				statusCode := c.Writer.Status()
				if statusCode == 0 {
					statusCode = 200 // Default status
				}
				go saveAuditLogAsync(c, "", statusCode, requestID, executionTime)
			}()
			c.Next()
			return
		}

		// Create custom response writer
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			status:         200,
		}
		c.Writer = w

		// Execute handler
		c.Next()

		// Calculate execution time
		executionTime := time.Since(startTime)

		// Get original response
		originalResponse := w.body.String()
		statusCode := w.status

		// Transform response to unified format
		unified := transformToUnifiedResponse(c, originalResponse, statusCode, requestID, executionTime)

		// Set proper headers and status code before writing response
		w.ResponseWriter.Header().Set("Content-Type", "application/json")
		w.ResponseWriter.WriteHeader(statusCode)

		// Write unified response to the actual response writer
		json.NewEncoder(w).Encode(unified)

		// 🔥 FIRE & FORGET - Async background tasks
		go saveAuditLogAsync(c, originalResponse, statusCode, requestID, executionTime)
		go sendNotificationAsync(c, unified)
	}
}

// transformToUnifiedResponse converts original response to unified format
func transformToUnifiedResponse(c *gin.Context, originalResponse string, statusCode int, requestID string, executionTime time.Duration) UnifiedResponse {
	isSuccess := statusCode >= 200 && statusCode < 300

	unified := UnifiedResponse{
		Success: isSuccess,
		Message: getAutoMessage(c.Request.Method, statusCode, isSuccess),
		Meta: &MetaInfo{
			RequestID:     requestID,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			ExecutionTime: fmt.Sprintf("%dms", executionTime.Milliseconds()),
			Method:        c.Request.Method,
			Path:          c.Request.URL.Path,
		},
	}

	if originalResponse != "" {
		var originalData interface{}
		if err := json.Unmarshal([]byte(originalResponse), &originalData); err == nil {
			if isSuccess {
				// Success response
				if dataMap, ok := originalData.(map[string]interface{}); ok {
					if data, exists := dataMap["data"]; exists {
						unified.Data = data
					} else {
						unified.Data = originalData
					}
					// Use custom message if provided
					if msg, exists := dataMap["message"]; exists {
						if msgStr, ok := msg.(string); ok && msgStr != "" {
							unified.Message = msgStr
						}
					}
				} else {
					unified.Data = originalData
				}
			} else {
				// Error response
				if errorMap, ok := originalData.(map[string]interface{}); ok {
					if errMsg, exists := errorMap["error"]; exists {
						unified.Error = &ErrorInfo{
							Code:    getErrorCode(statusCode),
							Details: fmt.Sprintf("%v", errMsg),
						}
					} else {
						unified.Error = &ErrorInfo{
							Code:    getErrorCode(statusCode),
							Details: originalResponse,
						}
					}
				} else {
					unified.Error = &ErrorInfo{
						Code:    getErrorCode(statusCode),
						Details: originalResponse,
					}
				}
			}
		}
	}

	return unified
}

// getAutoMessage generates appropriate success/error messages
func getAutoMessage(method string, statusCode int, isSuccess bool) string {
	if isSuccess {
		switch method {
		case "POST":
			return "Record created successfully"
		case "PUT", "PATCH":
			return "Record updated successfully"
		case "DELETE":
			return "Record deleted successfully"
		case "GET":
			return "Data retrieved successfully"
		default:
			return "Operation completed successfully"
		}
	} else {
		switch statusCode {
		case 400:
			return "Invalid request data"
		case 401:
			return "Authentication required"
		case 403:
			return "Permission denied"
		case 404:
			return "Resource not found"
		case 409:
			return "Resource already exists"
		case 422:
			return "Validation failed"
		case 500:
			return "Internal server error"
		default:
			return "Operation failed"
		}
	}
}

// getErrorCode generates error codes based on status
func getErrorCode(statusCode int) string {
	switch statusCode {
	case 400:
		return "BAD_REQUEST"
	case 401:
		return "UNAUTHORIZED"
	case 403:
		return "FORBIDDEN"
	case 404:
		return "NOT_FOUND"
	case 409:
		return "CONFLICT"
	case 422:
		return "VALIDATION_ERROR"
	case 500:
		return "INTERNAL_ERROR"
	default:
		return "UNKNOWN_ERROR"
	}
}

// saveAuditLogAsync saves audit log asynchronously
func saveAuditLogAsync(c *gin.Context, originalResponse string, statusCode int, requestID string, executionTime time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Audit log failed: %v\n", r)
		}
	}()

	// Get user ID from context (if available)
	var userID *uuid.UUID
	if userIDStr, exists := c.Get("user_id"); exists {
		if id, err := uuid.Parse(fmt.Sprintf("%v", userIDStr)); err == nil {
			userID = &id
		}
	}

	// Parse request body
	var requestBody interface{}
	if c.Request.Method != "GET" && c.Request.Method != "DELETE" {
		// Try to get request body from context first (if it was already read)
		if rawData, exists := c.Get("raw_body"); exists {
			if bodyBytes, ok := rawData.([]byte); ok {
				json.Unmarshal(bodyBytes, &requestBody)
			}
		} else {
			// For JSON requests, try to reconstruct from form values or params
			if c.Request.Header.Get("Content-Type") == "application/json" {
				// Body might be already consumed, skip for now
				requestBody = nil
			}
		}
	}

	// Parse response body
	var responseBody interface{}
	if originalResponse != "" {
		json.Unmarshal([]byte(originalResponse), &responseBody)
	}

	// Create audit log
	auditLog := notification.AuditLog{
		UserID:       userID,
		Method:       c.Request.Method,
		Path:         c.Request.URL.Path,
		StatusCode:   statusCode,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		Duration:     executionTime.Milliseconds(),
		RequestID:    requestID,
	}

	// Save to database (lazy initialization)
	db := database.GetDB()
	if db == nil {
		if err := database.InitDatabase(); err != nil {
			fmt.Printf("❌ Failed to initialize database for audit logging: %v\n", err)
			return
		}
		db = database.GetDB()
		if db == nil {
			fmt.Printf("❌ Database connection is still nil after initialization\n")
			return
		}
	}

	fmt.Printf("🔍 Attempting to save audit log: Method=%s, Path=%s, Status=%d, UserID=%v\n",
		auditLog.Method, auditLog.Path, auditLog.StatusCode, auditLog.UserID)

	if err := db.Create(&auditLog).Error; err != nil {
		fmt.Printf("❌ Failed to save audit log: %v\n", err)
		fmt.Printf("🔍 Audit log data: %+v\n", auditLog)
	} else {
		fmt.Printf("✅ Audit log saved successfully with ID: %s\n", auditLog.ID.String())
	}
}

// sendNotificationAsync sends real-time notification asynchronously
func sendNotificationAsync(c *gin.Context, unified UnifiedResponse) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Notification send failed: %v\n", r)
		}
	}()

	// Skip notifications for GET requests
	if c.Request.Method == "GET" {
		return
	}

	// Get user ID
	var userID *uuid.UUID
	if userIDStr, exists := c.Get("user_id"); exists {
		if id, err := uuid.Parse(fmt.Sprintf("%v", userIDStr)); err == nil {
			userID = &id
		}
	}

	// Skip if no user ID (anonymous requests)
	if userID == nil {
		return
	}

	// Determine notification level
	level := notification.NotificationLevelSuccess
	if !unified.Success {
		level = notification.NotificationLevelError
	}

	// Create notification title
	title := "✅ Success"
	if !unified.Success {
		title = "❌ Error"
	}

	// Create WebSocket message
	wsMessage := notification.WebSocketMessage{
		Type:      "notification",
		Level:     level,
		Title:     title,
		Message:   unified.Message,
		Timestamp: time.Now(),
		Action:    strings.ToLower(c.Request.Method),
		UserID:    userID,
	}

	// Send via WebSocket service
	sendToWebSocket(userID.String(), &wsMessage)

	fmt.Printf("📡 WebSocket message sent to user %s: %+v\n", userID.String(), wsMessage)
}

// sendToWebSocket sends message to WebSocket service
func sendToWebSocket(userID string, message *notification.WebSocketMessage) {
	// Get notification service URL from config
	cfg := config.GetConfig()
	url := cfg.NotificationServiceURL + "/ws/send"

	// Create request payload
	payload := map[string]interface{}{
		"user_id": userID,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ Error marshaling WebSocket message: %v\n", err)
		return
	}

	// Send async HTTP request to notification service
	go func() {
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("❌ Error sending WebSocket message: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			fmt.Printf("✅ WebSocket message sent successfully to user %s\n", userID)
		} else {
			fmt.Printf("❌ WebSocket service returned status: %d\n", resp.StatusCode)
		}
	}()
}

// shouldSkipUnifiedResponse checks if the request path should skip unified response format
func shouldSkipUnifiedResponse(c *gin.Context) bool {
	path := c.Request.URL.Path

	// Skip Swagger documentation paths
	excludePaths := []string{
		// "/swagger",
		"/docs",
		"/health",
		"/metrics",
	}

	for _, excludePath := range excludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}

	// Check if request is coming from Swagger UI by examining Referer header
	referer := c.Request.Header.Get("Referer")
	if strings.Contains(referer, "/swagger") || strings.Contains(referer, "/docs") {
		return true
	}

	// Check for swagger-ui specific query parameters
	if c.Query("swagger") != "" || c.Query("_swagger") != "" {
		return true
	}

	// Check User-Agent for swagger-ui
	userAgent := c.Request.Header.Get("User-Agent")
	return strings.Contains(userAgent, "swagger-ui")
}
