package handlers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"forgecrud-backend/document-service/services"
	"forgecrud-backend/shared/clients"
	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/document"
	docUtils "forgecrud-backend/shared/utils/document"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UploadDocument uploads a new document
// @Summary Upload a new document
// @Description Upload a new document to a specified folder
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param folder_id formData string true "Folder ID where the document will be uploaded"
// @Param user_id formData string false "User ID (for testing purposes)"
// @Param file formData file true "Document file to upload"
// @Param tags formData string false "Document tags"
// @Param description formData string false "Document description"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "Document uploaded successfully"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 404 {object} map[string]string "Folder not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents [post]
func UploadDocument(ctx *gin.Context) {
	db := database.GetDB()

	// Get folder ID
	folderID := ctx.PostForm("folder_id")
	if folderID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "folder_id is required"})
		return
	}

	// Validate folder exists
	var folder document.Folder
	if err := db.First(&folder, "id = ?", folderID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Get file from request
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Validate file
	if err := docUtils.ValidateUploadedFile(header); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate checksum
	checksum, err := docUtils.CalculateFileChecksum(file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate checksum"})
		return
	}

	// Reset file pointer after checksum calculation
	file.Seek(0, 0)

	// Calculate next version for this filename in this folder
	version := 1
	var existingDoc document.Document
	if err := db.Where("folder_id = ? AND file_name = ?", folderID, header.Filename).First(&existingDoc).Error; err == nil {
		// Same filename exists, get max version
		var maxVersion int
		db.Model(&document.DocumentVersion{}).
			Where("document_id = ?", existingDoc.ID).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion)
		version = maxVersion + 1
	}

	// Generate paths
	minioPath := docUtils.GenerateMinIOPath(folder.Path, header.Filename, version)
	displayPath := docUtils.GenerateDisplayPath(folder.Path, header.Filename, version)

	// Upload to MinIO
	minioService, err := services.NewMinIOService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service unavailable"})
		return
	}

	if err := minioService.UploadFile(context.Background(), file, header.Filename, folder.Path, header.Size); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Create document record
	doc := document.Document{
		ID:            uuid.New(),
		FileName:      header.Filename,
		OriginalName:  header.Filename,
		Path:          displayPath,
		FileSize:      header.Size,
		MimeType:      header.Header.Get("Content-Type"),
		FileExtension: filepath.Ext(header.Filename),
		FolderID:      uuid.MustParse(folderID),
		UploadedBy:    uuid.MustParse(ctx.PostForm("user_id")),
		ObjectKey:     minioPath,
		Checksum:      checksum,
		Tags:          ctx.PostForm("tags"),
		Description:   ctx.PostForm("description"),
	}

	if err := db.Create(&doc).Error; err != nil {
		// Cleanup MinIO file
		minioService.RemoveFile(context.Background(), header.Filename, folder.Path)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document"})
		return
	}

	// Create version record
	docVersion := document.DocumentVersion{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		Version:    version,
		ObjectKey:  minioPath,
		FileSize:   header.Size,
		Checksum:   checksum,
		CreatedBy:  doc.UploadedBy,
	}

	if err := db.Create(&docVersion).Error; err != nil {
		fmt.Printf("Warning: Failed to create version record: %v\n", err)
	}

	// Update folder statistics after successful upload
	if err := updateFolderStats(db, uuid.MustParse(folderID)); err != nil {
		fmt.Printf("Warning: Failed to update folder stats: %v\n", err)
	}

	// Load folder info for response
	db.Preload("Folder").First(&doc, doc.ID)

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Document uploaded successfully",
		"data":    docUtils.BuildDocumentResponse(&doc, db),
	})
}

