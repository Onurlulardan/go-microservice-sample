package notification

import (
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       *uuid.UUID  `json:"user_id,omitempty" gorm:"type:uuid;index"`
	Method       string      `json:"method" gorm:"type:varchar(10);not null"`
	Path         string      `json:"path" gorm:"type:varchar(500);not null"`
	StatusCode   int         `json:"status_code" gorm:"not null;index"`
	RequestBody  interface{} `json:"request_body,omitempty" gorm:"type:jsonb"`
	ResponseBody interface{} `json:"response_body,omitempty" gorm:"type:jsonb"`
	IPAddress    string      `json:"ip_address" gorm:"type:varchar(45)"`
	UserAgent    string      `json:"user_agent" gorm:"type:text"`
	Duration     int64       `json:"duration_ms" gorm:"not null"` // milliseconds
	RequestID    string      `json:"request_id" gorm:"type:varchar(100);index"`
	CreatedAt    time.Time   `json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName returns the table name for AuditLog
func (AuditLog) TableName() string {
	return "audit_logs"
}
