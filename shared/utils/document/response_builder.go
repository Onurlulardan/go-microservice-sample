package document

import (
	"forgecrud-backend/shared/database/models/document"

	"gorm.io/gorm"
)

// DocumentResponse API response structure
type DocumentResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
	Extension    string `json:"extension"`
	FolderID     string `json:"folder_id"`
	OwnerID      string `json:"owner_id"`
	OwnerType    string `json:"owner_type"`
	Version      int    `json:"version"`
	Tags         string `json:"tags"`
	Description  string `json:"description"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// BuildDocumentResponse creates a standardized document response
func BuildDocumentResponse(doc *document.Document, db *gorm.DB) DocumentResponse {
	// Get latest version
	var latestVersion document.DocumentVersion
	version := 1

	if err := db.Where("document_id = ?", doc.ID).
		Order("version DESC").
		First(&latestVersion).Error; err == nil {
		version = latestVersion.Version
	}

	return DocumentResponse{
		ID:           doc.ID.String(),
		Name:         doc.FileName,
		OriginalName: doc.OriginalName,
		Path:         doc.Path,
		Size:         doc.FileSize,
		MimeType:     doc.MimeType,
		Extension:    doc.FileExtension,
		FolderID:     doc.FolderID.String(),
		OwnerID:      doc.UploadedBy.String(),
		OwnerType:    doc.Folder.OwnerType,
		Version:      version,
		Tags:         doc.Tags,
		Description:  doc.Description,
		CreatedAt:    doc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    doc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
