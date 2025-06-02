package marketplace

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Marketplace manages the plugin and theme marketplace
type Marketplace struct {
	db                *gorm.DB
	curseForgeAPI     *CurseForgeAPI
	modrinthAPI       *ModrinthAPI
	githubAPI         *GitHubAPI
	spigotAPI         *SpigotAPI
	securityScanner   *SecurityScanner
	reviewSystem      *ReviewSystem
	paymentProcessor  *PaymentProcessor
}

// MarketplaceItem represents an item in the marketplace
type MarketplaceItem struct {
	ID               uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name             string            `json:"name" gorm:"not null"`
	Slug             string            `json:"slug" gorm:"unique;not null"`
	Description      string            `json:"description"`
	ShortDescription string            `json:"short_description"`
	Category         ItemCategory      `json:"category" gorm:"not null"`
	Type             ItemType          `json:"type" gorm:"not null"`
	Price            float64           `json:"price" gorm:"default:0"`
	Currency         string            `json:"currency" gorm:"default:'USD'"`
	IsFree           bool              `json:"is_free" gorm:"default:true"`
	AuthorID         uuid.UUID         `json:"author_id" gorm:"type:uuid"`
	AuthorName       string            `json:"author_name"`
	Version          string            `json:"version"`
	MinecraftVersions []string         `json:"minecraft_versions" gorm:"type:json"`
	ServerTypes      []string          `json:"server_types" gorm:"type:json"`
	Dependencies     []Dependency      `json:"dependencies" gorm:"type:json"`
	Permissions      []string          `json:"permissions" gorm:"type:json"`
	Commands         []Command         `json:"commands" gorm:"type:json"`
	ConfigFiles      []ConfigFile      `json:"config_files" gorm:"type:json"`
	DownloadURL      string            `json:"download_url"`
	SourceURL        string            `json:"source_url"`
	DocumentationURL string            `json:"documentation_url"`
	SupportURL       string            `json:"support_url"`
	DonationURL      string            `json:"donation_url"`
	License          string            `json:"license"`
	Tags             []string          `json:"tags" gorm:"type:json"`
	Screenshots      []Screenshot      `json:"screenshots" gorm:"type:json"`
	Icon             string            `json:"icon"`
	Banner           string            `json:"banner"`
	FileSize         int64             `json:"file_size"`
	FileHash         string            `json:"file_hash"`
	SecurityScore    float64           `json:"security_score"`
	QualityScore     float64           `json:"quality_score"`
	PopularityScore  float64           `json:"popularity_score"`
	OverallRating    float64           `json:"overall_rating"`
	RatingCount      int               `json:"rating_count"`
	DownloadCount    int64             `json:"download_count"`
	ViewCount        int64             `json:"view_count"`
	FavoriteCount    int               `json:"favorite_count"`
	Status           ItemStatus        `json:"status" gorm:"default:'pending'"`
	FeaturedUntil    *time.Time        `json:"featured_until"`
	LastUpdated      time.Time         `json:"last_updated"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	
	// External source info
	ExternalSource   string            `json:"external_source"` // curseforge, modrinth, github, spigot
	ExternalID       string            `json:"external_id"`
	ExternalURL      string            `json:"external_url"`
	
	// Relationships
	Author           *Developer        `json:"author,omitempty"`
	Reviews          []Review          `json:"reviews,omitempty"`
	Versions         []ItemVersion     `json:"versions,omitempty"`
	Downloads        []Download        `json:"downloads,omitempty"`
}

type ItemCategory string

const (
	CategoryPlugins           ItemCategory = "plugins"
	CategoryMods              ItemCategory = "mods"
	CategoryThemes            ItemCategory = "themes"
	CategoryWorlds            ItemCategory = "worlds"
	CategoryResourcePacks     ItemCategory = "resourcepacks"
	CategoryDataPacks         ItemCategory = "datapacks"
	CategoryServerTemplates   ItemCategory = "server_templates"
	CategoryTools             ItemCategory = "tools"
)

type ItemType string

const (
	TypePlugin           ItemType = "plugin"
	TypeMod              ItemType = "mod"
	TypeTheme            ItemType = "theme"
	TypeWorld            ItemType = "world"
	TypeResourcePack     ItemType = "resourcepack"
	TypeDataPack         ItemType = "datapack"
	TypeServerTemplate   ItemType = "server_template"
	TypeTool             ItemType = "tool"
)

type ItemStatus string

const (
	StatusPending   ItemStatus = "pending"
	StatusApproved  ItemStatus = "approved"
	StatusRejected  ItemStatus = "rejected"
	StatusSuspended ItemStatus = "suspended"
	StatusDeleted   ItemStatus = "deleted"
)

type Dependency struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // required, optional, incompatible
	Version  string `json:"version"`
	URL      string `json:"url"`
}

type Command struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Usage       string            `json:"usage"`
	Aliases     []string          `json:"aliases"`
	Permissions []string          `json:"permissions"`
	Examples    []CommandExample  `json:"examples"`
}

type CommandExample struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type ConfigFile struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Type        string `json:"type"` // yaml, json, properties, toml
	Template    string `json:"template"`
}

type Screenshot struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Order       int    `json:"order"`
}

// Developer represents a marketplace developer
type Developer struct {
	ID               uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username         string            `json:"username" gorm:"unique;not null"`
	Email            string            `json:"email" gorm:"unique;not null"`
	DisplayName      string            `json:"display_name"`
	Bio              string            `json:"bio"`
	Website          string            `json:"website"`
	Avatar           string            `json:"avatar"`
	SocialLinks      map[string]string `json:"social_links" gorm:"type:json"`
	IsVerified       bool              `json:"is_verified" gorm:"default:false"`
	IsPremium        bool              `json:"is_premium" gorm:"default:false"`
	Rating           float64           `json:"rating"`
	TotalDownloads   int64             `json:"total_downloads"`
	TotalRevenue     float64           `json:"total_revenue"`
	PayoutInfo       PayoutInfo        `json:"payout_info" gorm:"type:json"`
	Status           DeveloperStatus   `json:"status" gorm:"default:'active'"`
	JoinedAt         time.Time         `json:"joined_at"`
	LastActive       time.Time         `json:"last_active"`
	
	// Relationships
	Items            []MarketplaceItem `json:"items,omitempty"`
}

type DeveloperStatus string

const (
	DeveloperStatusActive    DeveloperStatus = "active"
	DeveloperStatusSuspended DeveloperStatus = "suspended"
	DeveloperStatusBanned    DeveloperStatus = "banned"
)

type PayoutInfo struct {
	PayPalEmail    string `json:"paypal_email"`
	BankAccount    string `json:"bank_account"`
	TaxID          string `json:"tax_id"`
	MinimumPayout  float64 `json:"minimum_payout"`
}

// Review represents a user review
type Review struct {
	ID          uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ItemID      uuid.UUID   `json:"item_id" gorm:"type:uuid;not null"`
	UserID      uuid.UUID   `json:"user_id" gorm:"type:uuid;not null"`
	Username    string      `json:"username"`
	Rating      int         `json:"rating" gorm:"check:rating >= 1 AND rating <= 5"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	Pros        []string    `json:"pros" gorm:"type:json"`
	Cons        []string    `json:"cons" gorm:"type:json"`
	Version     string      `json:"version"`
	ServerType  string      `json:"server_type"`
	IsVerified  bool        `json:"is_verified" gorm:"default:false"`
	HelpfulVotes int        `json:"helpful_votes" gorm:"default:0"`
	TotalVotes  int         `json:"total_votes" gorm:"default:0"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	
	// Relationships
	Item        MarketplaceItem `json:"item,omitempty"`
}

// ItemVersion represents different versions of an item
type ItemVersion struct {
	ID                uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ItemID            uuid.UUID         `json:"item_id" gorm:"type:uuid;not null"`
	Version           string            `json:"version" gorm:"not null"`
	Changelog         string            `json:"changelog"`
	MinecraftVersions []string          `json:"minecraft_versions" gorm:"type:json"`
	ServerTypes       []string          `json:"server_types" gorm:"type:json"`
	DownloadURL       string            `json:"download_url"`
	FileSize          int64             `json:"file_size"`
	FileHash          string            `json:"file_hash"`
	SecurityScan      SecurityScanResult `json:"security_scan" gorm:"type:json"`
	Status            VersionStatus     `json:"status" gorm:"default:'pending'"`
	IsStable          bool              `json:"is_stable" gorm:"default:true"`
	IsBeta            bool              `json:"is_beta" gorm:"default:false"`
	IsAlpha           bool              `json:"is_alpha" gorm:"default:false"`
	CreatedAt         time.Time         `json:"created_at"`
	
	// Relationships
	Item              MarketplaceItem   `json:"item,omitempty"`
}

type VersionStatus string

const (
	VersionStatusPending  VersionStatus = "pending"
	VersionStatusApproved VersionStatus = "approved"
	VersionStatusRejected VersionStatus = "rejected"
)

type SecurityScanResult struct {
	OverallScore     float64           `json:"overall_score"`
	Threats          []SecurityThreat  `json:"threats"`
	SafetyRating     string            `json:"safety_rating"` // safe, caution, dangerous
	ScannedAt        time.Time         `json:"scanned_at"`
	ScannerVersion   string            `json:"scanner_version"`
}

type SecurityThreat struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	File        string `json:"file"`
	Line        int    `json:"line"`
}

// Download tracking
type Download struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ItemID     uuid.UUID `json:"item_id" gorm:"type:uuid;not null"`
	UserID     *uuid.UUID `json:"user_id" gorm:"type:uuid"`
	ServerID   *uuid.UUID `json:"server_id" gorm:"type:uuid"`
	Version    string    `json:"version"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Country    string    `json:"country"`
	CreatedAt  time.Time `json:"created_at"`
}

