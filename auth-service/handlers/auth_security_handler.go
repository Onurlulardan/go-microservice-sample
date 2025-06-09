package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/auth"
	"forgecrud-backend/shared/utils/query"
)

// SessionResponse represents a user session in the response
type SessionResponse struct {
	ID               uuid.UUID `json:"id"`
	DeviceInfo       string    `json:"device_info"`
	IPAddress        string    `json:"ip_address"`
	LastUsedAt       time.Time `json:"last_used_at"`
	CreatedAt        time.Time `json:"created_at"`
	IsCurrentSession bool      `json:"is_current_session"`
}

// LoginHistoryResponse represents a login history entry in the response
type LoginHistoryResponse struct {
	ID          uuid.UUID `json:"id"`
	IPAddress   string    `json:"ip_address"`
	DeviceInfo  string    `json:"device_info"`
	Successful  bool      `json:"successful"`
	FailureType string    `json:"failure_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Location    string    `json:"location,omitempty"`
}

// SessionListResponse represents a list of user sessions
type SessionListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []SessionResponse  `json:"items"`
		Pagination PaginationResponse `json:"pagination"`
	} `json:"data"`
}

// LoginHistoryListResponse represents a list of login history entries
type LoginHistoryListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []LoginHistoryResponse `json:"items"`
		Pagination PaginationResponse     `json:"pagination"`
	} `json:"data"`
}

// PaginationResponse represents pagination information
type PaginationResponse struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// ListSessions lists all active sessions for the authenticated user
// @Summary List user sessions
// @Description Get all active sessions for the currently authenticated user
// @Tags sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param filters[is_active] query boolean false "Filter by active status"
// @Param sort[field] query string false "Sort field (created_at, updated_at, last_used_at)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Success 200 {object} handlers.SessionListResponse "List of user sessions"
// @Failure 401 {object} map[string]string "User not authenticated"
// @Failure 500 {object} map[string]string "Failed to retrieve sessions"
// @Router /auth/sessions [get]
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse query parameters using the shared utility
	params := query.ParseQueryParams(c)

	// Allowed filters for sessions (could add more if needed)
	allowedFilters := map[string]string{
		"is_active": "is_active",
	}

	// Allowed sort fields for sessions
	allowedSortFields := map[string]string{
		"created_at":   "created_at",
		"updated_at":   "updated_at",
		"last_used_at": "updated_at",
	}

	currentTokenHash, _ := c.Get("tokenHash")

	// Build base query - always filter by user and active status
	dbQuery := h.db.Model(&auth.UserSession{}).Where("user_id = ? AND is_active = ?", userID, true)

	// Apply filters (though for sessions we mainly just need active status)
	dbQuery = query.ApplyFilters(dbQuery, params.Filters, allowedFilters)

	// Apply sorting
	dbQuery = query.ApplySort(dbQuery, params.Sort, allowedSortFields)

	// Get total count
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count sessions"})
		return
	}

	// Apply pagination
	dbQuery = query.ApplyPagination(dbQuery, params.Page, params.Limit)

	// Get sessions
	var sessions []auth.UserSession
	if err := dbQuery.Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sessions"})
		return
	}

	var response []SessionResponse
	for _, session := range sessions {
		deviceInfo := parseUserAgent(session.UserAgent)

		isCurrentSession := false
		if currentTokenHash != nil && session.TokenHash == currentTokenHash.(string) {
			isCurrentSession = true
		}

		response = append(response, SessionResponse{
			ID:               session.ID,
			DeviceInfo:       deviceInfo,
			IPAddress:        session.IPAddress,
			LastUsedAt:       session.UpdatedAt,
			CreatedAt:        session.CreatedAt,
			IsCurrentSession: isCurrentSession,
		})
	}

	// Build pagination response
	paginationResponse := query.BuildPaginationResponse(params.Page, params.Limit, total)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      response,
			"pagination": paginationResponse,
		},
	})
}

// TerminateSession terminates a specific session
// @Summary Terminate session
// @Description Terminate a specific user session by ID
// @Tags sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Session ID to terminate"
// @Success 200 {object} map[string]string "Session terminated successfully"
// @Failure 400 {object} map[string]string "Session ID is required or invalid format"
// @Failure 401 {object} map[string]string "User not authenticated"
// @Failure 404 {object} map[string]string "Session not found"
// @Failure 500 {object} map[string]string "Failed to terminate session"
// @Router /auth/sessions/{id} [delete]
func (h *AuthHandler) TerminateSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID format"})
		return
	}

	currentTokenHash, _ := c.Get("tokenHash")

	var session auth.UserSession
	if err := h.db.Where("id = ? AND user_id = ?", sessionUUID, userID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or does not belong to the user"})
		return
	}

	if currentTokenHash != nil && session.TokenHash == currentTokenHash.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot terminate the current session"})
		return
	}

	if err := h.db.Model(&auth.UserSession{}).
		Where("id = ? AND user_id = ?", sessionUUID, userID).
		Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated successfully"})
}

// TerminateAllSessions terminates all sessions except the current one
// @Summary Terminate all sessions
// @Description Terminate all active sessions for the current user except the current session
// @Tags sessions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "All other sessions terminated successfully"
// @Failure 401 {object} map[string]string "User not authenticated"
// @Failure 500 {object} map[string]string "Failed to terminate sessions"
// @Router /auth/sessions/terminate-all [post]
func (h *AuthHandler) TerminateAllSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	currentTokenHash, _ := c.Get("tokenHash")

	if err := h.db.Model(&auth.UserSession{}).
		Where("user_id = ? AND token_hash != ? AND is_active = ?", userID, currentTokenHash, true).
		Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All other sessions terminated successfully"})
}

// GetLoginHistory retrieves the login history for the authenticated user
// @Summary Get login history
// @Description Get login history for the currently authenticated user
// @Tags auth-security
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param filters[successful] query boolean false "Filter by login success"
// @Param filters[from_date] query string false "Filter by date from (YYYY-MM-DD)"
// @Param filters[to_date] query string false "Filter by date to (YYYY-MM-DD)"
// @Param sort[field] query string false "Sort field (created_at, successful)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Success 200 {object} handlers.LoginHistoryListResponse "Login history list"
// @Failure 401 {object} map[string]string "User not authenticated"
// @Failure 500 {object} map[string]string "Failed to retrieve login history"
// @Router /auth/login-history [get]
func (h *AuthHandler) GetLoginHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse query parameters using the shared utility
	params := query.ParseQueryParams(c)

	// Allowed filters for login history
	allowedFilters := map[string]string{
		"successful": "successful",
		"from_date":  "created_at",
		"to_date":    "created_at",
	}

	// Allowed sort fields for login history
	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"successful": "successful",
	}

	// Get user email for filtering login attempts
	userEmail := getUserEmail(h.db, userID.(uuid.UUID))
	if userEmail == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user email"})
		return
	}

	// Build base query
	dbQuery := h.db.Model(&auth.LoginAttempt{}).Where("email = ?", userEmail)

	// Apply custom date filters if provided
	if fromDate := c.Query("filters[from_date]"); fromDate != "" {
		if parsedFromDate, err := time.Parse("2006-01-02", fromDate); err == nil {
			dbQuery = dbQuery.Where("created_at >= ?", parsedFromDate)
		}
	}
	if toDate := c.Query("filters[to_date]"); toDate != "" {
		if parsedToDate, err := time.Parse("2006-01-02", toDate); err == nil {
			parsedToDate = parsedToDate.AddDate(0, 0, 1)
			dbQuery = dbQuery.Where("created_at < ?", parsedToDate)
		}
	}

	// Apply standard filters (excluding date filters since they're handled above)
	filteredParams := make(map[string]string)
	for key, value := range params.Filters {
		if key != "from_date" && key != "to_date" {
			filteredParams[key] = value
		}
	}
	dbQuery = query.ApplyFilters(dbQuery, filteredParams, allowedFilters)

	// Apply sorting
	dbQuery = query.ApplySort(dbQuery, params.Sort, allowedSortFields)

	// Get total count
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count login history"})
		return
	}

	// Apply pagination
	dbQuery = query.ApplyPagination(dbQuery, params.Page, params.Limit)

	// Get login attempts
	var loginAttempts []auth.LoginAttempt
	if err := dbQuery.Find(&loginAttempts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve login history"})
		return
	}

	var response []LoginHistoryResponse
	for _, attempt := range loginAttempts {
		deviceInfo := parseUserAgent(attempt.UserAgent)

		response = append(response, LoginHistoryResponse{
			ID:          attempt.ID,
			IPAddress:   attempt.IPAddress,
			DeviceInfo:  deviceInfo,
			Successful:  attempt.Successful,
			FailureType: attempt.FailureType,
			CreatedAt:   attempt.CreatedAt,
		})
	}

	// Build pagination response
	paginationResponse := query.BuildPaginationResponse(params.Page, params.Limit, total)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      response,
			"pagination": paginationResponse,
		},
	})
}

// parseUserAgent extracts useful device info from user agent string
func parseUserAgent(userAgent string) string {
	if userAgent == "" {
		return "Unknown"
	}

	if strings.Contains(userAgent, "iPhone") || strings.Contains(userAgent, "iPad") {
		return "iOS Device"
	} else if strings.Contains(userAgent, "Android") {
		return "Android Device"
	} else if strings.Contains(userAgent, "Windows") {
		return "Windows"
	} else if strings.Contains(userAgent, "Mac") {
		return "MacOS"
	} else if strings.Contains(userAgent, "Linux") {
		return "Linux"
	}

	return "Other"
}

// getUserEmail gets the user's email based on their ID
func getUserEmail(db *gorm.DB, userID uuid.UUID) string {
	var user models.User
	if err := db.Select("email").Where("id = ?", userID).First(&user).Error; err != nil {
		return ""
	}
	return user.Email
}
