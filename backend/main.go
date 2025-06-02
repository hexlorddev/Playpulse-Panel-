package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"playpulse-panel/config"
	"playpulse-panel/database"
	"playpulse-panel/handlers/auth"
	"playpulse-panel/handlers/servers"
	"playpulse-panel/middleware"
	"playpulse-panel/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Seed database
	if err := database.Seed(); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	// Initialize services
	services.InitializeBackupService(cfg)
	services.StartMetricsCollector()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    100 * 1024 * 1024, // 100MB
	})

	// Setup middleware
	middleware.SetupMiddleware(app, cfg)

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		if err := database.Health(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":   "error",
				"database": "disconnected",
				"error":    err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status":   "ok",
			"database": "connected",
			"version":  "1.0.0",
		})
	})

	// API routes
	api := app.Group(cfg.Server.APIPrefix)

	// Public routes (no authentication required)
	public := api.Group("/public")
	public.Get("/info", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"name":        "Playpulse Panel",
			"version":     "1.0.0",
			"description": "Modern Game Server Control Panel",
			"author":      "hhexlorddev",
		})
	})

	// Auth routes
	authRoutes := api.Group("/auth")
	authRoutes.Post("/login", auth.Login)
	authRoutes.Post("/register", auth.Register)
	authRoutes.Post("/refresh", auth.RefreshToken)

	// Protected routes
	protected := api.Group("/", middleware.AuthRequired())
	
	// Auth protected routes
	authProtected := protected.Group("/auth")
	authProtected.Post("/logout", auth.Logout)
	authProtected.Get("/me", auth.Me)
	authProtected.Put("/profile", middleware.AuditLog("profile_update"), auth.UpdateProfile)
	authProtected.Put("/password", middleware.AuditLog("password_change"), auth.ChangePassword)

	// Server routes
	serverRoutes := protected.Group("/servers")
	serverRoutes.Get("/", servers.GetServers)
	serverRoutes.Post("/", middleware.AuditLog("server_create"), servers.CreateServer)

	// Server-specific routes (require server access)
	serverSpecific := serverRoutes.Group("/:serverId", middleware.ServerAccessRequired())
	serverSpecific.Get("/", servers.GetServer)
	serverSpecific.Put("/", middleware.AuditLog("server_update"), servers.UpdateServer)
	serverSpecific.Delete("/", middleware.AuditLog("server_delete"), servers.DeleteServer)
	
	// Server control
	serverSpecific.Post("/start", middleware.AuditLog("server_start"), servers.StartServer)
	serverSpecific.Post("/stop", middleware.AuditLog("server_stop"), servers.StopServer)
	serverSpecific.Post("/restart", middleware.AuditLog("server_restart"), servers.RestartServer)
	serverSpecific.Post("/command", middleware.AuditLog("server_command"), servers.SendCommand)
	
	// Server monitoring
	serverSpecific.Get("/logs", servers.GetServerLogs)
	serverSpecific.Get("/stats", servers.GetServerStats)

	// File management routes (to be implemented)
	fileRoutes := serverSpecific.Group("/files")
	fileRoutes.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "File management routes to be implemented"})
	})

	// Plugin management routes (to be implemented)
	pluginRoutes := serverSpecific.Group("/plugins")
	pluginRoutes.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Plugin management routes to be implemented"})
	})

	// Backup routes (to be implemented)
	backupRoutes := serverSpecific.Group("/backups")
	backupRoutes.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Backup routes to be implemented"})
	})

	// Schedule routes (to be implemented)
	scheduleRoutes := serverSpecific.Group("/schedules")
	scheduleRoutes.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Schedule routes to be implemented"})
	})

	// Admin routes
	adminRoutes := protected.Group("/admin", middleware.AdminRequired())
	adminRoutes.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Admin user management to be implemented"})
	})
	adminRoutes.Get("/settings", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Admin settings to be implemented"})
	})
	adminRoutes.Get("/audit", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Audit logs to be implemented"})
	})

	// WebSocket endpoint
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// Get user from JWT token (implement proper auth for WebSocket)
		services.HandleWebSocket(c, uuid.Nil) // For now, pass nil UUID
	}))

	// Serve static files (for frontend in production)
	if cfg.IsProduction() {
		app.Static("/", "./public")
		app.Static("*", "./public/index.html")
	}

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n🔄 Gracefully shutting down...")
		
		// Close database connection
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		
		// Shutdown server
		if err := app.Shutdown(); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		
		fmt.Println("✅ Playpulse Panel shut down successfully")
		os.Exit(0)
	}()

	// Start server
	fmt.Printf(`
🎮 Playpulse Panel Starting...

┌─────────────────────────────────────────────────┐
│                                                 │
│   🚀 Playpulse Panel v1.0.0                    │
│   🎯 Modern Game Server Control Panel          │
│   👨‍💻 Created by hhexlorddev                    │
│                                                 │
│   🌐 Server: http://localhost:%s              │
│   📚 API: http://localhost:%s%s               │
│   🔌 WebSocket: ws://localhost:%s/ws          │
│                                                 │
│   📖 Default Admin Credentials:                │
│   📧 Email: admin@playpulse.dev                │
│   🔑 Password: admin123                        │
│                                                 │
└─────────────────────────────────────────────────┘

`, cfg.Server.Port, cfg.Server.Port, cfg.Server.APIPrefix, cfg.Server.Port)

	log.Fatal(app.Listen(":" + cfg.Server.Port))
}