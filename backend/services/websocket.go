package services

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"playpulse-panel/database"
	"playpulse-panel/models"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	connections map[string]*websocket.Conn
	mutex       sync.RWMutex
}

var wsManager = &WebSocketManager{
	connections: make(map[string]*websocket.Conn),
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	ServerID  string      `json:"server_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// ConsoleMessage represents a console log message
type ConsoleMessage struct {
	Line      string `json:"line"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"` // "info", "warn", "error"
}

// StatsMessage represents server statistics
type StatsMessage struct {
	ServerID    string  `json:"server_id"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage int64   `json:"memory_usage"`
	DiskUsage   int64   `json:"disk_usage"`
	PlayerCount int     `json:"player_count"`
	TPS         float64 `json:"tps"`
	Status      string  `json:"status"`
}

// StatusMessage represents server status change
type StatusMessage struct {
	ServerID string `json:"server_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *websocket.Conn, userID uuid.UUID) {
	connectionID := uuid.New().String()
	
	// Store connection
	wsManager.mutex.Lock()
	wsManager.connections[connectionID] = c
	wsManager.mutex.Unlock()
	
	// Clean up on disconnect
	defer func() {
		wsManager.mutex.Lock()
		delete(wsManager.connections, connectionID)
		wsManager.mutex.Unlock()
		c.Close()
	}()

	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type: "welcome",
		Data: map[string]string{
			"message":       "Connected to Playpulse Panel",
			"connection_id": connectionID,
		},
	}
	c.WriteJSON(welcomeMsg)

	// Handle incoming messages
	for {
		var msg WebSocketMessage
		err := c.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		switch msg.Type {
		case "subscribe_server":
			handleServerSubscription(c, userID, msg)
		case "unsubscribe_server":
			handleServerUnsubscription(c, userID, msg)
		case "send_command":
			handleCommandMessage(c, userID, msg)
		case "ping":
			handlePingMessage(c, msg)
		}
	}
}

// BroadcastServerLog broadcasts server log messages to subscribed clients
func BroadcastServerLog(serverID uuid.UUID, logLine string) {
	message := WebSocketMessage{
		Type:     "console_log",
		ServerID: serverID.String(),
		Data: ConsoleMessage{
			Line:      logLine,
			Timestamp: getCurrentTimestamp(),
			Type:      "info",
		},
		Timestamp: getCurrentTimestamp(),
	}

	broadcastToServerSubscribers(serverID.String(), message)
}

// BroadcastServerStats broadcasts server statistics to subscribed clients
func BroadcastServerStats(serverID uuid.UUID, stats *ServerStats) {
	message := WebSocketMessage{
		Type:     "server_stats",
		ServerID: serverID.String(),
		Data: StatsMessage{
			ServerID:    serverID.String(),
			CPUUsage:    stats.CPUUsage,
			MemoryUsage: stats.MemoryUsage,
			DiskUsage:   stats.DiskUsage,
			PlayerCount: stats.PlayerCount,
			TPS:         stats.TPS,
			Status:      "running",
		},
		Timestamp: getCurrentTimestamp(),
	}

	broadcastToServerSubscribers(serverID.String(), message)
}

// BroadcastServerStatus broadcasts server status changes
func BroadcastServerStatus(serverID uuid.UUID, status models.ServerStatus, message string) {
	msg := WebSocketMessage{
		Type:     "server_status",
		ServerID: serverID.String(),
		Data: StatusMessage{
			ServerID: serverID.String(),
			Status:   string(status),
			Message:  message,
		},
		Timestamp: getCurrentTimestamp(),
	}

	broadcastToServerSubscribers(serverID.String(), msg)
}

// BroadcastToAll broadcasts a message to all connected clients
func BroadcastToAll(message WebSocketMessage) {
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	for _, conn := range wsManager.connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error broadcasting message: %v", err)
		}
	}
}

// Helper functions

