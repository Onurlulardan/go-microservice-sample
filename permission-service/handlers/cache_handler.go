package handlers

import (
	"net/http"
	"strconv"

	"forgecrud-backend/shared/utils/cache"

	"github.com/gin-gonic/gin"
)

// GetCacheStats returns cache statistics
// @Summary Get cache statistics
// @Description Get statistics about the permission cache
// @Tags cache
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Cache statistics"
// @Failure 503 {object} map[string]string "Cache manager not available"
// @Failure 500 {object} map[string]interface{} "Failed to get cache stats"
// @Router /permissions/cache/stats [get]
func GetCacheStats(c *gin.Context) {
	cacheManager := cache.GetCacheManager()
	if cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	stats, err := cacheManager.GetCacheStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get cache stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cache_stats": stats,
		"service":     "permission",
	})
}

// InvalidateUserPermissions invalidates all permissions for a specific user
// @Summary Invalidate user permissions cache
// @Description Invalidate all cached permissions for a specific user
// @Tags cache
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{} "Success message"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 500 {object} map[string]interface{} "Failed to invalidate cache"
// @Failure 503 {object} map[string]string "Cache manager not available"
// @Router /permissions/cache/invalidate/{user_id} [post]
func InvalidateUserPermissions(c *gin.Context) {
	userIDStr := c.Param("user_id")
	cacheManager := cache.GetCacheManager()
	if cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	// Parse user ID
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user ID",
			"details": "User ID must be a valid number",
		})
		return
	}

	// Invalidate all permissions for this user
	if err := cacheManager.InvalidateUserPermissions(uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to invalidate user permissions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User permissions cache invalidated successfully",
		"user_id": userID,
	})
}

// InvalidateRolePermissions invalidates all permissions for a specific role
// @Summary Invalidate role permissions cache
// @Description Invalidate all cached permissions for a specific role
// @Tags cache
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param role_id path string true "Role ID"
// @Success 200 {object} map[string]interface{} "Success message"
// @Failure 400 {object} map[string]interface{} "Invalid role ID"
// @Failure 500 {object} map[string]interface{} "Failed to invalidate cache"
// @Failure 503 {object} map[string]string "Cache manager not available"
// @Router /permissions/cache/invalidate/role/{role_id} [post]
func InvalidateRolePermissions(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	cacheManager := cache.GetCacheManager()
	if cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid role ID",
			"details": "Role ID must be a valid number",
		})
		return
	}

	if err := cacheManager.InvalidateRolePermissions(uint(roleID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to invalidate role permissions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role permissions cache invalidated successfully",
		"role_id": roleID,
	})
}

// InvalidateOrgPermissions invalidates all permissions for a specific organization
// @Summary Invalidate organization permissions cache
// @Description Invalidate all cached permissions for a specific organization
// @Tags cache
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param org_id path string true "Organization ID"
// @Success 200 {object} map[string]interface{} "Success message"
// @Failure 400 {object} map[string]interface{} "Invalid organization ID"
// @Failure 500 {object} map[string]interface{} "Failed to invalidate cache"
// @Failure 503 {object} map[string]string "Cache manager not available"
// @Router /permissions/cache/invalidate/org/{org_id} [post]
func InvalidateOrgPermissions(c *gin.Context) {
	orgIDStr := c.Param("org_id")
	cacheManager := cache.GetCacheManager()
	if cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	orgID, err := strconv.ParseUint(orgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid organization ID",
			"details": "Organization ID must be a valid number",
		})
		return
	}

	if err := cacheManager.InvalidateOrgPermissions(uint(orgID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to invalidate organization permissions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization permissions cache invalidated successfully",
		"org_id":  orgID,
	})
}

// InvalidateAllPermissions invalidates all permission caches
// @Summary Invalidate all permissions cache
// @Description Invalidate all cached permissions across the system
// @Tags cache
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Success message"
// @Failure 500 {object} map[string]interface{} "Failed to invalidate cache"
// @Failure 503 {object} map[string]string "Cache manager not available"
// @Router /permissions/cache/invalidate/all [post]
func InvalidateAllPermissions(c *gin.Context) {
	cacheManager := cache.GetCacheManager()
	if cacheManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Cache manager not available",
		})
		return
	}

	if err := cacheManager.InvalidateAllPermissions(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to invalidate all permissions",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All permissions cache invalidated successfully",
	})
}