// GetDocuments lists documents in a folder
// @Summary Get documents in a folder
// @Description Retrieve all documents in a specified folder
// @Tags documents
// @Accept json
// @Produce json
// @Param folder_id query string true "Folder ID to list documents from"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of documents"
// @Failure 400 {object} map[string]string "Missing or invalid folder_id"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents [get]
func GetDocuments(ctx *gin.Context) {
	db := database.GetDB()

	folderID := ctx.Query("folder_id")
	if folderID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "folder_id is required"})
		return
	}

	var documents []document.Document
	if err := db.Preload("Folder").Where("folder_id = ?", folderID).Find(&documents).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return
	}

	var response []docUtils.DocumentResponse
	for _, doc := range documents {
		response = append(response, docUtils.BuildDocumentResponse(&doc, db))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// GetDocument gets a single document
// @Summary Get document by ID
// @Description Get detailed information about a specific document
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Document details"
// @Failure 400 {object} map[string]string "Invalid document ID format"
// @Failure 404 {object} map[string]string "Document not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id} [get]
func GetDocument(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	var doc document.Document
	if err := db.Preload("Folder").First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    docUtils.BuildDocumentResponse(&doc, db),
	})
}

// DownloadDocument downloads a document file
// @Summary Download document file
// @Description Download the actual file content of a document
// @Tags documents
// @Accept json
// @Produce application/octet-stream
// @Param id path string true "Document ID" format(uuid)
// @Security BearerAuth
// @Success 200 {file} file "Document file content"
// @Failure 400 {object} map[string]string "Invalid document ID format"
// @Failure 404 {object} map[string]string "Document not found"
// @Failure 500 {object} map[string]string "Server error or storage unavailable"
// @Router /documents/{id}/download [get]
func DownloadDocument(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	var doc document.Document
	if err := db.Preload("Folder").First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Download from MinIO
	minioService, err := services.NewMinIOService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service unavailable"})
		return
	}

	fileName := filepath.Base(doc.ObjectKey)
	folderPath := filepath.Dir(doc.ObjectKey)

	fileReader, err := minioService.DownloadFile(context.Background(), fileName, folderPath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file"})
		return
	}
	defer fileReader.Close()

	// Set response headers
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", doc.OriginalName))
	ctx.Header("Content-Type", doc.MimeType)
	ctx.Header("Content-Length", fmt.Sprintf("%d", doc.FileSize))

	// Stream file to response
	ctx.DataFromReader(http.StatusOK, doc.FileSize, doc.MimeType, fileReader, nil)
}

// UpdateDocument updates document metadata
// @Summary Update document metadata
// @Description Update document tags and description
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Param tags formData string false "Updated document tags"
// @Param description formData string false "Updated document description"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Document updated successfully"
// @Failure 400 {object} map[string]string "Invalid document ID format"
// @Failure 404 {object} map[string]string "Document not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id} [put]
func UpdateDocument(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	var doc document.Document
	if err := db.Preload("Folder").First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Update fields
	updateData := map[string]interface{}{}

	if tags := ctx.PostForm("tags"); tags != "" {
		updateData["tags"] = tags
	}

	if description := ctx.PostForm("description"); description != "" {
		updateData["description"] = description
	}

	if len(updateData) > 0 {
		if err := db.Model(&doc).Updates(updateData).Error; err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
			return
		}
	}

	// Reload document
	db.Preload("Folder").First(&doc, documentID)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document updated successfully",
		"data":    docUtils.BuildDocumentResponse(&doc, db),
	})
}

