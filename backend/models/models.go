package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username          string         `json:"username" gorm:"uniqueIndex;not null" validate:"required,min=3,max=50"`
	Email             string         `json:"email" gorm:"uniqueIndex;not null" validate:"required,email"`
	Password          string         `json:"-" gorm:"not null"`
	FirstName         string         `json:"first_name" validate:"max=50"`
	LastName          string         `json:"last_name" validate:"max=50"`
	Avatar            string         `json:"avatar"`
	Role              UserRole       `json:"role" gorm:"default:'user'"`
	IsActive          bool           `json:"is_active" gorm:"default:true"`
	EmailVerified     bool           `json:"email_verified" gorm:"default:false"`
	TwoFactorEnabled  bool           `json:"two_factor_enabled" gorm:"default:false"`
	TwoFactorSecret   string         `json:"-"`
	LastLogin         *time.Time     `json:"last_login"`
	LoginAttempts     int            `json:"-" gorm:"default:0"`
	LockedUntil       *time.Time     `json:"-"`
	APIKey            string         `json:"api_key" gorm:"uniqueIndex"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Servers           []Server       `json:"servers,omitempty" gorm:"many2many:user_servers;"`
	AuditLogs         []AuditLog     `json:"audit_logs,omitempty"`
	Sessions          []UserSession  `json:"sessions,omitempty"`
}

type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleModerator UserRole = "moderator"
	RoleUser      UserRole = "user"
	RoleViewer    UserRole = "viewer"
)

// UserSession represents user login sessions
type UserSession struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID      `json:"user_id" gorm:"type:uuid;not null"`
	TokenHash    string         `json:"-" gorm:"not null"`
	RefreshToken string         `json:"-" gorm:"not null"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	ExpiresAt    time.Time      `json:"expires_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	
	User User `json:"user,omitempty"`
}

// Server represents a game server
type Server struct {
	ID              uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name            string          `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Description     string          `json:"description"`
	Type            ServerType      `json:"type" gorm:"not null"`
	Version         string          `json:"version"`
	Status          ServerStatus    `json:"status" gorm:"default:'stopped'"`
	Port            int             `json:"port" gorm:"uniqueIndex" validate:"required,min=1024,max=65535"`
	MemoryLimit     int64           `json:"memory_limit"` // in MB
	DiskLimit       int64           `json:"disk_limit"`   // in MB
	CPULimit        float64         `json:"cpu_limit"`    // percentage
	Path            string          `json:"path" gorm:"not null"`
	JavaPath        string          `json:"java_path"`
	JavaArgs        string          `json:"java_args"`
	ServerJar       string          `json:"server_jar"`
	StartCommand    string          `json:"start_command"`
	StopCommand     string          `json:"stop_command"`
	AutoRestart     bool            `json:"auto_restart" gorm:"default:true"`
	AutoStart       bool            `json:"auto_start" gorm:"default:false"`
	BackupEnabled   bool            `json:"backup_enabled" gorm:"default:true"`
	LastBackup      *time.Time      `json:"last_backup"`
	PID             int             `json:"pid" gorm:"default:0"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       gorm.DeletedAt  `json:"-" gorm:"index"`
	
	// Relationships
	Users           []User          `json:"users,omitempty" gorm:"many2many:user_servers;"`
	Plugins         []Plugin        `json:"plugins,omitempty"`
	Schedules       []Schedule      `json:"schedules,omitempty"`
	Backups         []Backup        `json:"backups,omitempty"`
	Metrics         []ServerMetric  `json:"metrics,omitempty"`
	AuditLogs       []AuditLog      `json:"audit_logs,omitempty"`
	Files           []ServerFile    `json:"files,omitempty"`
}

type ServerType string

const (
	ServerTypeMinecraft  ServerType = "minecraft"
	ServerTypePaper      ServerType = "paper"
	ServerTypeSpigot     ServerType = "spigot"
	ServerTypeFabric     ServerType = "fabric"
	ServerTypeForge      ServerType = "forge"
	ServerTypeVanilla    ServerType = "vanilla"
	ServerTypeBedrock    ServerType = "bedrock"
	ServerTypeProxy      ServerType = "proxy"
	ServerTypeOther      ServerType = "other"
)

type ServerStatus string

const (
	ServerStatusStopped  ServerStatus = "stopped"
	ServerStatusStarting ServerStatus = "starting"
	ServerStatusRunning  ServerStatus = "running"
	ServerStatusStopping ServerStatus = "stopping"
	ServerStatusCrashed  ServerStatus = "crashed"
	ServerStatusUnknown  ServerStatus = "unknown"
)

// Plugin represents installed plugins/mods
type Plugin struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ServerID     uuid.UUID      `json:"server_id" gorm:"type:uuid;not null"`
	Name         string         `json:"name" gorm:"not null"`
	Version      string         `json:"version"`
	Author       string         `json:"author"`
	Description  string         `json:"description"`
	FileName     string         `json:"file_name"`
	FilePath     string         `json:"file_path"`
	FileSize     int64          `json:"file_size"`
	Source       PluginSource   `json:"source"`
	SourceID     string         `json:"source_id"` // CurseForge/Modrinth ID
	IsEnabled    bool           `json:"is_enabled" gorm:"default:true"`
	Dependencies []string       `json:"dependencies" gorm:"type:text[]"`
	InstallDate  time.Time      `json:"install_date"`
	UpdateDate   *time.Time     `json:"update_date"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
	
	Server Server `json:"server,omitempty"`
}

