package main

import (
	"log"
	"net/http"
	"strings"

	"forgecrud-backend/core-service/handlers"
	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	router := gin.Default()

	// User routes
	router.GET("/api/users", handlers.GetUsers)
	router.GET("/api/users/:id", handlers.GetUser)
	router.POST("/api/users", handlers.CreateUser)
	router.PUT("/api/users/:id", handlers.UpdateUser)
	router.DELETE("/api/users/:id", handlers.DeleteUser)
	router.GET("/api/users/:id/permissions", handlers.GetUserPermissions)

	// Role routes
	router.GET("/api/roles", handlers.GetRoles)
	router.GET("/api/roles/:id", handlers.GetRole)
	router.POST("/api/roles", handlers.CreateRole)
	router.PUT("/api/roles/:id", handlers.UpdateRole)
	router.DELETE("/api/roles/:id", handlers.DeleteRole)
	router.GET("/api/roles/:id/permissions", handlers.GetRolePermissions)

	// Organization routes
	router.GET("/api/organizations", handlers.GetOrganizations)
	router.GET("/api/organizations/:id", handlers.GetOrganization)
	router.POST("/api/organizations", handlers.CreateOrganization)
	router.PUT("/api/organizations/:id", handlers.UpdateOrganization)
	router.DELETE("/api/organizations/:id", handlers.DeleteOrganization)
	router.GET("/api/organizations/:id/permissions", handlers.GetOrganizationPermissions)

	// Test endpoint
	router.GET("/api/core/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Core service working!",
			"service":  "core",
			"port":     "8003",
			"database": "connected",
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "core",
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Parse port from config URL
	port := strings.Split(config.GetConfig().CoreServiceURL, ":")[2]
	log.Printf("Core Service starting on port %s...", port)
	router.Run(":" + port)
}
