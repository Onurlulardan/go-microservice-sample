package document

import (
	"forgecrud-backend/shared/database/models/document"
	"time"

	"gorm.io/gorm"
)

// FolderResponse represents a folder response for API
type FolderResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ParentID  *string   `json:"parent_id,omitempty"`
	OwnerID   string    `json:"owner_id"`
	OwnerType string    `json:"owner_type"`
	FileCount int       `json:"file_count"`
	TotalSize int64     `json:"total_size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BuildFolderResponse converts folder model to response format
func BuildFolderResponse(folder *document.Folder) FolderResponse {
	response := FolderResponse{
		ID:        folder.ID.String(),
		Name:      folder.Name,
		Path:      folder.Path,
		OwnerID:   folder.OwnerID.String(),
		OwnerType: folder.OwnerType,
		FileCount: folder.FileCount,
		TotalSize: folder.TotalSize,
		CreatedAt: folder.CreatedAt,
		UpdatedAt: folder.UpdatedAt,
	}

	if folder.ParentID != nil {
		parentIDStr := folder.ParentID.String()
		response.ParentID = &parentIDStr
	}

	return response
}

// BuildFolderListResponse converts slice of folders to response format
func BuildFolderListResponse(folders []document.Folder) []FolderResponse {
	responses := make([]FolderResponse, len(folders))
	for i, folder := range folders {
		responses[i] = BuildFolderResponse(&folder)
	}
	return responses
}

// BuildDocumentListResponse builds response for multiple documents
func BuildDocumentListResponse(documents []document.Document, db *gorm.DB) []DocumentResponse {
	responses := make([]DocumentResponse, len(documents))

	for i, doc := range documents {
		responses[i] = BuildDocumentResponse(&doc, db)
	}

	return responses
}
