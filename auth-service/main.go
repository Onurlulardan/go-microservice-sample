package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forgecrud-backend/auth-service/handlers"
	"forgecrud-backend/auth-service/middleware"
	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// getIntConfig is a helper function to get integer configuration values
func getIntConfig(key string, defaultValue int) int {
	strValue := config.GetConfig().GetField(key)
	if strValue == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(strValue)
	if err != nil {
		log.Printf("Warning: Could not convert %s value '%s' to int, using default %d", key, strValue, defaultValue)
		return defaultValue
	}

	return intValue
}

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(database.GetDB())

	// Initialize rate limiter
	rateLimiterCleanupTime := 30 * time.Minute
	rateLimiter := middleware.NewRateLimiter(rateLimiterCleanupTime)

	// Rate limiting configs
	generalConfig := middleware.RateLimitConfig{
		MaxRequests:   getIntConfig("RateLimitMaxRequests", 100),
		TimeWindow:    time.Duration(getIntConfig("RateLimitTimeWindowSeconds", 60)) * time.Second,
		BlockDuration: time.Duration(getIntConfig("RateLimitBlockDurationMinutes", 15)) * time.Minute,
	}

	loginConfig := middleware.RateLimitConfig{
		MaxRequests:   getIntConfig("LoginRateLimitMaxAttempts", 5),
		TimeWindow:    time.Duration(getIntConfig("LoginRateLimitWindowSeconds", 300)) * time.Second,
		BlockDuration: time.Duration(getIntConfig("LoginRateLimitBlockMinutes", 30)) * time.Minute,
	}

	registerConfig := middleware.RateLimitConfig{
		MaxRequests:   getIntConfig("RegisterRateLimitMaxAttempts", 3),
		TimeWindow:    time.Duration(getIntConfig("RegisterRateLimitWindowHours", 24)) * time.Hour,
		BlockDuration: time.Duration(getIntConfig("RegisterRateLimitBlockHours", 48)) * time.Hour,
	}

	passwordResetConfig := middleware.RateLimitConfig{
		MaxRequests:   getIntConfig("PasswordResetMaxAttempts", 3),
		TimeWindow:    time.Duration(getIntConfig("PasswordResetWindowMinutes", 60)) * time.Minute,
		BlockDuration: time.Duration(getIntConfig("PasswordResetBlockHours", 24)) * time.Hour,
	}

	router := gin.Default()

	// Auth endpoints
	router.POST("/api/auth/login", rateLimiter.LoginRateLimitMiddleware(loginConfig), authHandler.Login)
	router.POST("/api/auth/logout", middleware.AuthMiddleware(), authHandler.Logout)
	router.POST("/api/auth/register", rateLimiter.RegistrationRateLimitMiddleware(registerConfig), authHandler.Register)
	router.POST("/api/auth/refresh", rateLimiter.RateLimitMiddleware(generalConfig), authHandler.Refresh)
	router.POST("/api/auth/validate", rateLimiter.RateLimitMiddleware(generalConfig), authHandler.Validate)
	router.POST("/api/auth/blacklist", middleware.AuthMiddleware(), authHandler.Blacklist)

	// Email verification endpoints
	router.POST("/api/auth/create-verification-token", rateLimiter.RateLimitMiddleware(generalConfig), authHandler.CreateVerificationToken)
	router.GET("/api/auth/verify-email/:token", authHandler.VerifyEmail)

	// Password management endpoints
	router.POST("/api/auth/change-password", middleware.AuthMiddleware(), authHandler.ChangePassword)
	router.POST("/api/auth/forgot-password", rateLimiter.PasswordResetRateLimitMiddleware(passwordResetConfig), authHandler.ForgotPassword)
	router.POST("/api/auth/reset-password", rateLimiter.PasswordResetRateLimitMiddleware(passwordResetConfig), authHandler.ResetPassword)

	// Security features endpoints
	router.GET("/api/auth/sessions", middleware.AuthMiddleware(), authHandler.ListSessions)
	router.DELETE("/api/auth/sessions/:id", middleware.AuthMiddleware(), authHandler.TerminateSession)
	router.DELETE("/api/auth/sessions", middleware.AuthMiddleware(), authHandler.TerminateAllSessions)
	router.POST("/api/auth/sessions/terminate-all", middleware.AuthMiddleware(), authHandler.TerminateAllSessions)
	router.GET("/api/auth/login-history", middleware.AuthMiddleware(), authHandler.GetLoginHistory)

	// Test endpoint
	router.GET("/api/auth/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Auth service working!",
			"service":  "auth",
			"port":     "8001",
			"database": "connected",
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "auth",
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := strings.Split(config.GetConfig().AuthServiceURL, ":")[2]
	log.Printf("Auth Service starting on port %s...", port)
	router.Run(":" + port)
}
