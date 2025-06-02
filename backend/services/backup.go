package services

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"playpulse-panel/config"
	"playpulse-panel/database"
	"playpulse-panel/models"
	"playpulse-panel/utils"

	"github.com/google/uuid"
)

// BackupService handles server backups
type BackupService struct {
	config *config.Config
}

var backupService *BackupService

// InitializeBackupService initializes the backup service
func InitializeBackupService(cfg *config.Config) {
	backupService = &BackupService{
		config: cfg,
	}
	
	// Start backup scheduler
	go backupService.startScheduler()
}

// CreateBackup creates a backup of a server
func CreateBackup(server *models.Server, backupName string) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}

	// Create backup record
	backup := models.Backup{
		ServerID:    server.ID,
		Name:        backupName,
		Description: fmt.Sprintf("Backup created at %s", time.Now().Format("2006-01-02 15:04:05")),
		Type:        models.BackupTypeManual,
		Status:      models.BackupStatusCreating,
	}

	if err := database.DB.Create(&backup).Error; err != nil {
		return fmt.Errorf("failed to create backup record: %v", err)
	}

	// Create backup in background
	go func() {
		if err := backupService.performBackup(server, &backup); err != nil {
			backup.Status = models.BackupStatusFailed
			database.DB.Save(&backup)
			return
		}

		backup.Status = models.BackupStatusCompleted
		database.DB.Save(&backup)
	}()

	return nil
}

// RestoreBackup restores a server from a backup
func RestoreBackup(server *models.Server, backupID uuid.UUID) error {
	var backup models.Backup
	if err := database.DB.First(&backup, backupID).Error; err != nil {
		return fmt.Errorf("backup not found: %v", err)
	}

	if backup.ServerID != server.ID {
		return fmt.Errorf("backup does not belong to this server")
	}

	if backup.Status != models.BackupStatusCompleted {
		return fmt.Errorf("backup is not completed")
	}

	// Stop server if running
	wasRunning := server.Status == models.ServerStatusRunning
	if wasRunning {
		if err := StopServer(server); err != nil {
			return fmt.Errorf("failed to stop server: %v", err)
		}
	}

	// Perform restore
	if err := backupService.performRestore(server, &backup); err != nil {
		return fmt.Errorf("failed to restore backup: %v", err)
	}

	// Start server if it was running
	if wasRunning {
		if err := StartServer(server); err != nil {
			return fmt.Errorf("failed to start server after restore: %v", err)
		}
	}

	return nil
}

// DeleteBackup deletes a backup
func DeleteBackup(backupID uuid.UUID) error {
	var backup models.Backup
	if err := database.DB.First(&backup, backupID).Error; err != nil {
		return fmt.Errorf("backup not found: %v", err)
	}

	// Delete backup file
	if backup.Path != "" && utils.FileExists(backup.Path) {
		if err := os.Remove(backup.Path); err != nil {
			return fmt.Errorf("failed to delete backup file: %v", err)
		}
	}

	// Delete backup record
	if err := database.DB.Delete(&backup).Error; err != nil {
		return fmt.Errorf("failed to delete backup record: %v", err)
	}

	return nil
}

// GetServerBackups returns all backups for a server
func GetServerBackups(serverID uuid.UUID) ([]models.Backup, error) {
	var backups []models.Backup
	err := database.DB.Where("server_id = ?", serverID).Order("created_at DESC").Find(&backups).Error
	return backups, err
}

// CleanupOldBackups removes old backups based on retention policy
func CleanupOldBackups(serverID uuid.UUID, maxBackups int) error {
	var backups []models.Backup
	err := database.DB.Where("server_id = ? AND status = ?", serverID, models.BackupStatusCompleted).
		Order("created_at DESC").Find(&backups).Error
	if err != nil {
		return err
	}

	if len(backups) <= maxBackups {
		return nil
	}

	// Delete oldest backups
	for i := maxBackups; i < len(backups); i++ {
		if err := DeleteBackup(backups[i].ID); err != nil {
			return fmt.Errorf("failed to delete old backup: %v", err)
		}
	}

	return nil
}

// Internal methods

