package services

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database/models/notification"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketManager handles all WebSocket connections
type WebSocketManager struct {
	clients    map[string]*websocket.Conn // userID -> connection
	mutex      sync.RWMutex
	upgrader   websocket.Upgrader
	register   chan *ClientConnection
	unregister chan *ClientConnection
	broadcast  chan *notification.WebSocketMessage
}

// ClientConnection represents a client WebSocket connection
type ClientConnection struct {
	UserID     string
	Connection *websocket.Conn
}

// Global WebSocket manager instance
var wsManager *WebSocketManager
var once sync.Once

// GetWebSocketManager returns singleton WebSocket manager
func GetWebSocketManager() *WebSocketManager {
	once.Do(func() {
		wsManager = &WebSocketManager{
			clients: make(map[string]*websocket.Conn),
			upgrader: websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					origin := r.Header.Get("Origin")

					// Get allowed origins from config
					allowedOrigins := []string{
						config.GetConfig().FrontendURL,
					}

					for _, allowed := range allowedOrigins {
						if origin == allowed {
							return true
						}
					}

					log.Printf("🚫 WebSocket connection rejected from origin: %s", origin)
					return false
				},
			},
			register:   make(chan *ClientConnection, 100),
			unregister: make(chan *ClientConnection, 100),
			broadcast:  make(chan *notification.WebSocketMessage, 1000),
		}
		go wsManager.run()
	})
	return wsManager
}

// run handles WebSocket manager event loop
func (wsm *WebSocketManager) run() {
	for {
		select {
		case client := <-wsm.register:
			wsm.registerClient(client)

		case client := <-wsm.unregister:
			wsm.unregisterClient(client)

		case message := <-wsm.broadcast:
			wsm.broadcastMessage(message)
		}
	}
}

// registerClient adds a new client connection
func (wsm *WebSocketManager) registerClient(client *ClientConnection) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	// Close existing connection if any
	if existingConn, exists := wsm.clients[client.UserID]; exists {
		existingConn.Close()
	}

	wsm.clients[client.UserID] = client.Connection
	log.Printf("🔌 WebSocket client connected: %s (Total: %d)", client.UserID, len(wsm.clients))

	// Send welcome message
	welcomeMsg := &notification.WebSocketMessage{
		Type:      "connection",
		Level:     notification.NotificationLevelInfo,
		Title:     "🔌 Connected",
		Message:   "WebSocket connection established",
		Timestamp: notification.GetCurrentTime(),
		UserID:    parseUUID(client.UserID),
	}
	wsm.sendToClient(client.UserID, welcomeMsg)
}

// unregisterClient removes a client connection
func (wsm *WebSocketManager) unregisterClient(client *ClientConnection) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	if _, exists := wsm.clients[client.UserID]; exists {
		delete(wsm.clients, client.UserID)
		client.Connection.Close()
		log.Printf("🔌 WebSocket client disconnected: %s (Total: %d)", client.UserID, len(wsm.clients))
	}
}

// broadcastMessage sends message to all connected clients
func (wsm *WebSocketManager) broadcastMessage(message *notification.WebSocketMessage) {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	successCount := 0
	failCount := 0

	for userID, conn := range wsm.clients {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("❌ Failed to send message to user %s: %v", userID, err)
			// Remove failed connection
			go func(uid string, connection *websocket.Conn) {
				wsm.unregister <- &ClientConnection{UserID: uid, Connection: connection}
			}(userID, conn)
			failCount++
		} else {
			successCount++
		}
	}

	log.Printf("📡 Broadcast sent: %d success, %d failed (Message: %s)",
		successCount, failCount, message.Message)
}

// SendToUser sends message to specific user
func (wsm *WebSocketManager) SendToUser(userID string, message *notification.WebSocketMessage) error {
	wsm.mutex.RLock()
	_, exists := wsm.clients[userID]
	wsm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("user %s not connected", userID)
	}

	return wsm.sendToClient(userID, message)
}

// sendToClient sends message to specific client connection
func (wsm *WebSocketManager) sendToClient(userID string, message *notification.WebSocketMessage) error {
	wsm.mutex.RLock()
	conn, exists := wsm.clients[userID]
	wsm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("user %s not connected", userID)
	}

	err := conn.WriteJSON(message)
	if err != nil {
		log.Printf("❌ Failed to send message to user %s: %v", userID, err)
		// Remove failed connection
		go func() {
			wsm.unregister <- &ClientConnection{UserID: userID, Connection: conn}
		}()
		return err
	}

	log.Printf("📱 Message sent to user %s: %s", userID, message.Message)
	return nil
}

// BroadcastToAll sends message to all connected clients
func (wsm *WebSocketManager) BroadcastToAll(message *notification.WebSocketMessage) {
	select {
	case wsm.broadcast <- message:
		// Message queued successfully
	default:
		log.Printf("⚠️ Broadcast queue full, dropping message: %s", message.Message)
	}
}

// HandleWebSocketConnection upgrades HTTP connection to WebSocket
func (wsm *WebSocketManager) HandleWebSocketConnection(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := wsm.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("❌ Failed to upgrade WebSocket: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	// Register client
	client := &ClientConnection{
		UserID:     userID,
		Connection: conn,
	}

	wsm.register <- client

	// Handle connection lifecycle
	defer func() {
		wsm.unregister <- client
	}()

	// Keep connection alive and handle incoming messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("❌ WebSocket error for user %s: %v", userID, err)
			}
			break
		}

		// Handle incoming messages (ping, pong, etc.)
		if msgType, ok := message["type"].(string); ok {
			switch msgType {
			case "ping":
				pongMsg := &notification.WebSocketMessage{
					Type:      "pong",
					Level:     notification.NotificationLevelInfo,
					Message:   "pong",
					Timestamp: notification.GetCurrentTime(),
					UserID:    parseUUID(userID),
				}
				wsm.sendToClient(userID, pongMsg)
			}
		}
	}
}

// GetConnectedUsers returns list of connected user IDs
func (wsm *WebSocketManager) GetConnectedUsers() []string {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	users := make([]string, 0, len(wsm.clients))
	for userID := range wsm.clients {
		users = append(users, userID)
	}
	return users
}

// GetConnectionCount returns number of active connections
func (wsm *WebSocketManager) GetConnectionCount() int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	return len(wsm.clients)
}

// parseUUID safely parses UUID string
func parseUUID(str string) *uuid.UUID {
	if id, err := uuid.Parse(str); err == nil {
		return &id
	}
	return nil
}