func handleServerSubscription(c *websocket.Conn, userID uuid.UUID, msg WebSocketMessage) {
	// Verify user has access to the server
	serverIDStr, ok := msg.Data.(map[string]interface{})["server_id"].(string)
	if !ok {
		sendErrorMessage(c, "Invalid server ID")
		return
	}

	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		sendErrorMessage(c, "Invalid server ID format")
		return
	}

	// Check if user has access to this server
	if !userHasServerAccess(userID, serverID) {
		sendErrorMessage(c, "Access denied to server")
		return
	}

	// Store subscription (you might want to track this per connection)
	response := WebSocketMessage{
		Type:     "subscribed",
		ServerID: serverIDStr,
		Data: map[string]string{
			"message": "Subscribed to server updates",
		},
		Timestamp: getCurrentTimestamp(),
	}

	c.WriteJSON(response)
}

func handleServerUnsubscription(c *websocket.Conn, userID uuid.UUID, msg WebSocketMessage) {
	serverIDStr, ok := msg.Data.(map[string]interface{})["server_id"].(string)
	if !ok {
		sendErrorMessage(c, "Invalid server ID")
		return
	}

	response := WebSocketMessage{
		Type:     "unsubscribed",
		ServerID: serverIDStr,
		Data: map[string]string{
			"message": "Unsubscribed from server updates",
		},
		Timestamp: getCurrentTimestamp(),
	}

	c.WriteJSON(response)
}

func handleCommandMessage(c *websocket.Conn, userID uuid.UUID, msg WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		sendErrorMessage(c, "Invalid command data")
		return
	}

	serverIDStr, ok := data["server_id"].(string)
	if !ok {
		sendErrorMessage(c, "Invalid server ID")
		return
	}

	command, ok := data["command"].(string)
	if !ok {
		sendErrorMessage(c, "Invalid command")
		return
	}

	serverID, err := uuid.Parse(serverIDStr)
	if err != nil {
		sendErrorMessage(c, "Invalid server ID format")
		return
	}

	// Check if user has access to this server
	if !userHasServerAccess(userID, serverID) {
		sendErrorMessage(c, "Access denied to server")
		return
	}

	// Get server and send command
	var server models.Server
	if err := database.DB.First(&server, serverID).Error; err != nil {
		sendErrorMessage(c, "Server not found")
		return
	}

	if err := SendServerCommand(&server, command); err != nil {
		sendErrorMessage(c, "Failed to send command: "+err.Error())
		return
	}

	// Send confirmation
	response := WebSocketMessage{
		Type:     "command_sent",
		ServerID: serverIDStr,
		Data: map[string]string{
			"command": command,
			"message": "Command sent successfully",
		},
		Timestamp: getCurrentTimestamp(),
	}

	c.WriteJSON(response)
}

func handlePingMessage(c *websocket.Conn, msg WebSocketMessage) {
	response := WebSocketMessage{
		Type: "pong",
		Data: map[string]string{
			"message": "pong",
		},
		Timestamp: getCurrentTimestamp(),
	}

	c.WriteJSON(response)
}

func sendErrorMessage(c *websocket.Conn, message string) {
	errorMsg := WebSocketMessage{
		Type: "error",
		Data: map[string]string{
			"message": message,
		},
		Timestamp: getCurrentTimestamp(),
	}

	c.WriteJSON(errorMsg)
}

func broadcastToServerSubscribers(serverID string, message WebSocketMessage) {
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	// In a real implementation, you'd track which connections are subscribed to which servers
	// For now, broadcast to all connections (they can filter on the client side)
	for _, conn := range wsManager.connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error broadcasting server message: %v", err)
		}
	}
}

func userHasServerAccess(userID, serverID uuid.UUID) bool {
	// Check if user has access to the server
	var user models.User
	if err := database.DB.Preload("Servers").First(&user, userID).Error; err != nil {
		return false
	}

	// Admin has access to all servers
	if user.Role == models.RoleAdmin {
		return true
	}

	// Check if user is associated with the server
	for _, server := range user.Servers {
		if server.ID == serverID {
			return true
		}
	}

	return false
}

func getCurrentTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// StartMetricsCollector starts the metrics collection goroutine
func StartMetricsCollector() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			collectAndBroadcastMetrics()
		}
	}()
}

func collectAndBroadcastMetrics() {
	var servers []models.Server
	database.DB.Where("status = ?", models.ServerStatusRunning).Find(&servers)

	for _, server := range servers {
		stats, err := GetServerStats(&server)
		if err != nil {
			log.Printf("Error collecting stats for server %s: %v", server.Name, err)
			continue
		}

		BroadcastServerStats(server.ID, stats)
	}
}