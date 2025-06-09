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

// UserResponse represents user data for API responses
type UserResponse struct {
	ID            uuid.UUID            `json:"id"`
	Email         string               `json:"email"`
	FirstName     string               `json:"first_name"`
	LastName      string               `json:"last_name"`
	Phone         string               `json:"phone"`
	Avatar        string               `json:"avatar"`
	Status        string               `json:"status"`
	EmailVerified bool                 `json:"email_verified"`
	Organization  *models.Organization `json:"organization,omitempty"`
	Role          *models.Role         `json:"role,omitempty"`
	CreatedAt     string               `json:"created_at"`
	UpdatedAt     string               `json:"updated_at"`
}

// CreateUserRequest represents request body for creating user
type CreateUserRequest struct {
	Email          string     `json:"email" binding:"required,email"`
	Password       string     `json:"password" binding:"required,min=6"`
	FirstName      string     `json:"first_name" binding:"required"`
	LastName       string     `json:"last_name" binding:"required"`
	Phone          string     `json:"phone"`
	Avatar         string     `json:"avatar"`
	OrganizationID *uuid.UUID `json:"organization_id"`
	RoleID         *uuid.UUID `json:"role_id"`
}

// UpdateUserRequest represents request body for updating user
type UpdateUserRequest struct {
	Email          string     `json:"email" binding:"omitempty,email"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	Phone          string     `json:"phone"`
	Avatar         string     `json:"avatar"`
	Status         string     `json:"status"`
	OrganizationID *uuid.UUID `json:"organization_id"`
	RoleID         *uuid.UUID `json:"role_id"`
}

// UserListResponse represents a list of users with pagination
type UserListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Items      []UserResponse     `json:"items"`
		Pagination PaginationResponse `json:"pagination"`
	} `json:"data"`
}

// SingleUserResponse represents a single user response
type SingleUserResponse struct {
	Success bool         `json:"success"`
	Data    UserResponse `json:"data"`
}

// PaginationResponse represents pagination information
type PaginationResponse struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// GetUsers retrieves all users with pagination and filtering
// @Summary Get all users
// @Description Get all users with pagination, filtering, sorting and search
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param search query string false "Search term across name and email"
// @Param filters[status] query string false "Filter by status (ACTIVE, INACTIVE, DELETED)"
// @Param filters[organization_id] query string false "Filter by organization ID"
// @Param filters[role_id] query string false "Filter by role ID"
// @Param sort[field] query string false "Sort field (email, first_name, last_name, created_at, updated_at)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Security BearerAuth
// @Success 200 {object} handlers.UserListResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [get]
func GetUsers(ctx *gin.Context) {
	db := database.DB

	// Parse standardized query parameters
	params := query.ParseQueryParams(ctx)

	// Define allowed filter fields
	allowedFilters := map[string]string{
		"status":          "status",
		"organization_id": "organization_id",
		"role_id":         "role_id",
	}

	// Define allowed sort fields
	allowedSortFields := map[string]string{
		"email":      "email",
		"first_name": "first_name",
		"last_name":  "last_name",
		"status":     "status",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	// Define search fields
	searchFields := []string{"first_name", "last_name", "email"}

	// Build base query
	baseQuery := db.Model(&models.User{}).
		Preload("Organization").
		Preload("Role")

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

	// Get users
	var users []models.User
	if err := finalQuery.Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve users",
			"message": err.Error(),
		})
		return
	}

	// Convert to response format
	var userResponses []UserResponse
	for _, user := range users {
		userResponse := UserResponse{
			ID:            user.ID,
			Email:         user.Email,
			FirstName:     user.FirstName,
			LastName:      user.LastName,
			Phone:         user.Phone,
			Avatar:        user.Avatar,
			Status:        user.Status,
			EmailVerified: user.EmailVerified,
			CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Add organization if exists
		if user.OrganizationID != nil {
			userResponse.Organization = &user.Organization
		}

		// Add role if exists
		if user.RoleID != nil {
			userResponse.Role = &user.Role
		}

		userResponses = append(userResponses, userResponse)
	}

	// Build pagination response
	pagination := query.BuildPaginationResponse(params.Page, params.Limit, total)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":      userResponses,
			"pagination": pagination,
		},
	})
}

// GetUser retrieves a single user by ID
// @Summary Get user by ID
// @Description Get detailed information about a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Security BearerAuth
// @Success 200 {object} handlers.SingleUserResponse
// @Failure 400 {object} map[string]string "Invalid user ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id} [get]
func GetUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB
	var user models.User

	if err := db.Preload("Organization").Preload("Role").First(&user, userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": "User with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve user",
			"message": err.Error(),
		})
		return
	}

	// Convert to response format
	userResponse := UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Status:        user.Status,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add organization if exists
	if user.OrganizationID != nil {
		userResponse.Organization = &user.Organization
	}

	// Add role if exists
	if user.RoleID != nil {
		userResponse.Role = &user.Role
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userResponse,
	})
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user with the provided information
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserRequest true "User information"
// @Security BearerAuth
// @Success 201 {object} handlers.SingleUserResponse "Created user"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Email already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users [post]
func CreateUser(ctx *gin.Context) {
	var request CreateUserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if email already exists
	var existingUser models.User
	if err := db.Where("email = ?", request.Email).First(&existingUser).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Email already exists",
			"message": "A user with this email already exists",
		})
		return
	}

	// Validate organization exists if provided
	if request.OrganizationID != nil {
		var org models.Organization
		if err := db.First(&org, *request.OrganizationID).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid organization ID",
				"message": "Organization not found",
			})
			return
		}
	}

	// Validate role exists if provided
	if request.RoleID != nil {
		var role models.Role
		if err := db.First(&role, *request.RoleID).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid role ID",
				"message": "Role not found",
			})
			return
		}
	}

	// Create new user
	user := models.User{
		Email:          request.Email,
		Password:       request.Password, // Note: In production, hash this password
		FirstName:      request.FirstName,
		LastName:       request.LastName,
		Phone:          request.Phone,
		Avatar:         request.Avatar,
		Status:         "ACTIVE",
		EmailVerified:  false,
		OrganizationID: request.OrganizationID,
		RoleID:         request.RoleID,
	}

	if err := db.Create(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"message": err.Error(),
		})
		return
	}

	// Load relations for response
	db.Preload("Organization").Preload("Role").First(&user, user.ID)

	// Convert to response format
	userResponse := UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Status:        user.Status,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add organization if exists
	if user.OrganizationID != nil {
		userResponse.Organization = &user.Organization
	}

	// Add role if exists
	if user.RoleID != nil {
		userResponse.Role = &user.Role
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "User created successfully",
		"data":    userResponse,
	})
}

// UpdateUser updates an existing user
// @Summary Update a user
// @Description Update an existing user's information
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param user body UpdateUserRequest true "Updated user information"
// @Security BearerAuth
// @Success 200 {object} handlers.SingleUserResponse "Updated user"
// @Failure 400 {object} map[string]string "Invalid request data or ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 409 {object} map[string]string "Email already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id} [put]
func UpdateUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user ID format",
			"message": err.Error(),
		})
		return
	}

	var request UpdateUserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"message": err.Error(),
		})
		return
	}

	db := database.DB
	var user models.User

	// Check if user exists
	if err := db.First(&user, userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": "User with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve user",
			"message": err.Error(),
		})
		return
	}

	// Check if email already exists for another user
	if request.Email != "" && request.Email != user.Email {
		var existingUser models.User
		if err := db.Where("email = ? AND id != ?", request.Email, userUUID).First(&existingUser).Error; err == nil {
			ctx.JSON(http.StatusConflict, gin.H{
				"error":   "Email already exists",
				"message": "Another user with this email already exists",
			})
			return
		}
	}

	// Validate organization exists if provided
	if request.OrganizationID != nil {
		var org models.Organization
		if err := db.First(&org, *request.OrganizationID).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid organization ID",
				"message": "Organization not found",
			})
			return
		}
	}

	// Validate role exists if provided
	if request.RoleID != nil {
		var role models.Role
		if err := db.First(&role, *request.RoleID).Error; err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid role ID",
				"message": "Role not found",
			})
			return
		}
	}

	// Update user fields
	updates := map[string]interface{}{}
	if request.Email != "" {
		updates["email"] = request.Email
	}
	if request.FirstName != "" {
		updates["first_name"] = request.FirstName
	}
	if request.LastName != "" {
		updates["last_name"] = request.LastName
	}
	if request.Phone != "" {
		updates["phone"] = request.Phone
	}
	if request.Avatar != "" {
		updates["avatar"] = request.Avatar
	}
	if request.Status != "" {
		updates["status"] = request.Status
	}
	if request.OrganizationID != nil {
		updates["organization_id"] = request.OrganizationID
	}
	if request.RoleID != nil {
		updates["role_id"] = request.RoleID
	}

	// Perform update
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user",
			"message": err.Error(),
		})
		return
	}

	// Load updated user with relations
	db.Preload("Organization").Preload("Role").First(&user, userUUID)

	// Convert to response format
	userResponse := UserResponse{
		ID:            user.ID,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		Status:        user.Status,
		EmailVerified: user.EmailVerified,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add organization if exists
	if user.OrganizationID != nil {
		userResponse.Organization = &user.Organization
	}

	// Add role if exists
	if user.RoleID != nil {
		userResponse.Role = &user.Role
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User updated successfully",
		"data":    userResponse,
	})
}

// DeleteUser deletes a user (soft delete)
// @Summary Delete a user
// @Description Soft delete a user by setting status to DELETED
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {object} map[string]string "Invalid user ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id} [delete]
func DeleteUser(ctx *gin.Context) {
	userID := ctx.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB
	var user models.User

	// Check if user exists
	if err := db.First(&user, userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": "User with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve user",
			"message": err.Error(),
		})
		return
	}

	// Soft delete by setting status to DELETED
	if err := db.Model(&user).Update("status", "DELETED").Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete user",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User deleted successfully",
	})
}

// GetUserPermissions retrieves all permissions for a specific user
// @Summary Get user permissions
// @Description Get all permissions assigned to a specific user including user-level, role-level and organization-level permissions
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User permissions data"
// @Failure 400 {object} map[string]string "Invalid user ID format"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /users/{id}/permissions [get]
func GetUserPermissions(ctx *gin.Context) {
	userID := ctx.Param("id")
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid user ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if user exists
	var user models.User
	if err := db.Preload("Organization").Preload("Role").First(&user, userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "User not found",
				"message": "User with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve user",
			"message": err.Error(),
		})
		return
	}

	// Get user-level permissions
	var userPermissions []models.Permission
	db.Preload("Resource").
		Preload("PermissionActions.Action").
		Where("target = ? AND user_id = ?", "USER", userUUID).
		Find(&userPermissions)

	// Get role-level permissions if user has a role
	var rolePermissions []models.Permission
	if user.RoleID != nil {
		db.Preload("Resource").
			Preload("PermissionActions.Action").
			Where("target = ? AND role_id = ?", "ROLE", *user.RoleID).
			Find(&rolePermissions)
	}

	// Get organization-level permissions if user has an organization
	var orgPermissions []models.Permission
	if user.OrganizationID != nil {
		db.Preload("Resource").
			Preload("PermissionActions.Action").
			Where("target = ? AND organization_id = ?", "ORGANIZATION", *user.OrganizationID).
			Find(&orgPermissions)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user": user,
			"permissions": gin.H{
				"user_permissions": userPermissions,
				"role_permissions": rolePermissions,
				"org_permissions":  orgPermissions,
			},
		},
	})
}
