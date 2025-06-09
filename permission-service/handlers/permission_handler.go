package handlers

import (
	"net/http"

	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/utils/query"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreatePermissionRequest represents the request body for creating a permission
type CreatePermissionRequest struct {
	ResourceID     uuid.UUID   `json:"resource_id" binding:"required"`
	Target         string      `json:"target" binding:"required,oneof=USER ROLE ORGANIZATION"`
	UserID         *uuid.UUID  `json:"user_id,omitempty"`
	RoleID         *uuid.UUID  `json:"role_id,omitempty"`
	OrganizationID *uuid.UUID  `json:"organization_id,omitempty"`
	ActionIDs      []uuid.UUID `json:"action_ids" binding:"required,min=1"`
}

// UpdatePermissionRequest represents the request body for updating a permission
type UpdatePermissionRequest struct {
	ResourceID     *uuid.UUID  `json:"resource_id,omitempty"`
	Target         *string     `json:"target,omitempty"`
	UserID         *uuid.UUID  `json:"user_id,omitempty"`
	RoleID         *uuid.UUID  `json:"role_id,omitempty"`
	OrganizationID *uuid.UUID  `json:"organization_id,omitempty"`
	ActionIDs      []uuid.UUID `json:"action_ids,omitempty"`
}

// PermissionResponse represents the response structure with actions included
type PermissionResponse struct {
	models.Permission
	Actions []models.Action `json:"actions"`
}

// Resource represents a resource in the system
type Resource struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// Action represents an action in the system
type Action struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// Permission represents a permission in the system
type Permission struct {
	ID                uuid.UUID          `json:"id"`
	Target            string             `json:"target"`
	ResourceID        uuid.UUID          `json:"resource_id"`
	UserID            *uuid.UUID         `json:"user_id,omitempty"`
	RoleID            *uuid.UUID         `json:"role_id,omitempty"`
	OrganizationID    *uuid.UUID         `json:"organization_id,omitempty"`
	Resource          Resource           `json:"resource"`
	Actions           []Action           `json:"actions"`
	CreatedAt         string             `json:"created_at"`
	UpdatedAt         string             `json:"updated_at"`
}

// PaginationResponse represents pagination information
type PaginationResponse struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// PermissionListResponse represents a list of permissions with pagination
type PermissionListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []Permission       `json:"items"`
		Pagination PaginationResponse `json:"pagination"`
	} `json:"data"`
}

// SinglePermissionResponse represents a single permission response
type SinglePermissionResponse struct {
	Success bool       `json:"success"`
	Data    Permission `json:"data"`
}

// CreatePermission creates a new permission with associated actions
// @Summary Create a new permission
// @Description Create a new permission with associated actions
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param permission body CreatePermissionRequest true "Permission data"
// @Success 201 {object} handlers.SinglePermissionResponse "Created permission"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 404 {object} map[string]string "Resource or action not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions [post]
func CreatePermission(c *gin.Context) {
	var req CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate target-specific requirements
	if err := validatePermissionTarget(req.Target, req.UserID, req.RoleID, req.OrganizationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid target configuration",
			"details": err.Error(),
		})
		return
	}

	db := database.GetDB()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Verify resource exists
	var resource models.Resource
	if err := tx.First(&resource, "id = ?", req.ResourceID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Verify all actions exist
	var actions []models.Action
	if err := tx.Find(&actions, "id IN ?", req.ActionIDs).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if len(actions) != len(req.ActionIDs) {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "One or more actions not found"})
		return
	}

	// Create permission
	permission := models.Permission{
		ResourceID:     req.ResourceID,
		Target:         req.Target,
		UserID:         req.UserID,
		RoleID:         req.RoleID,
		OrganizationID: req.OrganizationID,
	}

	if err := tx.Create(&permission).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create permission",
			"details": err.Error(),
		})
		return
	}

	// Create permission actions
	for _, actionID := range req.ActionIDs {
		permissionAction := models.PermissionAction{
			PermissionID: permission.ID,
			ActionID:     actionID,
		}
		if err := tx.Create(&permissionAction).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to create permission actions",
				"details": err.Error(),
			})
			return
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Fetch created permission with relations for response
	var createdPermission models.Permission
	db.Preload("Resource").
		Preload("User").
		Preload("Role").
		Preload("Organization").
		First(&createdPermission, "id = ?", permission.ID)

	// Get associated actions
	var permissionActions []models.PermissionAction
	db.Preload("Action").Find(&permissionActions, "permission_id = ?", permission.ID)

	var responseActions []models.Action
	for _, pa := range permissionActions {
		responseActions = append(responseActions, pa.Action)
	}

	response := PermissionResponse{
		Permission: createdPermission,
		Actions:    responseActions,
	}

	c.JSON(http.StatusCreated, response)
}

