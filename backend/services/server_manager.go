package services

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"playpulse-panel/database"
	"playpulse-panel/models"
	"playpulse-panel/utils"

	"github.com/google/uuid"
)

// ServerManager handles server operations
type ServerManager struct {
	processes map[uuid.UUID]*exec.Cmd
}

var manager = &ServerManager{
	processes: make(map[uuid.UUID]*exec.Cmd),
}

// ServerStats represents current server statistics
type ServerStats struct {
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  int64   `json:"memory_usage"`
	MemoryLimit  int64   `json:"memory_limit"`
	DiskUsage    int64   `json:"disk_usage"`
	DiskLimit    int64   `json:"disk_limit"`
	NetworkIn    int64   `json:"network_in"`
	NetworkOut   int64   `json:"network_out"`
	PlayerCount  int     `json:"player_count"`
	TPS          float64 `json:"tps"`
	MSPT         float64 `json:"mspt"`
	Uptime       int64   `json:"uptime"`
	IsOnline     bool    `json:"is_online"`
}

// StartServer starts a game server
func StartServer(server *models.Server) error {
	if server.Status == models.ServerStatusRunning {
		return fmt.Errorf("server is already running")
	}

	// Update status to starting
	server.Status = models.ServerStatusStarting
	database.DB.Save(server)

	// Ensure server directory exists
	if err := utils.CreateDirectory(server.Path); err != nil {
		server.Status = models.ServerStatusStopped
		database.DB.Save(server)
		return fmt.Errorf("failed to create server directory: %v", err)
	}

	// Check if server jar exists
	serverJarPath := filepath.Join(server.Path, server.ServerJar)
	if !utils.FileExists(serverJarPath) {
		// Try to download the server jar
		if err := SetupServerJar(server); err != nil {
			server.Status = models.ServerStatusStopped
			database.DB.Save(server)
			return fmt.Errorf("server jar not found and download failed: %v", err)
		}
	}

	// Parse Java arguments
	javaArgs := utils.ParseJavaArgs(server.JavaArgs)
	
	// Build command arguments
	args := append(javaArgs, "-jar", server.ServerJar)
	
	// Add nogui if not present
	hasNoGui := false
	for _, arg := range args {
		if arg == "nogui" {
			hasNoGui = true
			break
		}
	}
	if !hasNoGui {
		args = append(args, "nogui")
	}

	// Create command
	cmd := exec.Command(server.JavaPath, args...)
	cmd.Dir = server.Path
	
	// Set up pipes for stdin/stdout/stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		server.Status = models.ServerStatusStopped
		database.DB.Save(server)
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		server.Status = models.ServerStatusStopped
		database.DB.Save(server)
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		server.Status = models.ServerStatusStopped
		database.DB.Save(server)
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		server.Status = models.ServerStatusStopped
		database.DB.Save(server)
		return fmt.Errorf("failed to start server process: %v", err)
	}

	// Store process reference
	manager.processes[server.ID] = cmd

	// Update server with PID
	server.PID = cmd.Process.Pid
	server.Status = models.ServerStatusRunning
	database.DB.Save(server)

	// Handle process output
	go handleServerOutput(server, stdout, stderr)
	
	// Store stdin reference for sending commands
	go handleServerInput(server, stdin)

	// Monitor process
	go monitorServerProcess(server, cmd)

	return nil
}

// StopServer stops a game server
func StopServer(server *models.Server) error {
	if server.Status == models.ServerStatusStopped {
		return fmt.Errorf("server is already stopped")
	}

	// Update status
	server.Status = models.ServerStatusStopping
	database.DB.Save(server)

	// Try graceful shutdown first
	if server.StopCommand != "" {
		SendServerCommand(server, server.StopCommand)
	} else {
		SendServerCommand(server, "stop")
	}

	// Give it time to shutdown gracefully
	time.Sleep(10 * time.Second)

	// Force kill if still running
	if utils.IsProcessRunning(server.PID) {
		if err := utils.KillProcess(server.PID); err != nil {
			return fmt.Errorf("failed to kill server process: %v", err)
		}
	}

	// Clean up
	delete(manager.processes, server.ID)
	server.PID = 0
	server.Status = models.ServerStatusStopped
	database.DB.Save(server)

	return nil
}

// RestartServer restarts a game server
func RestartServer(server *models.Server) error {
	if server.Status == models.ServerStatusRunning {
		if err := StopServer(server); err != nil {
			return err
		}
		
		// Wait for server to fully stop
		for i := 0; i < 30; i++ {
			if server.Status == models.ServerStatusStopped {
				break
			}
			time.Sleep(1 * time.Second)
			database.DB.First(server, server.ID)
		}
	}

	return StartServer(server)
}

