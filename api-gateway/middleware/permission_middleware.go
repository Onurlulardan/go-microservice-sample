package middleware

import (
	"net/http"
	"strings"

	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/utils/permission"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RequirePermission creates a middleware that checks if user has specific permission
func RequirePermission(resourceSlug, actionSlug string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from JWT token
		userID, err := extractUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing token",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Check permission
		allowed, err := permission.CheckPermission(userID, resourceSlug, actionSlug)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permissions",
				"code":  "PERMISSION_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "FORBIDDEN",
				"details": gin.H{
					"required_resource": resourceSlug,
					"required_action":   actionSlug,
				},
			})
			c.Abort()
			return
		}

		// Add permission info to context for downstream services
		c.Set("user_id", userID)
		c.Set("resource", resourceSlug)
		c.Set("action", actionSlug)
		c.Set("permission_checked", true)

		c.Next()
	}
}

// RequireAnyPermission checks if user has ANY of the provided permissions
func RequireAnyPermission(permissions []struct{ Resource, Action string }) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := extractUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing token",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		// Prepare batch check
		var checks []permission.ResourceActionCheck
		for _, perm := range permissions {
			checks = append(checks, permission.ResourceActionCheck{
				ResourceSlug: perm.Resource,
				ActionSlug:   perm.Action,
			})
		}

		// Batch check permissions
		results, err := permission.BatchCheckPermissions(userID, checks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permissions",
				"code":  "PERMISSION_CHECK_FAILED",
			})
			c.Abort()
			return
		}

		// Check if user has ANY of the required permissions
		hasAnyPermission := false
		for _, result := range results {
			if result {
				hasAnyPermission = true
				break
			}
		}

		if !hasAnyPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "FORBIDDEN",
				"details": gin.H{
					"required_any_of": permissions,
				},
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("permission_checked", true)
		c.Next()
	}
}

// RequireAuthentication only checks if user is authenticated (no permission check)
func RequireAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := extractUserIDFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or missing token",
				"code":  "UNAUTHORIZED",
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// extractUserIDFromToken extracts user ID from JWT token
func extractUserIDFromToken(c *gin.Context) (string, error) {
	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", jwt.ErrInvalidKey
	}

	// Remove "Bearer " prefix
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return "", jwt.ErrInvalidKey
	}

	// Parse JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Get JWT secret from config
		cfg := config.GetConfig()
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", jwt.ErrInvalidKey
	}

	// Extract user ID from claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if userID, exists := claims["user_id"]; exists {
			if userIDStr, ok := userID.(string); ok {
				return userIDStr, nil
			}
		}
	}

	return "", jwt.ErrInvalidKey
}

// PermissionDebug middleware for debugging permission checks
// add autdit logs or other debugging information
func PermissionDebug() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Log permission check result after request
		if checked, exists := c.Get("permission_checked"); exists && checked.(bool) {
			userID, _ := c.Get("user_id")
			resource, _ := c.Get("resource")
			action, _ := c.Get("action")

			// TODO: Add proper logging
			_ = userID
			_ = resource
			_ = action
		}
	}
}