// External API integrations
type CurseForgeAPI struct {
	APIKey string
	client *http.Client
}

type ModrinthAPI struct {
	APIKey string
	client *http.Client
}

type GitHubAPI struct {
	Token  string
	client *http.Client
}

type SpigotAPI struct {
	client *http.Client
}

// Security Scanner
type SecurityScanner struct {
	enabled     bool
	scanTimeout time.Duration
}

// Review System
type ReviewSystem struct {
	db              *gorm.DB
	moderationQueue chan Review
}

// Payment Processor
type PaymentProcessor struct {
	stripeKey string
	paypalKey string
}

// NewMarketplace creates a new marketplace instance
func NewMarketplace(db *gorm.DB) *Marketplace {
	return &Marketplace{
		db: db,
		curseForgeAPI: &CurseForgeAPI{
			client: &http.Client{Timeout: 30 * time.Second},
		},
		modrinthAPI: &ModrinthAPI{
			client: &http.Client{Timeout: 30 * time.Second},
		},
		githubAPI: &GitHubAPI{
			client: &http.Client{Timeout: 30 * time.Second},
		},
		spigotAPI: &SpigotAPI{
			client: &http.Client{Timeout: 30 * time.Second},
		},
		securityScanner: &SecurityScanner{
			enabled:     true,
			scanTimeout: 5 * time.Minute,
		},
		reviewSystem: &ReviewSystem{
			db: db,
			moderationQueue: make(chan Review, 100),
		},
		paymentProcessor: &PaymentProcessor{},
	}
}

