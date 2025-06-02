package servers

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"playpulse-panel/config"
	"playpulse-panel/database"
	"playpulse-panel/models"
	"playpulse-panel/services"
	"playpulse-panel/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateServerRequest struct {
	Name         string             `json:"name" validate:"required,min=1,max=100"`
	Description  string             `json:"description"`
	Type         models.ServerType  `json:"type" validate:"required"`
	Version      string             `json:"version"`
	Port         int                `json:"port" validate:"required,min=1024,max=65535"`
	MemoryLimit  int64              `json:"memory_limit" validate:"required,min=512"`
	DiskLimit    int64              `json:"disk_limit" validate:"required,min=1024"`
	CPULimit     float64            `json:"cpu_limit" validate:"min=0,max=100"`
	JavaPath     string             `json:"java_path"`
	JavaArgs     string             `json:"java_args"`
	AutoRestart  bool               `json:"auto_restart"`
	AutoStart    bool               `json:"auto_start"`
}

type UpdateServerRequest struct {
	Name         string             `json:"name" validate:"min=1,max=100"`
	Description  string             `json:"description"`
	Version      string             `json:"version"`
	MemoryLimit  int64              `json:"memory_limit" validate:"min=512"`
	DiskLimit    int64              `json:"disk_limit" validate:"min=1024"`
	CPULimit     float64            `json:"cpu_limit" validate:"min=0,max=100"`
	JavaPath     string             `json:"java_path"`
	JavaArgs     string             `json:"java_args"`
	StartCommand string             `json:"start_command"`
	StopCommand  string             `json:"stop_command"`
	AutoRestart  *bool              `json:"auto_restart"`
	AutoStart    *bool              `json:"auto_start"`
}

// GetServers returns all servers for the current user
func GetServers(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	
	var servers []models.Server
	query := database.DB.Preload("Users").Preload("Metrics", func(db *gorm.DB) *gorm.DB {
		return db.Order("timestamp DESC").Limit(1)
	})

	if user.Role == models.RoleAdmin {
		// Admin can see all servers
		query = query.Find(&servers)
	} else {
		// Regular users can only see servers they have access to
		query = query.Joins("JOIN user_servers ON user_servers.server_id = servers.id").
			Where("user_servers.user_id = ?", user.ID).Find(&servers)
	}

	if query.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Database error",
			"message": "Unable to retrieve servers",
		})
	}

	// Update server status and metrics
	for i := range servers {
		services.UpdateServerStatus(&servers[i])
	}

	return c.JSON(servers)
}

// GetServer returns a specific server
func GetServer(c *fiber.Ctx) error {
	serverId := c.Locals("serverId").(uuid.UUID)
	
	var server models.Server
	err := database.DB.Preload("Users").
		Preload("Plugins").
		Preload("Schedules").
		Preload("Backups", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(10)
		}).
		Preload("Metrics", func(db *gorm.DB) *gorm.DB {
			return db.Order("timestamp DESC").Limit(100)
		}).
		First(&server, serverId).Error

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	// Update server status
	services.UpdateServerStatus(&server)

	return c.JSON(server)
}

// CreateServer creates a new server
func CreateServer(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var req CreateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	// Check if port is already in use
	var existingServer models.Server
	err := database.DB.Where("port = ?", req.Port).First(&existingServer).Error
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "Port already in use",
			"message": fmt.Sprintf("Port %d is already used by another server", req.Port),
		})
	}

	// Generate server path
	cfg, _ := config.Load()
	serverPath := filepath.Join(cfg.GameServers.DefaultServerPath, utils.SanitizeFilename(req.Name))
	
	// Validate server path
	if err := utils.ValidateServerPath(serverPath); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid server path",
			"message": err.Error(),
		})
	}

	// Create server directory
	if err := utils.CreateDirectory(serverPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Directory creation failed",
			"message": "Unable to create server directory",
		})
	}

	// Set default values
	javaPath := req.JavaPath
	if javaPath == "" {
		javaPath = cfg.GameServers.DefaultJavaPath
	}

	javaArgs := req.JavaArgs
	if javaArgs == "" {
		javaArgs = cfg.GameServers.DefaultJavaArgs
	}

	// Create server record
	server := models.Server{
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		Version:       req.Version,
		Status:        models.ServerStatusStopped,
		Port:          req.Port,
		MemoryLimit:   req.MemoryLimit,
		DiskLimit:     req.DiskLimit,
		CPULimit:      req.CPULimit,
		Path:          serverPath,
		JavaPath:      javaPath,
		JavaArgs:      javaArgs,
		AutoRestart:   req.AutoRestart,
		AutoStart:     req.AutoStart,
		BackupEnabled: true,
	}

	if err := database.DB.Create(&server).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Server creation failed",
			"message": "Unable to create server record",
		})
	}

	// Associate user with server (if not admin creating for others)
	if user.Role != models.RoleAdmin {
		database.DB.Model(&server).Association("Users").Append(&user)
	}

	// Download server jar based on type
	go services.SetupServerJar(&server)

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		ServerID:  &server.ID,
		Action:    "server_create",
		Details:   fmt.Sprintf("Created server: %s", server.Name),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.Status(fiber.StatusCreated).JSON(server)
}

