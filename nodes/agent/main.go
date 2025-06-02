package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/google/uuid"
)

// NodeAgent represents the agent running on each VPS node
type NodeAgent struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Location     string    `json:"location"`
	ControlPlane string    `json:"control_plane"`
	Status       string    `json:"status"`
	LastSeen     time.Time `json:"last_seen"`
	Capabilities []string  `json:"capabilities"`
	Resources    Resources `json:"resources"`
	conn         *websocket.Conn
}

type Resources struct {
	CPU    CPUInfo    `json:"cpu"`
	Memory MemoryInfo `json:"memory"`
	Disk   DiskInfo   `json:"disk"`
	Network NetworkInfo `json:"network"`
}

type CPUInfo struct {
	Cores       int     `json:"cores"`
	UsagePercent float64 `json:"usage_percent"`
	LoadAverage  float64 `json:"load_average"`
	Temperature  float64 `json:"temperature"`
}

type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsagePercent float64 `json:"usage_percent"`
	Swap        uint64  `json:"swap"`
}

type DiskInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsagePercent float64 `json:"usage_percent"`
	IORead      uint64  `json:"io_read"`
	IOWrite     uint64  `json:"io_write"`
}

type NetworkInfo struct {
	BytesReceived uint64 `json:"bytes_received"`
	BytesSent     uint64 `json:"bytes_sent"`
	PacketsReceived uint64 `json:"packets_received"`
	PacketsSent   uint64 `json:"packets_sent"`
	Interfaces    []NetworkInterface `json:"interfaces"`
}

type NetworkInterface struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Status    string `json:"status"`
	Speed     uint64 `json:"speed"`
}

type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	NodeID    string      `json:"node_id"`
}

type Command struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type ServerDeployment struct {
	ServerID    string            `json:"server_id"`
	ServerType  string            `json:"server_type"`
	Version     string            `json:"version"`
	Port        int               `json:"port"`
	Memory      int64             `json:"memory"`
	Environment map[string]string `json:"environment"`
}

func main() {
	agent := &NodeAgent{
		ID:           getNodeID(),
		Name:         getNodeName(),
		Location:     getNodeLocation(),
		ControlPlane: getControlPlaneURL(),
		Status:       "initializing",
		Capabilities: getNodeCapabilities(),
	}

	log.Printf("🚀 Playpulse Node Agent Starting")
	log.Printf("Node ID: %s", agent.ID)
	log.Printf("Node Name: %s", agent.Name)
	log.Printf("Location: %s", agent.Location)
	log.Printf("Control Plane: %s", agent.ControlPlane)

	// Connect to control plane
	if err := agent.connectToControlPlane(); err != nil {
		log.Fatalf("Failed to connect to control plane: %v", err)
	}

	// Start resource monitoring
	go agent.startResourceMonitoring()

	// Start command processor
	go agent.startCommandProcessor()

	// Start health check server
	go agent.startHealthCheckServer()

	log.Printf("✅ Node Agent running successfully")

	// Keep the agent running
	select {}
}

func (agent *NodeAgent) connectToControlPlane() error {
	wsURL := fmt.Sprintf("ws://%s/api/v1/nodes/connect", agent.ControlPlane)
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"Node-ID":       []string{agent.ID},
		"Node-Name":     []string{agent.Name},
		"Node-Location": []string{agent.Location},
		"Authorization": []string{fmt.Sprintf("Bearer %s", getNodeToken())},
	})
	
	if err != nil {
		return fmt.Errorf("failed to connect to control plane: %v", err)
	}

	agent.conn = conn
	agent.Status = "connected"

	// Send initial registration
	registrationMsg := Message{
		Type:      "node_registration",
		Data:      agent,
		Timestamp: time.Now(),
		NodeID:    agent.ID,
	}

	return agent.sendMessage(registrationMsg)
}

func (agent *NodeAgent) startResourceMonitoring() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		resources, err := agent.collectResources()
		if err != nil {
			log.Printf("Error collecting resources: %v", err)
			continue
		}

		agent.Resources = resources

		// Send resource update to control plane
		msg := Message{
			Type:      "resource_update",
			Data:      resources,
			Timestamp: time.Now(),
			NodeID:    agent.ID,
		}

		if err := agent.sendMessage(msg); err != nil {
			log.Printf("Error sending resource update: %v", err)
		}
	}
}