func (bs *BackupService) performBackup(server *models.Server, backup *models.Backup) error {
	// Create backup directory
	backupDir := filepath.Join(bs.config.Files.BackupPath, server.ID.String())
	if err := utils.CreateDirectory(backupDir); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupFilename := fmt.Sprintf("%s-%s.zip", server.Name, timestamp)
	backupPath := filepath.Join(backupDir, backupFilename)

	// Create backup zip file
	if err := bs.createZipBackup(server.Path, backupPath); err != nil {
		return fmt.Errorf("failed to create zip backup: %v", err)
	}

	// Get backup file size
	size, err := utils.GetFileSize(backupPath)
	if err != nil {
		return fmt.Errorf("failed to get backup size: %v", err)
	}

	// Update backup record
	backup.Path = backupPath
	backup.Size = size

	return nil
}

func (bs *BackupService) performRestore(server *models.Server, backup *models.Backup) error {
	// Create temporary restore directory
	tempDir := filepath.Join(os.TempDir(), "playpulse-restore-"+uuid.New().String())
	if err := utils.CreateDirectory(tempDir); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract backup to temp directory
	if err := bs.extractZipBackup(backup.Path, tempDir); err != nil {
		return fmt.Errorf("failed to extract backup: %v", err)
	}

	// Backup current server directory
	currentBackupPath := server.Path + ".bak." + time.Now().Format("20060102-150405")
	if err := os.Rename(server.Path, currentBackupPath); err != nil {
		return fmt.Errorf("failed to backup current server directory: %v", err)
	}

	// Move restored files to server directory
	if err := os.Rename(tempDir, server.Path); err != nil {
		// Restore original directory on failure
		os.Rename(currentBackupPath, server.Path)
		return fmt.Errorf("failed to restore server directory: %v", err)
	}

	// Remove backup of current directory
	go func() {
		time.Sleep(5 * time.Minute)
		os.RemoveAll(currentBackupPath)
	}()

	return nil
}

func (bs *BackupService) createZipBackup(sourceDir, backupPath string) error {
	zipFile, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip certain files/directories
		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip temporary files and logs (optional)
		if bs.shouldSkipFile(relativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		zipFileWriter, err := zipWriter.Create(relativePath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipFileWriter, file)
		return err
	})
}

func (bs *BackupService) extractZipBackup(backupPath, destDir string) error {
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		// Ensure the file path is within the destination directory
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bs *BackupService) shouldSkipFile(relativePath string) bool {
	skipPatterns := []string{
		"logs/",
		"crash-reports/",
		".tmp",
		".log",
		"session.lock",
		"console.log",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(relativePath, pattern) {
			return true
		}
	}

	return false
}

func (bs *BackupService) startScheduler() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		bs.performScheduledBackups()
	}
}

func (bs *BackupService) performScheduledBackups() {
	var servers []models.Server
	database.DB.Where("backup_enabled = ?", true).Find(&servers)

	for _, server := range servers {
		// Check if backup is needed
		if bs.needsBackup(&server) {
			backupName := fmt.Sprintf("scheduled-%s", time.Now().Format("20060102-150405"))
			
			backup := models.Backup{
				ServerID:    server.ID,
				Name:        backupName,
				Description: "Scheduled automatic backup",
				Type:        models.BackupTypeScheduled,
				Status:      models.BackupStatusCreating,
			}

			if err := database.DB.Create(&backup).Error; err != nil {
				continue
			}

			go func(s models.Server, b models.Backup) {
				if err := bs.performBackup(&s, &b); err != nil {
					b.Status = models.BackupStatusFailed
					database.DB.Save(&b)
					return
				}

				b.Status = models.BackupStatusCompleted
				database.DB.Save(&b)

				// Update server's last backup time
				s.LastBackup = &b.CreatedAt
				database.DB.Save(&s)

				// Clean up old backups
				CleanupOldBackups(s.ID, 10) // Keep last 10 backups
			}(server, backup)
		}
	}
}

func (bs *BackupService) needsBackup(server *models.Server) bool {
	// Backup every 24 hours
	backupInterval := 24 * time.Hour

	if server.LastBackup == nil {
		return true
	}

	return time.Since(*server.LastBackup) >= backupInterval
}