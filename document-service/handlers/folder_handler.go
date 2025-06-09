package handlers

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"forgecrud-backend/document-service/services"
	"forgecrud-backend/shared/clients"
	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/document"
	documentUtils "forgecrud-backend/shared/utils/document"
	"forgecrud-backend/shared/utils/query"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Request/Response structures
type CreateFolderRequest struct {
	Name      string  `json:"name" binding:"required"`
	ParentID  *string `json:"parent_id,omitempty"`
	OwnerID   string  `json:"owner_id" binding:"required"`
	OwnerType string  `json:"owner_type" binding:"required"`
}

type UpdateFolderRequest struct {
	Name string `json:"name" binding:"required"`
}

type MoveFolderRequest struct {
	TargetParentID *string `json:"target_parent_id"`
}

// GetFolders handles GET /folders - List folders with filtering and pagination
// @Summary Get all folders
// @Description Get all folders with pagination, filtering, sorting and search
// @Tags folders
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param search query string false "Search term across name and path"
// @Param filters[owner_id] query string false "Filter by owner ID"
// @Param filters[owner_type] query string false "Filter by owner type (user, organization)"
// @Param filters[parent_id] query string false "Filter by parent folder ID"
// @Param sort[field] query string false "Sort field (name, path, created_at, updated_at, file_count, total_size)"
// @Param sort[order] query string false "Sort order (asc, desc)"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of folders with pagination"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders [get]
func GetFolders(ctx *gin.Context) {
	db := database.DB

	// Parse query parameters
	params := query.ParseQueryParams(ctx)

	// Define allowed filter fields
	allowedFilters := map[string]string{
		"owner_id":   "owner_id",
		"owner_type": "owner_type",
		"parent_id":  "parent_id",
	}

	// Define allowed sort fields
	allowedSortFields := map[string]string{
		"name":       "name",
		"path":       "path",
		"created_at": "created_at",
		"updated_at": "updated_at",
		"file_count": "file_count",
		"total_size": "total_size",
	}

	// Define search fields
	searchFields := []string{"name", "path"}

	// Build query
	dbQuery := db.Model(&document.Folder{})

	// Apply filters, search, sorting, and pagination
	dbQuery = query.ApplyFilters(dbQuery, params.Filters, allowedFilters)
	dbQuery = query.ApplySearch(dbQuery, params.Search, searchFields)
	dbQuery = query.ApplySort(dbQuery, params.Sort, allowedSortFields)

	// Get total count for pagination
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to count folders",
			"message": err.Error(),
		})
		return
	}

	// Apply pagination
	dbQuery = query.ApplyPagination(dbQuery, params.Page, params.Limit)

	// Execute query
	var folders []document.Folder
	if err := dbQuery.Find(&folders).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folders",
			"message": err.Error(),
		})
		return
	}

	// Build response
	folderResponses := documentUtils.BuildFolderListResponse(folders)

	// Return paginated response
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    folderResponses,
		"pagination": gin.H{
			"page":  params.Page,
			"limit": params.Limit,
			"total": total,
		},
	})
}

// GetFolder handles GET /folders/:id - Get folder by ID
// @Summary Get folder by ID
// @Description Get detailed information about a specific folder
// @Tags folders
// @Accept json
// @Produce json
// @Param id path string true "Folder ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Folder details"
// @Failure 400 {object} map[string]string "Invalid folder ID format"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders/{id} [get]
func GetFolder(ctx *gin.Context) {
	folderID := ctx.Param("id")
	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	var folder document.Folder
	if err := db.First(&folder, folderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Folder not found",
				"message": "Folder with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folder",
			"message": err.Error(),
		})
		return
	}

	// Build response
	folderResponse := documentUtils.BuildFolderResponse(&folder)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    folderResponse,
	})
}

