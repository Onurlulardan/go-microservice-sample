package handlers

import (
	"net/http"

	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/utils/cache"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PermissionCheckRequest represents a single permission check request
type PermissionCheckRequest struct {
	UserID       string `json:"user_id" binding:"required"`
	ResourceSlug string `json:"resource_slug" binding:"required"`
	ActionSlug   string `json:"action_slug" binding:"required"`
}

// PermissionCheckResponse represents the response from permission check
type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// BatchPermissionCheckRequest represents batch permission check request
type BatchPermissionCheckRequest struct {
	UserID string                `json:"user_id" binding:"required"`
	Checks []ResourceActionCheck `json:"checks" binding:"required,min=1"`
}

type ResourceActionCheck struct {
	ResourceSlug string `json:"resource_slug" binding:"required"`
	ActionSlug   string `json:"action_slug" binding:"required"`
}

// BatchPermissionCheckResponse represents batch permission check response
type BatchPermissionCheckResponse struct {
	Results map[string]bool `json:"results"`
}

// CheckPermission checks if user has permission for specific resource and action
// @Summary Check single permission
// @Description Check if a user has permission for a specific resource and action
// @Tags permission-checks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param check body PermissionCheckRequest true "Permission check request"
// @Success 200 {object} PermissionCheckResponse "Permission check result"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Router /permissions/check [post]
func CheckPermission(c *gin.Context) {
	var req PermissionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check permission using 3-level hierarchy
	allowed, reason := checkPermissionHierarchy(userID, req.ResourceSlug, req.ActionSlug)

	response := PermissionCheckResponse{
		Allowed: allowed,
		Reason:  reason,
	}

	c.JSON(http.StatusOK, response)
}

// BatchCheckPermissions checks multiple permissions at once
// @Summary Check multiple permissions
// @Description Check multiple resource-action permissions for a user in a single request
// @Tags permission-checks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param batch body BatchPermissionCheckRequest true "Batch permission check request"
// @Success 200 {object} BatchPermissionCheckResponse "Batch permission check results"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Router /permissions/batch-check [post]
func BatchCheckPermissions(c *gin.Context) {
	var req BatchPermissionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	results := make(map[string]bool)

	// Check each permission
	for _, check := range req.Checks {
		key := check.ResourceSlug + ":" + check.ActionSlug
		allowed, _ := checkPermissionHierarchy(userID, check.ResourceSlug, check.ActionSlug)
		results[key] = allowed
	}

	response := BatchPermissionCheckResponse{
		Results: results,
	}

	c.JSON(http.StatusOK, response)
}

// checkPermissionHierarchy implements 3-level permission check logic with Redis cache
// Priority: 1. Cache lookup 2. User permissions 3. Role permissions 4. Organization permissions
func checkPermissionHierarchy(userID uuid.UUID, resourceSlug, actionSlug string) (bool, string) {
	userIDUint := uuidToUint(userID)

	// Try to get from cache first
	cacheManager := cache.GetCacheManager()
	if cacheManager != nil {
		if cacheData, found := cacheManager.GetPermissionCache(userIDUint, resourceSlug, actionSlug); found {
			return cacheData.HasPermission, "cached_" + cacheData.FoundAt
		}
	}

	db := database.GetDB()
	var allowed bool
	var foundAt string

	// 1. Check direct user permissions (highest priority)
	if hasDirectUserPermission(db, userID, resourceSlug, actionSlug) {
		allowed = true
		foundAt = "user"
	} else if hasRolePermission(db, userID, resourceSlug, actionSlug) {
		// 2. Check role-based permissions
		allowed = true
		foundAt = "role"
	} else if hasOrganizationPermission(db, userID, resourceSlug, actionSlug) {
		// 3. Check organization permissions (lowest priority)
		allowed = true
		foundAt = "organization"
	} else {
		allowed = false
		foundAt = "none"
	}

	// Cache the result if cache manager is available
	if cacheManager != nil {
		cacheData := &cache.PermissionCacheData{
			HasPermission: allowed,
			UserID:        userIDUint,
			Resource:      resourceSlug,
			Action:        actionSlug,
			FoundAt:       foundAt,
		}
		if err := cacheManager.SetPermissionCache(userIDUint, resourceSlug, actionSlug, cacheData); err != nil {
		}
	}

	if allowed {
		return true, foundAt + "_permission"
	}
	return false, "no_permission"
}

// uuidToUint converts UUID to uint for cache key
func uuidToUint(id uuid.UUID) uint {
	var hash uint32
	bytes := id[:]
	for i := 0; i < len(bytes); i += 4 {
		chunk := uint32(bytes[i])<<24 | uint32(bytes[i+1])<<16 | uint32(bytes[i+2])<<8 | uint32(bytes[i+3])
		hash ^= chunk
	}
	return uint(hash)
}

// hasDirectUserPermission checks if user has direct permission
func hasDirectUserPermission(db *gorm.DB, userID uuid.UUID, resourceSlug, actionSlug string) bool {
	var count int64

	// Check for specific resource permission or ALL resource permission
	err := db.Table("permissions p").
		Joins("JOIN resources r ON p.resource_id = r.id").
		Joins("JOIN permission_actions pa ON p.id = pa.permission_id").
		Joins("JOIN actions a ON pa.action_id = a.id").
		Where("p.target = ? AND p.user_id = ? AND (r.slug = ? OR r.slug = ?) AND a.slug = ?",
			"USER", userID, resourceSlug, "ALL", actionSlug).
		Count(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}

// hasRolePermission checks if user has permission through their role
func hasRolePermission(db *gorm.DB, userID uuid.UUID, resourceSlug, actionSlug string) bool {
	var count int64

	// Check for specific resource permission or ALL resource permission
	err := db.Table("permissions p").
		Joins("JOIN resources r ON p.resource_id = r.id").
		Joins("JOIN permission_actions pa ON p.id = pa.permission_id").
		Joins("JOIN actions a ON pa.action_id = a.id").
		Joins("JOIN users u ON p.role_id = u.role_id").
		Where("p.target = ? AND u.id = ? AND (r.slug = ? OR r.slug = ?) AND a.slug = ?",
			"ROLE", userID, resourceSlug, "ALL", actionSlug).
		Count(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}

// hasOrganizationPermission checks if user has permission through their organization
func hasOrganizationPermission(db *gorm.DB, userID uuid.UUID, resourceSlug, actionSlug string) bool {
	var count int64

	// Get user's organization first
	var user models.User
	if err := db.First(&user, "id = ?", userID).Error; err != nil {
		return false
	}

	if user.OrganizationID == nil {
		return false
	}

	// Check for specific resource permission or ALL resource permission
	err := db.Table("permissions p").
		Joins("JOIN resources r ON p.resource_id = r.id").
		Joins("JOIN permission_actions pa ON p.id = pa.permission_id").
		Joins("JOIN actions a ON pa.action_id = a.id").
		Where("p.target = ? AND p.organization_id = ? AND (r.slug = ? OR r.slug = ?) AND a.slug = ?",
			"ORGANIZATION", *user.OrganizationID, resourceSlug, "ALL", actionSlug).
		Count(&count).Error

	if err != nil {
		return false
	}

	return count > 0
}