// DeleteDocument deletes a document
// @Summary Delete a document
// @Description Delete a document and all its versions from storage and database
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Document deleted successfully"
// @Failure 400 {object} map[string]string "Invalid document ID format"
// @Failure 404 {object} map[string]string "Document not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id} [delete]
func DeleteDocument(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	var doc document.Document
	if err := db.First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Delete from MinIO
	minioService, err := services.NewMinIOService()
	if err == nil {
		var versions []document.DocumentVersion
		if err := db.Where("document_id = ?", doc.ID).Find(&versions).Error; err == nil {
			for _, version := range versions {
				if version.ObjectKey != "" {
					fileName := filepath.Base(version.ObjectKey)
					folderPath := filepath.Dir(version.ObjectKey)
					minioService.RemoveFile(context.Background(), fileName, folderPath)
				}
			}
		}

		// Delete main file if exists
		if doc.ObjectKey != "" {
			fileName := filepath.Base(doc.ObjectKey)
			folderPath := filepath.Dir(doc.ObjectKey)
			minioService.RemoveFile(context.Background(), fileName, folderPath)
		}
	}

	if err := db.Delete(&doc).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	// After successful deletion, get user info for notification
	var user models.User
	if err := db.Where("id = ?", doc.Folder.OwnerID).First(&user).Error; err != nil {
		fmt.Printf("Warning: Could not fetch user info for notification: %v\n", err)
	} else {
		notificationClient := clients.NewNotificationClient()

		go func() {
			err := notificationClient.SendUserActionEmail(clients.UserActionEmailRequest{
				AdminName:    "System Admin",
				UserName:     fmt.Sprintf("%s %s", user.FirstName, user.LastName),
				UserEmail:    user.Email,
				UserRole:     "",
				IPAddress:    ctx.ClientIP(),
				ActionType:   "Document Deletion",
				ResourceName: doc.OriginalName,
				Status:       "Completed",
				Priority:     "medium",
				PriorityText: "Medium",
				Description:  fmt.Sprintf("Document '%s' (%.2f KB) deleted from folder", doc.OriginalName, float64(doc.FileSize)/1024),
				Changes: []clients.UserActionChange{
					{
						Field:    "Document Status",
						OldValue: "Active",
						NewValue: "Deleted",
					},
					{
						Field:    "File Size",
						OldValue: fmt.Sprintf("%d bytes", doc.FileSize),
						NewValue: "0 bytes",
					},
				},
				Timestamp: time.Now().Format(time.RFC3339),
			})

			if err != nil {
				fmt.Printf("Warning: Failed to send document deletion notification: %v\n", err)
			}
		}()
	}

	// Update folder statistics after successful deletion
	if err := updateFolderStats(db, doc.FolderID); err != nil {
		fmt.Printf("Warning: Failed to update folder stats: %v\n", err)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document deleted successfully",
	})
}

// MoveDocumentRequest represents move request
type MoveDocumentRequest struct {
	TargetFolderID string `json:"target_folder_id" binding:"required"`
}

// MoveDocument moves a document to another folder
// @Summary Move document to another folder
// @Description Move a document and all its versions to a different folder
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Param request body MoveDocumentRequest true "Target folder information"
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Document moved successfully"
// @Failure 400 {object} map[string]string "Invalid request data or document ID format"
// @Failure 404 {object} map[string]string "Document or target folder not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id}/move [post]
func MoveDocument(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	var req MoveDocumentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get document
	var doc document.Document
	if err := db.Preload("Folder").First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Get target folder
	var targetFolder document.Folder
	if err := db.First(&targetFolder, "id = ?", req.TargetFolderID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Target folder not found"})
		return
	}

	// Move document
	if err := moveDocument(db, &doc, &targetFolder); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload document
	db.Preload("Folder").First(&doc, documentID)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Document moved successfully",
		"data":    docUtils.BuildDocumentResponse(&doc, db),
	})
}

