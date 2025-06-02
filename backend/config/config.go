package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	Database DatabaseConfig

	// JWT
	JWT JWTConfig

	// Server
	Server ServerConfig

	// External APIs
	ExternalAPIs ExternalAPIConfig

	// File Management
	Files FileConfig

	// Security
	Security SecurityConfig

	// Monitoring
	Monitoring MonitoringConfig

	// Game Servers
	GameServers GameServerConfig

	// Notifications
	Notifications NotificationConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret           string
	RefreshSecret    string
	ExpireHours      int
	RefreshExpireDays int
}

type ServerConfig struct {
	Port        string
	Environment string
	APIPrefix   string
	FrontendURL string
	CORSOrigins []string
	Debug       bool
	LogLevel    string
}

type ExternalAPIConfig struct {
	CurseForgeAPIKey string
	ModrinthAPIKey   string
}

type FileConfig struct {
	MaxFileSize string
	UploadPath  string
	BackupPath  string
}

type SecurityConfig struct {
	Enable2FA             bool
	MaxLoginAttempts      int
	LoginCooldownMinutes  int
	CleanupLogsDays       int
}

type MonitoringConfig struct {
	EnableMetrics      bool
	MetricsInterval    time.Duration
}

type GameServerConfig struct {
	DefaultServerPath string
	DefaultJavaPath   string
	DefaultJavaArgs   string
}

type NotificationConfig struct {
	Discord DiscordConfig
	Email   EmailConfig
}

type DiscordConfig struct {
	WebhookURL string
}

type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "playpulse"),
			Password: getEnv("DB_PASSWORD", "playpulse_password"),
			Name:     getEnv("DB_NAME", "playpulse_panel"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "default-secret-change-this"),
			RefreshSecret:     getEnv("JWT_REFRESH_SECRET", "default-refresh-secret-change-this"),
			ExpireHours:       getEnvInt("JWT_EXPIRE_HOURS", 24),
			RefreshExpireDays: getEnvInt("JWT_REFRESH_EXPIRE_DAYS", 30),
		},
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			Environment: getEnv("ENVIRONMENT", "development"),
			APIPrefix:   getEnv("API_PREFIX", "/api/v1"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
			CORSOrigins: strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"), ","),
			Debug:       getEnvBool("DEBUG", true),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
		ExternalAPIs: ExternalAPIConfig{
			CurseForgeAPIKey: getEnv("CURSEFORGE_API_KEY", ""),
			ModrinthAPIKey:   getEnv("MODRINTH_API_KEY", ""),
		},
		Files: FileConfig{
			MaxFileSize: getEnv("MAX_FILE_SIZE", "100MB"),
			UploadPath:  getEnv("UPLOAD_PATH", "./uploads"),
			BackupPath:  getEnv("BACKUP_PATH", "./backups"),
		},
		Security: SecurityConfig{
			Enable2FA:            getEnvBool("ENABLE_2FA", true),
			MaxLoginAttempts:     getEnvInt("MAX_LOGIN_ATTEMPTS", 5),
			LoginCooldownMinutes: getEnvInt("LOGIN_COOLDOWN_MINUTES", 15),
			CleanupLogsDays:      getEnvInt("CLEANUP_LOGS_DAYS", 30),
		},
		Monitoring: MonitoringConfig{
			EnableMetrics:   getEnvBool("ENABLE_METRICS", true),
			MetricsInterval: time.Duration(getEnvInt("METRICS_INTERVAL_SECONDS", 30)) * time.Second,
		},
		GameServers: GameServerConfig{
			DefaultServerPath: getEnv("DEFAULT_SERVER_PATH", "/opt/minecraft-servers"),
			DefaultJavaPath:   getEnv("DEFAULT_JAVA_PATH", "/usr/bin/java"),
			DefaultJavaArgs:   getEnv("DEFAULT_JAVA_ARGS", "-Xms1G -Xmx2G -XX:+UseG1GC"),
		},
		Notifications: NotificationConfig{
			Discord: DiscordConfig{
				WebhookURL: getEnv("DISCORD_WEBHOOK_URL", ""),
			},
			Email: EmailConfig{
				SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
				SMTPPort:     getEnv("SMTP_PORT", "587"),
				SMTPUser:     getEnv("SMTP_USER", ""),
				SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			},
		},
	}

	return config, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	return "host=" + c.Database.Host +
		" port=" + c.Database.Port +
		" user=" + c.Database.User +
		" password=" + c.Database.Password +
		" dbname=" + c.Database.Name +
		" sslmode=" + c.Database.SSLMode
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}