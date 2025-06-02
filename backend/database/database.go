package database

import (
	"fmt"
	"log"
	"time"

	"playpulse-panel/config"
	"playpulse-panel/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Initialize sets up the database connection
func Initialize(cfg *config.Config) error {
	var err error
	
	// Configure GORM logger
	var gormLogger logger.Interface
	if cfg.Server.Debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Connect to PostgreSQL
	DB, err = gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL database")
	return nil
}

// Migrate runs database migrations
func Migrate() error {
	log.Println("Running database migrations...")

	// Enable UUID extension
	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	// Auto-migrate all models
	err := DB.AutoMigrate(
		&models.User{},
		&models.UserSession{},
		&models.Server{},
		&models.Plugin{},
		&models.Schedule{},
		&models.Backup{},
		&models.ServerMetric{},
		&models.ServerFile{},
		&models.AuditLog{},
		&models.SystemSetting{},
		&models.Notification{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// Seed creates initial data
func Seed() error {
	log.Println("Seeding database with initial data...")

	// Check if admin user already exists
	var adminUser models.User
	if err := DB.Where("username = ?", "admin").First(&adminUser).Error; err == nil {
		log.Println("Admin user already exists, skipping seeding")
		return nil
	}

	// Create default admin user
	defaultAdmin := models.User{
		Username:      "admin",
		Email:         "admin@playpulse.dev",
		Password:      "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LrCLAKUP1gfgUmqfa", // "admin123"
		FirstName:     "System",
		LastName:      "Administrator",
		Role:          models.RoleAdmin,
		IsActive:      true,
		EmailVerified: true,
	}

	if err := DB.Create(&defaultAdmin).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Create default system settings
	defaultSettings := []models.SystemSetting{
		{
			Key:      "panel_name",
			Value:    "Playpulse Panel",
			Type:     "string",
			Category: "general",
		},
		{
			Key:      "panel_description",
			Value:    "Modern Game Server Control Panel",
			Type:     "string",
			Category: "general",
		},
		{
			Key:      "allow_registration",
			Value:    "false",
			Type:     "boolean",
			Category: "security",
		},
		{
			Key:      "default_server_memory",
			Value:    "2048",
			Type:     "number",
			Category: "servers",
		},
		{
			Key:      "max_backup_count",
			Value:    "10",
			Type:     "number",
			Category: "backups",
		},
		{
			Key:      "enable_discord_notifications",
			Value:    "false",
			Type:     "boolean",
			Category: "notifications",
		},
		{
			Key:      "enable_email_notifications",
			Value:    "false",
			Type:     "boolean",
			Category: "notifications",
		},
	}

	for _, setting := range defaultSettings {
		var existingSetting models.SystemSetting
		if err := DB.Where("key = ?", setting.Key).First(&existingSetting).Error; err != nil {
			if err := DB.Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to create setting %s: %w", setting.Key, err)
			}
		}
	}

	log.Println("Database seeding completed successfully")
	log.Println("Default admin credentials: admin@playpulse.dev / admin123")
	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Health checks database connectivity
func Health() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}