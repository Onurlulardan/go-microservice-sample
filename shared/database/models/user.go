package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email          string     `json:"email" gorm:"uniqueIndex;not null"`
	Password       string     `json:"-" gorm:"not null"`
	FirstName      string     `json:"first_name" gorm:"size:100"`
	LastName       string     `json:"last_name" gorm:"size:100"`
	Phone          string     `json:"phone" gorm:"size:20"`
	Avatar         string     `json:"avatar"`
	Status         string     `json:"status" gorm:"default:'ACTIVE'"`
	EmailVerified  bool       `json:"email_verified" gorm:"default:false"`
	OrganizationID *uuid.UUID `json:"organization_id" gorm:"type:uuid"`
	RoleID         *uuid.UUID `json:"role_id" gorm:"type:uuid"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	Organization Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
	Role         Role         `json:"role" gorm:"foreignKey:RoleID"`
}
