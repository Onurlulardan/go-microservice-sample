package auth

import (
	"time"

	"forgecrud-backend/shared/database/models"

	"github.com/google/uuid"
)

// PasswordResetToken - Şifre sıfırlama token'ları
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	Token     string     `json:"token" gorm:"size:255;uniqueIndex;not null"` // Reset token
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	Used      bool       `json:"used" gorm:"default:false"`
	Expired   bool       `json:"expired" gorm:"default:false"`
	UsedAt    *time.Time `json:"used_at"`
	IPAddress string     `json:"ip_address" gorm:"size:50"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	// Relations
	User models.User `json:"user" gorm:"foreignKey:UserID"`
}
