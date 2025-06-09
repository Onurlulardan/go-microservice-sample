package auth

import (
	"time"

	"forgecrud-backend/shared/database/models"

	"github.com/google/uuid"
)

// UserSession - JWT token ve session yönetimi
type UserSession struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	SessionID    string     `json:"session_id" gorm:"size:255;uniqueIndex;not null"` // Unique session identifier
	TokenHash    string     `json:"token_hash" gorm:"size:255;not null"`             // JWT token'ın hash'i
	RefreshToken string     `json:"refresh_token" gorm:"size:500"`                   // Refresh token
	DeviceInfo   string     `json:"device_info" gorm:"size:500"`                     // User-Agent, device bilgisi
	UserAgent    string     `json:"user_agent" gorm:"size:500"`                      // HTTP User-Agent
	IPAddress    string     `json:"ip_address" gorm:"size:50"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"not null"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relations
	User models.User `json:"user" gorm:"foreignKey:UserID"`
}
