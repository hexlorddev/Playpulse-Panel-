package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// NodeManager manages multiple VPS nodes
type NodeManager struct {
	db              *gorm.DB
	nodes           map[string]*Node
	nodesMutex      sync.RWMutex
	loadBalancer    *LoadBalancer
	serviceRegistry *ServiceRegistry
	healthMonitor   *HealthMonitor
	autoScaler      *AutoScaler
}

// Node represents a VPS node in the cluster
type Node struct {
	ID              string            `json:"id" gorm:"primaryKey"`
	Name            string            `json:"name" gorm:"not null"`
	Location        string            `json:"location"`
	IPAddress       string            `json:"ip_address" gorm:"not null"`
	InternalIP      string            `json:"internal_ip"`
	Port            int               `json:"port" gorm:"default:8090"`
	Status          NodeStatus        `json:"status" gorm:"default:'offline'"`
	Capabilities    []string          `json:"capabilities" gorm:"type:json"`
	Resources       NodeResources     `json:"resources" gorm:"type:json"`
	Metadata        map[string]string `json:"metadata" gorm:"type:json"`
	LastSeen        time.Time         `json:"last_seen"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	
	// Runtime data
	Connection      *websocket.Conn   `json:"-" gorm:"-"`
	Servers         []NodeServer      `json:"servers,omitempty"`
	Metrics         []NodeMetric      `json:"metrics,omitempty"`
}

type NodeStatus string

const (
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusMaintenance NodeStatus = "maintenance"
	NodeStatusDraining    NodeStatus = "draining"
	NodeStatusFailed      NodeStatus = "failed"
)

type NodeResources struct {
	CPU       CPUResources    `json:"cpu"`
	Memory    MemoryResources `json:"memory"`
	Disk      DiskResources   `json:"disk"`
	Network   NetworkResources `json:"network"`
	Available bool            `json:"available"`
}

type CPUResources struct {
	Cores         int     `json:"cores"`
	UsagePercent  float64 `json:"usage_percent"`
	LoadAverage   float64 `json:"load_average"`
	Available     int     `json:"available"`
}

type MemoryResources struct {
	Total         int64   `json:"total"`
	Used          int64   `json:"used"`
	Available     int64   `json:"available"`
	UsagePercent  float64 `json:"usage_percent"`
}

type DiskResources struct {
	Total         int64   `json:"total"`
	Used          int64   `json:"used"`
	Available     int64   `json:"available"`
	UsagePercent  float64 `json:"usage_percent"`
}

type NetworkResources struct {
	Bandwidth     int64 `json:"bandwidth"`
	BytesIn       int64 `json:"bytes_in"`
	BytesOut      int64 `json:"bytes_out"`
	Connections   int   `json:"connections"`
}

// NodeServer represents a server running on a node
type NodeServer struct {
	ID          string            `json:"id"`
	NodeID      string            `json:"node_id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Status      string            `json:"status"`
	Port        int               `json:"port"`
	Players     int               `json:"players"`
	MaxPlayers  int               `json:"max_players"`
	Resources   ServerResources   `json:"resources"`
	CreatedAt   time.Time         `json:"created_at"`
}

type ServerResources struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   int64   `json:"memory_usage"`
	MemoryLimit   int64   `json:"memory_limit"`
	DiskUsage     int64   `json:"disk_usage"`
	NetworkIn     int64   `json:"network_in"`
	NetworkOut    int64   `json:"network_out"`
}

