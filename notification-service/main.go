package main

import (
	"log"
	"net/http"
	"strings"

	"forgecrud-backend/notification-service/handlers"
	"forgecrud-backend/notification-service/services"
	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database"

	"github.com/gin-gonic/gin"
)

// @title Notification Service API
// @version 1.0
// @description Unified Response & Real-time Notifications
// @host localhost:8004
// @BasePath /api

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	router := gin.Default()

	// Initialize email service
	emailService := services.NewEmailService(config.GetConfig())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "notification-service",
			"status":  "healthy",
		})
	})

	// Email routes
	emailHandler := handlers.NewEmailHandler(emailService, config.GetConfig())
	emailRoutes := router.Group("/api/notifications/email")
	{
		emailRoutes.POST("/send", emailHandler.SendEmail)
		emailRoutes.POST("/welcome", emailHandler.SendWelcomeEmail)
		emailRoutes.POST("/password-reset", emailHandler.SendPasswordResetEmail)
		emailRoutes.POST("/verification", emailHandler.SendVerificationEmail)
		emailRoutes.POST("/resend-verification", emailHandler.ResendVerificationEmail)
	}

	// Notification routes
	router.GET("/api/notifications", handlers.GetNotifications)
	router.GET("/api/notifications/:id", handlers.GetNotification)
	router.POST("/api/notifications", handlers.CreateNotification)
	router.PUT("/api/notifications/:id/read", handlers.MarkAsRead)
	router.DELETE("/api/notifications/:id", handlers.DeleteNotification)

	// WebSocket endpoint
	router.GET("/ws/notifications/:user_id", handlers.HandleWebSocket)

	// WebSocket message sending endpoint (for API Gateway)
	router.POST("/ws/send", handlers.SendWebSocketMessage)

	port := strings.Split(config.GetConfig().NotificationServiceURL, ":")[2]
	log.Printf("🔔 Notification Service starting on port %s...", port)
	log.Fatal(router.Run(":" + port))
}
