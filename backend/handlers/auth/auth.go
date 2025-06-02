package auth

import (
	"time"

	"playpulse-panel/config"
	"playpulse-panel/database"
	"playpulse-panel/models"
	"playpulse-panel/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"max=50"`
	LastName  string `json:"last_name" validate:"max=50"`
}

type LoginResponse struct {
	User         models.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Login authenticates a user and returns JWT tokens
func Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	// Find user by email
	var user models.User
	err := database.DB.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid credentials",
			"message": "Email or password is incorrect",
		})
	}

	// Check if user is active
	if !user.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Account disabled",
			"message": "Your account has been disabled. Please contact an administrator.",
		})
	}

	// Check if user is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Account locked",
			"message": "Your account is temporarily locked due to too many failed login attempts.",
		})
	}

	// Verify password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		// Increment login attempts
		user.LoginAttempts++
		if user.LoginAttempts >= 5 {
			lockUntil := time.Now().Add(time.Minute * 15)
			user.LockedUntil = &lockUntil
		}
		database.DB.Save(&user)

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid credentials",
			"message": "Email or password is incorrect",
		})
	}

	// Reset login attempts
	user.LoginAttempts = 0
	user.LockedUntil = nil
	now := time.Now()
	user.LastLogin = &now
	database.DB.Save(&user)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Configuration error",
			"message": "Unable to load server configuration",
		})
	}

	// Generate tokens
	accessToken, err := utils.GenerateJWT(user.ID, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Token generation failed",
			"message": "Unable to generate access token",
		})
	}

	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Token generation failed",
			"message": "Unable to generate refresh token",
		})
	}

	// Save user session
	session := models.UserSession{
		UserID:       user.ID,
		TokenHash:    accessToken, // In production, hash this
		RefreshToken: refreshToken,
		IPAddress:    c.IP(),
		UserAgent:    c.Get("User-Agent"),
		ExpiresAt:    time.Now().Add(time.Hour * time.Duration(cfg.JWT.ExpireHours)),
	}
	database.DB.Create(&session)

	// Remove sensitive information
	user.Password = ""
	user.TwoFactorSecret = ""

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		Action:    "user_login",
		Details:   "User logged in successfully",
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(LoginResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    session.ExpiresAt,
	})
}

// Register creates a new user account
func Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	// Check if registration is allowed
	var registrationSetting models.SystemSetting
	err := database.DB.Where("key = ?", "allow_registration").First(&registrationSetting).Error
	if err == nil && registrationSetting.Value == "false" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "Registration disabled",
			"message": "Registration is currently disabled. Please contact an administrator.",
		})
	}

	// Validate username and email
	if !utils.ValidateUsername(req.Username) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid username",
			"message": "Username must be 3-50 characters and contain only letters, numbers, underscores, and hyphens",
		})
	}

	if !utils.ValidateEmail(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid email",
			"message": "Please provide a valid email address",
		})
	}

	// Check if user already exists
	var existingUser models.User
	err = database.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":   "User already exists",
			"message": "Username or email is already registered",
		})
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Password hashing failed",
			"message": "Unable to process password",
		})
	}

	// Create user
	user := models.User{
		Username:      req.Username,
		Email:         req.Email,
		Password:      hashedPassword,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Role:          models.RoleUser,
		IsActive:      true,
		EmailVerified: false, // In production, implement email verification
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "User creation failed",
			"message": "Unable to create user account",
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		Action:    "user_register",
		Details:   "User registered successfully",
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	// Remove sensitive information
	user.Password = ""

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"user":    user,
	})
}

// RefreshToken generates a new access token using refresh token
func RefreshToken(c *fiber.Ctx) error {
	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	// Find session by refresh token
	var session models.UserSession
	err := database.DB.Preload("User").Where("refresh_token = ? AND expires_at > ?", req.RefreshToken, time.Now()).First(&session).Error
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Invalid refresh token",
			"message": "Refresh token is invalid or expired",
		})
	}

	// Check if user is still active
	if !session.User.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Account disabled",
			"message": "Your account has been disabled",
		})
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Configuration error",
			"message": "Unable to load server configuration",
		})
	}

	// Generate new access token
	accessToken, err := utils.GenerateJWT(session.User.ID, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Token generation failed",
			"message": "Unable to generate access token",
		})
	}

	// Update session
	session.TokenHash = accessToken
	session.ExpiresAt = time.Now().Add(time.Hour * time.Duration(cfg.JWT.ExpireHours))
	database.DB.Save(&session)

	// Remove sensitive information
	session.User.Password = ""
	session.User.TwoFactorSecret = ""

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"expires_at":   session.ExpiresAt,
		"user":         session.User,
	})
}

// Logout invalidates the current session
func Logout(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	// Get Authorization header to find the session
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		token := authHeader[7:] // Remove "Bearer " prefix
		
		// Find and delete the session
		database.DB.Where("user_id = ? AND token_hash = ?", user.ID, token).Delete(&models.UserSession{})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		Action:    "user_logout",
		Details:   "User logged out",
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Me returns current user information
func Me(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	// Load user with relationships
	database.DB.Preload("Servers").First(&user, user.ID)

	// Remove sensitive information
	user.Password = ""
	user.TwoFactorSecret = ""

	return c.JSON(user)
}

// UpdateProfile updates user profile information
func UpdateProfile(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var req struct {
		FirstName string `json:"first_name" validate:"max=50"`
		LastName  string `json:"last_name" validate:"max=50"`
		Email     string `json:"email" validate:"email"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	// Check if email is already taken by another user
	if req.Email != "" && req.Email != user.Email {
		var existingUser models.User
		err := database.DB.Where("email = ? AND id != ?", req.Email, user.ID).First(&existingUser).Error
		if err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error":   "Email already taken",
				"message": "This email is already registered to another account",
			})
		}
	}

	// Update user fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Email != "" {
		user.Email = req.Email
		user.EmailVerified = false // Require re-verification
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Update failed",
			"message": "Unable to update profile",
		})
	}

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		Action:    "profile_update",
		Details:   "User updated profile information",
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	// Remove sensitive information
	user.Password = ""
	user.TwoFactorSecret = ""

	return c.JSON(user)
}

// ChangePassword changes user password
func ChangePassword(c *fiber.Ctx) error {
	user := c.Locals("user").(models.User)

	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request body",
			"message": err.Error(),
		})
	}

	// Get full user record with password
	var fullUser models.User
	if err := database.DB.First(&fullUser, user.ID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "User not found",
			"message": "User account not found",
		})
	}

	// Verify current password
	if !utils.CheckPasswordHash(req.CurrentPassword, fullUser.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid password",
			"message": "Current password is incorrect",
		})
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Password hashing failed",
			"message": "Unable to process new password",
		})
	}

	// Update password
	fullUser.Password = hashedPassword
	if err := database.DB.Save(&fullUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Password update failed",
			"message": "Unable to update password",
		})
	}

	// Invalidate all sessions except current one
	authHeader := c.Get("Authorization")
	currentToken := ""
	if authHeader != "" {
		currentToken = authHeader[7:] // Remove "Bearer " prefix
	}

	database.DB.Where("user_id = ? AND token_hash != ?", user.ID, currentToken).Delete(&models.UserSession{})

	// Create audit log
	auditLog := models.AuditLog{
		UserID:    user.ID,
		Action:    "password_change",
		Details:   "User changed password",
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	database.DB.Create(&auditLog)

	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}