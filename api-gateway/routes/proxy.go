package routes

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"forgecrud-backend/shared/config"

	"github.com/gin-gonic/gin"
)

// getServiceURLs returns service URLs from configuration
func getServiceURLs() map[string]string {
	cfg := config.GetConfig()
	return map[string]string{
		"auth":         cfg.AuthServiceURL,
		"permissions":  cfg.PermissionServiceURL,
		"core":         cfg.CoreServiceURL,
		"notification": cfg.NotificationServiceURL,
		"document":     cfg.DocumentServiceURL,
	}
}

// ProxyHandler handles requests and proxies them to the appropriate service
func ProxyToService(serviceName string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get service URLs
		serviceURLs := getServiceURLs()

		// Service URL lookup
		serviceURL, exists := serviceURLs[serviceName]
		if !exists {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Service not found", "service": serviceName})
			return
		}
		// Parse the service URL
		target, err := url.Parse(serviceURL)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid service URL", "service": serviceName})
			return
		}

		// Create a reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(target)

		// add request to proxy
		proxy.ServeHTTP(ctx.Writer, ctx.Request)
	}
}
