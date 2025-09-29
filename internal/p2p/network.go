package p2p

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

// Node represents a peer in the P2P network
type Node struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Port     int       `json:"port"`
	LastSeen time.Time `json:"last_seen"`
	Status   string    `json:"status"` // "online", "offline", "unreachable"
	Files    []string  `json:"files"`  // List of file IDs this node has
	Chunks   []string  `json:"chunks"` // List of chunk IDs this node has
}

// Network represents the P2P network
type Network struct {
	LocalNode       *Node
	Peers           map[string]*Node
	mu              sync.RWMutex
	heartbeatTicker *time.Ticker
	stopChan        chan bool
	store           storage.Storage            // Storage backend for serving chunks
	metaStore       *metadata.MetadataStore    // Metadata store for chunk mapping
}

// NetworkMessage represents messages exchanged between nodes
type NetworkMessage struct {
	Type      string      `json:"type"` // "ping", "pong", "file_request", "chunk_request", "node_register"
	From      string      `json:"from"`
	To        string      `json:"to"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewNetwork creates a new P2P network
func NewNetwork(address string, port int) *Network {
	return &Network{
		LocalNode: &Node{
			ID:       uuid.New().String(),
			Address:  address,
			Port:     port,
			LastSeen: time.Now(),
			Status:   "online",
			Files:    make([]string, 0),
			Chunks:   make([]string, 0),
		},
		Peers:    make(map[string]*Node),
		stopChan: make(chan bool),
		store:    nil, // Storage will be set later
	}
}

// SetStorage sets the storage backend for the network
func (n *Network) SetStorage(store storage.Storage) {
	n.store = store
}

// SetMetadataStore sets the metadata store for the network
func (n *Network) SetMetadataStore(metaStore *metadata.MetadataStore) {
	n.metaStore = metaStore
}

// Start initializes the P2P network
func (n *Network) Start() error {
	// Start heartbeat monitoring
	n.heartbeatTicker = time.NewTicker(30 * time.Second)
	go n.heartbeatMonitor()

	// Start HTTP server for P2P communication
	go n.startHTTPServer()

	fmt.Printf("🌐 P2P Network started - Node ID: %s\n", n.LocalNode.ID)
	return nil
}

// Stop shuts down the P2P network
func (n *Network) Stop() {
	if n.heartbeatTicker != nil {
		n.heartbeatTicker.Stop()
	}
	close(n.stopChan)
}

// RegisterPeer registers a new peer in the network
func (n *Network) RegisterPeer(node *Node) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if node.ID != n.LocalNode.ID {
		n.Peers[node.ID] = node
		fmt.Printf("📝 Registered peer: %s (%s:%d)\n", node.ID, node.Address, node.Port)
	}
}

// RemovePeer removes a peer from the network
func (n *Network) RemovePeer(nodeID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, exists := n.Peers[nodeID]; exists {
		delete(n.Peers, nodeID)
		fmt.Printf("❌ Removed peer: %s (%s:%d)\n", peer.ID, peer.Address, peer.Port)
	}
}

// GetPeers returns all active peers
func (n *Network) GetPeers() []*Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]*Node, 0, len(n.Peers))
	for _, peer := range n.Peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetPeerByID returns a specific peer by ID
func (n *Network) GetPeerByID(nodeID string) *Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.Peers[nodeID]
}

// UpdatePeerStatus updates the status of a peer
func (n *Network) UpdatePeerStatus(nodeID string, status string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, exists := n.Peers[nodeID]; exists {
		peer.Status = status
		peer.LastSeen = time.Now()
	}
}

// AddFileToNode adds a file to a node's file list
func (n *Network) AddFileToNode(nodeID string, fileID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if nodeID == n.LocalNode.ID {
		n.LocalNode.Files = append(n.LocalNode.Files, fileID)
	} else if peer, exists := n.Peers[nodeID]; exists {
		peer.Files = append(peer.Files, fileID)
	}
}

// AddChunkToNode adds a chunk to a node's chunk list
func (n *Network) AddChunkToNode(nodeID string, chunkID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if nodeID == n.LocalNode.ID {
		n.LocalNode.Chunks = append(n.LocalNode.Chunks, chunkID)
	} else if peer, exists := n.Peers[nodeID]; exists {
		peer.Chunks = append(peer.Chunks, chunkID)
	}
}

// FindNodesWithChunk finds all nodes that have a specific chunk
func (n *Network) FindNodesWithChunk(chunkID string) []*Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var nodes []*Node

	// Check local node
	for _, chunk := range n.LocalNode.Chunks {
		if chunk == chunkID {
			nodes = append(nodes, n.LocalNode)
			break
		}
	}

	// Check peers
	for _, peer := range n.Peers {
		if peer.Status == "online" {
			for _, chunk := range peer.Chunks {
				if chunk == chunkID {
					nodes = append(nodes, peer)
					break
				}
			}
		}
	}

	return nodes
}

// FindNodesWithFile finds all nodes that have a specific file
func (n *Network) FindNodesWithFile(fileID string) []*Node {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var nodes []*Node

	// Check local node
	for _, file := range n.LocalNode.Files {
		if file == fileID {
			nodes = append(nodes, n.LocalNode)
			break
		}
	}

	// Check peers
	for _, peer := range n.Peers {
		if peer.Status == "online" {
			for _, file := range peer.Files {
				if file == fileID {
					nodes = append(nodes, peer)
					break
				}
			}
		}
	}

	return nodes
}

// heartbeatMonitor monitors the health of all peers
func (n *Network) heartbeatMonitor() {
	for {
		select {
		case <-n.heartbeatTicker.C:
			n.checkPeerHealth()
		case <-n.stopChan:
			return
		}
	}
}

// checkPeerHealth checks the health of all peers
func (n *Network) checkPeerHealth() {
	peers := n.GetPeers()

	for _, peer := range peers {
		go n.pingPeer(peer)
	}
}

// pingPeer sends a ping to a peer to check its health
func (n *Network) pingPeer(peer *Node) {
	client := &http.Client{Timeout: 5 * time.Second}

	pingURL := fmt.Sprintf("http://%s:%d/ping", peer.Address, peer.Port)

	resp, err := client.Get(pingURL)
	if err != nil {
		n.UpdatePeerStatus(peer.ID, "offline")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		n.UpdatePeerStatus(peer.ID, "online")
	} else {
		n.UpdatePeerStatus(peer.ID, "unreachable")
	}
}

// startHTTPServer starts the HTTP server for P2P communication
func (n *Network) startHTTPServer() {
	mux := http.NewServeMux()

	// P2P endpoints
	mux.HandleFunc("/ping", n.HandlePing)
	mux.HandleFunc("/pong", n.HandlePong)
	mux.HandleFunc("/register", n.HandleRegister)
	mux.HandleFunc("/peers", n.HandleGetPeers)
	mux.HandleFunc("/file-request", n.HandleFileRequest)
	mux.HandleFunc("/chunk-request", n.HandleChunkRequest)
	mux.HandleFunc("/heartbeat", n.HandleHeartbeat)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", n.LocalNode.Port),
		Handler: mux,
	}

	fmt.Printf("🌐 P2P HTTP server starting on port %d\n", n.LocalNode.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("❌ P2P server failed: %v\n", err)
	}
}

// HTTP handlers for P2P communication
func (n *Network) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func (n *Network) HandlePong(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (n *Network) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var newNode Node
	if err := json.NewDecoder(r.Body).Decode(&newNode); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	n.RegisterPeer(&newNode)

	// Send back our node info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(n.LocalNode)
}

func (n *Network) HandleGetPeers(w http.ResponseWriter, r *http.Request) {
	peers := n.GetPeers()
	peers = append(peers, n.LocalNode) // Include local node

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peers)
}

func (n *Network) HandleFileRequest(w http.ResponseWriter, r *http.Request) {
	// Handle file requests from other nodes
	w.WriteHeader(http.StatusOK)
}

func (n *Network) HandleChunkRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get chunk ID from query parameters
	chunkID := r.URL.Query().Get("id")
	if chunkID == "" {
		http.Error(w, "Missing chunk ID", http.StatusBadRequest)
		return
	}

	// Check if local node has this chunk
	n.mu.RLock()
	hasChunk := false
	for _, chunk := range n.LocalNode.Chunks {
		if chunk == chunkID {
			hasChunk = true
			break
		}
	}
	n.mu.RUnlock()

	if !hasChunk {
		http.Error(w, "Chunk not found on this node", http.StatusNotFound)
		return
	}

	// Check if storage backend is available
	if n.store == nil {
		http.Error(w, "Storage backend not available", http.StatusInternalServerError)
		return
	}

	// Try to get chunk data from storage
	// First try direct access by chunkID (in case it's a hash)
	reader, err := n.store.Get(chunkID)
	if err != nil {
		// If direct access fails and we have metadata store, try to map UUID to storage path
		if n.metaStore != nil {
			if chunkMeta, metaErr := n.metaStore.GetChunkMetadata(chunkID); metaErr == nil {
				// Try using the path from metadata
				reader, err = n.store.Get(chunkMeta.Path)
				if err != nil {
					fmt.Printf("❌ Failed to retrieve chunk %s from storage path %s: %v\n", chunkID, chunkMeta.Path, err)
					http.Error(w, "Chunk not found in storage", http.StatusNotFound)
					return
				}
			} else {
				fmt.Printf("❌ Failed to retrieve chunk %s from storage and metadata: %v, %v\n", chunkID, err, metaErr)
				http.Error(w, "Chunk not found", http.StatusNotFound)
				return
			}
		} else {
			fmt.Printf("❌ Failed to retrieve chunk %s from storage: %v\n", chunkID, err)
			http.Error(w, "Chunk not found in storage", http.StatusNotFound)
			return
		}
	}
	defer reader.Close()

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	// Stream chunk data to the client
	if _, err := io.Copy(w, reader); err != nil {
		fmt.Printf("❌ Failed to stream chunk %s: %v\n", chunkID, err)
		return
	}
	
	fmt.Printf("📤 Successfully served chunk %s to remote node\n", chunkID)
}

func (n *Network) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("alive"))
}

// BroadcastMessage sends a message to all peers
func (n *Network) BroadcastMessage(msg *NetworkMessage) {
	peers := n.GetPeers()

	for _, peer := range peers {
		if peer.Status == "online" {
			go n.sendMessageToPeer(peer, msg)
		}
	}
}

// sendMessageToPeer sends a message to a specific peer
func (n *Network) sendMessageToPeer(peer *Node, msg *NetworkMessage) {
	client := &http.Client{Timeout: 10 * time.Second}

	msgData, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("❌ Failed to marshal message: %v\n", err)
		return
	}

	url := fmt.Sprintf("http://%s:%d/message", peer.Address, peer.Port)

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(msgData))
	if err != nil {
		fmt.Printf("❌ Failed to send message to %s: %v\n", peer.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("⚠️ Message to %s returned status: %d\n", peer.ID, resp.StatusCode)
	}
}