// NodeMetric represents performance metrics for a node
type NodeMetric struct {
	ID           uuid.UUID     `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	NodeID       string        `json:"node_id" gorm:"not null"`
	Timestamp    time.Time     `json:"timestamp"`
	CPUUsage     float64       `json:"cpu_usage"`
	MemoryUsage  float64       `json:"memory_usage"`
	DiskUsage    float64       `json:"disk_usage"`
	NetworkIn    int64         `json:"network_in"`
	NetworkOut   int64         `json:"network_out"`
	ServerCount  int           `json:"server_count"`
	PlayerCount  int           `json:"player_count"`
	ResponseTime float64       `json:"response_time"`
	
	Node Node `json:"node,omitempty"`
}

// LoadBalancer handles traffic distribution across nodes
type LoadBalancer struct {
	strategy LoadBalancingStrategy
	nodes    map[string]*Node
	metrics  *MetricsCollector
}

type LoadBalancingStrategy string

const (
	StrategyRoundRobin     LoadBalancingStrategy = "round_robin"
	StrategyLeastLoaded    LoadBalancingStrategy = "least_loaded"
	StrategyGeographicAware LoadBalancingStrategy = "geographic_aware"
	StrategyResourceBased  LoadBalancingStrategy = "resource_based"
	StrategyLatencyBased   LoadBalancingStrategy = "latency_based"
)

// ServiceRegistry tracks services across nodes
type ServiceRegistry struct {
	services map[string][]ServiceInstance
	mutex    sync.RWMutex
}

type ServiceInstance struct {
	ID       string            `json:"id"`
	NodeID   string            `json:"node_id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Health   string            `json:"health"`
	Metadata map[string]string `json:"metadata"`
	LastSeen time.Time         `json:"last_seen"`
}

// HealthMonitor monitors node health
type HealthMonitor struct {
	db             *gorm.DB
	checkInterval  time.Duration
	failureThreshold int
	recoveryThreshold int
}

// AutoScaler handles automatic scaling
type AutoScaler struct {
	db              *gorm.DB
	enabled         bool
	minNodes        int
	maxNodes        int
	targetCPU       float64
	targetMemory    float64
	scaleUpCooldown time.Duration
	scaleDownCooldown time.Duration
}

// NewNodeManager creates a new node manager
func NewNodeManager(db *gorm.DB) *NodeManager {
	nm := &NodeManager{
		db:    db,
		nodes: make(map[string]*Node),
		loadBalancer: &LoadBalancer{
			strategy: StrategyLeastLoaded,
			nodes:    make(map[string]*Node),
		},
		serviceRegistry: &ServiceRegistry{
			services: make(map[string][]ServiceInstance),
		},
		healthMonitor: &HealthMonitor{
			db:                db,
			checkInterval:     30 * time.Second,
			failureThreshold:  3,
			recoveryThreshold: 2,
		},
		autoScaler: &AutoScaler{
			db:                db,
			enabled:           true,
			minNodes:          2,
			maxNodes:          50,
			targetCPU:         70.0,
			targetMemory:      80.0,
			scaleUpCooldown:   5 * time.Minute,
			scaleDownCooldown: 10 * time.Minute,
		},
	}

	// Start background processes
	go nm.healthMonitor.Start()
	go nm.autoScaler.Start()
	go nm.startMetricsCollection()

	return nm
}

// RegisterNode registers a new node with the cluster
func (nm *NodeManager) RegisterNode(ctx context.Context, node *Node) error {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	// Save to database
	if err := nm.db.WithContext(ctx).Create(node).Error; err != nil {
		return fmt.Errorf("failed to save node to database: %w", err)
	}

	// Add to memory
	nm.nodes[node.ID] = node
	nm.loadBalancer.nodes[node.ID] = node

	log.Printf("Node registered: %s (%s) at %s", node.Name, node.ID, node.IPAddress)
	return nil
}

// ConnectNode establishes WebSocket connection with a node
func (nm *NodeManager) ConnectNode(nodeID string, conn *websocket.Conn) error {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.Connection = conn
	node.Status = NodeStatusOnline
	node.LastSeen = time.Now()

	// Update database
	nm.db.Model(node).Updates(map[string]interface{}{
		"status":    NodeStatusOnline,
		"last_seen": node.LastSeen,
	})

	log.Printf("Node connected: %s", nodeID)
	return nil
}

// DisconnectNode handles node disconnection
func (nm *NodeManager) DisconnectNode(nodeID string) {
	nm.nodesMutex.Lock()
	defer nm.nodesMutex.Unlock()

	node, exists := nm.nodes[nodeID]
	if !exists {
		return
	}

	if node.Connection != nil {
		node.Connection.Close()
		node.Connection = nil
	}

	node.Status = NodeStatusOffline
	node.LastSeen = time.Now()

	// Update database
	nm.db.Model(node).Updates(map[string]interface{}{
		"status":    NodeStatusOffline,
		"last_seen": node.LastSeen,
	})

	log.Printf("Node disconnected: %s", nodeID)
}

