package dfs

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/p2p"
	"github.com/sirupsen/logrus"
)

// NodeScore represents a node's suitability for storing a chunk
type NodeScore struct {
	Node            *p2p.Node
	Score           float64
	StorageLoad     float64  // 0.0 to 1.0
	NetworkLatency  time.Duration
	Reliability     float64  // 0.0 to 1.0
	Capacity        int64
	FreeSpace       int64
}

// DistributionStrategy defines different strategies for chunk distribution
type DistributionStrategy string

const (
	StrategyBalanced    DistributionStrategy = "balanced"     // Balance storage and performance
	StrategyPerformance DistributionStrategy = "performance" // Prioritize fast nodes
	StrategyReliability DistributionStrategy = "reliability" // Prioritize reliable nodes
	StrategyCapacity    DistributionStrategy = "capacity"    // Prioritize high-capacity nodes
)

// ChunkDistributor handles intelligent distribution of chunks across nodes
type ChunkDistributor struct {
	dfsCore  *DFSCore
	strategy DistributionStrategy
	logger   *logrus.Logger
}

// NewChunkDistributor creates a new chunk distributor
func NewChunkDistributor(dfsCore *DFSCore, strategy DistributionStrategy) *ChunkDistributor {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	return &ChunkDistributor{
		dfsCore:  dfsCore,
		strategy: strategy,
		logger:   logger,
	}
}

// SelectOptimalNodes selects the best nodes for storing chunk replicas
func (cd *ChunkDistributor) SelectOptimalNodes(chunkID string, replicaCount int, excludeNodes []string) ([]*p2p.Node, error) {
	availableNodes := cd.getAvailableNodes(excludeNodes)
	
	if len(availableNodes) < replicaCount {
		return nil, fmt.Errorf("insufficient nodes available: need %d, have %d", replicaCount, len(availableNodes))
	}

	// Score all available nodes
	nodeScores := cd.scoreNodes(availableNodes, chunkID)
	
	// Sort nodes by score (highest first)
	sort.Slice(nodeScores, func(i, j int) bool {
		return nodeScores[i].Score > nodeScores[j].Score
	})

	// Select top nodes, ensuring geographic/rack diversity if possible
	selectedNodes := cd.selectDiverseNodes(nodeScores, replicaCount)

	cd.logger.Infof("🎯 Selected %d optimal nodes for chunk %s using %s strategy", 
		len(selectedNodes), chunkID, cd.strategy)

	return selectedNodes, nil
}

// getAvailableNodes returns all healthy nodes excluding the specified ones
func (cd *ChunkDistributor) getAvailableNodes(excludeNodes []string) []*p2p.Node {
	allNodes := cd.dfsCore.getHealthyNodes()
	excludeSet := make(map[string]bool)
	
	for _, nodeID := range excludeNodes {
		excludeSet[nodeID] = true
	}

	var availableNodes []*p2p.Node
	for _, node := range allNodes {
		if !excludeSet[node.ID] {
			availableNodes = append(availableNodes, node)
		}
	}

	return availableNodes
}

// scoreNodes calculates scores for all nodes based on the current strategy
func (cd *ChunkDistributor) scoreNodes(nodes []*p2p.Node, chunkID string) []*NodeScore {
	nodeScores := make([]*NodeScore, len(nodes))

	for i, node := range nodes {
		health := cd.dfsCore.GetNodeHealth(node.ID)
		if health == nil {
			// Default health if not available
			health = &NodeHealth{
				NodeID:             node.ID,
				Status:             "unknown",
				StorageUtilization: 0.5,
				StorageCapacity:    100 * 1024 * 1024 * 1024, // 100GB
				StorageUsed:        50 * 1024 * 1024 * 1024,  // 50GB
				ChunkCount:         len(node.Chunks),
				NetworkLatency:     100 * time.Millisecond,
				ResponseTime:       50 * time.Millisecond,
				FailureCount:       0,
			}
		}

		nodeScore := &NodeScore{
			Node:            node,
			StorageLoad:     health.StorageUtilization,
			NetworkLatency:  health.NetworkLatency,
			Reliability:     cd.calculateReliability(health),
			Capacity:        health.StorageCapacity,
			FreeSpace:       health.StorageCapacity - health.StorageUsed,
		}

		nodeScore.Score = cd.calculateNodeScore(nodeScore)
		nodeScores[i] = nodeScore
	}

	return nodeScores
}