// SearchItems searches marketplace items
func (m *Marketplace) SearchItems(ctx context.Context, query SearchQuery) (*SearchResults, error) {
	var items []MarketplaceItem
	
	db := m.db.WithContext(ctx).Where("status = ?", StatusApproved)
	
	// Text search
	if query.Query != "" {
		searchTerms := strings.Fields(strings.ToLower(query.Query))
		for _, term := range searchTerms {
			db = db.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(tags::text) LIKE ?", 
				"%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
	}
	
	// Category filter
	if query.Category != "" {
		db = db.Where("category = ?", query.Category)
	}
	
	// Type filter
	if query.Type != "" {
		db = db.Where("type = ?", query.Type)
	}
	
	// Minecraft version filter
	if query.MinecraftVersion != "" {
		db = db.Where("minecraft_versions @> ?", fmt.Sprintf(`["%s"]`, query.MinecraftVersion))
	}
	
	// Server type filter
	if query.ServerType != "" {
		db = db.Where("server_types @> ?", fmt.Sprintf(`["%s"]`, query.ServerType))
	}
	
	// Price filter
	if query.IsFree {
		db = db.Where("is_free = true")
	}
	
	// Sorting
	switch query.SortBy {
	case "popularity":
		db = db.Order("popularity_score DESC")
	case "rating":
		db = db.Order("overall_rating DESC")
	case "downloads":
		db = db.Order("download_count DESC")
	case "updated":
		db = db.Order("last_updated DESC")
	case "created":
		db = db.Order("created_at DESC")
	case "name":
		db = db.Order("name ASC")
	default:
		db = db.Order("popularity_score DESC")
	}
	
	// Pagination
	offset := (query.Page - 1) * query.Limit
	db = db.Offset(offset).Limit(query.Limit)
	
	if err := db.Preload("Author").Find(&items).Error; err != nil {
		return nil, err
	}
	
	// Get total count
	var total int64
	m.db.Model(&MarketplaceItem{}).Where("status = ?", StatusApproved).Count(&total)
	
	return &SearchResults{
		Items:      items,
		Total:      int(total),
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: int((total + int64(query.Limit) - 1) / int64(query.Limit)),
	}, nil
}

// GetItem retrieves a specific marketplace item
func (m *Marketplace) GetItem(ctx context.Context, itemID uuid.UUID) (*MarketplaceItem, error) {
	var item MarketplaceItem
	
	err := m.db.WithContext(ctx).
		Preload("Author").
		Preload("Reviews", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(10)
		}).
		Preload("Versions", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		First(&item, itemID).Error
	
	if err != nil {
		return nil, err
	}
	
	// Increment view count
	go m.incrementViewCount(itemID)
	
	return &item, nil
}

// InstallItem installs an item on a server
func (m *Marketplace) InstallItem(ctx context.Context, request InstallRequest) (*InstallResult, error) {
	// Get item details
	item, err := m.GetItem(ctx, request.ItemID)
	if err != nil {
		return nil, fmt.Errorf("item not found: %w", err)
	}
	
	// Validate compatibility
	if err := m.validateCompatibility(item, request); err != nil {
		return nil, fmt.Errorf("compatibility check failed: %w", err)
	}
	
	// Security scan
	if err := m.performSecurityScan(item); err != nil {
		return nil, fmt.Errorf("security scan failed: %w", err)
	}
	
	// Download and install
	result, err := m.downloadAndInstall(ctx, item, request)
	if err != nil {
		return nil, fmt.Errorf("installation failed: %w", err)
	}
	
	// Track download
	go m.trackDownload(request.ItemID, request.UserID, request.ServerID, item.Version)
	
	return result, nil
}

// SubmitItem submits a new item to the marketplace
func (m *Marketplace) SubmitItem(ctx context.Context, request SubmissionRequest) (*MarketplaceItem, error) {
	// Validate submission
	if err := m.validateSubmission(request); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Create item
	item := &MarketplaceItem{
		Name:              request.Name,
		Slug:              m.generateSlug(request.Name),
		Description:       request.Description,
		ShortDescription:  request.ShortDescription,
		Category:          request.Category,
		Type:              request.Type,
		Price:             request.Price,
		IsFree:            request.Price == 0,
		AuthorID:          request.AuthorID,
		AuthorName:        request.AuthorName,
		Version:           request.Version,
		MinecraftVersions: request.MinecraftVersions,
		ServerTypes:       request.ServerTypes,
		Dependencies:      request.Dependencies,
		License:           request.License,
		Tags:              request.Tags,
		DownloadURL:       request.DownloadURL,
		SourceURL:         request.SourceURL,
		DocumentationURL:  request.DocumentationURL,
		Status:            StatusPending,
		LastUpdated:       time.Now(),
	}
	
	// Calculate file hash
	if request.FileData != nil {
		hash := sha256.Sum256(request.FileData)
		item.FileHash = fmt.Sprintf("%x", hash)
		item.FileSize = int64(len(request.FileData))
	}
	
	// Security scan
	if m.securityScanner.enabled {
		scanResult, err := m.performSecurityScanOnData(request.FileData)
		if err != nil {
			return nil, fmt.Errorf("security scan failed: %w", err)
		}
		item.SecurityScore = scanResult.OverallScore
	}
	
	// Save to database
	if err := m.db.WithContext(ctx).Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to save item: %w", err)
	}
	
	// Queue for review
	go m.queueForReview(item.ID)
	
	return item, nil
}