// GetFolderContents handles GET /folders/:id/contents - Get folder contents
// @Summary Get folder contents
// @Description Get all subfolders and documents in a specific folder
// @Tags folders
// @Accept json
// @Produce json
// @Param id path string true "Folder ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Folder contents"
// @Failure 400 {object} map[string]string "Invalid folder ID format"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders/{id}/contents [get]
func GetFolderContents(ctx *gin.Context) {
	folderID := ctx.Param("id")
	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if folder exists
	var folder document.Folder
	if err := db.First(&folder, folderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Folder not found",
				"message": "Folder with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folder",
			"message": err.Error(),
		})
		return
	}

	// Get subfolders
	var subfolders []document.Folder
	if err := db.Where("parent_id = ?", folderUUID).Find(&subfolders).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch subfolders",
			"message": err.Error(),
		})
		return
	}

	// Get documents
	var documents []document.Document
	if err := db.Where("folder_id = ?", folderUUID).Find(&documents).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch documents",
			"message": err.Error(),
		})
		return
	}

	// Build response
	folderResponse := documentUtils.BuildFolderResponse(&folder)
	subfolderResponses := documentUtils.BuildFolderListResponse(subfolders)
	documentResponses := documentUtils.BuildDocumentListResponse(documents, db)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"folder":     folderResponse,
			"subfolders": subfolderResponses,
			"documents":  documentResponses,
		},
	})
}

// CreateFolder handles POST /folders - Create new folder
// @Summary Create a new folder
// @Description Create a new folder with the provided information
// @Tags folders
// @Accept json
// @Produce json
// @Param folder body CreateFolderRequest true "Folder information"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "Created folder"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 409 {object} map[string]string "Folder already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders [post]
func CreateFolder(ctx *gin.Context) {
	var req CreateFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Validate folder name
	if err := documentUtils.ValidateFolderName(req.Name); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder name",
			"message": err.Error(),
		})
		return
	}

	// Parse owner ID
	ownerUUID, err := uuid.Parse(req.OwnerID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid owner ID format",
			"message": err.Error(),
		})
		return
	}

	// Validate owner type
	if req.OwnerType != "user" && req.OwnerType != "organization" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid owner type",
			"message": "Owner type must be 'user' or 'organization'",
		})
		return
	}

	var parentFolder *document.Folder
	var parentPath string

	// Validate parent folder if provided
	if req.ParentID != nil {
		parentUUID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid parent ID format",
				"message": err.Error(),
			})
			return
		}

		parentFolder = &document.Folder{}
		if err := db.First(parentFolder, parentUUID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Parent folder not found",
					"message": "The specified parent folder does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate parent folder",
				"message": err.Error(),
			})
			return
		}

		// Check owner consistency
		if parentFolder.OwnerID != ownerUUID || parentFolder.OwnerType != req.OwnerType {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Owner mismatch",
				"message": "Folder owner must match parent folder owner",
			})
			return
		}

		parentPath = parentFolder.Path
	}

	// Generate unique folder path
	folderPath := documentUtils.GenerateFolderPath(parentPath, req.Name)

	// Check if folder with same name exists in parent
	var existingFolder document.Folder
	query := db.Where("owner_id = ? AND owner_type = ? AND name = ?", ownerUUID, req.OwnerType, req.Name)

	if req.ParentID != nil {
		query = query.Where("parent_id = ?", *req.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if err := query.First(&existingFolder).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Folder already exists",
			"message": "A folder with this name already exists in the parent directory",
		})
		return
	}

	// Create folder
	folder := document.Folder{
		Name:      req.Name,
		Path:      folderPath,
		OwnerID:   ownerUUID,
		OwnerType: req.OwnerType,
		FileCount: 0,
		TotalSize: 0,
	}

	if req.ParentID != nil {
		parentUUID, _ := uuid.Parse(*req.ParentID)
		folder.ParentID = &parentUUID
	}

	if err := db.Create(&folder).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create folder",
			"message": err.Error(),
		})
		return
	}

	// Create folder in MinIO
	minioService, err := services.NewMinIOService()
	if err != nil {
		// Cleanup database record
		db.Delete(&folder)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Storage service unavailable",
			"message": err.Error(),
		})
		return
	}

	if err := minioService.CreateFolder(folder.Path); err != nil {
		// Cleanup database record
		db.Delete(&folder)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create folder in storage",
			"message": err.Error(),
		})
		return
	}

	// Build response
	folderResponse := documentUtils.BuildFolderResponse(&folder)

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Folder created successfully",
		"data":    folderResponse,
	})
}