// calculateReliability calculates a reliability score based on node health
func (cd *ChunkDistributor) calculateReliability(health *NodeHealth) float64 {
	if health.Status == "failed" {
		return 0.0
	}

	reliability := 1.0
	
	// Factor in failure count
	if health.FailureCount > 0 {
		reliability *= math.Pow(0.9, float64(health.FailureCount))
	}

	// Factor in last heartbeat recency
	timeSinceHeartbeat := time.Since(health.LastHeartbeat)
	if timeSinceHeartbeat > time.Minute {
		// Reduce reliability for stale heartbeats
		staleFactor := math.Max(0.1, 1.0 - timeSinceHeartbeat.Seconds()/300.0) // 5 minutes = 0 reliability
		reliability *= staleFactor
	}

	// Factor in status
	switch health.Status {
	case "healthy":
		// No penalty
	case "degraded":
		reliability *= 0.7
	case "failed":
		reliability = 0.0
	default:
		reliability *= 0.5 // Unknown status
	}

	return math.Max(0.0, math.Min(1.0, reliability))
}

// calculateNodeScore calculates the overall score for a node based on strategy
func (cd *ChunkDistributor) calculateNodeScore(nodeScore *NodeScore) float64 {
	switch cd.strategy {
	case StrategyBalanced:
		return cd.calculateBalancedScore(nodeScore)
	case StrategyPerformance:
		return cd.calculatePerformanceScore(nodeScore)
	case StrategyReliability:
		return cd.calculateReliabilityScore(nodeScore)
	case StrategyCapacity:
		return cd.calculateCapacityScore(nodeScore)
	default:
		return cd.calculateBalancedScore(nodeScore)
	}
}

// calculateBalancedScore calculates a balanced score considering all factors
func (cd *ChunkDistributor) calculateBalancedScore(nodeScore *NodeScore) float64 {
	// Normalize network latency (lower is better)
	latencyScore := 1.0 - math.Min(1.0, nodeScore.NetworkLatency.Seconds()/1.0)
	
	// Storage load score (lower utilization is better)
	storageScore := 1.0 - nodeScore.StorageLoad
	
	// Free space score (more free space is better)
	freeSpaceRatio := float64(nodeScore.FreeSpace) / float64(nodeScore.Capacity)
	freeSpaceScore := math.Min(1.0, freeSpaceRatio*2.0) // Cap at 1.0
	
	// Reliability score (higher is better)
	reliabilityScore := nodeScore.Reliability

	// Weighted combination
	score := (reliabilityScore * 0.35) +     // 35% reliability
		    (storageScore * 0.25) +          // 25% storage availability
		    (latencyScore * 0.25) +          // 25% network performance  
		    (freeSpaceScore * 0.15)          // 15% capacity

	return math.Max(0.0, math.Min(1.0, score))
}

// calculatePerformanceScore prioritizes network performance
func (cd *ChunkDistributor) calculatePerformanceScore(nodeScore *NodeScore) float64 {
	// Heavily weight network latency and response time
	latencyScore := 1.0 - math.Min(1.0, nodeScore.NetworkLatency.Seconds()/0.5)
	reliabilityScore := nodeScore.Reliability
	storageScore := 1.0 - nodeScore.StorageLoad

	score := (latencyScore * 0.6) +         // 60% network performance
		    (reliabilityScore * 0.3) +      // 30% reliability
		    (storageScore * 0.1)            // 10% storage availability

	return math.Max(0.0, math.Min(1.0, score))
}

// calculateReliabilityScore prioritizes node reliability
func (cd *ChunkDistributor) calculateReliabilityScore(nodeScore *NodeScore) float64 {
	// Heavily weight reliability
	reliabilityScore := nodeScore.Reliability
	storageScore := 1.0 - nodeScore.StorageLoad
	latencyScore := 1.0 - math.Min(1.0, nodeScore.NetworkLatency.Seconds()/1.0)

	score := (reliabilityScore * 0.7) +     // 70% reliability
		    (storageScore * 0.2) +          // 20% storage availability
		    (latencyScore * 0.1)            // 10% network performance

	return math.Max(0.0, math.Min(1.0, score))
}

// calculateCapacityScore prioritizes storage capacity
func (cd *ChunkDistributor) calculateCapacityScore(nodeScore *NodeScore) float64 {
	// Heavily weight available storage capacity
	freeSpaceRatio := float64(nodeScore.FreeSpace) / float64(nodeScore.Capacity)
	capacityScore := math.Min(1.0, freeSpaceRatio*2.0)
	
	storageScore := 1.0 - nodeScore.StorageLoad
	reliabilityScore := nodeScore.Reliability

	score := (capacityScore * 0.5) +        // 50% free capacity
		    (storageScore * 0.3) +          // 30% storage load
		    (reliabilityScore * 0.2)        // 20% reliability

	return math.Max(0.0, math.Min(1.0, score))
}

