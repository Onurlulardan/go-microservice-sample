package notification

import (
	"time"

	"github.com/google/uuid"
)

// NotificationLevel represents the severity level of a notification
type NotificationLevel string

const (
	NotificationLevelSuccess NotificationLevel = "success"
	NotificationLevelError   NotificationLevel = "error"
	NotificationLevelWarning NotificationLevel = "warning"
	NotificationLevelInfo    NotificationLevel = "info"
)

// Notification represents a real-time notification
type Notification struct {
	ID        uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    *uuid.UUID        `json:"user_id,omitempty" gorm:"type:uuid;index"`
	Type      string            `json:"type" gorm:"type:varchar(50);not null"`
	Level     NotificationLevel `json:"level" gorm:"type:varchar(20);not null;default:'info'"`
	Title     string            `json:"title" gorm:"type:varchar(200);not null"`
	Message   string            `json:"message" gorm:"type:text;not null"`
	Action    string            `json:"action,omitempty" gorm:"type:varchar(100)"`
	EntityID  *uuid.UUID        `json:"entity_id,omitempty" gorm:"type:uuid"`
	Entity    string            `json:"entity,omitempty" gorm:"type:varchar(100)"`
	Data      interface{}       `json:"data,omitempty" gorm:"type:jsonb"`
	IsRead    bool              `json:"is_read" gorm:"default:false;index"`
	CreatedAt time.Time         `json:"created_at" gorm:"autoCreateTime;index"`
	ReadAt    *time.Time        `json:"read_at,omitempty"`
}

// TableName returns the table name for Notification
func (Notification) TableName() string {
	return "notifications"
}

// WebSocketMessage represents a WebSocket message format
type WebSocketMessage struct {
	Type      string            `json:"type"`
	Level     NotificationLevel `json:"level"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Timestamp time.Time         `json:"timestamp"`
	Action    string            `json:"action,omitempty"`
	EntityID  *uuid.UUID        `json:"entity_id,omitempty"`
	Entity    string            `json:"entity,omitempty"`
	UserID    *uuid.UUID        `json:"user_id,omitempty"`
	Data      interface{}       `json:"data,omitempty"`
}

// GetCurrentTime returns current time for WebSocket messages
func GetCurrentTime() time.Time {
	return time.Now().UTC()
}
