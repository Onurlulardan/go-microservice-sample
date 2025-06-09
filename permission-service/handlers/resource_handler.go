package handlers

import (
	"net/http"
	"strings"

	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/utils/query"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateResourceRequest represents the request body for creating a resource
type CreateResourceRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// UpdateResourceRequest represents the request body for updating a resource
type UpdateResourceRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// ResourceResponse represents a resource in the system
type ResourceResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// ResourceListResponse represents a list of resources with pagination
type ResourceListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []ResourceResponse `json:"items"`
		Pagination PaginationResponse `json:"pagination"`
	} `json:"data"`
}

// SingleResourceResponse represents a single resource response
type SingleResourceResponse struct {
	Success bool             `json:"success"`
	Data    ResourceResponse `json:"data"`
}

// CreateResource creates a new resource
// @Summary Create a new resource
// @Description Create a new resource for permissions
// @Tags resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param resource body CreateResourceRequest true "Resource data"
// @Success 201 {object} handlers.SingleResourceResponse "Created resource"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 409 {object} map[string]string "Resource with this slug already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /permissions/resources [post]
func CreateResource(c *gin.Context) {
	var req CreateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Slug == "" {
		req.Slug = generateSlug(req.Name)
	}

	// Validate slug uniqueness
	var existingResource models.Resource
	if err := database.DB.Where("slug = ?", req.Slug).First(&existingResource).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Resource with this slug already exists",
		})
		return
	}

	resource := models.Resource{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := database.DB.Create(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create resource",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Resource created successfully",
		"resource": resource,
	})
}

// GetResources returns a list of all resources with pagination
// @Summary Get all resources
// @Description Get all resources with pagination, filtering, sorting, and search
// @Tags resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Results per page (default: 10)"
// @Param filters[name] query string false "Filter by name"
// @Param filters[slug] query string false "Filter by slug"
// @Param sort[field] query string false "Sort field (name, slug, created_at, updated_at)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Param search query string false "Search term"
// @Success 200 {object} handlers.ResourceListResponse "List of resources"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions/resources [get]
func GetResources(c *gin.Context) {
	db := database.DB

	// Parse standardized query parameters
	params := query.ParseQueryParams(c)

	// Define allowed filter fields
	allowedFilters := map[string]string{
		"name": "name",
		"slug": "slug",
	}

	// Define allowed sort fields
	allowedSortFields := map[string]string{
		"name":       "name",
		"slug":       "slug",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	// Define search fields
	searchFields := []string{"name", "slug", "description"}

	// Build base query
	baseQuery := db.Model(&models.Resource{})

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

	// Get resources
	var resources []models.Resource
	if err := finalQuery.Find(&resources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch resources",
			"details": err.Error(),
		})
		return
	}

	// Build pagination response
	pagination := query.BuildPaginationResponse(params.Page, params.Limit, total)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      resources,
			"pagination": pagination,
		},
	})
}

// GetResource returns a single resource by ID
// @Summary Get a resource by ID
// @Description Get detailed information about a specific resource
// @Tags resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} handlers.SingleResourceResponse "Resource details"
// @Failure 400 {object} map[string]string "Invalid resource ID format"
// @Failure 404 {object} map[string]string "Resource not found"
// @Router /permissions/resources/{id} [get]
func GetResource(c *gin.Context) {
	id := c.Param("id")

	resourceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid resource ID format",
		})
		return
	}

	var resource models.Resource
	if err := database.DB.First(&resource, resourceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Resource not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resource": resource,
	})
}

// UpdateResource updates an existing resource
// @Summary Update a resource
// @Description Update an existing resource's details
// @Tags resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resource ID" format(uuid)
// @Param resource body UpdateResourceRequest true "Updated resource data"
// @Success 200 {object} handlers.SingleResourceResponse "Updated resource"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 404 {object} map[string]string "Resource not found"
// @Failure 409 {object} map[string]string "Resource with this slug already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /permissions/resources/{id} [put]
func UpdateResource(c *gin.Context) {
	id := c.Param("id")

	resourceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid resource ID format",
		})
		return
	}

	var req UpdateResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	var resource models.Resource
	if err := database.DB.First(&resource, resourceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Resource not found",
		})
		return
	}

	// Check if it's a system resource and prevent modification of critical fields
	if resource.IsSystem {
		// System resources can only have their description updated
		if req.Name != "" || req.Slug != "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Cannot modify system resource",
				"message": "System resources name and slug cannot be modified. Only description can be updated.",
			})
			return
		}
	}

	if req.Name != "" {
		resource.Name = req.Name
	}
	if req.Slug != "" {
		var existingResource models.Resource
		if err := database.DB.Where("slug = ? AND id != ?", req.Slug, resourceID).First(&existingResource).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Resource with this slug already exists",
			})
			return
		}
		resource.Slug = req.Slug
	}
	if req.Description != "" {
		resource.Description = req.Description
	}

	if err := database.DB.Save(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update resource",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Resource updated successfully",
		"resource": resource,
	})
}

// DeleteResource deletes a resource by ID
// @Summary Delete a resource
// @Description Delete a resource if it's not being used in any permissions
// @Tags resources
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid resource ID format"
// @Failure 404 {object} map[string]string "Resource not found"
// @Failure 409 {object} map[string]interface{} "Resource is being used in permissions"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /permissions/resources/{id} [delete]
func DeleteResource(c *gin.Context) {
	id := c.Param("id")

	resourceID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid resource ID format",
		})
		return
	}

	var resource models.Resource
	if err := database.DB.First(&resource, resourceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Resource not found",
		})
		return
	}

	// Check if it's a system resource
	if resource.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Cannot delete system resource",
			"message": "System resources are protected and cannot be deleted",
		})
		return
	}

	var permissionCount int64
	database.DB.Model(&models.Permission{}).Where("resource_id = ?", resourceID).Count(&permissionCount)
	if permissionCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Cannot delete resource",
			"message": "Resource is being used in permissions",
			"count":   permissionCount,
		})
		return
	}

	if err := database.DB.Delete(&resource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete resource",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Resource deleted successfully",
	})
}

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}
