package main

import (
	"forgecrud-backend/document-service/services"
	"forgecrud-backend/shared/config"
	"log"
	"strings"

	"forgecrud-backend/document-service/handlers"
	"forgecrud-backend/shared/database"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize MinIO service
	minioService, err := services.NewMinIOService()
	if err != nil {
		log.Fatalf("❌ Failed to initialize MinIO service: %v", err)
	}

	// Test MinIO connection
	if err := minioService.TestConnection(); err != nil {
		log.Fatalf("❌ MinIO connection test failed: %v", err)
	}

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	// Initialize Gin router
	router := gin.Default()

	//Folder Routes
	router.GET("/api/folders", handlers.GetFolders)
	router.GET("/api/folders/:id", handlers.GetFolder)
	router.GET("/api/folders/:id/contents", handlers.GetFolderContents)
	router.POST("/api/folders", handlers.CreateFolder)
	router.PUT("/api/folders/:id", handlers.UpdateFolder)
	router.POST("/api/folders/:id/move", handlers.MoveFolder)
	router.DELETE("/api/folders/:id", handlers.DeleteFolder)
	router.GET("/api/folders/:id/download", handlers.DownloadFolder)

	// Document Routes
	router.POST("/api/documents", handlers.UploadDocument)
	router.GET("/api/documents", handlers.GetDocuments)
	router.GET("/api/documents/:id", handlers.GetDocument)
	router.GET("/api/documents/:id/download", handlers.DownloadDocument)
	router.PUT("/api/documents/:id", handlers.UpdateDocument)
	router.POST("/api/documents/:id/move", handlers.MoveDocument)
	router.DELETE("/api/documents/:id", handlers.DeleteDocument)
	router.POST("/documents/:id/copy", handlers.CopyDocument)

	// Document Version Routes
	router.GET("/api/documents/:id/versions", handlers.GetDocumentVersions)
	router.GET("/api/documents/:id/versions/latest", handlers.GetLatestDocumentVersion)
	router.POST("/api/documents/:id/versions", handlers.UploadDocumentVersion)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "document-service",
			"message": "Document service is running",
		})
	})

	// Start server
	// Parse port from config URL
	port := strings.Split(config.GetConfig().DocumentServiceURL, ":")[2]
	log.Printf("Document Service starting on port %s...", port)
	router.Run(":" + port)
}