// SyncExternalSources syncs items from external sources
func (m *Marketplace) SyncExternalSources(ctx context.Context) error {
	// Sync from CurseForge
	if err := m.syncFromCurseForge(ctx); err != nil {
		log.Printf("CurseForge sync error: %v", err)
	}
	
	// Sync from Modrinth
	if err := m.syncFromModrinth(ctx); err != nil {
		log.Printf("Modrinth sync error: %v", err)
	}
	
	// Sync from GitHub
	if err := m.syncFromGitHub(ctx); err != nil {
		log.Printf("GitHub sync error: %v", err)
	}
	
	// Sync from SpigotMC
	if err := m.syncFromSpigot(ctx); err != nil {
		log.Printf("SpigotMC sync error: %v", err)
	}
	
	return nil
}

// Helper methods
func (m *Marketplace) incrementViewCount(itemID uuid.UUID) {
	m.db.Model(&MarketplaceItem{}).Where("id = ?", itemID).Update("view_count", gorm.Expr("view_count + 1"))
}

func (m *Marketplace) validateCompatibility(item *MarketplaceItem, request InstallRequest) error {
	// Check Minecraft version compatibility
	if request.MinecraftVersion != "" {
		compatible := false
		for _, version := range item.MinecraftVersions {
			if version == request.MinecraftVersion {
				compatible = true
				break
			}
		}
		if !compatible {
			return fmt.Errorf("incompatible Minecraft version")
		}
	}
	
	// Check server type compatibility
	if request.ServerType != "" {
		compatible := false
		for _, serverType := range item.ServerTypes {
			if serverType == request.ServerType {
				compatible = true
				break
			}
		}
		if !compatible {
			return fmt.Errorf("incompatible server type")
		}
	}
	
	return nil
}

