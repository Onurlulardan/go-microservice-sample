package models

import (
	"time"

	"github.com/google/uuid"
)

// Resources table
type Resource struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Slug        string    `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`
	IsSystem    bool      `json:"is_system" gorm:"default:false;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Actions table
type Action struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"size:100;not null"`
	Slug        string    `json:"slug" gorm:"size:100;uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`
	IsSystem    bool      `json:"is_system" gorm:"default:false;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permissions table (3 Seviyeli Hedef Sistem)
type Permission struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ResourceID     uuid.UUID  `json:"resource_id" gorm:"type:uuid;not null"`
	Target         string     `json:"target" gorm:"type:varchar(20);not null"` // USER, ROLE, ORGANIZATION
	UserID         *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	RoleID         *uuid.UUID `json:"role_id" gorm:"type:uuid"`
	OrganizationID *uuid.UUID `json:"organization_id" gorm:"type:uuid"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	Resource          Resource           `json:"resource" gorm:"foreignKey:ResourceID"`
	User              *User              `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Role              *Role              `json:"role,omitempty" gorm:"foreignKey:RoleID"`
	Organization      *Organization      `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	PermissionActions []PermissionAction `json:"permission_actions" gorm:"foreignKey:PermissionID"`
}

// Permission Actions (Many-to-Many relationship between Permissions and Actions)
type PermissionAction struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:uuid;not null"`
	ActionID     uuid.UUID `json:"action_id" gorm:"type:uuid;not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relations
	Permission Permission `json:"permission" gorm:"foreignKey:PermissionID"`
	Action     Action     `json:"action" gorm:"foreignKey:ActionID"`
}