// UpdateFolder handles PUT /folders/:id - Update folder
// @Summary Update a folder
// @Description Update an existing folder's name
// @Tags folders
// @Accept json
// @Produce json
// @Param id path string true "Folder ID" format(uuid)
// @Param folder body UpdateFolderRequest true "Updated folder information"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Updated folder"
// @Failure 400 {object} map[string]string "Invalid request data or ID format"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 409 {object} map[string]string "Folder name conflict"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders/{id} [put]
func UpdateFolder(ctx *gin.Context) {
	folderID := ctx.Param("id")
	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder ID format",
			"message": err.Error(),
		})
		return
	}

	var req UpdateFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Validate folder name
	if err := documentUtils.ValidateFolderName(req.Name); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder name",
			"message": err.Error(),
		})
		return
	}

	// Check if folder exists
	var folder document.Folder
	if err := db.First(&folder, folderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Folder not found",
				"message": "Folder with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folder",
			"message": err.Error(),
		})
		return
	}

	// Check if name is different
	if folder.Name == req.Name {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "No changes",
			"message": "Folder name is already the same",
		})
		return
	}

	// Check for name conflicts in parent
	var existingFolder document.Folder
	query := db.Where("owner_id = ? AND owner_type = ? AND name = ? AND id != ?",
		folder.OwnerID, folder.OwnerType, req.Name, folderUUID)

	if folder.ParentID != nil {
		query = query.Where("parent_id = ?", *folder.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if err := query.First(&existingFolder).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Folder name conflict",
			"message": "A folder with this name already exists in the parent directory",
		})
		return
	}

	// Generate new path
	var parentPath string
	if folder.ParentID != nil {
		var parentFolder document.Folder
		if err := db.First(&parentFolder, *folder.ParentID).Error; err == nil {
			parentPath = parentFolder.Path
		}
	}

	newPath := documentUtils.GenerateFolderPath(parentPath, req.Name)

	// Start transaction for updating folder and all subfolders
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update folder
	if err := tx.Model(&folder).Updates(map[string]interface{}{
		"name": req.Name,
		"path": newPath,
	}).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update folder",
			"message": err.Error(),
		})
		return
	}

	// Update all subfolders' paths if any
	var subfolders []document.Folder
	if err := tx.Where("path LIKE ?", folder.Path+"/%").Find(&subfolders).Error; err == nil {
		for _, subfolder := range subfolders {
			// Calculate new subfolder path
			oldPrefix := folder.Path
			newPrefix := newPath
			newSubfolderPath := newPrefix + subfolder.Path[len(oldPrefix):]

			if err := tx.Model(&subfolder).Update("path", newSubfolderPath).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to update subfolder paths",
					"message": err.Error(),
				})
				return
			}
		}
	}

	// Update documents' paths in this folder and subfolders
	var documents []document.Document
	if err := tx.Where("folder_id = ?", folderUUID).Find(&documents).Error; err == nil {
		for _, doc := range documents {
			// Update document path
			newDocPath := filepath.Join(newPath, doc.FileName)
			if err := tx.Model(&doc).Update("path", newDocPath).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Failed to update document paths",
					"message": err.Error(),
				})
				return
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to commit updates",
			"message": err.Error(),
		})
		return
	}

	// Refresh folder data
	db.First(&folder, folderUUID)
	folderResponse := documentUtils.BuildFolderResponse(&folder)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Folder updated successfully",
		"data":    folderResponse,
	})
}

