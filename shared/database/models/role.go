package models

import (
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name           string     `json:"name" gorm:"size:100;not null"`
	Description    string     `json:"description" gorm:"type:text"`
	IsDefault      bool       `json:"is_default" gorm:"default:false"`
	OrganizationID *uuid.UUID `json:"organization_id" gorm:"type:uuid"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	Organization Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
}
