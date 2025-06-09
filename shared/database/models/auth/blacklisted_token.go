package auth

import (
	"time"

	"github.com/google/uuid"
)

// BlacklistedToken
type BlacklistedToken struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID        uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	TokenHash     string    `json:"token_hash" gorm:"size:255;not null"`
	ExpiresAt     time.Time `json:"expires_at" gorm:"not null"`
	BlacklistedAt time.Time `json:"blacklisted_at" gorm:"not null"`
	Reason        string    `json:"reason" gorm:"size:255"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}