// MoveFolder handles PUT /folders/:id/move - Move folder to another parent
// @Summary Move folder to another parent
// @Description Move a folder and all its contents to a different parent folder
// @Tags folders
// @Accept json
// @Produce json
// @Param id path string true "Folder ID" format(uuid)
// @Param request body MoveFolderRequest true "Target parent folder information"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Folder moved successfully"
// @Failure 400 {object} map[string]string "Invalid request data or folder ID format"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 409 {object} map[string]string "Folder name conflict or circular dependency"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders/{id}/move [post]
func MoveFolder(ctx *gin.Context) {
	folderID := ctx.Param("id")
	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder ID format",
			"message": err.Error(),
		})
		return
	}

	var req MoveFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if folder exists
	var folder document.Folder
	if err := db.First(&folder, folderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Folder not found",
				"message": "Folder with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folder",
			"message": err.Error(),
		})
		return
	}

	var targetParentFolder *document.Folder
	var targetParentPath string

	// Validate target parent if provided
	if req.TargetParentID != nil {
		targetParentUUID, err := uuid.Parse(*req.TargetParentID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid target parent ID format",
				"message": err.Error(),
			})
			return
		}

		// Prevent moving to self or subfolder
		if targetParentUUID == folderUUID {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid move operation",
				"message": "Cannot move folder to itself",
			})
			return
		}

		targetParentFolder = &document.Folder{}
		if err := db.First(targetParentFolder, targetParentUUID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"error":   "Target parent folder not found",
					"message": "The specified target parent folder does not exist",
				})
				return
			}
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to validate target parent folder",
				"message": err.Error(),
			})
			return
		}

		// Check owner consistency
		if targetParentFolder.OwnerID != folder.OwnerID || targetParentFolder.OwnerType != folder.OwnerType {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Owner mismatch",
				"message": "Target parent folder must have the same owner",
			})
			return
		}

		// Prevent circular dependency - check if target is a subfolder
		if isSubfolderOf(db, targetParentUUID, folderUUID) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "Circular dependency",
				"message": "Cannot move folder to its own subfolder",
			})
			return
		}

		targetParentPath = targetParentFolder.Path
	}

	// Check if same parent (no move needed)
	if (req.TargetParentID == nil && folder.ParentID == nil) ||
		(req.TargetParentID != nil && folder.ParentID != nil && *req.TargetParentID == folder.ParentID.String()) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "No move needed",
			"message": "Folder is already in the target location",
		})
		return
	}

	// Check for name conflicts in target parent
	var existingFolder document.Folder
	query := db.Where("owner_id = ? AND owner_type = ? AND name = ? AND id != ?",
		folder.OwnerID, folder.OwnerType, folder.Name, folderUUID)

	if req.TargetParentID != nil {
		targetParentUUID, _ := uuid.Parse(*req.TargetParentID)
		query = query.Where("parent_id = ?", targetParentUUID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if err := query.First(&existingFolder).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Folder name conflict",
			"message": "A folder with this name already exists in the target directory",
		})
		return
	}

	// Generate new path
	newPath := documentUtils.GenerateFolderPath(targetParentPath, folder.Name)

	// Store original path before updating
	oldPath := folder.Path

	// Start transaction for moving folder and updating all subfolders
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update folder
	updateData := map[string]interface{}{
		"path": newPath,
	}

	if req.TargetParentID != nil {
		targetParentUUID, _ := uuid.Parse(*req.TargetParentID)
		updateData["parent_id"] = targetParentUUID
	} else {
		updateData["parent_id"] = nil
	}

	if err := tx.Model(&folder).Updates(updateData).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to move folder",
			"message": err.Error(),
		})
		return
	}

	// Update all subfolders' paths
	if err := updateSubfolderPaths(tx, folder.Path, newPath); err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update subfolder paths",
			"message": err.Error(),
		})
		return
	}

	// Update documents' paths in this folder and subfolders
	if err := updateDocumentPaths(tx, folder.Path, newPath); err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update document paths",
			"message": err.Error(),
		})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to commit move operation",
			"message": err.Error(),
		})
		return
	}

	// Move folder in MinIO after successful database update
	minioService, err := services.NewMinIOService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Storage service unavailable",
			"message": err.Error(),
		})
		return
	}

	if err := minioService.MoveFolder(oldPath, newPath); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to move folder in storage",
			"message": err.Error(),
		})
		return
	}

	// Refresh folder data
	db.First(&folder, folderUUID)
	folderResponse := documentUtils.BuildFolderResponse(&folder)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Folder moved successfully",
		"data":    folderResponse,
	})
}