// moveDocument helper function to move document and its versions
func moveDocument(db *gorm.DB, doc *document.Document, targetFolder *document.Folder) error {
	// Store original folder ID before updating
	oldFolderID := doc.FolderID
	oldFolderPath := doc.Folder.Path

	minioService, err := services.NewMinIOService()
	if err != nil {
		return fmt.Errorf("storage service unavailable: %v", err)
	}

	// Get all versions
	var versions []document.DocumentVersion
	if err := db.Where("document_id = ?", doc.ID).Find(&versions).Error; err != nil {
		return fmt.Errorf("failed to get document versions: %v", err)
	}

	// Store version updates before DB changes
	type VersionUpdate struct {
		Version      document.DocumentVersion
		OldMinIOPath string
		NewMinIOPath string
		NewObjectKey string
	}

	var versionUpdates []VersionUpdate

	// Prepare version updates using simple folder path + filename
	for _, version := range versions {
		oldMinIOPath := filepath.Join(oldFolderPath, doc.FileName)
		newMinIOPath := filepath.Join(targetFolder.Path, doc.FileName)

		fileName := filepath.Base(version.ObjectKey)
		newObjectKey := filepath.Join(targetFolder.Path, fileName)

		versionUpdates = append(versionUpdates, VersionUpdate{
			Version:      version,
			OldMinIOPath: oldMinIOPath,
			NewMinIOPath: newMinIOPath,
			NewObjectKey: newObjectKey,
		})

		// Update version record in DB
		if err := db.Model(&version).Update("object_key", newObjectKey).Error; err != nil {
			return fmt.Errorf("failed to update version %d: %v", version.Version, err)
		}
	}

	// Now move files in MinIO after DB is updated
	for _, update := range versionUpdates {
		if err := minioService.MoveObject(update.OldMinIOPath, update.NewMinIOPath); err != nil {
			return fmt.Errorf("failed to move version %d: %v", update.Version.Version, err)
		}

		fmt.Printf("Moved version %d from %s to %s\n", update.Version.Version, update.OldMinIOPath, update.NewMinIOPath)
	}

	// Update document record
	// Get latest version number
	latestVersion := 1
	for _, v := range versions {
		if v.Version > latestVersion {
			latestVersion = v.Version
		}
	}

	newDisplayPath := docUtils.GenerateDisplayPath(targetFolder.Path, doc.FileName, latestVersion)
	newObjectKey := ""
	if doc.ObjectKey != "" {
		fileName := filepath.Base(doc.ObjectKey)
		newObjectKey = filepath.Join(targetFolder.Path, fileName)
	}

	updateData := map[string]interface{}{
		"folder_id": targetFolder.ID,
		"path":      newDisplayPath,
	}

	if newObjectKey != "" {
		updateData["object_key"] = newObjectKey
	}

	if err := db.Model(doc).Updates(updateData).Error; err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}

	// Update folder statistics for both old and new folders
	if err := updateFolderStats(db, oldFolderID); err != nil {
		fmt.Printf("Warning: Failed to update old folder stats: %v\n", err)
	}
	if err := updateFolderStats(db, targetFolder.ID); err != nil {
		fmt.Printf("Warning: Failed to update target folder stats: %v\n", err)
	}

	return nil
}

// GetDocumentVersions gets all versions of a document
// @Summary Get all versions of a document
// @Description Retrieve all versions of a specific document ordered by version number
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of document versions"
// @Failure 404 {object} map[string]string "Document not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id}/versions [get]
func GetDocumentVersions(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	// Check if document exists
	var doc document.Document
	if err := db.First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Get all versions
	var versions []document.DocumentVersion
	if err := db.Where("document_id = ?", documentID).Order("version DESC").Find(&versions).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch document versions"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    versions,
	})
}

// GetLatestDocumentVersion gets the latest version of a document
// @Summary Get latest version of a document
// @Description Retrieve the most recent version of a specific document
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Latest document version"
// @Failure 404 {object} map[string]string "Document or version not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id}/versions/latest [get]
func GetLatestDocumentVersion(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	// Check if document exists
	var doc document.Document
	if err := db.First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Get latest version
	var version document.DocumentVersion
	if err := db.Where("document_id = ?", documentID).Order("version DESC").First(&version).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No versions found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    version,
	})
}

