package handlers

import (
	"net/http"

	"forgecrud-backend/notification-service/services"
	"forgecrud-backend/shared/database/models/notification"

	"github.com/gin-gonic/gin"
)

// HandleWebSocket handles WebSocket connection requests
// @Summary WebSocket Connection
// @Description Establish WebSocket connection for real-time notifications
// @Tags websocket
// @Param user_id path string true "User ID"
// @Router /ws/notifications/{user_id} [get]
func HandleWebSocket(c *gin.Context) {
	wsManager := services.GetWebSocketManager()
	wsManager.HandleWebSocketConnection(c)
}

// SendWebSocketMessage sends message via WebSocket service (for API Gateway)
// @Summary Send WebSocket Message
// @Description Send real-time message to specific user via WebSocket
// @Tags websocket
// @Accept json
// @Produce json
// @Param payload body SendMessageRequest true "Message payload"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /ws/send [post]
func SendWebSocketMessage(c *gin.Context) {
	var request SendMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	wsManager := services.GetWebSocketManager()

	// Send message to specific user
	if err := wsManager.SendToUser(request.UserID, request.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket message sent successfully",
		"user_id": request.UserID,
	})
}

// SendMessageRequest represents the request payload for sending WebSocket messages
type SendMessageRequest struct {
	UserID  string                         `json:"user_id" binding:"required"`
	Message *notification.WebSocketMessage `json:"message" binding:"required"`
}