type PluginSource string

const (
	PluginSourceCurseForge PluginSource = "curseforge"
	PluginSourceModrinth   PluginSource = "modrinth"
	PluginSourceManual     PluginSource = "manual"
	PluginSourceGitHub     PluginSource = "github"
)

// Schedule represents scheduled tasks
type Schedule struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ServerID    uuid.UUID      `json:"server_id" gorm:"type:uuid;not null"`
	Name        string         `json:"name" gorm:"not null"`
	Action      ScheduleAction `json:"action" gorm:"not null"`
	Command     string         `json:"command"`
	CronPattern string         `json:"cron_pattern" gorm:"not null"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	LastRun     *time.Time     `json:"last_run"`
	NextRun     *time.Time     `json:"next_run"`
	RunCount    int            `json:"run_count" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
	
	Server Server `json:"server,omitempty"`
}

type ScheduleAction string

const (
	ScheduleActionRestart ScheduleAction = "restart"
	ScheduleActionStop    ScheduleAction = "stop"
	ScheduleActionStart   ScheduleAction = "start"
	ScheduleActionCommand ScheduleAction = "command"
	ScheduleActionBackup  ScheduleAction = "backup"
)

// Backup represents server backups
type Backup struct {
	ID          uuid.UUID    `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ServerID    uuid.UUID    `json:"server_id" gorm:"type:uuid;not null"`
	Name        string       `json:"name" gorm:"not null"`
	Description string       `json:"description"`
	Path        string       `json:"path" gorm:"not null"`
	Size        int64        `json:"size"`
	Type        BackupType   `json:"type"`
	Status      BackupStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	
	Server Server `json:"server,omitempty"`
}

type BackupType string

const (
	BackupTypeManual    BackupType = "manual"
	BackupTypeScheduled BackupType = "scheduled"
	BackupTypeAutomatic BackupType = "automatic"
)

type BackupStatus string

const (
	BackupStatusCreating  BackupStatus = "creating"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
)

// ServerMetric represents server performance metrics
type ServerMetric struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ServerID     uuid.UUID `json:"server_id" gorm:"type:uuid;not null"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage"`  // in MB
	DiskUsage    int64     `json:"disk_usage"`    // in MB
	NetworkIn    int64     `json:"network_in"`    // in bytes
	NetworkOut   int64     `json:"network_out"`   // in bytes
	PlayerCount  int       `json:"player_count"`
	TPS          float64   `json:"tps"`
	MSPT         float64   `json:"mspt"`
	Timestamp    time.Time `json:"timestamp"`
	
	Server Server `json:"server,omitempty"`
}

// ServerFile represents files in server directory
type ServerFile struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ServerID  uuid.UUID `json:"server_id" gorm:"type:uuid;not null"`
	Path      string    `json:"path" gorm:"not null"`
	Name      string    `json:"name" gorm:"not null"`
	Size      int64     `json:"size"`
	Type      FileType  `json:"type"`
	MimeType  string    `json:"mime_type"`
	IsHidden  bool      `json:"is_hidden"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	Server Server `json:"server,omitempty"`
}

type FileType string

const (
	FileTypeFile      FileType = "file"
	FileTypeDirectory FileType = "directory"
	FileTypeSymlink   FileType = "symlink"
)

// AuditLog represents system audit logs
type AuditLog struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid"`
	ServerID  *uuid.UUID `json:"server_id" gorm:"type:uuid"`
	Action    string    `json:"action" gorm:"not null"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	
	User   User    `json:"user,omitempty"`
	Server *Server `json:"server,omitempty"`
}

// SystemSetting represents system-wide settings
type SystemSetting struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	Value     string    `json:"value"`
	Type      string    `json:"type" gorm:"default:'string'"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Notification represents system notifications
type Notification struct {
	ID        uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID          `json:"user_id" gorm:"type:uuid;not null"`
	ServerID  *uuid.UUID         `json:"server_id" gorm:"type:uuid"`
	Title     string             `json:"title" gorm:"not null"`
	Message   string             `json:"message" gorm:"not null"`
	Type      NotificationType   `json:"type" gorm:"not null"`
	Priority  NotificationPriority `json:"priority" gorm:"default:'medium'"`
	IsRead    bool               `json:"is_read" gorm:"default:false"`
	CreatedAt time.Time          `json:"created_at"`
	ReadAt    *time.Time         `json:"read_at"`
	
	User   User    `json:"user,omitempty"`
	Server *Server `json:"server,omitempty"`
}

type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
)

type NotificationPriority string

const (
	NotificationPriorityLow    NotificationPriority = "low"
	NotificationPriorityMedium NotificationPriority = "medium"
	NotificationPriorityHigh   NotificationPriority = "high"
)

// BeforeCreate hooks
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.APIKey == "" {
		u.APIKey = uuid.New().String()
	}
	return nil
}

func (s *Server) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}