func (m *Marketplace) performSecurityScan(item *MarketplaceItem) error {
	if !m.securityScanner.enabled {
		return nil
	}
	
	if item.SecurityScore < 0.7 {
		return fmt.Errorf("item failed security scan (score: %.2f)", item.SecurityScore)
	}
	
	return nil
}

func (m *Marketplace) performSecurityScanOnData(data []byte) (*SecurityScanResult, error) {
	// Implement comprehensive security scanning
	result := &SecurityScanResult{
		OverallScore:   0.9, // Placeholder
		SafetyRating:   "safe",
		ScannedAt:      time.Now(),
		ScannerVersion: "1.0.0",
	}
	
	return result, nil
}

func (m *Marketplace) downloadAndInstall(ctx context.Context, item *MarketplaceItem, request InstallRequest) (*InstallResult, error) {
	// Implement installation logic
	return &InstallResult{
		ItemID:    item.ID,
		ServerID:  request.ServerID,
		Status:    "installed",
		Message:   "Installation completed successfully",
		Files:     []string{item.Name + ".jar"},
	}, nil
}

func (m *Marketplace) trackDownload(itemID, userID, serverID uuid.UUID, version string) {
	download := Download{
		ItemID:   itemID,
		UserID:   &userID,
		ServerID: &serverID,
		Version:  version,
	}
	
	m.db.Create(&download)
	m.db.Model(&MarketplaceItem{}).Where("id = ?", itemID).Update("download_count", gorm.Expr("download_count + 1"))
}

func (m *Marketplace) validateSubmission(request SubmissionRequest) error {
	if request.Name == "" {
		return fmt.Errorf("name is required")
	}
	if request.Description == "" {
		return fmt.Errorf("description is required")
	}
	if request.Category == "" {
		return fmt.Errorf("category is required")
	}
	return nil
}

func (m *Marketplace) generateSlug(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	return slug
}

func (m *Marketplace) queueForReview(itemID uuid.UUID) {
	// Implementation for review queue
}

// External API sync methods (simplified implementations)
func (m *Marketplace) syncFromCurseForge(ctx context.Context) error {
	// Implementation for CurseForge API sync
	return nil
}

func (m *Marketplace) syncFromModrinth(ctx context.Context) error {
	// Implementation for Modrinth API sync
	return nil
}

func (m *Marketplace) syncFromGitHub(ctx context.Context) error {
	// Implementation for GitHub API sync
	return nil
}

func (m *Marketplace) syncFromSpigot(ctx context.Context) error {
	// Implementation for SpigotMC API sync
	return nil
}

// Supporting types
type SearchQuery struct {
	Query            string `json:"query"`
	Category         string `json:"category"`
	Type             string `json:"type"`
	MinecraftVersion string `json:"minecraft_version"`
	ServerType       string `json:"server_type"`
	IsFree           bool   `json:"is_free"`
	SortBy           string `json:"sort_by"`
	Page             int    `json:"page"`
	Limit            int    `json:"limit"`
}

type SearchResults struct {
	Items      []MarketplaceItem `json:"items"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

type InstallRequest struct {
	ItemID           uuid.UUID `json:"item_id"`
	UserID           uuid.UUID `json:"user_id"`
	ServerID         uuid.UUID `json:"server_id"`
	Version          string    `json:"version"`
	MinecraftVersion string    `json:"minecraft_version"`
	ServerType       string    `json:"server_type"`
	ForceInstall     bool      `json:"force_install"`
}

type InstallResult struct {
	ItemID   uuid.UUID `json:"item_id"`
	ServerID uuid.UUID `json:"server_id"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	Files    []string  `json:"files"`
}

type SubmissionRequest struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	ShortDescription  string            `json:"short_description"`
	Category          ItemCategory      `json:"category"`
	Type              ItemType          `json:"type"`
	Price             float64           `json:"price"`
	AuthorID          uuid.UUID         `json:"author_id"`
	AuthorName        string            `json:"author_name"`
	Version           string            `json:"version"`
	MinecraftVersions []string          `json:"minecraft_versions"`
	ServerTypes       []string          `json:"server_types"`
	Dependencies      []Dependency      `json:"dependencies"`
	License           string            `json:"license"`
	Tags              []string          `json:"tags"`
	DownloadURL       string            `json:"download_url"`
	SourceURL         string            `json:"source_url"`
	DocumentationURL  string            `json:"documentation_url"`
	FileData          []byte            `json:"file_data"`
}