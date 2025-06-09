package auth

import (
	"time"

	"forgecrud-backend/shared/database/models"

	"github.com/google/uuid"
)

// EmailVerificationToken - Email doğrulama token'ları
type EmailVerificationToken struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     uuid.UUID  `json:"user_id" gorm:"type:uuid;not null"`
	Token      string     `json:"token" gorm:"size:255;uniqueIndex;not null"` // Verification token
	Email      string     `json:"email" gorm:"size:255;not null"`             // Doğrulanacak email
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	Verified   bool       `json:"verified" gorm:"default:false"`
	VerifiedAt *time.Time `json:"verified_at"`
	IPAddress  string     `json:"ip_address" gorm:"size:50"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relations
	User models.User `json:"user" gorm:"foreignKey:UserID"`
}