// SendServerCommand sends a command to a running server
func SendServerCommand(server *models.Server, command string) error {
	if server.Status != models.ServerStatusRunning {
		return fmt.Errorf("server is not running")
	}

	cmd, exists := manager.processes[server.ID]
	if !exists {
		return fmt.Errorf("server process not found")
	}

	stdin := cmd.Stdin
	if stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	// Send command
	_, err := io.WriteString(stdin, command+"\n")
	return err
}

// UpdateServerStatus updates the server status based on process state
func UpdateServerStatus(server *models.Server) {
	if server.PID > 0 {
		if utils.IsProcessRunning(server.PID) {
			if server.Status != models.ServerStatusRunning {
				server.Status = models.ServerStatusRunning
				database.DB.Save(server)
			}
		} else {
			if server.Status != models.ServerStatusStopped {
				server.Status = models.ServerStatusStopped
				server.PID = 0
				database.DB.Save(server)
				delete(manager.processes, server.ID)
			}
		}
	} else {
		if server.Status != models.ServerStatusStopped {
			server.Status = models.ServerStatusStopped
			database.DB.Save(server)
		}
	}
}

// GetServerStats returns current server statistics
func GetServerStats(server *models.Server) (*ServerStats, error) {
	stats := &ServerStats{
		MemoryLimit: server.MemoryLimit,
		DiskLimit:   server.DiskLimit,
		IsOnline:    server.Status == models.ServerStatusRunning,
	}

	if server.PID > 0 && utils.IsProcessRunning(server.PID) {
		// Get process statistics
		if cpuUsage, memUsage, err := getProcessStats(server.PID); err == nil {
			stats.CPUUsage = cpuUsage
			stats.MemoryUsage = memUsage
		}

		// Get disk usage
		if diskUsed, _, err := utils.GetDiskUsage(server.Path); err == nil {
			stats.DiskUsage = diskUsed
		}

		// Get player count and TPS from server logs (implement based on server type)
		stats.PlayerCount = getPlayerCount(server)
		stats.TPS = getTPS(server)
		stats.MSPT = getMSPT(server)
	}

	// Save metrics to database
	metric := models.ServerMetric{
		ServerID:    server.ID,
		CPUUsage:    stats.CPUUsage,
		MemoryUsage: stats.MemoryUsage,
		DiskUsage:   stats.DiskUsage,
		NetworkIn:   stats.NetworkIn,
		NetworkOut:  stats.NetworkOut,
		PlayerCount: stats.PlayerCount,
		TPS:         stats.TPS,
		MSPT:        stats.MSPT,
		Timestamp:   time.Now(),
	}
	database.DB.Create(&metric)

	return stats, nil
}

// GetServerLogs returns server console logs
func GetServerLogs(server *models.Server, lines int) ([]string, error) {
	logFile := filepath.Join(server.Path, "logs", "latest.log")
	
	if !utils.FileExists(logFile) {
		return []string{}, nil
	}

	file, err := os.Open(logFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logLines []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		logLines = append(logLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last N lines
	if len(logLines) > lines {
		return logLines[len(logLines)-lines:], nil
	}

	return logLines, nil
}

// SetupServerJar downloads and sets up the server jar
func SetupServerJar(server *models.Server) error {
	var downloadURL string
	var fileName string

	switch server.Type {
	case models.ServerTypePaper:
		downloadURL = getPaperDownloadURL(server.Version)
		fileName = fmt.Sprintf("paper-%s.jar", server.Version)
	case models.ServerTypeSpigot:
		downloadURL = getSpigotDownloadURL(server.Version)
		fileName = fmt.Sprintf("spigot-%s.jar", server.Version)
	case models.ServerTypeVanilla:
		downloadURL = getVanillaDownloadURL(server.Version)
		fileName = fmt.Sprintf("server-%s.jar", server.Version)
	case models.ServerTypeFabric:
		downloadURL = getFabricDownloadURL(server.Version)
		fileName = fmt.Sprintf("fabric-server-%s.jar", server.Version)
	case models.ServerTypeForge:
		downloadURL = getForgeDownloadURL(server.Version)
		fileName = fmt.Sprintf("forge-server-%s.jar", server.Version)
	default:
		return fmt.Errorf("unsupported server type: %s", server.Type)
	}

	if downloadURL == "" {
		return fmt.Errorf("download URL not available for server type %s version %s", server.Type, server.Version)
	}

	// Download the jar file
	jarPath := filepath.Join(server.Path, fileName)
	if err := downloadFile(downloadURL, jarPath); err != nil {
		return fmt.Errorf("failed to download server jar: %v", err)
	}

	// Update server jar in database
	server.ServerJar = fileName
	database.DB.Save(server)

	// Create server.properties if it doesn't exist
	createDefaultServerProperties(server)

	// Accept EULA
	acceptEULA(server)

	return nil
}

// Helper functions

func handleServerOutput(server *models.Server, stdout, stderr io.ReadCloser) {
	// Create log file
	logPath := filepath.Join(server.Path, "console.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer logFile.Close()

	// Handle stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			logFile.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), line))
			
			// Broadcast to WebSocket clients
			BroadcastServerLog(server.ID, line)
		}
	}()

	// Handle stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			logFile.WriteString(fmt.Sprintf("[%s] ERROR: %s\n", time.Now().Format("2006-01-02 15:04:05"), line))
			
			// Broadcast to WebSocket clients
			BroadcastServerLog(server.ID, "ERROR: "+line)
		}
	}()
}

