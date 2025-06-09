package document

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Folder represents a document folder
type Folder struct {
	ID       uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name     string     `gorm:"not null" json:"name"`
	Path     string     `gorm:"not null;unique" json:"path"`
	ParentID *uuid.UUID `gorm:"type:uuid" json:"parent_id"`
	Parent   *Folder    `gorm:"foreignKey:ParentID" json:"parent,omitempty"`

	// Owner context
	OwnerID   uuid.UUID `gorm:"type:uuid;not null" json:"owner_id"`
	OwnerType string    `gorm:"not null" json:"owner_type"` // "user", "organization"

	// Stats
	FileCount int   `gorm:"default:0" json:"file_count"`
	TotalSize int64 `gorm:"default:0" json:"total_size"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Document represents a document file
type Document struct {
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`

	// File information
	FileName      string `gorm:"not null" json:"file_name"`
	OriginalName  string `gorm:"not null" json:"original_name"`
	FileSize      int64  `gorm:"not null" json:"file_size"`
	MimeType      string `gorm:"not null" json:"mime_type"`
	FileExtension string `gorm:"not null" json:"file_extension"`
	Checksum      string `gorm:"not null" json:"checksum"`

	// Storage
	FolderID   uuid.UUID `gorm:"type:uuid;not null" json:"folder_id"`
	Folder     Folder    `gorm:"foreignKey:FolderID" json:"folder,omitempty"`
	BucketName string    `gorm:"not null" json:"bucket_name"`
	ObjectKey  string    `gorm:"not null;unique" json:"object_key"`
	Path       string    `gorm:"not null" json:"path"`

	// Metadata
	Description string `gorm:"type:text" json:"description"`
	Tags        string `gorm:"type:text" json:"tags"`

	// OCR
	OCRStatus string `gorm:"default:'pending'" json:"ocr_status"` // pending, processing, completed, failed
	OCRText   string `gorm:"type:text" json:"ocr_text"`

	// Processing
	HasThumbnail  bool   `gorm:"default:false" json:"has_thumbnail"`
	ThumbnailPath string `json:"thumbnail_path"`

	// Owner
	UploadedBy uuid.UUID `gorm:"type:uuid;not null" json:"uploaded_by"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// DocumentVersion represents version history
type DocumentVersion struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null" json:"document_id"`
	Document   Document  `gorm:"foreignKey:DocumentID" json:"document,omitempty"`
	Version    int       `gorm:"not null" json:"version"`
	ObjectKey  string    `gorm:"not null" json:"object_key"`
	FileSize   int64     `gorm:"not null" json:"file_size"`
	Checksum   string    `gorm:"not null" json:"checksum"`
	CreatedBy  uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}
