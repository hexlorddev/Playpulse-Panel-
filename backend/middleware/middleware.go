package middleware

import (
	"fmt"
	"strings"
	"time"

	"playpulse-panel/config"
	"playpulse-panel/database"
	"playpulse-panel/models"
	"playpulse-panel/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SetupMiddleware configures all middleware
func SetupMiddleware(app *fiber.App, cfg *config.Config) {
	// Recover middleware
	app.Use(recover.New())

	// Logger middleware
	if cfg.Server.Debug {
		app.Use(logger.New(logger.Config{
			Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
			TimeFormat: "2006-01-02 15:04:05",
			TimeZone:   "UTC",
		}))
	}

	// CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.Server.CORSOrigins, ","),
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With",
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rate limiting middleware
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
		},
	}))

	// Request ID middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestId", uuid.New().String())
		return c.Next()
	})
}

// AuthRequired middleware for protected routes
func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Missing authorization header",
				"message": "Authorization header is required",
			})
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid authorization header",
				"message": "Authorization header must start with 'Bearer '",
			})
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Missing token",
				"message": "JWT token is required",
			})
		}

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			cfg, _ := config.Load()
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid token",
				"message": "JWT token is invalid or expired",
			})
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid token claims",
				"message": "Unable to parse token claims",
			})
		}

		// Get user ID from claims
		userIdStr, ok := claims["user_id"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid user ID in token",
				"message": "User ID not found in token",
			})
		}

		userId, err := uuid.Parse(userIdStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid user ID format",
				"message": "User ID must be a valid UUID",
			})
		}

		// Get user from database
		var user models.User
		if err := database.DB.Where("id = ? AND is_active = ?", userId, true).First(&user).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "User not found",
				"message": "User associated with token not found or inactive",
			})
		}

		// Store user in context
		c.Locals("user", user)
		c.Locals("userId", userId)

		return c.Next()
	}
}

// RoleRequired middleware for role-based access control
func RoleRequired(roles ...models.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(models.User)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "User not authenticated",
				"message": "Please login to access this resource",
			})
		}

		// Check if user has required role
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Insufficient permissions",
				"message": "You don't have permission to access this resource",
			})
		}

		return c.Next()
	}
}

// AdminRequired middleware for admin-only routes
func AdminRequired() fiber.Handler {
	return RoleRequired(models.RoleAdmin)
}

// ServerAccessRequired middleware for server-specific access
func ServerAccessRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(models.User)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "User not authenticated",
				"message": "Please login to access this resource",
			})
		}

		// Get server ID from URL params
		serverIdStr := c.Params("serverId")
		if serverIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Missing server ID",
				"message": "Server ID is required",
			})
		}

		serverId, err := uuid.Parse(serverIdStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid server ID",
				"message": "Server ID must be a valid UUID",
			})
		}

		// Check if user is admin (admins have access to all servers)
		if user.Role == models.RoleAdmin {
			c.Locals("serverId", serverId)
			return c.Next()
		}

		// Check if user has access to this server
		var server models.Server
		err = database.DB.Preload("Users").Where("id = ?", serverId).First(&server).Error
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "Server not found",
				"message": "The requested server does not exist",
			})
		}

		// Check if user is associated with this server
		hasAccess := false
		for _, serverUser := range server.Users {
			if serverUser.ID == user.ID {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "Access denied",
				"message": "You don't have access to this server",
			})
		}

		c.Locals("serverId", serverId)
		c.Locals("server", server)
		return c.Next()
	}
}

// APIKeyAuth middleware for API key authentication
func APIKeyAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Missing API key",
				"message": "X-API-Key header is required",
			})
		}

		// Find user by API key
		var user models.User
		if err := database.DB.Where("api_key = ? AND is_active = ?", apiKey, true).First(&user).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid API key",
				"message": "The provided API key is invalid",
			})
		}

		// Store user in context
		c.Locals("user", user)
		c.Locals("userId", user.ID)

		return c.Next()
	}
}

// AuditLog middleware for logging user actions
func AuditLog(action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Execute the next handler first
		err := c.Next()

		// Log the action if request was successful
		if c.Response().StatusCode() < 400 {
			go func() {
				user, ok := c.Locals("user").(models.User)
				if !ok {
					return
				}

				serverId, _ := c.Locals("serverId").(uuid.UUID)
				var serverIdPtr *uuid.UUID
				if serverId != uuid.Nil {
					serverIdPtr = &serverId
				}

				auditLog := models.AuditLog{
					UserID:    user.ID,
					ServerID:  serverIdPtr,
					Action:    action,
					Details:   utils.GetRequestDetails(c),
					IPAddress: c.IP(),
					UserAgent: c.Get("User-Agent"),
				}

				database.DB.Create(&auditLog)
			}()
		}

		return err
	}
}

// ErrorHandler handles application errors
func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log error
	fmt.Printf("Error: %v\n", err)

	return c.Status(code).JSON(fiber.Map{
		"error":     message,
		"timestamp": time.Now().UTC(),
		"path":      c.Path(),
		"method":    c.Method(),
	})
}