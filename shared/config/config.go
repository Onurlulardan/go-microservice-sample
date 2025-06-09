package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT
	JWTSecret            string
	JWTExpireHours       string
	JWTRefreshExpireDays string

	// API Gateway URL
	APIGatewayURL string

	// Super Admin
	SuperAdminEmail    string
	SuperAdminPassword string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       string

	// Email Configuration
	EmailFrom     string
	EmailFromName string
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPUseTLS    bool

	// Rate Limiting
	RateLimitMaxRequests          string
	RateLimitTimeWindowSeconds    string
	RateLimitBlockDurationMinutes string

	// Login Rate Limiting
	LoginRateLimitMaxAttempts   string
	LoginRateLimitWindowSeconds string
	LoginRateLimitBlockMinutes  string

	// Register Rate Limiting
	RegisterRateLimitMaxAttempts string
	RegisterRateLimitWindowHours string
	RegisterRateLimitBlockHours  string

	// Password Reset Rate Limiting
	PasswordResetMaxAttempts   string
	PasswordResetWindowMinutes string
	PasswordResetBlockHours    string

	// Frontend URL
	FrontendURL string

	// Service URLs (Dynamic based on environment)
	AuthServiceURL         string
	PermissionServiceURL   string
	CoreServiceURL         string
	NotificationServiceURL string
	DocumentServiceURL     string

	// MinIO Configuration
	MinIOServerURL    string
	MinIORootUser     string
	MinIORootPassword string
	MinIOUseSSL       bool
	MinIOBucketName   string

	// Document Service Configuration
	DocumentServiceMaxFileSize  string
	DocumentServiceAllowedTypes string
}

var cfg *Config

// LoadConfig loads configuration from environment variables
func LoadConfig() {
	envPaths := []string{
		".env",
		"../.env",
		"../../.env",
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("✅ Environment loaded from: %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	cfg = &Config{
		// Database
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "forgecrud"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		// JWT
		JWTSecret:            getEnv("JWT_SECRET", "your-secret-key-change-this"),
		JWTExpireHours:       getEnv("JWT_EXPIRE_HOURS", "3"),
		JWTRefreshExpireDays: getEnv("JWT_REFRESH_EXPIRE_DAYS", "1"),

		// API Gateway URL
		APIGatewayURL: getEnv("API_GATEWAY_URL", "http://localhost:8000"),

		// Super Admin
		SuperAdminEmail:    getEnv("SUPER_ADMIN_EMAIL", "admin@forgecrud.com"),
		SuperAdminPassword: getEnv("SUPER_ADMIN_PASSWORD", "admin123"),

		// Redis
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnv("REDIS_DB", "0"),

		// Email Configuration
		EmailFrom:     getEnv("EMAIL_FROM", "noreply@forgecrud.com"),
		EmailFromName: getEnv("EMAIL_FROM_NAME", "ForgeCRUD"),
		SMTPHost:      getEnv("SMTP_HOST", "smtp.example.com"),
		SMTPPort:      getEnv("SMTP_PORT", "587"),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPUseTLS:    getEnvAsBool("SMTP_USE_TLS", false),

		// Rate Limiting - Genel
		RateLimitMaxRequests:          getEnv("RATE_LIMIT_MAX_REQUESTS", "100"),
		RateLimitTimeWindowSeconds:    getEnv("RATE_LIMIT_TIME_WINDOW_SECONDS", "60"),
		RateLimitBlockDurationMinutes: getEnv("RATE_LIMIT_BLOCK_DURATION_MINUTES", "15"),

		// Login Rate Limiting
		LoginRateLimitMaxAttempts:   getEnv("LOGIN_RATE_LIMIT_MAX_ATTEMPTS", "5"),
		LoginRateLimitWindowSeconds: getEnv("LOGIN_RATE_LIMIT_WINDOW_SECONDS", "300"),
		LoginRateLimitBlockMinutes:  getEnv("LOGIN_RATE_LIMIT_BLOCK_MINUTES", "30"),

		// Register Rate Limiting
		RegisterRateLimitMaxAttempts: getEnv("REGISTER_RATE_LIMIT_MAX_ATTEMPTS", "3"),
		RegisterRateLimitWindowHours: getEnv("REGISTER_RATE_LIMIT_WINDOW_HOURS", "24"),
		RegisterRateLimitBlockHours:  getEnv("REGISTER_RATE_LIMIT_BLOCK_HOURS", "48"),

		// Password Reset Rate Limiting
		PasswordResetMaxAttempts:   getEnv("PASSWORD_RESET_MAX_ATTEMPTS", "3"),
		PasswordResetWindowMinutes: getEnv("PASSWORD_RESET_WINDOW_MINUTES", "60"),
		PasswordResetBlockHours:    getEnv("PASSWORD_RESET_BLOCK_HOURS", "24"),

		// Frontend URL
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),

		// Service URLs - Environment-based configuration
		AuthServiceURL:         getEnv("AUTH_SERVICE_URL", "http://localhost:8001"),
		PermissionServiceURL:   getEnv("PERMISSION_SERVICE_URL", "http://localhost:8002"),
		CoreServiceURL:         getEnv("CORE_SERVICE_URL", "http://localhost:8003"),
		NotificationServiceURL: getEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8004"),
		DocumentServiceURL:     getEnv("DOCUMENT_SERVICE_URL", "http://localhost:8005"),

		// MinIO Configuration
		MinIOServerURL:    getEnv("MINIO_SERVER_URL", "http://localhost:9000"),
		MinIORootUser:     getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinIORootPassword: getEnv("MINIO_ROOT_PASSWORD", "minioadmin"),
		MinIOUseSSL:       getEnvAsBool("MINIO_USE_SSL", false),
		MinIOBucketName:   getEnv("MINIO_BUCKET_NAME", "forgecrud-documents"),

		// Document Service Configuration
		DocumentServiceMaxFileSize:  getEnv("DOCUMENT_SERVICE_MAX_FILE_SIZE", "100MB"),
		DocumentServiceAllowedTypes: getEnv("DOCUMENT_SERVICE_ALLOWED_TYPES", ".pdf,.doc,.docx,.txt,.jpg,.jpeg,.png"),
	}

	log.Println("✅ Configuration loaded successfully")
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	if cfg == nil {
		LoadConfig()
	}
	return cfg
}

