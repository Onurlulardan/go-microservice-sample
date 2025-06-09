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

// RoleResponse represents role data for API responses
type RoleResponse struct {
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	IsDefault      bool                 `json:"is_default"`
	Organization   *models.Organization `json:"organization,omitempty"`
	OrganizationID *uuid.UUID           `json:"organization_id"`
	CreatedAt      string               `json:"created_at"`
	UpdatedAt      string               `json:"updated_at"`
}

// CreateRoleRequest represents request body for creating role
type CreateRoleRequest struct {
	Name           string     `json:"name" binding:"required"`
	Description    string     `json:"description"`
	IsDefault      bool       `json:"is_default"`
	OrganizationID *uuid.UUID `json:"organization_id"`
}

// UpdateRoleRequest represents request body for updating role
type UpdateRoleRequest struct {
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	IsDefault      bool       `json:"is_default"`
	OrganizationID *uuid.UUID `json:"organization_id"`
}

// RoleListResponse represents a list of roles with pagination
type RoleListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []RoleResponse     `json:"items"`
		Pagination PaginationResponse `json:"pagination"`
	} `json:"data"`
}

// SingleRoleResponse represents a single role response
type SingleRoleResponse struct {
	Success bool         `json:"success"`
	Data    RoleResponse `json:"data"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetRoles retrieves all roles with pagination and filtering
// @Summary Get all roles
// @Description Get all roles with pagination, filtering, sorting and search
// @Tags roles
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param search query string false "Search term across name and description"
// @Param filters[organization_id] query string false "Filter by organization ID"
// @Param filters[is_default] query string false "Filter by default status (true, false)"
// @Param sort[field] query string false "Sort field (name, description, is_default, created_at, updated_at)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Security BearerAuth
// @Success 200 {object} handlers.RoleListResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /roles [get]
func GetRoles(ctx *gin.Context) {
	db := database.DB

	// Parse standardized query parameters
	params := query.ParseQueryParams(ctx)

	// Define allowed filter fields
	allowedFilters := map[string]string{
		"organization_id": "organization_id",
		"is_default":      "is_default",
	}

	// Define allowed sort fields
	allowedSortFields := map[string]string{
		"name":        "name",
		"description": "description",
		"is_default":  "is_default",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
	}

	// Define search fields
	searchFields := []string{"name", "description"}

	// Build base query
	baseQuery := db.Model(&models.Role{})

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

	// Get roles
	var roles []models.Role
	if err := finalQuery.Find(&roles).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve roles",
			"message": err.Error(),
		})
		return
	}

	// Convert to response format
	var roleResponses []RoleResponse
	for _, role := range roles {
		roleResponse := RoleResponse{
			ID:             role.ID,
			Name:           role.Name,
			Description:    role.Description,
			IsDefault:      role.IsDefault,
			OrganizationID: role.OrganizationID,
			CreatedAt:      role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Add organization if exists
		if role.OrganizationID != nil {
			var org models.Organization
			if err := db.First(&org, *role.OrganizationID).Error; err == nil {
				roleResponse.Organization = &org
			}
		}

		roleResponses = append(roleResponses, roleResponse)
	}

	// Build pagination response
	pagination := query.BuildPaginationResponse(params.Page, params.Limit, total)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"roles":      roleResponses,
			"pagination": pagination,
		},
	})
}

// GetRole retrieves a single role by ID
// @Summary Get role by ID
// @Description Get detailed information about a specific role
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} handlers.SingleRoleResponse
// @Failure 400 {object} map[string]string "Invalid role ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Role not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /roles/{id} [get]
func GetRole(ctx *gin.Context) {
	roleID := ctx.Param("id")
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid role ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	var role models.Role
	if err := db.Preload("Organization").First(&role, roleUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Role not found",
				"message": "Role with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve role",
			"message": err.Error(),
		})
		return
	}

	roleResponse := RoleResponse{
		ID:             role.ID,
		Name:           role.Name,
		Description:    role.Description,
		IsDefault:      role.IsDefault,
		OrganizationID: role.OrganizationID,
		CreatedAt:      role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add organization if exists
	if role.OrganizationID != nil {
		var org models.Organization
		if err := db.First(&org, *role.OrganizationID).Error; err == nil {
			roleResponse.Organization = &org
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    roleResponse,
	})
}

// CreateRole creates a new role
// @Summary Create a new role
// @Description Create a new role with the provided information
// @Tags roles
// @Accept json
// @Produce json
// @Param role body CreateRoleRequest true "Role information"
// @Security BearerAuth
// @Success 201 {object} handlers.SingleRoleResponse "Created role"
// @Failure 400 {object} map[string]string "Invalid request data or organization not found"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Role name already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /roles [post]
func CreateRole(ctx *gin.Context) {
	var req CreateRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if organization exists (if provided)
	if req.OrganizationID != nil {
		var org models.Organization
		if err := db.First(&org, *req.OrganizationID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Organization not found",
					"message": "The specified organization does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate organization",
				"message": err.Error(),
			})
			return
		}
	}

	// Check if role name already exists in the same organization
	var existingRole models.Role
	query := db.Where("name = ?", req.Name)
	if req.OrganizationID != nil {
		query = query.Where("organization_id = ?", *req.OrganizationID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	if err := query.First(&existingRole).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Role name already exists",
			"message": "A role with this name already exists in the specified organization",
		})
		return
	}

	// Create new role
	role := models.Role{
		Name:           req.Name,
		Description:    req.Description,
		IsDefault:      req.IsDefault,
		OrganizationID: req.OrganizationID,
	}

	if err := db.Create(&role).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create role",
			"message": err.Error(),
		})
		return
	}

	// Load organization relation
	db.Preload("Organization").First(&role, role.ID)

	roleResponse := RoleResponse{
		ID:             role.ID,
		Name:           role.Name,
		Description:    role.Description,
		IsDefault:      role.IsDefault,
		OrganizationID: role.OrganizationID,
		CreatedAt:      role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Role created successfully",
		"data":    roleResponse,
	})
}

// UpdateRole updates an existing role
// @Summary Update a role
// @Description Update an existing role's information
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID" format(uuid)
// @Param role body UpdateRoleRequest true "Updated role information"
// @Security BearerAuth
// @Success 200 {object} handlers.SingleRoleResponse "Updated role"
// @Failure 400 {object} map[string]string "Invalid request data or ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Role not found"
// @Failure 409 {object} map[string]string "Role name already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /roles/{id} [put]
func UpdateRole(ctx *gin.Context) {
	roleID := ctx.Param("id")
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid role ID format",
			"message": err.Error(),
		})
		return
	}

	var req UpdateRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if role exists
	var role models.Role
	if err := db.First(&role, roleUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Role not found",
				"message": "Role with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve role",
			"message": err.Error(),
		})
		return
	}

	// Check if organization exists (if provided)
	if req.OrganizationID != nil {
		var org models.Organization
		if err := db.First(&org, *req.OrganizationID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Organization not found",
					"message": "The specified organization does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate organization",
				"message": err.Error(),
			})
			return
		}
	}

	// Check if role name already exists (if name is being changed)
	if req.Name != "" && req.Name != role.Name {
		var existingRole models.Role
		query := db.Where("name = ? AND id != ?", req.Name, roleUUID)
		if req.OrganizationID != nil {
			query = query.Where("organization_id = ?", *req.OrganizationID)
		} else {
			query = query.Where("organization_id IS NULL")
		}

		if err := query.First(&existingRole).Error; err == nil {
			ctx.JSON(http.StatusConflict, gin.H{
				"error":   "Role name already exists",
				"message": "A role with this name already exists in the specified organization",
			})
			return
		}
	}

	// Update role fields
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	role.IsDefault = req.IsDefault
	if req.OrganizationID != nil {
		role.OrganizationID = req.OrganizationID
	}

	if err := db.Save(&role).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update role",
			"message": err.Error(),
		})
		return
	}

	// Load organization relation
	db.Preload("Organization").First(&role, role.ID)

	roleResponse := RoleResponse{
		ID:             role.ID,
		Name:           role.Name,
		Description:    role.Description,
		IsDefault:      role.IsDefault,
		OrganizationID: role.OrganizationID,
		CreatedAt:      role.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      role.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role updated successfully",
		"data":    roleResponse,
	})
}

// DeleteRole deletes a role (soft delete by setting inactive status)
// @Summary Delete a role
// @Description Delete a role if it's not being used by any users
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} handlers.SuccessResponse "Success message"
// @Failure 400 {object} map[string]string "Invalid role ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Role not found"
// @Failure 409 {object} map[string]string "Role is in use"
// @Failure 500 {object} map[string]string "Server error"
// @Router /roles/{id} [delete]
func DeleteRole(ctx *gin.Context) {
	roleID := ctx.Param("id")
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid role ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if role exists
	var role models.Role
	if err := db.First(&role, roleUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Role not found",
				"message": "Role with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve role",
			"message": err.Error(),
		})
		return
	}

	// Check if role is being used by any users
	var userCount int64
	db.Model(&models.User{}).Where("role_id = ?", roleUUID).Count(&userCount)
	if userCount > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Role is in use",
			"message": "Cannot delete role that is assigned to users",
		})
		return
	}

	// Delete the role
	if err := db.Delete(&role).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete role",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role deleted successfully",
	})
}

// GetRolePermissions retrieves all permissions for a specific role
// @Summary Get role permissions
// @Description Get all permissions assigned to a specific role including role-level and organization-level permissions
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Role permissions data"
// @Failure 400 {object} map[string]string "Invalid role ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Role not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /roles/{id}/permissions [get]
func GetRolePermissions(ctx *gin.Context) {
	roleID := ctx.Param("id")
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid role ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if role exists
	var role models.Role
	if err := db.Preload("Organization").First(&role, roleUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Role not found",
				"message": "Role with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve role",
			"message": err.Error(),
		})
		return
	}

	// Get role-level permissions
	var rolePermissions []models.Permission
	db.Preload("Resource").
		Preload("PermissionActions.Action").
		Where("target = ? AND role_id = ?", "ROLE", roleUUID).
		Find(&rolePermissions)

	// Get organization-level permissions if role has an organization
	var orgPermissions []models.Permission
	if role.OrganizationID != nil {
		db.Preload("Resource").
			Preload("PermissionActions.Action").
			Where("target = ? AND organization_id = ?", "ORGANIZATION", *role.OrganizationID).
			Find(&orgPermissions)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"role": role,
			"permissions": gin.H{
				"role_permissions": rolePermissions,
				"org_permissions":  orgPermissions,
			},
		},
	})
}