// GetPermissions retrieves all permissions with optional filtering
// @Summary Get all permissions
// @Description Get all permissions with pagination, filtering, sorting, and search
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Results per page (default: 10)"
// @Param filters[target] query string false "Filter by target (USER, ROLE, ORGANIZATION)"
// @Param filters[resource_id] query string false "Filter by resource ID"
// @Param filters[user_id] query string false "Filter by user ID"
// @Param filters[role_id] query string false "Filter by role ID"
// @Param filters[organization_id] query string false "Filter by organization ID"
// @Param sort[field] query string false "Sort field (target, created_at, updated_at)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Param search query string false "Search term"
// @Success 200 {object} handlers.PermissionListResponse "List of permissions"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions [get]
func GetPermissions(c *gin.Context) {
	db := database.GetDB()

	// Parse standardized query parameters
	params := query.ParseQueryParams(c)

	// Define allowed filter fields (frontend field -> database field mapping)
	allowedFilters := map[string]string{
		"target":          "target",
		"resource_id":     "resource_id",
		"user_id":         "user_id",
		"role_id":         "role_id",
		"organization_id": "organization_id",
	}

	// Define allowed sort fields
	allowedSortFields := map[string]string{
		"target":     "target",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	// Define search fields
	searchFields := []string{"target"}

	// Build base query
	baseQuery := db.Model(&models.Permission{}).
		Preload("Resource").
		Preload("User").
		Preload("Role").
		Preload("Organization")

	// Apply filters
	filteredQuery := query.ApplyFilters(baseQuery, params.Filters, allowedFilters)

	// Apply search
	searchedQuery := query.ApplySearch(filteredQuery, params.Search, searchFields)

	// Get total count
	var total int64
	searchedQuery.Count(&total)

	// Apply sorting and pagination
	finalQuery := query.ApplySort(searchedQuery, params.Sort, allowedSortFields)
	finalQuery = query.ApplyPagination(finalQuery, params.Page, params.Limit)

	// Get permissions
	var permissions []models.Permission
	if err := finalQuery.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get actions for each permission
	var responses []PermissionResponse
	for _, permission := range permissions {
		var permissionActions []models.PermissionAction
		db.Preload("Action").Find(&permissionActions, "permission_id = ?", permission.ID)

		var actions []models.Action
		for _, pa := range permissionActions {
			actions = append(actions, pa.Action)
		}

		responses = append(responses, PermissionResponse{
			Permission: permission,
			Actions:    actions,
		})
	}

	// Build pagination response
	pagination := query.BuildPaginationResponse(params.Page, params.Limit, total)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      responses,
			"pagination": pagination,
		},
	})
}

