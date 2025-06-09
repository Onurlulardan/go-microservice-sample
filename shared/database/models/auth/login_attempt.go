package auth

import (
	"time"

	"github.com/google/uuid"
)

// LoginAttempt - Giriş denemeleri ve rate limiting
type LoginAttempt struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string     `json:"email" gorm:"size:255;not null"`
	IPAddress    string     `json:"ip_address" gorm:"size:50;not null"`
	UserAgent    string     `json:"user_agent" gorm:"type:text"`
	Successful   bool       `json:"successful" gorm:"default:false"`
	FailureType  string     `json:"failure_type" gorm:"size:100"` // wrong_password, user_not_found, account_locked
	Attempts     int        `json:"attempts" gorm:"default:1"`
	LastAttempt  time.Time  `json:"last_attempt" gorm:"not null"`
	BlockedUntil *time.Time `json:"blocked_until"`
	Location     string     `json:"location" gorm:"size:255"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