// GetField returns a configuration field by name
func (c *Config) GetField(key string) string {
	switch key {
	// Database
	case "DBHost":
		return c.DBHost
	case "DBPort":
		return c.DBPort
	case "DBUser":
		return c.DBUser
	case "DBPassword":
		return c.DBPassword
	case "DBName":
		return c.DBName
	case "DBSSLMode":
		return c.DBSSLMode

	// Services
	case "APIGatewayURL":
		return c.APIGatewayURL

	// JWT
	case "JWTSecret":
		return c.JWTSecret
	case "JWTExpireHours":
		return c.JWTExpireHours

	// Rate Limiting
	case "RateLimitMaxRequests":
		return c.RateLimitMaxRequests
	case "RateLimitTimeWindowSeconds":
		return c.RateLimitTimeWindowSeconds
	case "RateLimitBlockDurationMinutes":
		return c.RateLimitBlockDurationMinutes
	case "LoginRateLimitMaxAttempts":
		return c.LoginRateLimitMaxAttempts
	case "LoginRateLimitWindowSeconds":
		return c.LoginRateLimitWindowSeconds
	case "LoginRateLimitBlockMinutes":
		return c.LoginRateLimitBlockMinutes
	case "RegisterRateLimitMaxAttempts":
		return c.RegisterRateLimitMaxAttempts
	case "RegisterRateLimitWindowHours":
		return c.RegisterRateLimitWindowHours
	case "RegisterRateLimitBlockHours":
		return c.RegisterRateLimitBlockHours
	case "PasswordResetMaxAttempts":
		return c.PasswordResetMaxAttempts
	case "PasswordResetWindowMinutes":
		return c.PasswordResetWindowMinutes
	case "PasswordResetBlockHours":
		return c.PasswordResetBlockHours

	// Service URLs
	case "AuthServiceURL":
		return c.AuthServiceURL
	case "PermissionServiceURL":
		return c.PermissionServiceURL
	case "CoreServiceURL":
		return c.CoreServiceURL
	case "NotificationServiceURL":
		return c.NotificationServiceURL

	default:
		return ""
	}
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets environment variable as integer with default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetRateLimitMaxRequests returns the rate limit max requests as integer
func (c *Config) GetRateLimitMaxRequests() int {
	if value, err := strconv.Atoi(c.RateLimitMaxRequests); err == nil {
		return value
	}
	return 100
}

// GetRateLimitTimeWindowSeconds returns the rate limit time window as integer
func (c *Config) GetRateLimitTimeWindowSeconds() int {
	if value, err := strconv.Atoi(c.RateLimitTimeWindowSeconds); err == nil {
		return value
	}
	return 60
}

// GetRateLimitBlockDurationMinutes returns the rate limit block duration as integer
func (c *Config) GetRateLimitBlockDurationMinutes() int {
	if value, err := strconv.Atoi(c.RateLimitBlockDurationMinutes); err == nil {
		return value
	}
	return 15
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