// GetPermission retrieves a single permission by ID
// @Summary Get a permission by ID
// @Description Get detailed information about a specific permission
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Permission ID" format(uuid)
// @Success 200 {object} handlers.SinglePermissionResponse "Permission details"
// @Failure 400 {object} map[string]string "Invalid permission ID"
// @Failure 404 {object} map[string]string "Permission not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions/{id} [get]
func GetPermission(c *gin.Context) {
	id := c.Param("id")
	permissionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	db := database.GetDB()

	var permission models.Permission
	if err := db.Preload("Resource").
		Preload("User").
		Preload("Role").
		Preload("Organization").
		First(&permission, "id = ?", permissionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Get associated actions
	var permissionActions []models.PermissionAction
	db.Preload("Action").Find(&permissionActions, "permission_id = ?", permission.ID)

	var actions []models.Action
	for _, pa := range permissionActions {
		actions = append(actions, pa.Action)
	}

	response := PermissionResponse{
		Permission: permission,
		Actions:    actions,
	}

	c.JSON(http.StatusOK, response)
}

// UpdatePermission updates an existing permission
// UpdatePermission updates a permission by ID
// @Summary Update a permission
// @Description Update an existing permission
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Permission ID" format(uuid)
// @Param permission body UpdatePermissionRequest true "Updated permission data"
// @Success 200 {object} handlers.SinglePermissionResponse "Updated permission"
// @Failure 400 {object} map[string]interface{} "Invalid request format or validation error"
// @Failure 404 {object} map[string]string "Permission, resource, or action not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions/{id} [put]
func UpdatePermission(c *gin.Context) {
	id := c.Param("id")
	permissionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	var req UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	db := database.GetDB()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if permission exists
	var permission models.Permission
	if err := tx.First(&permission, "id = ?", permissionID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Update permission fields
	updates := make(map[string]interface{})

	if req.ResourceID != nil {
		// Verify resource exists
		var resource models.Resource
		if err := tx.First(&resource, "id = ?", *req.ResourceID).Error; err != nil {
			tx.Rollback()
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			}
			return
		}
		updates["resource_id"] = *req.ResourceID
	}

	if req.Target != nil {
		// Validate target with current/new IDs
		targetUserID := req.UserID
		targetRoleID := req.RoleID
		targetOrgID := req.OrganizationID

		// Use existing values if not provided in update
		if targetUserID == nil {
			targetUserID = permission.UserID
		}
		if targetRoleID == nil {
			targetRoleID = permission.RoleID
		}
		if targetOrgID == nil {
			targetOrgID = permission.OrganizationID
		}

		if err := validatePermissionTarget(*req.Target, targetUserID, targetRoleID, targetOrgID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid target configuration",
				"details": err.Error(),
			})
			return
		}
		updates["target"] = *req.Target
	}

	if req.UserID != nil {
		updates["user_id"] = *req.UserID
	}
	if req.RoleID != nil {
		updates["role_id"] = *req.RoleID
	}
	if req.OrganizationID != nil {
		updates["organization_id"] = *req.OrganizationID
	}

	// Update permission
	if len(updates) > 0 {
		if err := tx.Model(&permission).Updates(updates).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to update permission",
				"details": err.Error(),
			})
			return
		}
	}

	// Update actions if provided
	if len(req.ActionIDs) > 0 {
		// Verify all actions exist
		var actions []models.Action
		if err := tx.Find(&actions, "id IN ?", req.ActionIDs).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		if len(actions) != len(req.ActionIDs) {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "One or more actions not found"})
			return
		}

		// Delete existing permission actions
		if err := tx.Delete(&models.PermissionAction{}, "permission_id = ?", permissionID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permission actions"})
			return
		}

		// Create new permission actions
		for _, actionID := range req.ActionIDs {
			permissionAction := models.PermissionAction{
				PermissionID: permissionID,
				ActionID:     actionID,
			}
			if err := tx.Create(&permissionAction).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to create permission actions",
					"details": err.Error(),
				})
				return
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Fetch updated permission with relations for response
	var updatedPermission models.Permission
	db.Preload("Resource").
		Preload("User").
		Preload("Role").
		Preload("Organization").
		First(&updatedPermission, "id = ?", permissionID)

	// Get associated actions
	var permissionActions []models.PermissionAction
	db.Preload("Action").Find(&permissionActions, "permission_id = ?", permissionID)

	var responseActions []models.Action
	for _, pa := range permissionActions {
		responseActions = append(responseActions, pa.Action)
	}

	response := PermissionResponse{
		Permission: updatedPermission,
		Actions:    responseActions,
	}

	c.JSON(http.StatusOK, response)
}

// DeletePermission deletes a permission and its associated actions
// DeletePermission deletes a permission by ID
// @Summary Delete a permission
// @Description Delete a permission and its associated actions
// @Tags permissions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Permission ID" format(uuid)
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid permission ID"
// @Failure 404 {object} map[string]string "Permission not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions/{id} [delete]
func DeletePermission(c *gin.Context) {
	id := c.Param("id")
	permissionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	db := database.GetDB()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if permission exists
	var permission models.Permission
	if err := tx.First(&permission, "id = ?", permissionID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Permission not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Delete associated permission actions first
	if err := tx.Delete(&models.PermissionAction{}, "permission_id = ?", permissionID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission actions"})
		return
	}

	// Delete permission
	if err := tx.Delete(&permission).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// Helper function to validate permission target configuration
func validatePermissionTarget(target string, userID, roleID, organizationID *uuid.UUID) error {
	switch target {
	case "USER":
		if userID == nil {
			return &ValidationError{Field: "user_id", Message: "user_id is required for USER target"}
		}
		if roleID != nil || organizationID != nil {
			return &ValidationError{Field: "target", Message: "only user_id should be set for USER target"}
		}
	case "ROLE":
		if roleID == nil {
			return &ValidationError{Field: "role_id", Message: "role_id is required for ROLE target"}
		}
		if userID != nil || organizationID != nil {
			return &ValidationError{Field: "target", Message: "only role_id should be set for ROLE target"}
		}
	case "ORGANIZATION":
		if organizationID == nil {
			return &ValidationError{Field: "organization_id", Message: "organization_id is required for ORGANIZATION target"}
		}
		if userID != nil || roleID != nil {
			return &ValidationError{Field: "target", Message: "only organization_id should be set for ORGANIZATION target"}
		}
	default:
		return &ValidationError{Field: "target", Message: "target must be USER, ROLE, or ORGANIZATION"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