func handleServerInput(server *models.Server, stdin io.WriteCloser) {
	// Store stdin reference for sending commands
	// This would be used by SendServerCommand
}

func monitorServerProcess(server *models.Server, cmd *exec.Cmd) {
	// Wait for process to exit
	err := cmd.Wait()
	
	// Clean up
	delete(manager.processes, server.ID)
	server.PID = 0
	
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() == 0 {
					server.Status = models.ServerStatusStopped
				} else {
					server.Status = models.ServerStatusCrashed
				}
			}
		} else {
			server.Status = models.ServerStatusCrashed
		}
	} else {
		server.Status = models.ServerStatusStopped
	}
	
	database.DB.Save(server)
	
	// Auto-restart if enabled and crashed
	if server.AutoRestart && server.Status == models.ServerStatusCrashed {
		time.Sleep(5 * time.Second)
		StartServer(server)
	}
}

func getProcessStats(pid int) (float64, int64, error) {
	// Read process stats from /proc/[pid]/stat
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statFile)
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 24 {
		return 0, 0, fmt.Errorf("invalid stat file format")
	}

	// Memory usage is in field 23 (RSS in pages)
	rss, err := strconv.ParseInt(fields[23], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	// Convert pages to bytes (assuming 4KB pages)
	memoryUsage := rss * 4096

	// CPU usage calculation would require tracking over time
	// For now, return 0 (implement proper CPU monitoring)
	cpuUsage := 0.0

	return cpuUsage, memoryUsage, nil
}

func getPlayerCount(server *models.Server) int {
	// Parse player count from server logs or status
	// Implementation depends on server type
	return 0
}

func getTPS(server *models.Server) float64 {
	// Parse TPS from server logs
	// Implementation depends on server type
	return 20.0
}

func getMSPT(server *models.Server) float64 {
	// Parse MSPT from server logs
	// Implementation depends on server type
	return 50.0
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func createDefaultServerProperties(server *models.Server) {
	propertiesPath := filepath.Join(server.Path, "server.properties")
	if utils.FileExists(propertiesPath) {
		return
	}

	properties := fmt.Sprintf(`# Minecraft server properties
server-port=%d
max-players=20
level-name=world
gamemode=survival
difficulty=easy
online-mode=true
white-list=false
motd=A Playpulse Server
`, server.Port)

	os.WriteFile(propertiesPath, []byte(properties), 0644)
}

func acceptEULA(server *models.Server) {
	eulaPath := filepath.Join(server.Path, "eula.txt")
	eula := "eula=true\n"
	os.WriteFile(eulaPath, []byte(eula), 0644)
}

// Download URL functions (implement actual API calls)
func getPaperDownloadURL(version string) string {
	// Implement Paper API call to get download URL
	return fmt.Sprintf("https://papermc.io/api/v2/projects/paper/versions/%s/builds/latest/downloads/paper-%s-latest.jar", version, version)
}

func getSpigotDownloadURL(version string) string {
	// Spigot doesn't provide direct downloads, would need BuildTools
	return ""
}

func getVanillaDownloadURL(version string) string {
	// Implement Minecraft version manifest API
	return fmt.Sprintf("https://launcher.mojang.com/v1/objects/server.jar")
}

func getFabricDownloadURL(version string) string {
	// Implement Fabric API call
	return fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s/stable/server/jar", version)
}

func getForgeDownloadURL(version string) string {
	// Implement Forge API call
	return ""
}