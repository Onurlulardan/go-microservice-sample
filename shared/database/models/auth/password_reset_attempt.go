package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordResetAttempt tracks password reset attempts for rate limiting
type PasswordResetAttempt struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email      string    `gorm:"type:varchar(255);not null;index"`
	IPAddress  string    `gorm:"type:varchar(45);index"`
	UserAgent  string    `gorm:"type:varchar(255)"`
	Successful bool      `gorm:"default:false"`
	CreatedAt  time.Time `gorm:"not null"`
}

// BeforeCreate will set ID if not set
func (a *PasswordResetAttempt) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