// DeleteFolder handles DELETE /folders/:id - Delete folder
// @Summary Delete a folder
// @Description Delete an empty folder (folder must not contain any subfolders or documents)
// @Tags folders
// @Accept json
// @Produce json
// @Param id path string true "Folder ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Folder deleted successfully"
// @Failure 400 {object} map[string]string "Invalid folder ID format"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 409 {object} map[string]string "Folder contains subfolders or documents"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders/{id} [delete]
func DeleteFolder(ctx *gin.Context) {
	folderID := ctx.Param("id")
	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Check if folder exists
	var folder document.Folder
	if err := db.First(&folder, folderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Folder not found",
				"message": "Folder with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folder",
			"message": err.Error(),
		})
		return
	}

	// Check if folder has subfolders
	var subfolderCount int64
	db.Model(&document.Folder{}).Where("parent_id = ?", folderUUID).Count(&subfolderCount)
	if subfolderCount > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Folder has subfolders",
			"message": "Cannot delete folder that contains subfolders",
		})
		return
	}

	// Check if folder has documents
	var documentCount int64
	db.Model(&document.Document{}).Where("folder_id = ?", folderUUID).Count(&documentCount)
	if documentCount > 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error":   "Folder has documents",
			"message": "Cannot delete folder that contains documents",
		})
		return
	}

	minioService, err := services.NewMinIOService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Storage service unavailable",
			"message": err.Error(),
		})
		return
	}

	// MinIO'dan folder'ı sil
	if err := minioService.DeleteFolder(folder.Path); err != nil {
		fmt.Printf("Warning: Failed to delete folder from MinIO: %v\n", err)
	}

	// Delete folder
	if err := db.Delete(&folder).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete folder",
			"message": err.Error(),
		})
		return
	}

	// After successful deletion, get user info for notification
	var user models.User
	if err := db.Where("id = ?", folder.OwnerID).First(&user).Error; err != nil {
		fmt.Printf("Warning: Could not fetch user info for notification: %v\n", err)
	} else {
		notificationClient := clients.NewNotificationClient()

		go func() { // Async olarak gönder, response'u bloklamasın
			err := notificationClient.SendUserActionEmail(clients.UserActionEmailRequest{
				AdminName:    "System Admin",
				UserName:     fmt.Sprintf("%s %s", user.FirstName, user.LastName),
				UserEmail:    user.Email,
				UserRole:     folder.OwnerType,
				IPAddress:    ctx.ClientIP(),
				ActionType:   "Folder Deletion",
				ResourceName: folder.Name,
				Status:       "Completed",
				Priority:     "high",
				PriorityText: "High",
				Description: fmt.Sprintf("Folder '%s' deleted from path '%s' (contained %d files, %.2f KB total)",
					folder.Name, folder.Path, folder.FileCount, float64(folder.TotalSize)/1024),
				Changes: []clients.UserActionChange{
					{
						Field:    "Folder Status",
						OldValue: "Active",
						NewValue: "Deleted",
					},
					{
						Field:    "Folder Path",
						OldValue: folder.Path,
						NewValue: "N/A",
					},
					{
						Field:    "File Count",
						OldValue: fmt.Sprintf("%d files", folder.FileCount),
						NewValue: "0 files",
					},
					{
						Field:    "Total Size",
						OldValue: fmt.Sprintf("%d bytes", folder.TotalSize),
						NewValue: "0 bytes",
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
			})

			if err != nil {
				fmt.Printf("Warning: Failed to send folder deletion notification: %v\n", err)
			}
		}()
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Folder deleted successfully",
	})
}

// Helper functions

// isSubfolderOf checks if targetID is a subfolder of parentID
func isSubfolderOf(db *gorm.DB, targetID, parentID uuid.UUID) bool {
	var folder document.Folder
	if err := db.First(&folder, targetID).Error; err != nil {
		return false
	}

	if folder.ParentID == nil {
		return false
	}

	if *folder.ParentID == parentID {
		return true
	}

	return isSubfolderOf(db, *folder.ParentID, parentID)
}

// updateSubfolderPaths updates paths for all subfolders
func updateSubfolderPaths(tx *gorm.DB, oldParentPath, newParentPath string) error {
	var subfolders []document.Folder
	if err := tx.Where("path LIKE ?", oldParentPath+"/%").Find(&subfolders).Error; err != nil {
		return err
	}

	for _, subfolder := range subfolders {
		newSubfolderPath := newParentPath + subfolder.Path[len(oldParentPath):]
		if err := tx.Model(&subfolder).Update("path", newSubfolderPath).Error; err != nil {
			return err
		}
	}

	return nil
}