// UpdateServer updates server configuration
func UpdateServer(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	serverId := c.Locals("serverId").(uuid.UUID)

	var req UpdateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	// Check if server is running (some settings can't be changed while running)
	if server.Status == models.ServerStatusRunning {
		restrictedFields := []string{"memory_limit", "java_path", "java_args"}
		// In a real implementation, you'd check which fields are being changed
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Server is running",
			"message": "Stop the server before changing these settings",
		})
	}

	// Update fields
	if req.Name != "" {
		server.Name = req.Name
	}
	if req.Description != "" {
		server.Description = req.Description
	}
	if req.Version != "" {
		server.Version = req.Version
	}
	if req.MemoryLimit > 0 {
		server.MemoryLimit = req.MemoryLimit
	}
	if req.DiskLimit > 0 {
		server.DiskLimit = req.DiskLimit
	}
	if req.CPULimit >= 0 {
		server.CPULimit = req.CPULimit
	}
	if req.JavaPath != "" {
		server.JavaPath = req.JavaPath
	}
	if req.JavaArgs != "" {
		server.JavaArgs = req.JavaArgs
	}
	if req.StartCommand != "" {
		server.StartCommand = req.StartCommand
	}
	if req.StopCommand != "" {
		server.StopCommand = req.StopCommand
	}
	if req.AutoRestart != nil {
		server.AutoRestart = *req.AutoRestart
	}
	if req.AutoStart != nil {
		server.AutoStart = *req.AutoStart
	}

	if err := database.DB.Save(&server).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Update failed",
			"message": "Unable to update server configuration",
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		ServerID:  &server.ID,
		Action:    "server_update",
		Details:   fmt.Sprintf("Updated server configuration: %s", server.Name),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(server)
}

// DeleteServer deletes a server
func DeleteServer(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	serverId := c.Locals("serverId").(uuid.UUID)

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	// Stop server if running
	if server.Status == models.ServerStatusRunning {
		services.StopServer(&server)
	}

	// Delete server files (optional - add confirmation parameter)
	deleteFiles := c.Query("delete_files", "false")
	if deleteFiles == "true" {
		go func() {
			// Create backup before deletion
			services.CreateBackup(&server, "pre_deletion_backup")
			// Remove server directory (implement with caution)
		}()
	}

	// Delete server record
	if err := database.DB.Delete(&server).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Deletion failed",
			"message": "Unable to delete server",
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		Action:    "server_delete",
		Details:   fmt.Sprintf("Deleted server: %s", server.Name),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Server deleted successfully",
	})
}

// StartServer starts a server
func StartServer(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	serverId := c.Locals("serverId").(uuid.UUID)

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	if server.Status == models.ServerStatusRunning {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Server already running",
			"message": "The server is already running",
		})
	}

	// Start server
	if err := services.StartServer(&server); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Start failed",
			"message": err.Error(),
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		ServerID:  &server.ID,
		Action:    "server_start",
		Details:   fmt.Sprintf("Started server: %s", server.Name),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Server start command sent",
		"status":  server.Status,
	})
}

// StopServer stops a server
func StopServer(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	serverId := c.Locals("serverId").(uuid.UUID)

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	if server.Status == models.ServerStatusStopped {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Server already stopped",
			"message": "The server is already stopped",
		})
	}

	// Stop server
	if err := services.StopServer(&server); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Stop failed",
			"message": err.Error(),
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		ServerID:  &server.ID,
		Action:    "server_stop",
		Details:   fmt.Sprintf("Stopped server: %s", server.Name),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Server stop command sent",
		"status":  server.Status,
	})
}

// RestartServer restarts a server
func RestartServer(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	serverId := c.Locals("serverId").(uuid.UUID)

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	// Restart server
	if err := services.RestartServer(&server); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Restart failed",
			"message": err.Error(),
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		ServerID:  &server.ID,
		Action:    "server_restart",
		Details:   fmt.Sprintf("Restarted server: %s", server.Name),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Server restart command sent",
		"status":  server.Status,
	})
}

// SendCommand sends a command to the server
func SendCommand(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)
	serverId := c.Locals("serverId").(uuid.UUID)

	var req struct {
		Command string `json:"command" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	if server.Status != models.ServerStatusRunning {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Server not running",
			"message": "The server must be running to send commands",
		})
	}

	// Send command to server
	if err := services.SendServerCommand(&server, req.Command); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Command failed",
			"message": err.Error(),
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		ServerID:  &server.ID,
		Action:    "server_command",
		Details:   fmt.Sprintf("Sent command to server %s: %s", server.Name, req.Command),
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Command sent successfully",
	})
}

// GetServerLogs returns server console logs
func GetServerLogs(c *fiber.Ctx) error {
	serverId := c.Locals("serverId").(uuid.UUID)

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	// Get query parameters
	lines := 100
	if l := c.Query("lines"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			lines = parsed
		}
	}

	// Get logs from service
	logs, err := services.GetServerLogs(&server, lines)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to retrieve logs",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"lines": len(logs),
	})
}

// GetServerStats returns current server statistics
func GetServerStats(c *fiber.Ctx) error {
	serverId := c.Locals("serverId").(uuid.UUID)

	var server models.Server
	if err := database.DB.First(&server, serverId).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Server not found",
			"message": "The requested server does not exist",
		})
	}

	// Get current stats
	stats, err := services.GetServerStats(&server)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to retrieve stats",
			"message": err.Error(),
		})
	}

	return c.JSON(stats)
}