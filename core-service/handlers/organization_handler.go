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

// OrganizationResponse represents organization data for API responses
type OrganizationResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Status    string     `json:"status"`
	OwnerID   uuid.UUID  `json:"owner_id"`
	ParentID  *uuid.UUID `json:"parent_id"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

// CreateOrganizationRequest represents request body for creating organization
type CreateOrganizationRequest struct {
	Name     string     `json:"name" binding:"required"`
	Slug     string     `json:"slug" binding:"required"`
	Status   string     `json:"status"`
	OwnerID  uuid.UUID  `json:"owner_id" binding:"required"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// UpdateOrganizationRequest represents request body for updating organization
type UpdateOrganizationRequest struct {
	Name     string     `json:"name"`
	Slug     string     `json:"slug"`
	Status   string     `json:"status"`
	OwnerID  *uuid.UUID `json:"owner_id"`
	ParentID *uuid.UUID `json:"parent_id"`
}

// OrganizationListResponse represents a list of organizations with pagination
type OrganizationListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []OrganizationResponse `json:"items"`
		Pagination PaginationResponse     `json:"pagination"`
	} `json:"data"`
}

// SingleOrganizationResponse represents a single organization response
type SingleOrganizationResponse struct {
	Success bool                 `json:"success"`
	Data    OrganizationResponse `json:"data"`
}

// GetOrganizations retrieves all organizations with pagination and filtering
// @Summary Get all organizations
// @Description Get all organizations with pagination, filtering, sorting and search
// @Tags organizations
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param search query string false "Search term across name and slug"
// @Param filters[status] query string false "Filter by status (ACTIVE, INACTIVE)"
// @Param filters[owner_id] query string false "Filter by owner ID"
// @Param filters[parent_id] query string false "Filter by parent organization ID"
// @Param sort[field] query string false "Sort field (name, slug, status, created_at, updated_at)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Security BearerAuth
// @Success 200 {object} handlers.OrganizationListResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /organizations [get]
func GetOrganizations(ctx *gin.Context) {
	db := database.DB

	// Parse query parameters using shared utility
	params := query.ParseQueryParams(ctx)

	// Define allowed filter fields
	allowedFilters := map[string]string{
		"status":    "status",
		"owner_id":  "owner_id",
		"parent_id": "parent_id",
	}

	// Define allowed sort fields
	allowedSortFields := map[string]string{
		"name":       "name",
		"slug":       "slug",
		"status":     "status",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	// Define search fields
	searchFields := []string{"name", "slug"}

	// Build query
	dbQuery := db.Model(&models.Organization{})

	// Apply filters, search, sorting, and pagination
	dbQuery = query.ApplyFilters(dbQuery, params.Filters, allowedFilters)
	dbQuery = query.ApplySearch(dbQuery, params.Search, searchFields)
	dbQuery = query.ApplySort(dbQuery, params.Sort, allowedSortFields)

	// Get total count before pagination
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to count organizations",
			"message": err.Error(),
		})
		return
	}

	// Apply pagination
	dbQuery = query.ApplyPagination(dbQuery, params.Page, params.Limit)

	// Get organizations
	var organizations []models.Organization
	if err := dbQuery.Find(&organizations).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve organizations",
			"message": err.Error(),
		})
		return
	}

	// Convert to response format
	var orgResponses []OrganizationResponse
	for _, org := range organizations {
		orgResponse := OrganizationResponse{
			ID:        org.ID,
			Name:      org.Name,
			Slug:      org.Slug,
			Status:    org.Status,
			OwnerID:   org.OwnerID,
			ParentID:  org.ParentID,
			CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		orgResponses = append(orgResponses, orgResponse)
	}

	// Build pagination response
	pagination := query.BuildPaginationResponse(params.Page, params.Limit, total)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      orgResponses,
			"pagination": pagination,
		},
	})
}

// GetOrganization retrieves a single organization by ID
// @Summary Get organization by ID
// @Description Get detailed information about a specific organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} handlers.SingleOrganizationResponse
// @Failure 400 {object} map[string]string "Invalid organization ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Organization not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /organizations/{id} [get]
func GetOrganization(ctx *gin.Context) {
	orgID := ctx.Param("id")
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid organization ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	var org models.Organization
	if err := db.First(&org, orgUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Organization not found",
				"message": "Organization with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve organization",
			"message": err.Error(),
		})
		return
	}

	orgResponse := OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    org.Status,
		OwnerID:   org.OwnerID,
		ParentID:  org.ParentID,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    orgResponse,
	})
}

// CreateOrganization creates a new organization
// @Summary Create a new organization
// @Description Create a new organization with the provided information
// @Tags organizations
// @Accept json
// @Produce json
// @Param organization body CreateOrganizationRequest true "Organization information"
// @Security BearerAuth
// @Success 201 {object} handlers.SingleOrganizationResponse "Created organization"
// @Failure 400 {object} map[string]string "Invalid request data or owner/parent not found"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Slug already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /organizations [post]
func CreateOrganization(ctx *gin.Context) {
	var req CreateOrganizationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if owner exists
	var owner models.User
	if err := db.First(&owner, req.OwnerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Owner not found",
				"message": "The specified owner does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to validate owner",
			"message": err.Error(),
		})
		return
	}

	// Check if parent organization exists (if provided)
	if req.ParentID != nil {
		var parentOrg models.Organization
		if err := db.First(&parentOrg, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Parent organization not found",
					"message": "The specified parent organization does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate parent organization",
				"message": err.Error(),
			})
			return
		}
	}

	// Check if slug already exists
	var existingOrg models.Organization
	if err := db.Where("slug = ?", req.Slug).First(&existingOrg).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Slug already exists",
			"message": "An organization with this slug already exists",
		})
		return
	}

	// Set default status if not provided
	if req.Status == "" {
		req.Status = "ACTIVE"
	}

	// Create new organization
	org := models.Organization{
		Name:     req.Name,
		Slug:     req.Slug,
		Status:   req.Status,
		OwnerID:  req.OwnerID,
		ParentID: req.ParentID,
	}

	if err := db.Create(&org).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create organization",
			"message": err.Error(),
		})
		return
	}

	orgResponse := OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    org.Status,
		OwnerID:   org.OwnerID,
		ParentID:  org.ParentID,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Organization created successfully",
		"data":    orgResponse,
	})
}