// selectDiverseNodes selects nodes ensuring geographic/network diversity
func (cd *ChunkDistributor) selectDiverseNodes(nodeScores []*NodeScore, count int) []*p2p.Node {
	if len(nodeScores) <= count {
		nodes := make([]*p2p.Node, len(nodeScores))
		for i, score := range nodeScores {
			nodes[i] = score.Node
		}
		return nodes
	}

	selected := make([]*p2p.Node, 0, count)
	selectedIPs := make(map[string]bool)

	// First pass: select highest scoring nodes from different IP addresses
	for _, nodeScore := range nodeScores {
		if len(selected) >= count {
			break
		}

		// Simple diversity check based on IP address
		if !selectedIPs[nodeScore.Node.Address] {
			selected = append(selected, nodeScore.Node)
			selectedIPs[nodeScore.Node.Address] = true
		}
	}

	// Second pass: fill remaining slots with highest scoring nodes
	for _, nodeScore := range nodeScores {
		if len(selected) >= count {
			break
		}

		alreadySelected := false
		for _, selectedNode := range selected {
			if selectedNode.ID == nodeScore.Node.ID {
				alreadySelected = true
				break
			}
		}

		if !alreadySelected {
			selected = append(selected, nodeScore.Node)
		}
	}

	return selected
}

// RebalanceChunks rebalances chunk distribution across the network
func (cd *ChunkDistributor) RebalanceChunks() error {
	cd.logger.Info("🔄 Starting intelligent chunk rebalancing...")

	allReplicas := cd.dfsCore.GetAllReplicaInfo()
	rebalanceCount := 0

	for chunkID, replica := range allReplicas {
		if cd.shouldRebalanceChunk(replica) {
			if err := cd.rebalanceChunk(chunkID, replica); err != nil {
				cd.logger.Errorf("❌ Failed to rebalance chunk %s: %v", chunkID, err)
			} else {
				rebalanceCount++
			}
		}
	}

	cd.logger.Infof("✅ Rebalancing completed: %d chunks rebalanced", rebalanceCount)
	return nil
}

// shouldRebalanceChunk determines if a chunk needs rebalancing
func (cd *ChunkDistributor) shouldRebalanceChunk(replica *ReplicaInfo) bool {
	// Check if under-replicated
	if len(replica.CurrentReplicas) < replica.DesiredReplicas {
		return true
	}

	// Check if replicas are on poorly performing nodes
	poorPerformingCount := 0
	for _, nodeID := range replica.CurrentReplicas {
		health := cd.dfsCore.GetNodeHealth(nodeID)
		if health != nil && (health.Status == "degraded" || health.StorageUtilization > 0.9) {
			poorPerformingCount++
		}
	}

	// Rebalance if more than half the replicas are on poor performing nodes
	return poorPerformingCount > len(replica.CurrentReplicas)/2
}

// rebalanceChunk rebalances replicas for a specific chunk
func (cd *ChunkDistributor) rebalanceChunk(chunkID string, replica *ReplicaInfo) error {
	cd.logger.Infof("⚖️ Rebalancing chunk %s", chunkID)

	// Find better nodes for this chunk
	optimalNodes, err := cd.SelectOptimalNodes(chunkID, replica.DesiredReplicas, replica.CurrentReplicas)
	if err != nil {
		return fmt.Errorf("failed to find optimal nodes: %v", err)
	}

	// Calculate which replicas to move
	nodesToAdd := make([]*p2p.Node, 0)
	nodesToRemove := make([]string, 0)

	// Simple approach: if we found significantly better nodes, migrate to them
	currentScores := cd.scoreCurrentReplicas(replica)
	newScores := cd.scoreNodes(optimalNodes, chunkID)

	avgCurrentScore := cd.calculateAverageScore(currentScores)
	avgNewScore := cd.calculateAverageScore(newScores)

	if avgNewScore > avgCurrentScore*1.2 { // 20% improvement threshold
		// Plan migration to better nodes
		for _, newScore := range newScores {
			if !cd.nodeInCurrentReplicas(newScore.Node.ID, replica.CurrentReplicas) {
				nodesToAdd = append(nodesToAdd, newScore.Node)
			}
		}

		// Remove worst performing current replicas
		sort.Slice(currentScores, func(i, j int) bool {
			return currentScores[i].Score < currentScores[j].Score
		})

		removeCount := int(math.Min(float64(len(nodesToAdd)), float64(len(currentScores))/2.0))
		for i := 0; i < int(removeCount); i++ {
			nodesToRemove = append(nodesToRemove, currentScores[i].Node.ID)
		}
	}

	// Execute the rebalancing
	for _, node := range nodesToAdd {
		if err := cd.dfsCore.createReplicaOnNode(chunkID, node.ID); err != nil {
			cd.logger.Errorf("❌ Failed to create replica on node %s: %v", node.ID, err)
		} else {
			cd.logger.Infof("✅ Created new replica of chunk %s on node %s", chunkID, node.ID)
		}
	}

	for _, nodeID := range nodesToRemove {
		cd.removeReplicaFromNode(chunkID, nodeID)
		cd.logger.Infof("🗑️ Removed replica of chunk %s from node %s", chunkID, nodeID)
	}

	return nil
}

