package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"forgecrud-backend/api-gateway/middleware"
	"forgecrud-backend/api-gateway/routes"
	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/utils/permission"

	_ "forgecrud-backend/docs/swagger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title ForgeCRUD API
// @version 1.0
// @description Complete API documentation for the ForgeCRUD microservices platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.forgecrud.com/support
// @contact.email support@forgecrud.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8000
// @BasePath /api
// @schemes http https

// @tag.name auth
// @tag.description Authentication operations
// @tag.docs.url http://forgecrud.com/docs/auth
// @tag.docs.description Authentication detailed documentation

// @tag.name users
// @tag.description User management operations

// @tag.name roles
// @tag.description Role management operations

// @tag.name organizations
// @tag.description Organization management operations

// @tag.name permissions
// @tag.description Permission management operations

// @tag.name resources
// @tag.description Resource management operations

// @tag.name actions
// @tag.description Action management operations

// @tag.name documents
// @tag.description Document management operations

// @tag.name folders
// @tag.description Folder management operations

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.

func main() {
	// Load configuration
	config.LoadConfig()
	cfg := config.GetConfig()

	// Initialize permission client with config-based URL
	permission.InitPermissionClient(cfg.PermissionServiceURL)

	// Initialize global rate limiter
	rateLimiter := middleware.NewRateLimiter(5 * time.Minute) // Cleanup every 5 minutes

	// Global rate limit configuration from environment variables
	globalRateConfig := middleware.NewRateLimitConfig()

	// Gin router oluştur
	router := gin.Default()

	// Add CORS middleware
	router.Use(cors.Default())

	// Global rate limiter middleware
	router.Use(rateLimiter.GlobalRateLimitMiddleware(globalRateConfig))

	// Add unified response middleware (transforms all service responses)
	router.Use(middleware.UnifiedResponseMiddleware())

	// Health check endpoint
	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "API Gateway is running", "Port": "8000"})
	})

	// Test endpoint
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API Gateway working!",
			"service": "gateway",
		})
	})

	// Auth routes (no permission required for login/register)
	// Note: Auth Service has its own internal rate limiting
	router.Any("/api/auth/*path",
		routes.ProxyToService("auth"))

	// Protected routes with permission checks

	// Permission service routes
	// Permission Management routes
	router.GET("/api/permissions",
		middleware.RequirePermission("permissions", "read"),
		routes.ProxyToService("permissions"))
	router.POST("/api/permissions",
		middleware.RequirePermission("permissions", "create"),
		routes.ProxyToService("permissions"))
	router.PUT("/api/permissions/:id",
		middleware.RequirePermission("permissions", "update"),
		routes.ProxyToService("permissions"))
	router.DELETE("/api/permissions/:id",
		middleware.RequirePermission("permissions", "delete"),
		routes.ProxyToService("permissions"))

	// Resource Management routes
	router.GET("/api/permissions/resources",
		middleware.RequirePermission("permissions", "read"),
		routes.ProxyToService("permissions"))
	router.POST("/api/permissions/resources",
		middleware.RequirePermission("permissions", "create"),
		routes.ProxyToService("permissions"))
	router.PUT("/api/permissions/resources/:id",
		middleware.RequirePermission("permissions", "update"),
		routes.ProxyToService("permissions"))
	router.DELETE("/api/permissions/resources/:id",
		middleware.RequirePermission("permissions", "delete"),
		routes.ProxyToService("permissions"))

	// Action Management routes
	router.GET("/api/permissions/actions",
		middleware.RequirePermission("permissions", "read"),
		routes.ProxyToService("permissions"))
	router.POST("/api/permissions/actions",
		middleware.RequirePermission("permissions", "create"),
		routes.ProxyToService("permissions"))
	router.PUT("/api/permissions/actions/:id",
		middleware.RequirePermission("permissions", "update"),
		routes.ProxyToService("permissions"))
	router.DELETE("/api/permissions/actions/:id",
		middleware.RequirePermission("permissions", "delete"),
		routes.ProxyToService("permissions"))

	// Cache operations (admin only)
	router.Any("/api/permissions/cache/*path",
		middleware.RequirePermission("permissions", "manage"),
		routes.ProxyToService("permissions"))

	// Core service routes
	router.GET("/api/users",
		middleware.RequirePermission("users", "read"),
		routes.ProxyToService("core"))
	router.POST("/api/users",
		middleware.RequirePermission("users", "create"),
		routes.ProxyToService("core"))
	router.PUT("/api/users/:id",
		middleware.RequirePermission("users", "update"),
		routes.ProxyToService("core"))
	router.DELETE("/api/users/:id",
		middleware.RequirePermission("users", "delete"),
		routes.ProxyToService("core"))
	router.GET("/api/users/:id/permissions",
		middleware.RequirePermission("users", "read"),
		routes.ProxyToService("core"))

	// Role routes
	router.GET("/api/roles",
		middleware.RequirePermission("roles", "read"),
		routes.ProxyToService("core"))
	router.POST("/api/roles",
		middleware.RequirePermission("roles", "create"),
		routes.ProxyToService("core"))
	router.PUT("/api/roles/:id",
		middleware.RequirePermission("roles", "update"),
		routes.ProxyToService("core"))
	router.DELETE("/api/roles/:id",
		middleware.RequirePermission("roles", "delete"),
		routes.ProxyToService("core"))
	router.GET("/api/roles/:id/permissions",
		middleware.RequirePermission("roles", "read"),
		routes.ProxyToService("core"))

	// Organization routes
	router.GET("/api/organizations",
		middleware.RequirePermission("organizations", "read"),
		routes.ProxyToService("core"))
	router.POST("/api/organizations",
		middleware.RequirePermission("organizations", "create"),
		routes.ProxyToService("core"))
	router.PUT("/api/organizations/:id",
		middleware.RequirePermission("organizations", "update"),
		routes.ProxyToService("core"))
	router.DELETE("/api/organizations/:id",
		middleware.RequirePermission("organizations", "delete"),
		routes.ProxyToService("core"))
	router.GET("/api/organizations/:id/permissions",
		middleware.RequirePermission("organizations", "read"),
		routes.ProxyToService("core"))

	// Notification service routes
	router.GET("/api/notifications",
		middleware.RequirePermission("notifications", "read"),
		routes.ProxyToService("notification"))
	router.POST("/api/notifications",
		middleware.RequirePermission("notifications", "create"),
		routes.ProxyToService("notification"))
	router.GET("/api/notifications/:id",
		middleware.RequirePermission("notifications", "read"),
		routes.ProxyToService("notification"))
	router.PUT("/api/notifications/:id",
		middleware.RequirePermission("notifications", "update"),
		routes.ProxyToService("notification"))
	router.DELETE("/api/notifications/:id",
		middleware.RequirePermission("notifications", "delete"),
		routes.ProxyToService("notification"))

	// Email service routes
	// Protected route - only admin/system can send arbitrary emails
	router.POST("/api/notifications/email/send",
		middleware.RequirePermission("notifications", "create"),
		routes.ProxyToService("notification"))

	router.POST("/api/notifications/email/welcome",
		routes.ProxyToService("notification"))
	router.POST("/api/notifications/email/password-reset",
		routes.ProxyToService("notification"))
	router.POST("/api/notifications/email/verification",
		routes.ProxyToService("notification"))
	router.POST("/api/notifications/email/resend-verification",
		routes.ProxyToService("notification"))

	// WebSocket routes
	router.GET("/ws/notifications/:user_id",
		middleware.RequirePermission("notifications", "read"),
		routes.ProxyToService("notification"))

	// Document service routes
	// Folder routes
	router.GET("/api/folders",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.POST("/api/folders",
		middleware.RequirePermission("file-management", "create"),
		routes.ProxyToService("document"))
	router.GET("/api/folders/:id",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.PUT("/api/folders/:id",
		middleware.RequirePermission("file-management", "update"),
		routes.ProxyToService("document"))
	router.POST("/api/folders/:id/move",
		middleware.RequirePermission("file-management", "update"),
		routes.ProxyToService("document"))
	router.DELETE("/api/folders/:id",
		middleware.RequirePermission("file-management", "delete"),
		routes.ProxyToService("document"))
	router.GET("/api/folders/:id/contents",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.GET("/api/folders/:id/download",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))

	// Document routes
	router.GET("/api/documents",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.POST("/api/documents",
		middleware.RequirePermission("file-management", "create"),
		routes.ProxyToService("document"))
	router.GET("/api/documents/:id",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.GET("/api/documents/:id/download",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.PUT("/api/documents/:id",
		middleware.RequirePermission("file-management", "update"),
		routes.ProxyToService("document"))
	router.DELETE("/api/documents/:id",
		middleware.RequirePermission("file-management", "delete"),
		routes.ProxyToService("document"))
	router.POST("/api/documents/:id/move",
		middleware.RequirePermission("file-management", "update"),
		routes.ProxyToService("document"))
	router.POST("/api/documents/:id/copy",
		middleware.RequirePermission("file-management", "update"),
		routes.ProxyToService("document"))

	// Document version routes
	router.GET("/api/documents/:id/versions",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.GET("/api/documents/:id/versions/latest",
		middleware.RequirePermission("file-management", "read"),
		routes.ProxyToService("document"))
	router.POST("/api/documents/:id/versions",
		middleware.RequirePermission("file-management", "create"),
		routes.ProxyToService("document"))

	// Swagger documentation UI
	// Swagger documentation UI - conditional olarak ekleyelim
	router.GET("/swagger/*any", func(c *gin.Context) {
		// Development environment'ta swagger'ı göster
		if gin.Mode() == gin.DebugMode {
			ginSwagger.WrapHandler(swaggerFiles.Handler)(c)
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Swagger documentation not available in production",
			})
		}
	})

	// Server Start
	port := strings.Split(config.GetConfig().APIGatewayURL, ":")[2]
	log.Printf("API Gateway is running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