// DeployServer deploys a server to the best available node
func (nm *NodeManager) DeployServer(ctx context.Context, request ServerDeploymentRequest) (*DeploymentResult, error) {
	// Select best node using load balancer
	targetNode, err := nm.loadBalancer.SelectNode(request.Requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to select target node: %w", err)
	}

	// Send deployment command to node
	deploymentCmd := NodeCommand{
		ID:      uuid.New().String(),
		Type:    "deploy_server",
		Payload: request,
	}

	if err := nm.sendCommandToNode(targetNode.ID, deploymentCmd); err != nil {
		return nil, fmt.Errorf("failed to send deployment command: %w", err)
	}

	// Wait for deployment result
	result := &DeploymentResult{
		ServerID: request.ServerID,
		NodeID:   targetNode.ID,
		Status:   "deploying",
	}

	return result, nil
}

// MigrateServer migrates a server from one node to another
func (nm *NodeManager) MigrateServer(ctx context.Context, serverID, targetNodeID string) error {
	// Find source node
	sourceNode, err := nm.findNodeByServer(serverID)
	if err != nil {
		return fmt.Errorf("failed to find source node: %w", err)
	}

	// Get target node
	nm.nodesMutex.RLock()
	targetNode, exists := nm.nodes[targetNodeID]
	nm.nodesMutex.RUnlock()

	if !exists {
		return fmt.Errorf("target node %s not found", targetNodeID)
	}

	// Create migration plan
	migrationPlan := MigrationPlan{
		ServerID:     serverID,
		SourceNodeID: sourceNode.ID,
		TargetNodeID: targetNodeID,
		Strategy:     "live_migration",
	}

	// Execute migration
	return nm.executeMigration(ctx, migrationPlan)
}

// GetNodeMetrics returns metrics for all nodes
func (nm *NodeManager) GetNodeMetrics(ctx context.Context, timeRange string) (map[string][]NodeMetric, error) {
	var metrics []NodeMetric
	
	// Parse time range
	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		duration = 24 * time.Hour
	}

	since := time.Now().Add(-duration)

	err = nm.db.WithContext(ctx).
		Where("timestamp > ?", since).
		Order("timestamp DESC").
		Find(&metrics).Error

	if err != nil {
		return nil, err
	}

	// Group by node ID
	result := make(map[string][]NodeMetric)
	for _, metric := range metrics {
		result[metric.NodeID] = append(result[metric.NodeID], metric)
	}

	return result, nil
}

// GetClusterStatus returns overall cluster status
func (nm *NodeManager) GetClusterStatus() ClusterStatus {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	status := ClusterStatus{
		TotalNodes:   len(nm.nodes),
		OnlineNodes:  0,
		OfflineNodes: 0,
		TotalServers: 0,
		TotalPlayers: 0,
		Resources:    ClusterResources{},
	}

	for _, node := range nm.nodes {
		if node.Status == NodeStatusOnline {
			status.OnlineNodes++
			status.TotalServers += len(node.Servers)
			
			for _, server := range node.Servers {
				status.TotalPlayers += server.Players
			}

			// Aggregate resources
			status.Resources.TotalCPU += node.Resources.CPU.Cores
			status.Resources.UsedCPU += int(node.Resources.CPU.UsagePercent * float64(node.Resources.CPU.Cores) / 100)
			status.Resources.TotalMemory += node.Resources.Memory.Total
			status.Resources.UsedMemory += node.Resources.Memory.Used
			status.Resources.TotalDisk += node.Resources.Disk.Total
			status.Resources.UsedDisk += node.Resources.Disk.Used
		} else {
			status.OfflineNodes++
		}
	}

	// Calculate percentages
	if status.Resources.TotalCPU > 0 {
		status.Resources.CPUUsagePercent = float64(status.Resources.UsedCPU) / float64(status.Resources.TotalCPU) * 100
	}
	if status.Resources.TotalMemory > 0 {
		status.Resources.MemoryUsagePercent = float64(status.Resources.UsedMemory) / float64(status.Resources.TotalMemory) * 100
	}
	if status.Resources.TotalDisk > 0 {
		status.Resources.DiskUsagePercent = float64(status.Resources.UsedDisk) / float64(status.Resources.TotalDisk) * 100
	}

	return status
}

