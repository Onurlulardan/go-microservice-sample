package main

import (
	"log"
	"net/http"
	"strings"

	"forgecrud-backend/permission-service/handlers"
	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database"
	"forgecrud-backend/shared/utils/cache"

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

	// Initialize Redis Cache Manager
	if err := cache.InitCacheManager(); err != nil {
		log.Printf("⚠️  Warning: Redis cache not available: %v", err)
		log.Println("🔄 Service will continue without caching...")
	} else {
		// Test cache connection
		cacheManager := cache.GetCacheManager()
		if cacheManager != nil {
			if err := cacheManager.TestConnection(); err != nil {
				log.Printf("⚠️  Warning: Redis connection test failed: %v", err)
			}
		}
	}

	router := gin.Default()

	// Resource Management Routes
	router.GET("/api/permissions/resources", handlers.GetResources)
	router.POST("/api/permissions/resources", handlers.CreateResource)
	router.GET("/api/permissions/resources/:id", handlers.GetResource)
	router.PUT("/api/permissions/resources/:id", handlers.UpdateResource)
	router.DELETE("/api/permissions/resources/:id", handlers.DeleteResource)

	// Action Management Routes
	router.GET("/api/permissions/actions", handlers.GetActions)
	router.POST("/api/permissions/actions", handlers.CreateAction)
	router.GET("/api/permissions/actions/:id", handlers.GetAction)
	router.PUT("/api/permissions/actions/:id", handlers.UpdateAction)
	router.DELETE("/api/permissions/actions/:id", handlers.DeleteAction)

	// Permission Management Routes
	router.GET("/api/permissions", handlers.GetPermissions)
	router.POST("/api/permissions", handlers.CreatePermission)
	router.GET("/api/permissions/:id", handlers.GetPermission)
	router.PUT("/api/permissions/:id", handlers.UpdatePermission)
	router.DELETE("/api/permissions/:id", handlers.DeletePermission)

	// Permission Check Routes
	router.POST("/api/permissions/check", handlers.CheckPermission)
	router.POST("/api/permissions/batch-check", handlers.BatchCheckPermissions)

	// Cache Management Routes
	router.GET("/api/permissions/cache/stats", handlers.GetCacheStats)
	router.POST("/api/permissions/cache/invalidate/:user_id", handlers.InvalidateUserPermissions)
	router.POST("/api/permissions/cache/invalidate/role/:role_id", handlers.InvalidateRolePermissions)
	router.POST("/api/permissions/cache/invalidate/org/:org_id", handlers.InvalidateOrgPermissions)
	router.POST("/api/permissions/cache/invalidate/all", handlers.InvalidateAllPermissions)

	// Test endpoint
	router.GET("/api/permission/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Permission service working!",
			"service":  "permission",
			"port":     "8002",
			"database": "connected",
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "permission",
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := strings.Split(config.GetConfig().PermissionServiceURL, ":")[2]
	log.Printf("Permission Service starting on port %s...", port)
	router.Run(":" + port)
}
