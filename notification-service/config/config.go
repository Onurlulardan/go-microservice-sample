// filepath: /Users/onuraltuntas/noyan/repos/forgeCRUD/backend/notification-service/config/config.go
package config

import (
	"os"
	"strconv"

	sharedConfig "forgecrud-backend/shared/config"
)

type NotificationConfig struct {
	*sharedConfig.Config

	EmailConfig EmailConfig
}

type EmailConfig struct {
	EnableEmailNotification bool
	QueueSize               int
	RetryAttempts           int
	RetryDelay              int
	Templates               EmailTemplates
}

type EmailTemplates struct {
	WelcomeTemplate       string
	PasswordResetTemplate string
	CriticalErrorTemplate string
	UserActionTemplate    string
	SystemAlertTemplate   string
}

var notificationConfig *NotificationConfig

func LoadNotificationConfig() *NotificationConfig {
	if notificationConfig != nil {
		return notificationConfig
	}

	baseConfig := sharedConfig.GetConfig()

	notificationConfig = &NotificationConfig{
		Config: baseConfig,
		EmailConfig: EmailConfig{
			EnableEmailNotification: getEnvAsBool("EMAIL_NOTIFICATION_ENABLE", true),
			QueueSize:               getEnvAsInt("EMAIL_QUEUE_SIZE", 1000),
			RetryAttempts:           getEnvAsInt("EMAIL_RETRY_ATTEMPTS", 3),
			RetryDelay:              getEnvAsInt("EMAIL_RETRY_DELAY", 30),
			Templates: EmailTemplates{
				WelcomeTemplate:       getEnv("EMAIL_TEMPLATE_WELCOME", "welcome_verification.html"),
				PasswordResetTemplate: getEnv("EMAIL_TEMPLATE_PASSWORD_RESET", "password_reset.html"),
				CriticalErrorTemplate: getEnv("EMAIL_TEMPLATE_CRITICAL", "critical_error.html"),
				UserActionTemplate:    getEnv("EMAIL_TEMPLATE_USER_ACTION", "user_action.html"),
				SystemAlertTemplate:   getEnv("EMAIL_TEMPLATE_SYSTEM_ALERT", "system_alert.html"),
			},
		},
	}

	return notificationConfig
}

func GetNotificationConfig() *NotificationConfig {
	if notificationConfig == nil {
		return LoadNotificationConfig()
	}
	return notificationConfig
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