// UploadDocumentVersion uploads a new version of an existing document
// @Summary Upload new document version
// @Description Upload a new version of an existing document
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Param file formData file true "Document file to upload"
// @Param user_id formData string false "User ID (for testing purposes)"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "Document version uploaded successfully"
// @Failure 400 {object} map[string]string "Invalid request data"
// @Failure 404 {object} map[string]string "Document not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id}/versions [post]
func UploadDocumentVersion(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")

	// Get existing document
	var doc document.Document
	if err := db.Preload("Folder").First(&doc, "id = ?", documentID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Get file from request
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Validate file
	if err := docUtils.ValidateUploadedFile(header); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Calculate checksum
	checksum, err := docUtils.CalculateFileChecksum(file)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate checksum"})
		return
	}

	// Reset file pointer after checksum calculation
	file.Seek(0, 0)

	// Get next version number
	var maxVersion int
	db.Model(&document.DocumentVersion{}).
		Where("document_id = ?", doc.ID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)
	newVersion := maxVersion + 1

	// Generate paths for new version
	minioPath := docUtils.GenerateMinIOPath(doc.Folder.Path, header.Filename, newVersion)

	// Upload to MinIO
	minioService, err := services.NewMinIOService()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service unavailable"})
		return
	}

	if err := minioService.UploadFile(context.Background(), file, header.Filename, doc.Folder.Path, header.Size); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Create version record
	docVersion := document.DocumentVersion{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		Version:    newVersion,
		ObjectKey:  minioPath,
		FileSize:   header.Size,
		Checksum:   checksum,
		CreatedBy:  uuid.MustParse(ctx.PostForm("user_id")),
	}

	if err := db.Create(&docVersion).Error; err != nil {
		minioService.RemoveFile(context.Background(), header.Filename, doc.Folder.Path)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save version"})
		return
	}

	// Update main document to point to latest version
	newDisplayPath := docUtils.GenerateDisplayPath(doc.Folder.Path, header.Filename, newVersion)
	updateData := map[string]interface{}{
		"path":       newDisplayPath,
		"object_key": minioPath,
		"file_size":  header.Size,
		"checksum":   checksum,
	}

	if err := db.Model(&doc).Updates(updateData).Error; err != nil {
		fmt.Printf("Warning: Failed to update main document record: %v\n", err)
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Document version uploaded successfully",
		"data":    docVersion,
	})
}

// CopyDocumentRequest represents copy request
type CopyDocumentRequest struct {
	TargetFolderID string `json:"target_folder_id" binding:"required"`
}

// CopyDocument copies a document to another folder
// @Summary Copy document to another folder
// @Description Copy a document and its latest version to a different folder with automatic naming
// @Tags documents
// @Accept json
// @Produce json
// @Param id path string true "Document ID" format(uuid)
// @Param request body CopyDocumentRequest true "Target folder information"
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "Document copied successfully"
// @Failure 400 {object} map[string]string "Invalid request data or document ID format"
// @Failure 404 {object} map[string]string "Document or target folder not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /documents/{id}/copy [post]
func CopyDocument(ctx *gin.Context) {
	db := database.GetDB()

	documentID := ctx.Param("id")
	docUUID, err := uuid.Parse(documentID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID format"})
		return
	}

	var req CopyDocumentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get original document
	var originalDoc document.Document
	if err := db.Preload("Folder").First(&originalDoc, docUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch document"})
		return
	}

	// Get target folder
	targetFolderUUID, err := uuid.Parse(req.TargetFolderID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target folder ID format"})
		return
	}

	var targetFolder document.Folder
	if err := db.First(&targetFolder, targetFolderUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Target folder not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch target folder"})
		return
	}

	// Generate unique name with "Copy" suffix
	newFileName := generateCopyName(db, originalDoc.OriginalName, targetFolderUUID)

	// Copy document
	copiedDoc, err := copyDocument(db, &originalDoc, &targetFolder, newFileName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Document copied successfully",
		"data": gin.H{
			"id":            copiedDoc.ID,
			"original_name": copiedDoc.OriginalName,
			"file_name":     copiedDoc.FileName,
			"folder_id":     copiedDoc.FolderID,
			"folder_name":   targetFolder.Name,
			"file_size":     copiedDoc.FileSize,
			"created_at":    copiedDoc.CreatedAt,
		},
	})
}

