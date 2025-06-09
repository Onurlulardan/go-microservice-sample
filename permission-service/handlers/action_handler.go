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

// CreateActionRequest represents the request body for creating an action
type CreateActionRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// UpdateActionRequest represents the request body for updating an action
type UpdateActionRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// ActionResponse represents an action in the system
type ActionResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// ActionListResponse represents a list of actions with pagination
type ActionListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []ActionResponse   `json:"items"`
		Pagination PaginationResponse `json:"pagination"`
	} `json:"data"`
}

// SingleActionResponse represents a single action response
type SingleActionResponse struct {
	Success bool           `json:"success"`
	Data    ActionResponse `json:"data"`
}

// CreateAction creates a new action
// @Summary Create a new action
// @Description Create a new action for permissions
// @Tags actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param action body CreateActionRequest true "Action data"
// @Success 201 {object} handlers.SingleActionResponse "Created action"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 409 {object} map[string]string "Action with this slug already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /permissions/actions [post]
func CreateAction(c *gin.Context) {
	var req CreateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if req.Slug == "" {
		req.Slug = generateActionSlug(req.Name)
	}

	// Validate slug uniqueness
	var existingAction models.Action
	if err := database.DB.Where("slug = ?", req.Slug).First(&existingAction).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Action with this slug already exists",
		})
		return
	}

	action := models.Action{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	}

	if err := database.DB.Create(&action).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create action",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Action created successfully",
		"action":  action,
	})
}

// GetActions returns a list of all actions with pagination
// @Summary Get all actions
// @Description Get all actions with pagination, filtering, sorting, and search
// @Tags actions
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
// @Success 200 {object} handlers.ActionListResponse "List of actions"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /permissions/actions [get]
func GetActions(c *gin.Context) {
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
	baseQuery := db.Model(&models.Action{})

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

	// Get actions
	var actions []models.Action
	if err := finalQuery.Find(&actions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch actions",
			"details": err.Error(),
		})
		return
	}

	// Build pagination response
	pagination := query.BuildPaginationResponse(params.Page, params.Limit, total)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      actions,
			"pagination": pagination,
		},
	})
}

// GetAction returns a single action by ID
// @Summary Get an action by ID
// @Description Get detailed information about a specific action
// @Tags actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Action ID" format(uuid)
// @Success 200 {object} handlers.SingleActionResponse "Action details"
// @Failure 400 {object} map[string]string "Invalid action ID format"
// @Failure 404 {object} map[string]string "Action not found"
// @Router /permissions/actions/{id} [get]
func GetAction(c *gin.Context) {
	id := c.Param("id")

	actionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid action ID format",
		})
		return
	}

	var action models.Action
	if err := database.DB.First(&action, actionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Action not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"action": action,
	})
}

// UpdateAction updates an existing action
// @Summary Update an action
// @Description Update an existing action's details
// @Tags actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Action ID" format(uuid)
// @Param action body UpdateActionRequest true "Updated action data"
// @Success 200 {object} handlers.SingleActionResponse "Updated action"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 404 {object} map[string]string "Action not found"
// @Failure 409 {object} map[string]string "Action with this slug already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /permissions/actions/{id} [put]
func UpdateAction(c *gin.Context) {
	id := c.Param("id")

	actionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid action ID format",
		})
		return
	}

	var req UpdateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	var action models.Action
	if err := database.DB.First(&action, actionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Action not found",
		})
		return
	}

	// Check if it's a system action and prevent modification of critical fields
	if action.IsSystem {
		// System actions can only have their description updated
		if req.Name != "" || req.Slug != "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Cannot modify system action",
				"message": "System actions name and slug cannot be modified. Only description can be updated.",
			})
			return
		}
	}

	if req.Name != "" {
		action.Name = req.Name
	}
	if req.Slug != "" {
		var existingAction models.Action
		if err := database.DB.Where("slug = ? AND id != ?", req.Slug, actionID).First(&existingAction).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Action with this slug already exists",
			})
			return
		}
		action.Slug = req.Slug
	}
	if req.Description != "" {
		action.Description = req.Description
	}

	if err := database.DB.Save(&action).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update action",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Action updated successfully",
		"action":  action,
	})
}

// DeleteAction deletes an action by ID
// @Summary Delete an action
// @Description Delete an action if it's not being used in any permissions
// @Tags actions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Action ID" format(uuid)
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid action ID format"
// @Failure 404 {object} map[string]string "Action not found"
// @Failure 409 {object} map[string]interface{} "Action is being used in permissions"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /permissions/actions/{id} [delete]
func DeleteAction(c *gin.Context) {
	id := c.Param("id")

	actionID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid action ID format",
		})
		return
	}

	var action models.Action
	if err := database.DB.First(&action, actionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Action not found",
		})
		return
	}

	// Check if it's a system action
	if action.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Cannot delete system action",
			"message": "System actions are protected and cannot be deleted",
		})
		return
	}

	var permissionActionCount int64
	database.DB.Model(&models.PermissionAction{}).Where("action_id = ?", actionID).Count(&permissionActionCount)
	if permissionActionCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Cannot delete action",
			"message": "Action is being used in permission actions",
			"count":   permissionActionCount,
		})
		return
	}

	if err := database.DB.Delete(&action).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete action",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Action deleted successfully",
	})
}

// generateActionSlug creates a URL-friendly slug from an action name
func generateActionSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	return slug
}