func (agent *NodeAgent) collectResources() (Resources, error) {
	// CPU Information
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return Resources{}, err
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		return Resources{}, err
	}

	// Memory Information
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return Resources{}, err
	}

	swapInfo, err := mem.SwapMemory()
	if err != nil {
		return Resources{}, err
	}

	// Disk Information
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return Resources{}, err
	}

	// Network Information
	netStats, err := net.IOCounters(false)
	if err != nil {
		return Resources{}, err
	}

	var netInfo NetworkInfo
	if len(netStats) > 0 {
		netInfo = NetworkInfo{
			BytesReceived:   netStats[0].BytesRecv,
			BytesSent:       netStats[0].BytesSent,
			PacketsReceived: netStats[0].PacketsRecv,
			PacketsSent:     netStats[0].PacketsSent,
		}
	}

	// Get network interfaces
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if len(iface.Addrs) > 0 {
				netInfo.Interfaces = append(netInfo.Interfaces, NetworkInterface{
					Name:   iface.Name,
					IP:     iface.Addrs[0].Addr,
					Status: "up",
				})
			}
		}
	}

	resources := Resources{
		CPU: CPUInfo{
			Cores:        len(cpuInfo),
			UsagePercent: cpuPercent[0],
		},
		Memory: MemoryInfo{
			Total:        memInfo.Total,
			Available:    memInfo.Available,
			Used:         memInfo.Used,
			UsagePercent: memInfo.UsedPercent,
			Swap:         swapInfo.Used,
		},
		Disk: DiskInfo{
			Total:        diskInfo.Total,
			Available:    diskInfo.Free,
			Used:         diskInfo.Used,
			UsagePercent: diskInfo.UsedPercent,
		},
		Network: netInfo,
	}

	return resources, nil
}

func (agent *NodeAgent) startCommandProcessor() {
	for {
		var msg Message
		err := agent.conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			// Attempt to reconnect
			agent.reconnect()
			continue
		}

		go agent.processCommand(msg)
	}
}

func (agent *NodeAgent) processCommand(msg Message) {
	switch msg.Type {
	case "deploy_server":
		agent.deployServer(msg.Data)
	case "stop_server":
		agent.stopServer(msg.Data)
	case "restart_server":
		agent.restartServer(msg.Data)
	case "update_server":
		agent.updateServer(msg.Data)
	case "get_server_status":
		agent.getServerStatus(msg.Data)
	case "execute_command":
		agent.executeCommand(msg.Data)
	case "file_operation":
		agent.handleFileOperation(msg.Data)
	case "node_update":
		agent.updateNode(msg.Data)
	case "health_check":
		agent.respondHealthCheck()
	default:
		log.Printf("Unknown command type: %s", msg.Type)
	}
}

func (agent *NodeAgent) deployServer(data interface{}) {
	deployment, ok := data.(ServerDeployment)
	if !ok {
		log.Printf("Invalid deployment data")
		return
	}

	log.Printf("🚀 Deploying server %s of type %s", deployment.ServerID, deployment.ServerType)

	// Create server directory
	serverPath := fmt.Sprintf("/opt/playpulse/servers/%s", deployment.ServerID)
	if err := os.MkdirAll(serverPath, 0755); err != nil {
		agent.sendError("Failed to create server directory", err)
		return
	}

	// Download server software based on type
	if err := agent.downloadServerSoftware(deployment, serverPath); err != nil {
		agent.sendError("Failed to download server software", err)
		return
	}

	// Create Docker container for the server
	if err := agent.createServerContainer(deployment, serverPath); err != nil {
		agent.sendError("Failed to create server container", err)
		return
	}

	// Start the server
	if err := agent.startServerContainer(deployment.ServerID); err != nil {
		agent.sendError("Failed to start server container", err)
		return
	}

	// Send success response
	response := Message{
		Type: "server_deployed",
		Data: map[string]interface{}{
			"server_id": deployment.ServerID,
			"status":    "running",
			"node_id":   agent.ID,
		},
		Timestamp: time.Now(),
		NodeID:    agent.ID,
	}

	agent.sendMessage(response)
	log.Printf("✅ Server %s deployed successfully", deployment.ServerID)
}