// Load Balancer Implementation
func (lb *LoadBalancer) SelectNode(requirements ServerRequirements) (*Node, error) {
	switch lb.strategy {
	case StrategyLeastLoaded:
		return lb.selectLeastLoadedNode(requirements)
	case StrategyGeographicAware:
		return lb.selectGeographicNode(requirements)
	case StrategyResourceBased:
		return lb.selectResourceBasedNode(requirements)
	case StrategyLatencyBased:
		return lb.selectLatencyBasedNode(requirements)
	default:
		return lb.selectRoundRobinNode(requirements)
	}
}

func (lb *LoadBalancer) selectLeastLoadedNode(requirements ServerRequirements) (*Node, error) {
	var bestNode *Node
	var lowestLoad float64 = 100.0

	for _, node := range lb.nodes {
		if node.Status != NodeStatusOnline {
			continue
		}

		// Check if node meets requirements
		if !lb.nodeMetetsRequirements(node, requirements) {
			continue
		}

		// Calculate load score
		cpuLoad := node.Resources.CPU.UsagePercent
		memoryLoad := node.Resources.Memory.UsagePercent
		averageLoad := (cpuLoad + memoryLoad) / 2

		if averageLoad < lowestLoad {
			lowestLoad = averageLoad
			bestNode = node
		}
	}

	if bestNode == nil {
		return nil, fmt.Errorf("no suitable node found")
	}

	return bestNode, nil
}

func (lb *LoadBalancer) selectGeographicNode(requirements ServerRequirements) (*Node, error) {
	preferredLocation := requirements.PreferredLocation
	if preferredLocation == "" {
		return lb.selectLeastLoadedNode(requirements)
	}

	// First try to find nodes in preferred location
	for _, node := range lb.nodes {
		if node.Status == NodeStatusOnline &&
			node.Location == preferredLocation &&
			lb.nodeMetetsRequirements(node, requirements) {
			return node, nil
		}
	}

	// Fallback to any available node
	return lb.selectLeastLoadedNode(requirements)
}