// generateCopyName generates unique name with "Copy" suffix and incremental numbers
func generateCopyName(db *gorm.DB, originalName string, targetFolderID uuid.UUID) string {
	// Extract file name and extension
	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)

	// Start with "Copy of OriginalName"
	baseName := fmt.Sprintf("Copy of %s", nameWithoutExt)
	candidateName := baseName + ext

	// Check if this name exists in target folder
	var count int64
	db.Model(&document.Document{}).Where("folder_id = ? AND original_name = ?", targetFolderID, candidateName).Count(&count)

	// If doesn't exist, use it
	if count == 0 {
		return candidateName
	}

	// If exists, try with incremental numbers
	counter := 1
	for {
		candidateName = fmt.Sprintf("%s_%d%s", baseName, counter, ext)

		db.Model(&document.Document{}).Where("folder_id = ? AND original_name = ?", targetFolderID, candidateName).Count(&count)

		if count == 0 {
			return candidateName
		}

		counter++

		// Safety limit to prevent infinite loop
		if counter > 1000 {
			// Use timestamp as fallback
			candidateName = fmt.Sprintf("%s_%d%s", baseName, time.Now().Unix(), ext)
			break
		}
	}

	return candidateName
}

// copyDocument helper function
func copyDocument(db *gorm.DB, originalDoc *document.Document, targetFolder *document.Folder, newFileName string) (*document.Document, error) {
	minioService, err := services.NewMinIOService()
	if err != nil {
		return nil, fmt.Errorf("storage service unavailable: %v", err)
	}

	// Generate new paths
	newMinIOPath := docUtils.GenerateMinIOPath(targetFolder.Path, newFileName, 1)
	newDisplayPath := docUtils.GenerateDisplayPath(targetFolder.Path, newFileName, 1)

	// Copy file in MinIO
	oldObjectKey := originalDoc.ObjectKey
	if err := minioService.CopyObject(oldObjectKey, newMinIOPath); err != nil {
		return nil, fmt.Errorf("failed to copy file in storage: %v", err)
	}

	// Create new document record
	copiedDoc := document.Document{
		ID:            uuid.New(),
		FileName:      newFileName,
		OriginalName:  newFileName,
		Path:          newDisplayPath,
		FileSize:      originalDoc.FileSize,
		MimeType:      originalDoc.MimeType,
		FileExtension: originalDoc.FileExtension,
		FolderID:      targetFolder.ID,
		UploadedBy:    originalDoc.UploadedBy,
		ObjectKey:     newMinIOPath,
		Checksum:      originalDoc.Checksum,
		Tags:          originalDoc.Tags,
		Description:   fmt.Sprintf("Copy of: %s", originalDoc.Description),
	}

	if err := db.Create(&copiedDoc).Error; err != nil {
		// Cleanup MinIO if database save fails
		fileName := filepath.Base(newMinIOPath)
		folderPath := filepath.Dir(newMinIOPath)
		minioService.RemoveFile(context.Background(), fileName, folderPath)
		return nil, fmt.Errorf("failed to save copied document: %v", err)
	}

	// Create version record
	docVersion := document.DocumentVersion{
		ID:         uuid.New(),
		DocumentID: copiedDoc.ID,
		Version:    1,
		ObjectKey:  newMinIOPath,
		FileSize:   originalDoc.FileSize,
		Checksum:   originalDoc.Checksum,
		CreatedBy:  originalDoc.UploadedBy,
	}

	if err := db.Create(&docVersion).Error; err != nil {
		return nil, fmt.Errorf("failed to create version record: %v", err)
	}

	// Update folder statistics
	if err := updateFolderStats(db, targetFolder.ID); err != nil {
		fmt.Printf("Warning: Failed to update folder stats: %v", err)
	}

	return &copiedDoc, nil
}
