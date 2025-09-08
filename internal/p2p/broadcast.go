package p2p

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// BroadcastManager handles broadcasting messages to all peers
type BroadcastManager struct {
	network         *TCPNetwork
	messageQueue    chan *BroadcastMessage
	subscribers     map[string]chan *BroadcastMessage
	messageHistory  map[string]*BroadcastMessage
	broadcastStats  *BroadcastStats
	maxQueueSize    int
	maxHistorySize  int
	mu              sync.RWMutex
	running         bool
	stopChan        chan bool
}

// BroadcastMessage represents a message to be broadcasted
type BroadcastMessage struct {
	ID          string                 `json:"id"`
	Type        BroadcastType          `json:"type"`
	From        string                 `json:"from"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	TTL         int                    `json:"ttl"` // Time to live (hops)
	Recipients  []string               `json:"recipients,omitempty"` // Specific recipients, empty means all
	Priority    BroadcastPriority      `json:"priority"`
	Reliable    bool                   `json:"reliable"` // Require acknowledgment
}

// BroadcastType defines different types of broadcast messages
type BroadcastType string

const (
	BroadcastTypeFileAnnouncement BroadcastType = "file_announcement"
	BroadcastTypeFileRequest      BroadcastType = "file_request"
	BroadcastTypeChunkAnnouncement BroadcastType = "chunk_announcement"
	BroadcastTypeChunkRequest     BroadcastType = "chunk_request"
	BroadcastTypeNodeJoin         BroadcastType = "node_join"
	BroadcastTypeNodeLeave        BroadcastType = "node_leave"
	BroadcastTypeNetworkUpdate    BroadcastType = "network_update"
	BroadcastTypeStreamStart      BroadcastType = "stream_start"
	BroadcastTypeStreamEnd        BroadcastType = "stream_end"
	BroadcastTypeCustom           BroadcastType = "custom"
)

// BroadcastPriority defines message priority levels
type BroadcastPriority int

const (
	PriorityLow BroadcastPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// BroadcastStats tracks broadcasting statistics
type BroadcastStats struct {
	TotalSent        int64
	TotalReceived    int64
	TotalDelivered   int64
	TotalFailed      int64
	AverageLatency   time.Duration
	LastBroadcastTime time.Time
	mu               sync.RWMutex
}

// NewBroadcastManager creates a new broadcast manager
func NewBroadcastManager(network *TCPNetwork) *BroadcastManager {
	return &BroadcastManager{
		network:        network,
		messageQueue:   make(chan *BroadcastMessage, 1000),
		subscribers:    make(map[string]chan *BroadcastMessage),
		messageHistory: make(map[string]*BroadcastMessage),
		broadcastStats: &BroadcastStats{},
		maxQueueSize:   1000,
		maxHistorySize: 10000,
		stopChan:       make(chan bool),
		running:        false,
	}
}

// Start begins the broadcast manager
func (bm *BroadcastManager) Start() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.running {
		return fmt.Errorf("broadcast manager already running")
	}

	bm.running = true

	// Start message processor
	go bm.processMessages()

	// Start statistics updater
	go bm.updateStatistics()

	// Register broadcast message handler with TCP network
	bm.network.RegisterMessageHandler(MessageTypeBroadcast, bm.handleIncomingBroadcast)

	fmt.Printf("📡 Broadcast Manager started\n")
	return nil
}

// Stop stops the broadcast manager
func (bm *BroadcastManager) Stop() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if !bm.running {
		return nil
	}

	bm.running = false
	close(bm.stopChan)

	// Close all subscriber channels
	for _, ch := range bm.subscribers {
		close(ch)
	}

	fmt.Printf("🛑 Broadcast Manager stopped\n")
	return nil
}

// BroadcastOptions provides options for broadcasting
type BroadcastOptions struct {
	Priority   BroadcastPriority
	TTL        int
	Recipients []string // Specific recipients, empty means all
	Reliable   bool     // Require acknowledgment
}

// Broadcast sends a message to all connected peers
func (bm *BroadcastManager) Broadcast(msgType BroadcastType, data map[string]interface{}, priority BroadcastPriority) (string, error) {
	return bm.BroadcastWithOptions(msgType, data, BroadcastOptions{
		Priority: priority,
		TTL:      5, // Default TTL
		Reliable: false,
	})
}

// BroadcastWithOptions sends a broadcast message with specific options
func (bm *BroadcastManager) BroadcastWithOptions(msgType BroadcastType, data map[string]interface{}, options BroadcastOptions) (string, error) {
	message := &BroadcastMessage{
		ID:         uuid.New().String(),
		Type:       msgType,
		From:       bm.network.LocalNode.ID,
		Data:       data,
		Timestamp:  time.Now(),
		TTL:        options.TTL,
		Recipients: options.Recipients,
		Priority:   options.Priority,
		Reliable:   options.Reliable,
	}

	// Add to queue
	select {
	case bm.messageQueue <- message:
		fmt.Printf("📤 Queued broadcast message: %s (Type: %s, Priority: %d)\n", 
			message.ID, message.Type, message.Priority)
		return message.ID, nil
	default:
		return "", fmt.Errorf("broadcast queue full")
	}
}

// BroadcastFileAnnouncement announces a new file to the network
func (bm *BroadcastManager) BroadcastFileAnnouncement(fileID, fileName string, fileSize int64, chunkCount int) error {
	data := map[string]interface{}{
		"file_id":     fileID,
		"file_name":   fileName,
		"file_size":   fileSize,
		"chunk_count": chunkCount,
		"node_id":     bm.network.LocalNode.ID,
	}

	_, err := bm.Broadcast(BroadcastTypeFileAnnouncement, data, PriorityNormal)
	return err
}

// processMessages processes queued broadcast messages
func (bm *BroadcastManager) processMessages() {
	for bm.running {
		select {
		case message := <-bm.messageQueue:
			if message != nil {
				bm.sendBroadcastMessage(message)
			}
		case <-bm.stopChan:
			return
		}
	}
}

// sendBroadcastMessage sends a broadcast message to peers
func (bm *BroadcastManager) sendBroadcastMessage(message *BroadcastMessage) {
	if message.TTL <= 0 {
		fmt.Printf("⚠️ Message %s TTL expired, dropping\n", message.ID)
		return
	}

	// Store in history
	bm.mu.Lock()
	if len(bm.messageHistory) >= bm.maxHistorySize {
		// Remove oldest message
		for id := range bm.messageHistory {
			delete(bm.messageHistory, id)
			break
		}
	}
	bm.messageHistory[message.ID] = message
	bm.mu.Unlock()

	// Get target peers
	peers := bm.getTargetPeers(message)

	// Send to all target peers
	for _, peer := range peers {
		go func(p *TCPPeer) {
			if err := bm.network.sendMessageToPeer(p, MessageTypeBroadcast, message); err != nil {
				fmt.Printf("❌ Failed to broadcast to peer %s: %v\n", p.ID, err)
				bm.updateStats(false)
			} else {
				bm.updateStats(true)
			}
		}(peer)
	}

	fmt.Printf("📡 Broadcasted message %s to %d peers (Type: %s)\n", 
		message.ID, len(peers), message.Type)

	// Update statistics
	bm.broadcastStats.mu.Lock()
	bm.broadcastStats.TotalSent++
	bm.broadcastStats.LastBroadcastTime = time.Now()
	bm.broadcastStats.mu.Unlock()

	// Notify subscribers
	bm.notifySubscribers(message)
}

// getTargetPeers determines which peers should receive the message
func (bm *BroadcastManager) getTargetPeers(message *BroadcastMessage) []*TCPPeer {
	allPeers := bm.network.GetConnectedPeers()

	// If specific recipients are specified
	if len(message.Recipients) > 0 {
		targetPeers := make([]*TCPPeer, 0)
		for _, peer := range allPeers {
			for _, recipientID := range message.Recipients {
				if peer.ID == recipientID {
					targetPeers = append(targetPeers, peer)
					break
				}
			}
		}
		return targetPeers
	}

	// Return all connected peers
	return allPeers
}

// handleIncomingBroadcast handles incoming broadcast messages
func (bm *BroadcastManager) handleIncomingBroadcast(peer *TCPPeer, msg *TCPMessage) error {
	var broadcastMsg BroadcastMessage
	if err := json.Unmarshal(msg.Data, &broadcastMsg); err != nil {
		return fmt.Errorf("failed to unmarshal broadcast message: %v", err)
	}

	fmt.Printf("📥 Received broadcast message: %s from %s (Type: %s)\n", 
		broadcastMsg.ID, peer.ID, broadcastMsg.Type)

	// Check if we've seen this message before
	bm.mu.RLock()
	_, seen := bm.messageHistory[broadcastMsg.ID]
	bm.mu.RUnlock()

	if seen {
		fmt.Printf("🔄 Duplicate broadcast message %s, ignoring\n", broadcastMsg.ID)
		return nil
	}

	// Store in history
	bm.mu.Lock()
	bm.messageHistory[broadcastMsg.ID] = &broadcastMsg
	bm.mu.Unlock()

	// Update statistics
	bm.broadcastStats.mu.Lock()
	bm.broadcastStats.TotalReceived++
	bm.broadcastStats.mu.Unlock()

	// Notify local subscribers
	bm.notifySubscribers(&broadcastMsg)

	// Forward message if TTL > 1 (flooding protocol)
	if broadcastMsg.TTL > 1 {
		broadcastMsg.TTL--
		go func() {
			time.Sleep(100 * time.Millisecond) // Small delay to prevent loops
			bm.forwardBroadcastMessage(&broadcastMsg, peer.ID)
		}()
	}

	return nil
}

// forwardBroadcastMessage forwards a broadcast message to other peers
func (bm *BroadcastManager) forwardBroadcastMessage(message *BroadcastMessage, excludePeerID string) {
	peers := bm.network.GetConnectedPeers()

	for _, peer := range peers {
		if peer.ID != excludePeerID && peer.ID != message.From {
			go func(p *TCPPeer) {
				if err := bm.network.sendMessageToPeer(p, MessageTypeBroadcast, message); err != nil {
					fmt.Printf("❌ Failed to forward broadcast to peer %s: %v\n", p.ID, err)
				}
			}(peer)
		}
	}

	fmt.Printf("🔄 Forwarded broadcast message %s to %d peers\n", message.ID, len(peers)-1)
}

// Subscribe subscribes to broadcast messages of specific types
func (bm *BroadcastManager) Subscribe(subscriberID string, msgTypes []BroadcastType) (chan *BroadcastMessage, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.subscribers[subscriberID]; exists {
		return nil, fmt.Errorf("subscriber %s already exists", subscriberID)
	}

	subscribeChan := make(chan *BroadcastMessage, 100)
	bm.subscribers[subscriberID] = subscribeChan

	fmt.Printf("📋 Subscriber %s registered for %d message types\n", subscriberID, len(msgTypes))
	return subscribeChan, nil
}

// notifySubscribers notifies all subscribers of a broadcast message
func (bm *BroadcastManager) notifySubscribers(message *BroadcastMessage) {
	bm.mu.RLock()
	subscribers := make(map[string]chan *BroadcastMessage)
	for id, ch := range bm.subscribers {
		subscribers[id] = ch
	}
	bm.mu.RUnlock()

	for subscriberID, ch := range subscribers {
		select {
		case ch <- message:
			// Message sent successfully
		default:
			// Channel full, skip this subscriber
			fmt.Printf("⚠️ Subscriber %s channel full, skipping message %s\n", 
				subscriberID, message.ID)
		}
	}
}

// updateStats updates broadcast statistics
func (bm *BroadcastManager) updateStats(success bool) {
	bm.broadcastStats.mu.Lock()
	defer bm.broadcastStats.mu.Unlock()

	if success {
		bm.broadcastStats.TotalDelivered++
	} else {
		bm.broadcastStats.TotalFailed++
	}
}

// updateStatistics periodically updates statistics
func (bm *BroadcastManager) updateStatistics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for bm.running {
		select {
		case <-ticker.C:
			bm.cleanupHistory()
		case <-bm.stopChan:
			return
		}
	}
}

// cleanupHistory removes old messages from history
func (bm *BroadcastManager) cleanupHistory() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour) // Keep messages for 1 hour
	cleaned := 0

	for id, message := range bm.messageHistory {
		if message.Timestamp.Before(cutoff) {
			delete(bm.messageHistory, id)
			cleaned++
		}
	}

	if cleaned > 0 {
		fmt.Printf("🧹 Cleaned up %d old broadcast messages\n", cleaned)
	}
}

// GetStatistics returns current broadcast statistics
func (bm *BroadcastManager) GetStatistics() BroadcastStats {
	bm.broadcastStats.mu.RLock()
	defer bm.broadcastStats.mu.RUnlock()

	return *bm.broadcastStats
}
