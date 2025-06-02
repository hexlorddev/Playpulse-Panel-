package utils

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT generates a JWT token for a user
func GenerateJWT(userID uuid.UUID, secret string, expireHours int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * time.Duration(expireHours)).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken generates a refresh token
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// Generate2FASecret generates a 2FA secret
func Generate2FASecret() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// GetRequestDetails extracts request details for audit logging
func GetRequestDetails(c *fiber.Ctx) string {
	details := map[string]interface{}{
		"method": c.Method(),
		"path":   c.Path(),
		"params": c.AllParams(),
	}

	if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
		var body map[string]interface{}
		if err := json.Unmarshal(c.Body(), &body); err == nil {
			// Remove sensitive fields
			delete(body, "password")
			delete(body, "token")
			delete(body, "secret")
			details["body"] = body
		}
	}

	detailsJSON, _ := json.Marshal(details)
	return string(detailsJSON)
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// CreateDirectory creates a directory if it doesn't exist
func CreateDirectory(path string) error {
	if !FileExists(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// RunCommand executes a system command
func RunCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunCommandInDir executes a command in a specific directory
func RunCommandInDir(dir, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// StartProcess starts a process and returns its PID
func StartProcess(command string, args []string, workDir string) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	
	return cmd.Process.Pid, nil
}

// KillProcess kills a process by PID
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	
	return process.Kill()
}

// IsProcessRunning checks if a process is running
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	
	// Send signal 0 to check if process exists
	err = process.Signal(os.Signal(nil))
	return err == nil
}

// GetSystemMemory returns system memory information
func GetSystemMemory() (int64, int64, error) {
	// Read /proc/meminfo
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	
	lines := strings.Split(string(data), "\n")
	var total, available int64
	
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseInt(fields[1], 10, 64)
				total = val * 1024 // Convert from KB to bytes
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseInt(fields[1], 10, 64)
				available = val * 1024 // Convert from KB to bytes
			}
		}
	}
	
	return total, available, nil
}

// GetDiskUsage returns disk usage for a path
func GetDiskUsage(path string) (int64, int64, error) {
	var total, available int64
	
	cmd := exec.Command("df", "-B1", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			total, _ = strconv.ParseInt(fields[1], 10, 64)
			available, _ = strconv.ParseInt(fields[3], 10, 64)
		}
	}
	
	return total, available, nil
}

// FormatBytes formats bytes into human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SanitizeFilename removes dangerous characters from filename
func SanitizeFilename(filename string) string {
	// Remove path separators and dangerous characters
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	result := filename
	for _, char := range forbidden {
		result = strings.ReplaceAll(result, char, "_")
	}
	
	// Remove leading/trailing spaces and dots
	result = strings.Trim(result, " .")
	
	// Ensure filename is not empty
	if result == "" {
		result = "unnamed"
	}
	
	return result
}

// ValidateServerPath checks if a server path is valid and safe
func ValidateServerPath(path string) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}
	
	// Check if path contains dangerous patterns
	dangerous := []string{"../", "..", "/etc/", "/var/", "/usr/", "/bin/", "/sbin/"}
	for _, pattern := range dangerous {
		if strings.Contains(absPath, pattern) {
			return fmt.Errorf("path contains dangerous pattern: %s", pattern)
		}
	}
	
	return nil
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes), nil
}

// ParseJavaArgs parses Java arguments string into a slice
func ParseJavaArgs(args string) []string {
	if args == "" {
		return []string{}
	}
	
	// Split by spaces but preserve quoted strings
	var result []string
	var current strings.Builder
	inQuotes := false
	
	for i, char := range args {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
				continue
			}
			fallthrough
		default:
			current.WriteRune(char)
		}
		
		// Add the last argument if we're at the end
		if i == len(args)-1 && current.Len() > 0 {
			result = append(result, current.String())
		}
	}
	
	return result
}

// ValidateEmail validates an email address
func ValidateEmail(email string) bool {
	// Simple email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	
	if !strings.Contains(parts[1], ".") {
		return false
	}
	
	return true
}

// ValidateUsername validates a username
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	
	// Only allow alphanumeric characters, underscores, and hyphens
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_' || char == '-') {
			return false
		}
	}
	
	return true
}

// TimeAgo returns a human readable time difference
func TimeAgo(t time.Time) string {
	now := time.Now().UTC()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}