func (agent *NodeAgent) downloadServerSoftware(deployment ServerDeployment, serverPath string) error {
	var downloadURL string
	var fileName string

	switch deployment.ServerType {
	case "minecraft-paper":
		downloadURL = fmt.Sprintf("https://papermc.io/api/v2/projects/paper/versions/%s/builds/latest/downloads/paper-%s-latest.jar", deployment.Version, deployment.Version)
		fileName = "server.jar"
	case "minecraft-fabric":
		downloadURL = fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s/stable/server/jar", deployment.Version)
		fileName = "server.jar"
	case "minecraft-forge":
		// Forge requires special handling
		fileName = "server.jar"
	case "valheim":
		downloadURL = "steamcmd://install/896660"
		fileName = "valheim_dedicated_server"
	case "rust":
		downloadURL = "steamcmd://install/258550"
		fileName = "rust_dedicated_server"
	default:
		return fmt.Errorf("unsupported server type: %s", deployment.ServerType)
	}

	// Download the server software
	if err := agent.downloadFile(downloadURL, fmt.Sprintf("%s/%s", serverPath, fileName)); err != nil {
		return err
	}

	return nil
}

func (agent *NodeAgent) createServerContainer(deployment ServerDeployment, serverPath string) error {
	// Create Docker container configuration
	containerConfig := map[string]interface{}{
		"Image": getServerImage(deployment.ServerType),
		"Cmd": []string{
			"java",
			fmt.Sprintf("-Xmx%dM", deployment.Memory),
			"-jar",
			"server.jar",
			"nogui",
		},
		"WorkingDir": "/server",
		"ExposedPorts": map[string]interface{}{
			fmt.Sprintf("%d/tcp", deployment.Port): struct{}{},
		},
		"Env": buildEnvironmentVariables(deployment.Environment),
		"HostConfig": map[string]interface{}{
			"PortBindings": map[string]interface{}{
				fmt.Sprintf("%d/tcp", deployment.Port): []map[string]interface{}{
					{
						"HostPort": fmt.Sprintf("%d", deployment.Port),
					},
				},
			},
			"Binds": []string{
				fmt.Sprintf("%s:/server", serverPath),
			},
			"Memory":     deployment.Memory * 1024 * 1024, // Convert MB to bytes
			"RestartPolicy": map[string]interface{}{
				"Name": "unless-stopped",
			},
		},
	}

	// Create container using Docker API
	cmd := exec.Command("docker", "create",
		"--name", deployment.ServerID,
		"--memory", fmt.Sprintf("%dm", deployment.Memory),
		"-p", fmt.Sprintf("%d:%d", deployment.Port, deployment.Port),
		"-v", fmt.Sprintf("%s:/server", serverPath),
		"--restart", "unless-stopped",
		getServerImage(deployment.ServerType),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	return nil
}

func (agent *NodeAgent) startServerContainer(serverID string) error {
	cmd := exec.Command("docker", "start", serverID)
	return cmd.Run()
}

func (agent *NodeAgent) stopServer(data interface{}) {
	serverID, ok := data.(string)
	if !ok {
		log.Printf("Invalid server ID")
		return
	}

	log.Printf("🛑 Stopping server %s", serverID)

	cmd := exec.Command("docker", "stop", serverID)
	if err := cmd.Run(); err != nil {
		agent.sendError("Failed to stop server", err)
		return
	}

	response := Message{
		Type: "server_stopped",
		Data: map[string]interface{}{
			"server_id": serverID,
			"status":    "stopped",
			"node_id":   agent.ID,
		},
		Timestamp: time.Now(),
		NodeID:    agent.ID,
	}

	agent.sendMessage(response)
	log.Printf("✅ Server %s stopped successfully", serverID)
}

func (agent *NodeAgent) restartServer(data interface{}) {
	serverID, ok := data.(string)
	if !ok {
		log.Printf("Invalid server ID")
		return
	}

	log.Printf("🔄 Restarting server %s", serverID)

	cmd := exec.Command("docker", "restart", serverID)
	if err := cmd.Run(); err != nil {
		agent.sendError("Failed to restart server", err)
		return
	}

	response := Message{
		Type: "server_restarted",
		Data: map[string]interface{}{
			"server_id": serverID,
			"status":    "running",
			"node_id":   agent.ID,
		},
		Timestamp: time.Now(),
		NodeID:    agent.ID,
	}

	agent.sendMessage(response)
	log.Printf("✅ Server %s restarted successfully", serverID)
}

func (agent *NodeAgent) startHealthCheckServer() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		healthStatus := map[string]interface{}{
			"status":    "healthy",
			"node_id":   agent.ID,
			"timestamp": time.Now(),
			"resources": agent.Resources,
			"uptime":    getUptime(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(healthStatus)
	})

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Prometheus metrics endpoint
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "# HELP node_cpu_usage_percent CPU usage percentage\n")
		fmt.Fprintf(w, "# TYPE node_cpu_usage_percent gauge\n")
		fmt.Fprintf(w, "node_cpu_usage_percent{node_id=\"%s\"} %.2f\n", agent.ID, agent.Resources.CPU.UsagePercent)
		
		fmt.Fprintf(w, "# HELP node_memory_usage_percent Memory usage percentage\n")
		fmt.Fprintf(w, "# TYPE node_memory_usage_percent gauge\n")
		fmt.Fprintf(w, "node_memory_usage_percent{node_id=\"%s\"} %.2f\n", agent.ID, agent.Resources.Memory.UsagePercent)
	})

	log.Printf("🏥 Health check server starting on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}

func (agent *NodeAgent) sendMessage(msg Message) error {
	if agent.conn == nil {
		return fmt.Errorf("no connection to control plane")
	}
	return agent.conn.WriteJSON(msg)
}

func (agent *NodeAgent) sendError(message string, err error) {
	errorMsg := Message{
		Type: "error",
		Data: map[string]interface{}{
			"message": message,
			"error":   err.Error(),
		},
		Timestamp: time.Now(),
		NodeID:    agent.ID,
	}
	agent.sendMessage(errorMsg)
}

func (agent *NodeAgent) reconnect() {
	for {
		log.Printf("🔄 Attempting to reconnect to control plane...")
		if err := agent.connectToControlPlane(); err == nil {
			log.Printf("✅ Reconnected to control plane")
			return
		}
		time.Sleep(10 * time.Second)
	}
}

// Helper functions
func getNodeID() string {
	// Try to get node ID from environment or generate new one
	if nodeID := os.Getenv("PLAYPULSE_NODE_ID"); nodeID != "" {
		return nodeID
	}
	return uuid.New().String()
}

func getNodeName() string {
	if nodeName := os.Getenv("PLAYPULSE_NODE_NAME"); nodeName != "" {
		return nodeName
	}
	hostname, _ := os.Hostname()
	return hostname
}

func getNodeLocation() string {
	if location := os.Getenv("PLAYPULSE_NODE_LOCATION"); location != "" {
		return location
	}
	return "unknown"
}

func getControlPlaneURL() string {
	if url := os.Getenv("PLAYPULSE_CONTROL_PLANE"); url != "" {
		return url
	}
	return "localhost:8080"
}

func getNodeToken() string {
	return os.Getenv("PLAYPULSE_NODE_TOKEN")
}

func getNodeCapabilities() []string {
	capabilities := []string{"docker", "containers"}
	
	// Check for specific capabilities
	if _, err := exec.LookPath("java"); err == nil {
		capabilities = append(capabilities, "java")
	}
	
	if _, err := exec.LookPath("node"); err == nil {
		capabilities = append(capabilities, "nodejs")
	}
	
	if _, err := exec.LookPath("python3"); err == nil {
		capabilities = append(capabilities, "python")
	}
	
	return capabilities
}

func getServerImage(serverType string) string {
	images := map[string]string{
		"minecraft-paper":  "playpulse/minecraft-paper:latest",
		"minecraft-fabric": "playpulse/minecraft-fabric:latest",
		"minecraft-forge":  "playpulse/minecraft-forge:latest",
		"valheim":         "playpulse/valheim:latest",
		"rust":            "playpulse/rust:latest",
	}
	
	if image, exists := images[serverType]; exists {
		return image
	}
	return "playpulse/generic:latest"
}

func buildEnvironmentVariables(env map[string]string) []string {
	var envVars []string
	for key, value := range env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}
	return envVars
}

func (agent *NodeAgent) downloadFile(url, filepath string) error {
	// Implement file download logic
	cmd := exec.Command("wget", "-O", filepath, url)
	return cmd.Run()
}

func getUptime() time.Duration {
	info, _ := host.Info()
	return time.Duration(info.Uptime) * time.Second
}

func (agent *NodeAgent) updateServer(data interface{}) {
	// Implement server update logic
}

func (agent *NodeAgent) getServerStatus(data interface{}) {
	// Implement server status check
}

func (agent *NodeAgent) executeCommand(data interface{}) {
	// Implement command execution
}

func (agent *NodeAgent) handleFileOperation(data interface{}) {
	// Implement file operations
}

func (agent *NodeAgent) updateNode(data interface{}) {
	// Implement node update logic
}

func (agent *NodeAgent) respondHealthCheck() {
	response := Message{
		Type: "health_response",
		Data: map[string]interface{}{
			"status":    "healthy",
			"resources": agent.Resources,
			"uptime":    getUptime(),
		},
		Timestamp: time.Now(),
		NodeID:    agent.ID,
	}
	agent.sendMessage(response)
}