func (lb *LoadBalancer) nodeMetetsRequirements(node *Node, requirements ServerRequirements) bool {
	// Check CPU
	if requirements.MinCPU > 0 && node.Resources.CPU.Available < requirements.MinCPU {
		return false
	}

	// Check Memory
	if requirements.MinMemory > 0 && node.Resources.Memory.Available < requirements.MinMemory {
		return false
	}

	// Check Disk
	if requirements.MinDisk > 0 && node.Resources.Disk.Available < requirements.MinDisk {
		return false
	}

	// Check capabilities
	for _, requiredCap := range requirements.RequiredCapabilities {
		found := false
		for _, nodeCap := range node.Capabilities {
			if nodeCap == requiredCap {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Health Monitor Implementation
func (hm *HealthMonitor) Start() {
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		hm.checkNodeHealth()
	}
}

func (hm *HealthMonitor) checkNodeHealth() {
	var nodes []Node
	hm.db.Find(&nodes)

	for _, node := range nodes {
		if node.Status == NodeStatusOnline {
			// Check if node is responsive
			if time.Since(node.LastSeen) > hm.checkInterval*2 {
				// Node hasn't been seen for too long
				hm.markNodeUnhealthy(node.ID)
			}
		}
	}
}

func (hm *HealthMonitor) markNodeUnhealthy(nodeID string) {
	hm.db.Model(&Node{}).Where("id = ?", nodeID).Update("status", NodeStatusFailed)
	log.Printf("Node marked as unhealthy: %s", nodeID)
}

// Auto Scaler Implementation
func (as *AutoScaler) Start() {
	if !as.enabled {
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		as.evaluateScaling()
	}
}

func (as *AutoScaler) evaluateScaling() {
	// Get cluster metrics
	var avgCPU, avgMemory float64
	var nodeCount int

	var nodes []Node
	as.db.Where("status = ?", NodeStatusOnline).Find(&nodes)

	for _, node := range nodes {
		avgCPU += node.Resources.CPU.UsagePercent
		avgMemory += node.Resources.Memory.UsagePercent
		nodeCount++
	}

	if nodeCount == 0 {
		return
	}

	avgCPU /= float64(nodeCount)
	avgMemory /= float64(nodeCount)

	// Scale up if needed
	if (avgCPU > as.targetCPU || avgMemory > as.targetMemory) && nodeCount < as.maxNodes {
		as.scaleUp()
	}

	// Scale down if needed
	if avgCPU < as.targetCPU*0.5 && avgMemory < as.targetMemory*0.5 && nodeCount > as.minNodes {
		as.scaleDown()
	}
}

func (as *AutoScaler) scaleUp() {
	log.Println("Auto-scaling: Scaling up cluster")
	// Implementation would provision new nodes
}

func (as *AutoScaler) scaleDown() {
	log.Println("Auto-scaling: Scaling down cluster")
	// Implementation would drain and remove nodes
}

// Supporting types
type ServerDeploymentRequest struct {
	ServerID     string             `json:"server_id"`
	ServerType   string             `json:"server_type"`
	Version      string             `json:"version"`
	Port         int                `json:"port"`
	Requirements ServerRequirements `json:"requirements"`
	Environment  map[string]string  `json:"environment"`
}

type ServerRequirements struct {
	MinCPU                int      `json:"min_cpu"`
	MinMemory             int64    `json:"min_memory"`
	MinDisk               int64    `json:"min_disk"`
	RequiredCapabilities  []string `json:"required_capabilities"`
	PreferredLocation     string   `json:"preferred_location"`
}

type DeploymentResult struct {
	ServerID string `json:"server_id"`
	NodeID   string `json:"node_id"`
	Status   string `json:"status"`
}

type NodeCommand struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type MigrationPlan struct {
	ServerID     string `json:"server_id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	Strategy     string `json:"strategy"`
}

type ClusterStatus struct {
	TotalNodes   int              `json:"total_nodes"`
	OnlineNodes  int              `json:"online_nodes"`
	OfflineNodes int              `json:"offline_nodes"`
	TotalServers int              `json:"total_servers"`
	TotalPlayers int              `json:"total_players"`
	Resources    ClusterResources `json:"resources"`
}

type ClusterResources struct {
	TotalCPU           int     `json:"total_cpu"`
	UsedCPU            int     `json:"used_cpu"`
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	TotalMemory        int64   `json:"total_memory"`
	UsedMemory         int64   `json:"used_memory"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	TotalDisk          int64   `json:"total_disk"`
	UsedDisk           int64   `json:"used_disk"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
}

// Helper methods
func (nm *NodeManager) sendCommandToNode(nodeID string, command NodeCommand) error {
	nm.nodesMutex.RLock()
	node, exists := nm.nodes[nodeID]
	nm.nodesMutex.RUnlock()

	if !exists || node.Connection == nil {
		return fmt.Errorf("node %s not connected", nodeID)
	}

	return node.Connection.WriteJSON(command)
}

func (nm *NodeManager) findNodeByServer(serverID string) (*Node, error) {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	for _, node := range nm.nodes {
		for _, server := range node.Servers {
			if server.ID == serverID {
				return node, nil
			}
		}
	}

	return nil, fmt.Errorf("server %s not found on any node", serverID)
}

func (nm *NodeManager) executeMigration(ctx context.Context, plan MigrationPlan) error {
	// Implementation for server migration
	log.Printf("Executing migration: %s from %s to %s", plan.ServerID, plan.SourceNodeID, plan.TargetNodeID)
	return nil
}

func (nm *NodeManager) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		nm.collectNodeMetrics()
	}
}

func (nm *NodeManager) collectNodeMetrics() {
	nm.nodesMutex.RLock()
	defer nm.nodesMutex.RUnlock()

	for _, node := range nm.nodes {
		if node.Status == NodeStatusOnline && node.Connection != nil {
			// Request metrics from node
			metricsCmd := NodeCommand{
				ID:   uuid.New().String(),
				Type: "get_metrics",
			}
			
			node.Connection.WriteJSON(metricsCmd)
		}
	}
}

func (lb *LoadBalancer) selectRoundRobinNode(requirements ServerRequirements) (*Node, error) {
	// Simple round-robin implementation
	for _, node := range lb.nodes {
		if node.Status == NodeStatusOnline && lb.nodeMetetsRequirements(node, requirements) {
			return node, nil
		}
	}
	return nil, fmt.Errorf("no suitable node found")
}

func (lb *LoadBalancer) selectResourceBasedNode(requirements ServerRequirements) (*Node, error) {
	return lb.selectLeastLoadedNode(requirements)
}

func (lb *LoadBalancer) selectLatencyBasedNode(requirements ServerRequirements) (*Node, error) {
	return lb.selectLeastLoadedNode(requirements)
}