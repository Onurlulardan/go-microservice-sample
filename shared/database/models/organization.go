package models

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string     `json:"name" gorm:"size:200;not null"`
	Slug      string     `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	Status    string     `json:"status" gorm:"default:'ACTIVE'"`
	OwnerID   uuid.UUID  `json:"owner_id" gorm:"type:uuid;not null"`
	ParentID  *uuid.UUID `json:"parent_id" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