// updateDocumentPaths updates paths for all documents in folder and subfolders
func updateDocumentPaths(tx *gorm.DB, oldFolderPath, newFolderPath string) error {
	var documents []document.Document

	// Get documents in the folder and all subfolders
	if err := tx.Joins("JOIN folders ON documents.folder_id = folders.id").
		Where("folders.path = ? OR folders.path LIKE ?", oldFolderPath, oldFolderPath+"/%").
		Find(&documents).Error; err != nil {
		return err
	}

	for _, doc := range documents {
		// Get the folder path for this document
		var docFolder document.Folder
		if err := tx.First(&docFolder, doc.FolderID).Error; err != nil {
			continue
		}

		// Calculate new document path
		newDocPath := filepath.Join(docFolder.Path, doc.FileName)
		if err := tx.Model(&doc).Update("path", newDocPath).Error; err != nil {
			return err
		}
	}

	return nil
}

// updateFolderStats recalculates and updates folder statistics (file_count and total_size)
// Includes files from this folder AND all subfolders recursively
func updateFolderStats(db *gorm.DB, folderID uuid.UUID) error {
	var stats struct {
		FileCount int64
		TotalSize int64
	}

	// Get folder path first
	var folder document.Folder
	if err := db.First(&folder, folderID).Error; err != nil {
		return err
	}

	// Calculate stats for this folder AND all subfolders recursively
	if err := db.Model(&document.Document{}).
		Joins("JOIN folders ON documents.folder_id = folders.id").
		Where("folders.path = ? OR folders.path LIKE ?", folder.Path, folder.Path+"/%").
		Select("COUNT(*) as file_count, COALESCE(SUM(documents.file_size), 0) as total_size").
		Scan(&stats).Error; err != nil {
		return err
	}

	// Update folder with recursive stats
	return db.Model(&document.Folder{}).
		Where("id = ?", folderID).
		Updates(map[string]interface{}{
			"file_count": stats.FileCount,
			"total_size": stats.TotalSize,
		}).Error
}

// DownloadFolder downloads folder as ZIP archive
// @Summary Download folder as ZIP
// @Description Download a folder and all its contents as a ZIP archive (recursive)
// @Tags folders
// @Accept json
// @Produce application/zip
// @Param id path string true "Folder ID" format(uuid)
// @Security BearerAuth
// @Success 200 {file} file "ZIP archive containing folder contents"
// @Failure 400 {object} map[string]string "Invalid folder ID format"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /folders/{id}/download [get]
func DownloadFolder(ctx *gin.Context) {
	folderID := ctx.Param("id")
	folderUUID, err := uuid.Parse(folderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid folder ID format",
			"message": err.Error(),
		})
		return
	}

	db := database.DB

	// Get folder
	var folder document.Folder
	if err := db.First(&folder, folderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error":   "Folder not found",
				"message": "Folder with the given ID does not exist",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch folder",
			"message": err.Error(),
		})
		return
	}

	// Get all documents in folder and subfolders recursively
	documents, err := getAllDocumentsInFolder(db, folderUUID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get folder contents",
			"message": err.Error(),
		})
		return
	}

	if len(documents) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Empty folder",
			"message": "Folder contains no documents to download",
		})
		return
	}

	// Initialize MinIO service
	minioService, err := services.NewMinIOService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Storage service unavailable",
			"message": err.Error(),
		})
		return
	}

	// Create ZIP file name
	zipFileName := fmt.Sprintf("%s.zip", documentUtils.SanitizeFileName(folder.Name))

	// Set response headers for ZIP download
	ctx.Header("Content-Type", "application/zip")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))
	ctx.Header("Cache-Control", "no-cache")

	// Create ZIP writer that writes directly to response
	zipWriter := zip.NewWriter(ctx.Writer)
	defer zipWriter.Close()

	// Track statistics
	addedFiles := 0
	totalSize := int64(0)
	errors := []string{}

	// Add each document to ZIP with proper folder structure
	for _, doc := range documents {
		if err := addDocumentToZip(zipWriter, minioService, &doc, folder.Path); err != nil {
			errorMsg := fmt.Sprintf("Failed to add %s: %v", doc.OriginalName, err)
			errors = append(errors, errorMsg)
			fmt.Printf("Warning: %s\n", errorMsg)
			continue
		}
		addedFiles++
		totalSize += doc.FileSize
	}

	// Log download statistics
	fmt.Printf("✅ Folder '%s' downloaded as ZIP: %d files, %.2f MB\n",
		folder.Name, addedFiles, float64(totalSize)/(1024*1024))

}