// scoreCurrentReplicas scores the current replica nodes
func (cd *ChunkDistributor) scoreCurrentReplicas(replica *ReplicaInfo) []*NodeScore {
	currentNodes := make([]*p2p.Node, 0, len(replica.CurrentReplicas))
	
	for _, nodeID := range replica.CurrentReplicas {
		if nodeID == cd.dfsCore.network.LocalNode.ID {
			currentNodes = append(currentNodes, cd.dfsCore.network.LocalNode)
		} else {
			peer := cd.dfsCore.network.GetPeerByID(nodeID)
			if peer != nil {
				currentNodes = append(currentNodes, peer)
			}
		}
	}

	return cd.scoreNodes(currentNodes, replica.ChunkID)
}

// calculateAverageScore calculates the average score of a set of node scores
func (cd *ChunkDistributor) calculateAverageScore(scores []*NodeScore) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	total := 0.0
	for _, score := range scores {
		total += score.Score
	}

	return total / float64(len(scores))
}

// nodeInCurrentReplicas checks if a node is already in the current replicas
func (cd *ChunkDistributor) nodeInCurrentReplicas(nodeID string, currentReplicas []string) bool {
	for _, replicaNodeID := range currentReplicas {
		if nodeID == replicaNodeID {
			return true
		}
	}
	return false
}

// removeReplicaFromNode removes a replica from a specific node
func (cd *ChunkDistributor) removeReplicaFromNode(chunkID, nodeID string) {
	// TODO: Implement actual replica removal
	// This would involve deleting the chunk from the node's storage
	
	// For now, just update the replica info
	replica := cd.dfsCore.GetReplicaInfo(chunkID)
	if replica != nil {
		updatedReplicas := make([]string, 0)
		for _, replicaNodeID := range replica.CurrentReplicas {
			if replicaNodeID != nodeID {
				updatedReplicas = append(updatedReplicas, replicaNodeID)
			}
		}
		replica.CurrentReplicas = updatedReplicas
		delete(replica.Health, nodeID)
	}
}

// GetDistributionStats returns statistics about chunk distribution
func (cd *ChunkDistributor) GetDistributionStats() map[string]interface{} {
	allReplicas := cd.dfsCore.GetAllReplicaInfo()
	allHealth := cd.dfsCore.GetAllNodeHealth()

	stats := map[string]interface{}{
		"total_chunks":           len(allReplicas),
		"avg_replicas_per_chunk": 0.0,
		"distribution_efficiency": 0.0,
		"load_balance_score":     0.0,
		"strategy":               string(cd.strategy),
	}

	if len(allReplicas) > 0 {
		totalReplicas := 0
		for _, replica := range allReplicas {
			totalReplicas += len(replica.CurrentReplicas)
		}
		stats["avg_replicas_per_chunk"] = float64(totalReplicas) / float64(len(allReplicas))
	}

	// Calculate load balance score
	if len(allHealth) > 0 {
		utilizationSum := 0.0
		for _, health := range allHealth {
			utilizationSum += health.StorageUtilization
		}
		avgUtilization := utilizationSum / float64(len(allHealth))
		
		// Calculate variance from average
		varianceSum := 0.0
		for _, health := range allHealth {
			diff := health.StorageUtilization - avgUtilization
			varianceSum += diff * diff
		}
		variance := varianceSum / float64(len(allHealth))
		
		// Load balance score: higher score = more balanced (lower variance)
		stats["load_balance_score"] = math.Max(0.0, 1.0 - variance)
	}

	return stats
}
