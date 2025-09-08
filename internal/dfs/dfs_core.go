package dfs

import (
	"fmt"
	"sync"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/distributor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/p2p"
	"github.com/jaywantadh/DisktroByte/internal/storage"
	"github.com/sirupsen/logrus"
)

// DFSConfig holds configuration for the distributed file system
type DFSConfig struct {
	MinReplicaCount      int           `json:"min_replica_count"`      // Minimum replicas per chunk
	MaxReplicaCount      int           `json:"max_replica_count"`      // Maximum replicas per chunk
	DefaultReplicaCount  int           `json:"default_replica_count"`  // Default replicas per chunk
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`     // How often to send heartbeats
	NodeTimeoutDuration  time.Duration `json:"node_timeout_duration"`  // When to consider a node dead
	RebalanceInterval    time.Duration `json:"rebalance_interval"`     // How often to rebalance replicas
	ChunkSize            int64         `json:"chunk_size"`             // Size of each chunk in bytes
	CompressionEnabled   bool          `json:"compression_enabled"`    // Enable compression
	EncryptionEnabled    bool          `json:"encryption_enabled"`     // Enable encryption
	DeduplicationEnabled bool          `json:"deduplication_enabled"`  // Enable deduplication
}

// DefaultDFSConfig returns a default configuration
func DefaultDFSConfig() *DFSConfig {
	return &DFSConfig{
		MinReplicaCount:      2,
		MaxReplicaCount:      5,
		DefaultReplicaCount:  3,
		HeartbeatInterval:    15 * time.Second,
		NodeTimeoutDuration:  60 * time.Second,
		RebalanceInterval:    5 * time.Minute,
		ChunkSize:            1024 * 1024, // 1MB chunks
		CompressionEnabled:   true,
		EncryptionEnabled:    true,
		DeduplicationEnabled: true,
	}
}

// NodeHealth represents the health status of a node
type NodeHealth struct {
	NodeID            string        `json:"node_id"`
	Status            string        `json:"status"`                // "healthy", "degraded", "failed"
	LastHeartbeat     time.Time     `json:"last_heartbeat"`
	ResponseTime      time.Duration `json:"response_time"`
	FailureCount      int           `json:"failure_count"`
	StorageUtilization float64      `json:"storage_utilization"`   // 0.0 to 1.0
	StorageCapacity   int64         `json:"storage_capacity"`      // bytes
	StorageUsed       int64         `json:"storage_used"`          // bytes
	ChunkCount        int           `json:"chunk_count"`
	NetworkLatency    time.Duration `json:"network_latency"`
}

// ReplicaInfo represents information about chunk replicas
type ReplicaInfo struct {
	ChunkID       string              `json:"chunk_id"`
	FileID        string              `json:"file_id"`
	CurrentReplicas []string          `json:"current_replicas"`      // Node IDs that have the chunk
	DesiredReplicas int               `json:"desired_replicas"`      // How many replicas we want
	Health         map[string]string  `json:"health"`                // Health status per replica
	LastVerified   time.Time          `json:"last_verified"`
}

// DFSCore manages the distributed file system
type DFSCore struct {
	config       *DFSConfig
	network      *p2p.Network
	distributor  *distributor.Distributor
	storage      storage.Storage
	metaStore    *metadata.MetadataStore
	
	// Health monitoring
	nodeHealth   map[string]*NodeHealth
	healthMu     sync.RWMutex
	
	// Replica management
	replicaInfo  map[string]*ReplicaInfo
	replicaMu    sync.RWMutex
	
	// Background tasks
	heartbeatTicker   *time.Ticker
	rebalanceTicker   *time.Ticker
	stopChan          chan bool
	
	// Advanced storage optimization
	OptimizedStorage  *OptimizedStorage
	
	logger *logrus.Logger
}

// NewDFSCore creates a new distributed file system core
func NewDFSCore(config *DFSConfig, network *p2p.Network, distributor *distributor.Distributor, storage storage.Storage, metaStore *metadata.MetadataStore) *DFSCore {
	if config == nil {
		config = DefaultDFSConfig()
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &DFSCore{
		config:      config,
		network:     network,
		distributor: distributor,
		storage:     storage,
		metaStore:   metaStore,
		nodeHealth:  make(map[string]*NodeHealth),
		replicaInfo: make(map[string]*ReplicaInfo),
		stopChan:    make(chan bool),
		logger:      logger,
	}
}

// Start initializes and starts the DFS core system
func (dfs *DFSCore) Start() error {
	dfs.logger.Info("🚀 Starting DFS Core System")
	
	// Initialize node health for all known peers
	dfs.initializeNodeHealth()
	
	// Start heartbeat monitoring
	dfs.heartbeatTicker = time.NewTicker(dfs.config.HeartbeatInterval)
	go dfs.heartbeatMonitor()
	
	// Start replica rebalancing
	dfs.rebalanceTicker = time.NewTicker(dfs.config.RebalanceInterval)
	go dfs.rebalanceMonitor()
	
	// Start health monitoring
	go dfs.healthMonitor()
	
	// Start replica verification
	go dfs.replicaVerificationMonitor()
	
	// Initialize optimized storage if storage is available
	if dfs.storage != nil {
		optimizedStorage, err := NewOptimizedStorage("./optimized_storage")
		if err != nil {
			dfs.logger.Warnf("⚠️ Failed to initialize optimized storage: %v", err)
		} else {
			dfs.OptimizedStorage = optimizedStorage
			dfs.logger.Info("💾 Optimized Storage System initialized")
		}
	}
	
	dfs.logger.Info("✅ DFS Core System started successfully")
	return nil
}

// Stop shuts down the DFS core system
func (dfs *DFSCore) Stop() {
	dfs.logger.Info("🛑 Stopping DFS Core System")
	
	if dfs.heartbeatTicker != nil {
		dfs.heartbeatTicker.Stop()
	}
	if dfs.rebalanceTicker != nil {
		dfs.rebalanceTicker.Stop()
	}
	
	close(dfs.stopChan)
	
	// Close optimized storage if available
	if dfs.OptimizedStorage != nil {
		if err := dfs.OptimizedStorage.Close(); err != nil {
			dfs.logger.Errorf("❌ Failed to close optimized storage: %v", err)
		}
	}
	
	dfs.logger.Info("✅ DFS Core System stopped")
}

// initializeNodeHealth initializes health tracking for all nodes
func (dfs *DFSCore) initializeNodeHealth() {
	dfs.healthMu.Lock()
	defer dfs.healthMu.Unlock()
	
	// Initialize local node health
	dfs.nodeHealth[dfs.network.LocalNode.ID] = &NodeHealth{
		NodeID:            dfs.network.LocalNode.ID,
		Status:            "healthy",
		LastHeartbeat:     time.Now(),
		ResponseTime:      0,
		FailureCount:      0,
		StorageUtilization: 0.0,
		StorageCapacity:   1024 * 1024 * 1024 * 100, // 100GB default
		StorageUsed:       0,
		ChunkCount:        len(dfs.network.LocalNode.Chunks),
		NetworkLatency:    0,
	}
	
	// Initialize peer node health
	peers := dfs.network.GetPeers()
	for _, peer := range peers {
		dfs.nodeHealth[peer.ID] = &NodeHealth{
			NodeID:            peer.ID,
			Status:            "unknown",
			LastHeartbeat:     peer.LastSeen,
			ResponseTime:      0,
			FailureCount:      0,
			StorageUtilization: 0.0,
			StorageCapacity:   1024 * 1024 * 1024 * 100, // 100GB default
			StorageUsed:       0,
			ChunkCount:        len(peer.Chunks),
			NetworkLatency:    0,
		}
	}
	
	dfs.logger.Infof("🔍 Initialized health monitoring for %d nodes", len(dfs.nodeHealth))
}

// heartbeatMonitor monitors node heartbeats
func (dfs *DFSCore) heartbeatMonitor() {
	for {
		select {
		case <-dfs.heartbeatTicker.C:
			dfs.performHeartbeatCheck()
		case <-dfs.stopChan:
			return
		}
	}
}

// performHeartbeatCheck performs heartbeat checks on all nodes
func (dfs *DFSCore) performHeartbeatCheck() {
	peers := dfs.network.GetPeers()
	
	for _, peer := range peers {
		go dfs.checkNodeHeartbeat(peer)
	}
}

// checkNodeHeartbeat checks the heartbeat of a specific node
func (dfs *DFSCore) checkNodeHeartbeat(node *p2p.Node) {
	start := time.Now()
	
	// TODO: Send heartbeat message to node
	// msg := &p2p.NetworkMessage{
	//	Type:      "heartbeat",
	//	From:      dfs.network.LocalNode.ID,
	//	To:        node.ID,
	//	Data:      map[string]interface{}{"timestamp": time.Now()},
	//	Timestamp: time.Now(),
	// }
	
	// TODO: Implement actual network heartbeat
	// For now, simulate based on last seen time
	responseTime := time.Since(start)
	isHealthy := time.Since(node.LastSeen) < dfs.config.NodeTimeoutDuration
	
	dfs.updateNodeHealth(node.ID, isHealthy, responseTime)
}

// updateNodeHealth updates the health status of a node
func (dfs *DFSCore) updateNodeHealth(nodeID string, isHealthy bool, responseTime time.Duration) {
	dfs.healthMu.Lock()
	defer dfs.healthMu.Unlock()
	
	health, exists := dfs.nodeHealth[nodeID]
	if !exists {
		health = &NodeHealth{
			NodeID:        nodeID,
			Status:        "unknown",
			LastHeartbeat: time.Time{},
			FailureCount:  0,
		}
		dfs.nodeHealth[nodeID] = health
	}
	
	health.ResponseTime = responseTime
	health.NetworkLatency = responseTime
	
	if isHealthy {
		health.LastHeartbeat = time.Now()
		health.FailureCount = 0
		
		if health.Status != "healthy" {
			dfs.logger.Infof("✅ Node %s is now healthy", nodeID)
			health.Status = "healthy"
		}
	} else {
		health.FailureCount++
		
		if health.FailureCount >= 3 {
			if health.Status != "failed" {
				dfs.logger.Warnf("❌ Node %s marked as failed (failure count: %d)", nodeID, health.FailureCount)
				health.Status = "failed"
				
				// Trigger replica recovery for this node
				go dfs.handleNodeFailure(nodeID)
			}
		} else if health.FailureCount >= 1 {
			health.Status = "degraded"
		}
	}
}

// handleNodeFailure handles node failure by redistributing its chunks
func (dfs *DFSCore) handleNodeFailure(nodeID string) {
	dfs.logger.Warnf("🚨 Handling failure of node %s", nodeID)
	
	// Find all chunks that were stored on the failed node
	failedChunks := dfs.findChunksOnNode(nodeID)
	
	for _, chunkID := range failedChunks {
		dfs.logger.Infof("🔄 Recovering chunk %s from failed node %s", chunkID, nodeID)
		go dfs.recoverChunk(chunkID, nodeID)
	}
}

// findChunksOnNode finds all chunks stored on a specific node
func (dfs *DFSCore) findChunksOnNode(nodeID string) []string {
	dfs.replicaMu.RLock()
	defer dfs.replicaMu.RUnlock()
	
	var chunks []string
	for chunkID, replica := range dfs.replicaInfo {
		for _, nodeWithChunk := range replica.CurrentReplicas {
			if nodeWithChunk == nodeID {
				chunks = append(chunks, chunkID)
				break
			}
		}
	}
	
	return chunks
}

// recoverChunk recovers a chunk by creating new replicas
func (dfs *DFSCore) recoverChunk(chunkID, failedNodeID string) error {
	dfs.replicaMu.Lock()
	replica, exists := dfs.replicaInfo[chunkID]
	if !exists {
		dfs.replicaMu.Unlock()
		return fmt.Errorf("chunk %s not found in replica info", chunkID)
	}
	
	// Remove failed node from current replicas
	var updatedReplicas []string
	for _, nodeID := range replica.CurrentReplicas {
		if nodeID != failedNodeID {
			updatedReplicas = append(updatedReplicas, nodeID)
		}
	}
	replica.CurrentReplicas = updatedReplicas
	
	// Check if we need more replicas
	replicasNeeded := replica.DesiredReplicas - len(replica.CurrentReplicas)
	dfs.replicaMu.Unlock()
	
	if replicasNeeded > 0 {
		return dfs.createAdditionalReplicas(chunkID, replicasNeeded)
	}
	
	return nil
}

// createAdditionalReplicas creates additional replicas for a chunk
func (dfs *DFSCore) createAdditionalReplicas(chunkID string, count int) error {
	dfs.logger.Infof("📋 Creating %d additional replicas for chunk %s", count, chunkID)
	
	// Find healthy nodes for new replicas
	healthyNodes := dfs.getHealthyNodes()
	
	// Remove nodes that already have this chunk
	availableNodes := dfs.filterNodesWithoutChunk(healthyNodes, chunkID)
	
	if len(availableNodes) < count {
		dfs.logger.Warnf("⚠️ Only %d nodes available for %d needed replicas of chunk %s", 
			len(availableNodes), count, chunkID)
	}
	
	// Create replicas on available nodes
	created := 0
	for _, node := range availableNodes {
		if created >= count {
			break
		}
		
		if err := dfs.createReplicaOnNode(chunkID, node.ID); err != nil {
			dfs.logger.Errorf("❌ Failed to create replica of chunk %s on node %s: %v", 
				chunkID, node.ID, err)
		} else {
			created++
			dfs.logger.Infof("✅ Created replica of chunk %s on node %s", chunkID, node.ID)
		}
	}
	
	return nil
}

// getHealthyNodes returns all nodes with healthy status
func (dfs *DFSCore) getHealthyNodes() []*p2p.Node {
	dfs.healthMu.RLock()
	defer dfs.healthMu.RUnlock()
	
	var healthyNodes []*p2p.Node
	
	// Check local node
	if health, exists := dfs.nodeHealth[dfs.network.LocalNode.ID]; exists && health.Status == "healthy" {
		healthyNodes = append(healthyNodes, dfs.network.LocalNode)
	}
	
	// Check peer nodes
	peers := dfs.network.GetPeers()
	for _, peer := range peers {
		if health, exists := dfs.nodeHealth[peer.ID]; exists && health.Status == "healthy" {
			healthyNodes = append(healthyNodes, peer)
		}
	}
	
	return healthyNodes
}

// filterNodesWithoutChunk filters out nodes that already have the chunk
func (dfs *DFSCore) filterNodesWithoutChunk(nodes []*p2p.Node, chunkID string) []*p2p.Node {
	dfs.replicaMu.RLock()
	replica, exists := dfs.replicaInfo[chunkID]
	dfs.replicaMu.RUnlock()
	
	if !exists {
		return nodes
	}
	
	var availableNodes []*p2p.Node
	for _, node := range nodes {
		hasChunk := false
		for _, nodeWithChunk := range replica.CurrentReplicas {
			if node.ID == nodeWithChunk {
				hasChunk = true
				break
			}
		}
		if !hasChunk {
			availableNodes = append(availableNodes, node)
		}
	}
	
	return availableNodes
}

// createReplicaOnNode creates a replica of a chunk on a specific node
func (dfs *DFSCore) createReplicaOnNode(chunkID, nodeID string) error {
	// TODO: Implement actual chunk transfer to the node
	// For now, just update the replica info
	
	dfs.replicaMu.Lock()
	defer dfs.replicaMu.Unlock()
	
	if replica, exists := dfs.replicaInfo[chunkID]; exists {
		replica.CurrentReplicas = append(replica.CurrentReplicas, nodeID)
		replica.Health[nodeID] = "healthy"
		
		// Add chunk to node's chunk list
		dfs.network.AddChunkToNode(nodeID, chunkID)
	}
	
	return nil
}

// GetSystemStats returns comprehensive DFS system statistics
func (dfs *DFSCore) GetSystemStats() map[string]interface{} {
	dfs.healthMu.RLock()
	dfs.replicaMu.RLock()
	defer dfs.healthMu.RUnlock()
	defer dfs.replicaMu.RUnlock()
	
	stats := map[string]interface{}{
		"total_nodes":       len(dfs.nodeHealth),
		"healthy_nodes":     0,
		"degraded_nodes":    0,
		"failed_nodes":      0,
		"total_chunks":      len(dfs.replicaInfo),
		"total_replicas":    0,
		"under_replicated":  0,
		"over_replicated":   0,
		"storage_capacity":  int64(0),
		"storage_used":      int64(0),
		"storage_efficiency": 0.0,
	}
	
	// Calculate node statistics
	for _, health := range dfs.nodeHealth {
		switch health.Status {
		case "healthy":
			stats["healthy_nodes"] = stats["healthy_nodes"].(int) + 1
		case "degraded":
			stats["degraded_nodes"] = stats["degraded_nodes"].(int) + 1
		case "failed":
			stats["failed_nodes"] = stats["failed_nodes"].(int) + 1
		}
		
		stats["storage_capacity"] = stats["storage_capacity"].(int64) + health.StorageCapacity
		stats["storage_used"] = stats["storage_used"].(int64) + health.StorageUsed
	}
	
	// Calculate replica statistics
	for _, replica := range dfs.replicaInfo {
		replicaCount := len(replica.CurrentReplicas)
		stats["total_replicas"] = stats["total_replicas"].(int) + replicaCount
		
		if replicaCount < replica.DesiredReplicas {
			stats["under_replicated"] = stats["under_replicated"].(int) + 1
		} else if replicaCount > replica.DesiredReplicas {
			stats["over_replicated"] = stats["over_replicated"].(int) + 1
		}
	}
	
	// Calculate storage efficiency
	if stats["storage_capacity"].(int64) > 0 {
		efficiency := float64(stats["storage_used"].(int64)) / float64(stats["storage_capacity"].(int64))
		stats["storage_efficiency"] = efficiency
	}
	
	return stats
}

// rebalanceMonitor monitors and rebalances replicas across nodes
func (dfs *DFSCore) rebalanceMonitor() {
	for {
		select {
		case <-dfs.rebalanceTicker.C:
			dfs.performRebalancing()
		case <-dfs.stopChan:
			return
		}
	}
}

// performRebalancing checks and rebalances chunk replicas
func (dfs *DFSCore) performRebalancing() {
	dfs.logger.Info("🔄 Performing replica rebalancing...")
	
	dfs.replicaMu.RLock()
	rebalanceList := make([]*ReplicaInfo, 0)
	for _, replica := range dfs.replicaInfo {
		currentCount := len(replica.CurrentReplicas)
		if currentCount < replica.DesiredReplicas {
			rebalanceList = append(rebalanceList, replica)
		}
	}
	dfs.replicaMu.RUnlock()
	
	for _, replica := range rebalanceList {
		replicasNeeded := replica.DesiredReplicas - len(replica.CurrentReplicas)
		dfs.logger.Infof("⚖️ Chunk %s needs %d more replicas", replica.ChunkID, replicasNeeded)
		go dfs.createAdditionalReplicas(replica.ChunkID, replicasNeeded)
	}
	
	if len(rebalanceList) > 0 {
		dfs.logger.Infof("✅ Rebalancing completed for %d chunks", len(rebalanceList))
	} else {
		dfs.logger.Debug("✨ All chunks are properly replicated")
	}
}

// healthMonitor continuously monitors overall system health
func (dfs *DFSCore) healthMonitor() {
	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()
	
	for {
		select {
		case <-healthTicker.C:
			dfs.performHealthCheck()
		case <-dfs.stopChan:
			return
		}
	}
}

// performHealthCheck performs comprehensive system health check
func (dfs *DFSCore) performHealthCheck() {
	stats := dfs.GetSystemStats()
	
	healthyNodes := stats["healthy_nodes"].(int)
	totalNodes := stats["total_nodes"].(int)
	underReplicated := stats["under_replicated"].(int)
	
	if totalNodes > 0 {
		healthPercentage := float64(healthyNodes) / float64(totalNodes) * 100
		
		if healthPercentage < 70 {
			dfs.logger.Warnf("⚠️ System health is degraded: %.1f%% nodes healthy (%d/%d)", 
				healthPercentage, healthyNodes, totalNodes)
		} else if healthPercentage < 90 {
			dfs.logger.Infof("⚡ System health is moderate: %.1f%% nodes healthy (%d/%d)", 
				healthPercentage, healthyNodes, totalNodes)
		}
	}
	
	if underReplicated > 0 {
		dfs.logger.Warnf("📊 %d chunks are under-replicated", underReplicated)
	}
}

// replicaVerificationMonitor verifies replica integrity
func (dfs *DFSCore) replicaVerificationMonitor() {
	verificationTicker := time.NewTicker(10 * time.Minute)
	defer verificationTicker.Stop()
	
	for {
		select {
		case <-verificationTicker.C:
			dfs.performReplicaVerification()
		case <-dfs.stopChan:
			return
		}
	}
}

// performReplicaVerification verifies the integrity of replicas
func (dfs *DFSCore) performReplicaVerification() {
	dfs.logger.Info("🔍 Performing replica integrity verification...")
	
	dfs.replicaMu.RLock()
	verificationList := make([]*ReplicaInfo, 0)
	for _, replica := range dfs.replicaInfo {
		// Verify replicas that haven't been checked recently
		if time.Since(replica.LastVerified) > 30*time.Minute {
			verificationList = append(verificationList, replica)
		}
	}
	dfs.replicaMu.RUnlock()
	
	for _, replica := range verificationList {
		go dfs.verifyChunkReplicas(replica)
	}
	
	dfs.logger.Infof("🔎 Verifying %d chunk replicas", len(verificationList))
}

// verifyChunkReplicas verifies all replicas of a specific chunk
func (dfs *DFSCore) verifyChunkReplicas(replica *ReplicaInfo) {
	healthyReplicas := make([]string, 0)
	
	for _, nodeID := range replica.CurrentReplicas {
		if dfs.verifyReplicaOnNode(replica.ChunkID, nodeID) {
			healthyReplicas = append(healthyReplicas, nodeID)
			replica.Health[nodeID] = "healthy"
		} else {
			replica.Health[nodeID] = "corrupted"
			dfs.logger.Warnf("❌ Replica of chunk %s on node %s is corrupted", 
				replica.ChunkID, nodeID)
		}
	}
	
	// Update replica info
	dfs.replicaMu.Lock()
	replica.CurrentReplicas = healthyReplicas
	replica.LastVerified = time.Now()
	dfs.replicaMu.Unlock()
	
	// Create new replicas if needed
	if len(healthyReplicas) < replica.DesiredReplicas {
		replicasNeeded := replica.DesiredReplicas - len(healthyReplicas)
		dfs.logger.Infof("🔧 Creating %d replacement replicas for chunk %s", 
			replicasNeeded, replica.ChunkID)
		go dfs.createAdditionalReplicas(replica.ChunkID, replicasNeeded)
	}
}

// verifyReplicaOnNode verifies a replica on a specific node
func (dfs *DFSCore) verifyReplicaOnNode(chunkID, nodeID string) bool {
	// TODO: Implement actual chunk integrity verification
	// This would involve checking chunk hash, accessibility, etc.
	
	// For now, simulate verification based on node health
	dfs.healthMu.RLock()
	health, exists := dfs.nodeHealth[nodeID]
	dfs.healthMu.RUnlock()
	
	if !exists {
		return false
	}
	
	return health.Status == "healthy"
}

// RegisterChunk registers a new chunk in the DFS system
func (dfs *DFSCore) RegisterChunk(chunkID, fileID string, nodeIDs []string) {
	dfs.replicaMu.Lock()
	defer dfs.replicaMu.Unlock()
	
	replicaInfo := &ReplicaInfo{
		ChunkID:         chunkID,
		FileID:          fileID,
		CurrentReplicas: make([]string, len(nodeIDs)),
		DesiredReplicas: dfs.config.DefaultReplicaCount,
		Health:          make(map[string]string),
		LastVerified:    time.Now(),
	}
	
	copy(replicaInfo.CurrentReplicas, nodeIDs)
	
	for _, nodeID := range nodeIDs {
		replicaInfo.Health[nodeID] = "healthy"
	}
	
	dfs.replicaInfo[chunkID] = replicaInfo
	
	dfs.logger.Infof("📝 Registered chunk %s with %d replicas", chunkID, len(nodeIDs))
}

// UnregisterChunk removes a chunk from the DFS system
func (dfs *DFSCore) UnregisterChunk(chunkID string) {
	dfs.replicaMu.Lock()
	defer dfs.replicaMu.Unlock()
	
	if _, exists := dfs.replicaInfo[chunkID]; exists {
		delete(dfs.replicaInfo, chunkID)
		dfs.logger.Infof("🗑️ Unregistered chunk %s", chunkID)
	}
}

// GetNodeHealth returns the health status of a node
func (dfs *DFSCore) GetNodeHealth(nodeID string) *NodeHealth {
	dfs.healthMu.RLock()
	defer dfs.healthMu.RUnlock()
	
	if health, exists := dfs.nodeHealth[nodeID]; exists {
		return health
	}
	return nil
}

// GetAllNodeHealth returns health status of all nodes
func (dfs *DFSCore) GetAllNodeHealth() map[string]*NodeHealth {
	dfs.healthMu.RLock()
	defer dfs.healthMu.RUnlock()
	
	health := make(map[string]*NodeHealth)
	for nodeID, nodeHealth := range dfs.nodeHealth {
		health[nodeID] = nodeHealth
	}
	return health
}

// GetReplicaInfo returns replica information for a chunk
func (dfs *DFSCore) GetReplicaInfo(chunkID string) *ReplicaInfo {
	dfs.replicaMu.RLock()
	defer dfs.replicaMu.RUnlock()
	
	if replica, exists := dfs.replicaInfo[chunkID]; exists {
		return replica
	}
	return nil
}

// GetAllReplicaInfo returns replica information for all chunks
func (dfs *DFSCore) GetAllReplicaInfo() map[string]*ReplicaInfo {
	dfs.replicaMu.RLock()
	defer dfs.replicaMu.RUnlock()
	
	replicas := make(map[string]*ReplicaInfo)
	for chunkID, replicaInfo := range dfs.replicaInfo {
		replicas[chunkID] = replicaInfo
	}
	return replicas
}