// getAllDocumentsInFolder gets all documents in folder and subfolders recursively
func getAllDocumentsInFolder(db *gorm.DB, folderID uuid.UUID) ([]document.Document, error) {
	var documents []document.Document

	// Get documents directly in this folder
	if err := db.Preload("Folder").Where("folder_id = ?", folderID).Find(&documents).Error; err != nil {
		return nil, fmt.Errorf("failed to get documents in folder: %v", err)
	}

	// Get all subfolders recursively
	subfolders, err := getAllSubfolders(db, folderID)
	if err != nil {
		return documents, nil // Return what we have so far
	}

	// Get documents from all subfolders
	for _, subfolder := range subfolders {
		var subDocuments []document.Document
		if err := db.Preload("Folder").Where("folder_id = ?", subfolder.ID).Find(&subDocuments).Error; err == nil {
			documents = append(documents, subDocuments...)
		}
	}

	return documents, nil
}

// getAllSubfolders gets all subfolders recursively
func getAllSubfolders(db *gorm.DB, parentID uuid.UUID) ([]document.Folder, error) {
	var allSubfolders []document.Folder

	// Get direct subfolders
	var directSubfolders []document.Folder
	if err := db.Where("parent_id = ?", parentID).Find(&directSubfolders).Error; err != nil {
		return nil, err
	}

	// Add direct subfolders to result
	allSubfolders = append(allSubfolders, directSubfolders...)

	// Recursively get subfolders of each direct subfolder
	for _, subfolder := range directSubfolders {
		nestedSubfolders, err := getAllSubfolders(db, subfolder.ID)
		if err == nil {
			allSubfolders = append(allSubfolders, nestedSubfolders...)
		}
	}

	return allSubfolders, nil
}

// addDocumentToZip adds a document to the ZIP archive with proper folder structure
func addDocumentToZip(zipWriter *zip.Writer, minioService *services.MinIOService, doc *document.Document, baseFolderPath string) error {
	// Download file from MinIO
	fileName := filepath.Base(doc.ObjectKey)
	folderPath := filepath.Dir(doc.ObjectKey)

	fileReader, err := minioService.DownloadFile(context.Background(), fileName, folderPath)
	if err != nil {
		return fmt.Errorf("failed to download file from storage: %v", err)
	}
	defer fileReader.Close()

	// Calculate relative path for ZIP (preserve folder structure)
	relativePath := calculateRelativePath(doc.Folder.Path, baseFolderPath, doc.OriginalName)

	// Create file entry in ZIP
	zipFileHeader := &zip.FileHeader{
		Name:   relativePath,
		Method: zip.Deflate,
	}

	// Set modification time if available
	if !doc.CreatedAt.IsZero() {
		zipFileHeader.Modified = doc.CreatedAt
	}

	zipFile, err := zipWriter.CreateHeader(zipFileHeader)
	if err != nil {
		return fmt.Errorf("failed to create ZIP entry: %v", err)
	}

	// Copy file content to ZIP
	_, err = io.Copy(zipFile, fileReader)
	if err != nil {
		return fmt.Errorf("failed to write file to ZIP: %v", err)
	}

	return nil
}

// calculateRelativePath calculates the relative path for a file in the ZIP
func calculateRelativePath(documentFolderPath, baseFolderPath, fileName string) string {
	// Remove base folder path from document folder path
	relativeFolderPath := strings.TrimPrefix(documentFolderPath, baseFolderPath)
	relativeFolderPath = strings.TrimPrefix(relativeFolderPath, "/")

	// If document is in a subfolder, include the subfolder path
	if relativeFolderPath != "" {
		return filepath.Join(relativeFolderPath, fileName)
	}

	// Document is directly in the base folder
	return fileName
}
