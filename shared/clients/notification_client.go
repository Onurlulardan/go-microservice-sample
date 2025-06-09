package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"forgecrud-backend/shared/config"
)

// NotificationClient handles communication with notification service
type NotificationClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewNotificationClient creates a new notification client
func NewNotificationClient() *NotificationClient {
	cfg := config.GetConfig()
	return &NotificationClient{
		baseURL: cfg.APIGatewayURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Email request structs
type WelcomeEmailRequest struct {
	Email            string `json:"email"`
	Name             string `json:"name"`
	VerificationCode string `json:"verification_code"`
}

type PasswordResetEmailRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type CriticalErrorEmailRequest struct {
	AdminName          string   `json:"admin_name"`
	ErrorType          string   `json:"error_type"`
	ServiceName        string   `json:"service_name"`
	ErrorMessage       string   `json:"error_message"`
	StackTrace         string   `json:"stack_trace,omitempty"`
	AffectedServices   []string `json:"affected_services"`
	RecommendedActions []string `json:"recommended_actions"`
	Timestamp          string   `json:"timestamp"`
}

type SystemAlertEmailRequest struct {
	UserName         string   `json:"user_name"`
	AlertType        string   `json:"alert_type"`
	AlertTypeText    string   `json:"alert_type_text"`
	Message          string   `json:"message"`
	Category         string   `json:"category"`
	Severity         string   `json:"severity,omitempty"`
	AffectedServices string   `json:"affected_services,omitempty"`
	Details          string   `json:"details,omitempty"`
	StartTime        string   `json:"start_time,omitempty"`
	EndTime          string   `json:"end_time,omitempty"`
	Duration         string   `json:"duration,omitempty"`
	ActionRequired   string   `json:"action_required,omitempty"`
	NextSteps        []string `json:"next_steps,omitempty"`
	ContactInfo      string   `json:"contact_info,omitempty"`
	Timestamp        string   `json:"timestamp"`
}

type UserActionEmailRequest struct {
	AdminName    string             `json:"admin_name"`
	UserName     string             `json:"user_name"`
	UserEmail    string             `json:"user_email"`
	UserRole     string             `json:"user_role"`
	IPAddress    string             `json:"ip_address"`
	ActionType   string             `json:"action_type"`
	ResourceName string             `json:"resource_name"`
	Status       string             `json:"status"`
	Priority     string             `json:"priority"`
	PriorityText string             `json:"priority_text"`
	Description  string             `json:"description,omitempty"`
	Changes      []UserActionChange `json:"changes,omitempty"`
	Timestamp    string             `json:"timestamp"`
}

type UserActionChange struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// EmailResponse represents email service response
type EmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SentAt  string `json:"sent_at"`
}

// 🎯 Template-based email methods

// SendWelcomeEmail sends welcome verification email
func (nc *NotificationClient) SendWelcomeEmail(to, name, verificationCode string) error {
	request := WelcomeEmailRequest{
		Email:            to,
		Name:             name,
		VerificationCode: verificationCode,
	}
	return nc.sendEmailRequest("/api/notifications/email/verification", request)
}

// SendPasswordResetEmail sends password reset email
func (nc *NotificationClient) SendPasswordResetEmail(to, name, token string) error {
	request := PasswordResetEmailRequest{
		Email: to,
		Name:  name,
		Token: token,
	}
	return nc.sendEmailRequest("/api/notifications/email/password-reset", request)
}

// SendCriticalErrorEmail sends critical error notification to admins
func (nc *NotificationClient) SendCriticalErrorEmail(req CriticalErrorEmailRequest) error {
	return nc.sendEmailRequest("/api/notifications/email/critical-error", req)
}

// SendSystemAlertEmail sends system alert notifications
func (nc *NotificationClient) SendSystemAlertEmail(req SystemAlertEmailRequest) error {
	return nc.sendEmailRequest("/api/notifications/email/system-alert", req)
}

// SendUserActionEmail sends user action notifications
func (nc *NotificationClient) SendUserActionEmail(req UserActionEmailRequest) error {
	return nc.sendEmailRequest("/api/notifications/email/user-action", req)
}

// Generic email sender
func (nc *NotificationClient) sendEmailRequest(endpoint string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	url := fmt.Sprintf("%s%s", nc.baseURL, endpoint)
	resp, err := nc.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification service returned status: %d", resp.StatusCode)
	}

	return nil
}