// UpdateOrganization updates an existing organization
// @Summary Update an organization
// @Description Update an existing organization's information
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Param organization body UpdateOrganizationRequest true "Updated organization information"
// @Security BearerAuth
// @Success 200 {object} handlers.SingleOrganizationResponse "Updated organization"
// @Failure 400 {object} map[string]string "Invalid request data or ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Organization not found"
// @Failure 409 {object} map[string]string "Slug already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /organizations/{id} [put]
func UpdateOrganization(ctx *gin.Context) {
	orgID := ctx.Param("id")
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid organization ID format",
			"message": err.Error(),
		})
		return
	}

	var req UpdateOrganizationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if organization exists
	var org models.Organization
	if err := db.First(&org, orgUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Organization not found",
				"message": "Organization with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve organization",
			"message": err.Error(),
		})
		return
	}

	// Check if owner exists (if provided)
	if req.OwnerID != nil {
		var owner models.User
		if err := db.First(&owner, *req.OwnerID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Owner not found",
					"message": "The specified owner does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate owner",
				"message": err.Error(),
			})
			return
		}
	}

	// Check if parent organization exists (if provided)
	if req.ParentID != nil {
		var parentOrg models.Organization
		if err := db.First(&parentOrg, *req.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Parent organization not found",
					"message": "The specified parent organization does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate parent organization",
				"message": err.Error(),
			})
			return
		}
	}

	// Check if slug already exists (if slug is being changed)
	if req.Slug != "" && req.Slug != org.Slug {
		var existingOrg models.Organization
		if err := db.Where("slug = ? AND id != ?", req.Slug, orgUUID).First(&existingOrg).Error; err == nil {
			ctx.JSON(http.StatusConflict, gin.H{
				"error":   "Slug already exists",
				"message": "An organization with this slug already exists",
			})
			return
		}
	}

	// Update organization fields
	if req.Name != "" {
		org.Name = req.Name
	}
	if req.Slug != "" {
		org.Slug = req.Slug
	}
	if req.Status != "" {
		org.Status = req.Status
	}
	if req.OwnerID != nil {
		org.OwnerID = *req.OwnerID
	}
	if req.ParentID != nil {
		org.ParentID = req.ParentID
	}

	if err := db.Save(&org).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update organization",
			"message": err.Error(),
		})
		return
	}

	orgResponse := OrganizationResponse{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    org.Status,
		OwnerID:   org.OwnerID,
		ParentID:  org.ParentID,
		CreatedAt: org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization updated successfully",
		"data":    orgResponse,
	})
}

// DeleteOrganization deletes an organization (soft delete by setting inactive status)
// @Summary Delete an organization
// @Description Delete an organization if it has no child organizations, users, or roles
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} handlers.SuccessResponse "Success message"
// @Failure 400 {object} map[string]string "Invalid organization ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Organization not found"
// @Failure 409 {object} map[string]string "Organization has dependencies"
// @Failure 500 {object} map[string]string "Server error"
// @Router /organizations/{id} [delete]
func DeleteOrganization(ctx *gin.Context) {
	orgID := ctx.Param("id")
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid organization ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if organization exists
	var org models.Organization
	if err := db.First(&org, orgUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Organization not found",
				"message": "Organization with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve organization",
			"message": err.Error(),
		})
		return
	}

	// Check if organization has child organizations
	var childCount int64
	db.Model(&models.Organization{}).Where("parent_id = ?", orgUUID).Count(&childCount)
	if childCount > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Organization has child organizations",
			"message": "Cannot delete organization that has child organizations",
		})
		return
	}

	// Check if organization has users
	var userCount int64
	db.Model(&models.User{}).Where("organization_id = ?", orgUUID).Count(&userCount)
	if userCount > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Organization has users",
			"message": "Cannot delete organization that has users",
		})
		return
	}

	// Check if organization has roles
	var roleCount int64
	db.Model(&models.Role{}).Where("organization_id = ?", orgUUID).Count(&roleCount)
	if roleCount > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Organization has roles",
			"message": "Cannot delete organization that has roles",
		})
		return
	}

	// Delete the organization
	if err := db.Delete(&org).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete organization",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Organization deleted successfully",
	})
}

// GetOrganizationPermissions retrieves all permissions for a specific organization
// @Summary Get organization permissions
// @Description Get all permissions assigned to a specific organization
// @Tags organizations
// @Accept json
// @Produce json
// @Param id path string true "Organization ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Organization permissions data"
// @Failure 400 {object} map[string]string "Invalid organization ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Organization not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /organizations/{id}/permissions [get]
func GetOrganizationPermissions(ctx *gin.Context) {
	orgID := ctx.Param("id")
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid organization ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if organization exists
	var org models.Organization
	if err := db.First(&org, orgUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Organization not found",
				"message": "Organization with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve organization",
			"message": err.Error(),
		})
		return
	}

	// Get organization-level permissions
	var orgPermissions []models.Permission
	db.Preload("Resource").
		Preload("PermissionActions.Action").
		Where("target = ? AND organization_id = ?", "ORGANIZATION", orgUUID).
		Find(&orgPermissions)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"organization": org,
			"permissions":  orgPermissions,
		},
	})
}
