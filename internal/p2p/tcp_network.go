package p2p

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TCPNetwork represents a custom TCP-based P2P network
type TCPNetwork struct {
	LocalNode       *Node
	Peers           map[string]*TCPPeer
	connections     map[string]net.Conn
	listener        net.Listener
	mu              sync.RWMutex
	stopChan        chan bool
	messageHandlers map[MessageType]MessageHandler
	running         bool
}

// TCPPeer represents a TCP peer connection
type TCPPeer struct {
	*Node
	Connection   net.Conn
	LastPing     time.Time
	Connected    bool
	Reader       *bufio.Reader
	Writer       *bufio.Writer
	writeMutex   sync.Mutex
}

// MessageType represents different types of P2P messages
type MessageType byte

const (
	MessageTypeHandshake MessageType = iota
	MessageTypeHandshakeReply
	MessageTypePing
	MessageTypePong
	MessageTypeFileRequest
	MessageTypeFileResponse
	MessageTypeChunkRequest
	MessageTypeChunkResponse
	MessageTypeBroadcast
	MessageTypeNodeDiscovery
	MessageTypeNodeAnnouncement
)

// TCPMessage represents a message sent over TCP
type TCPMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Timestamp time.Time   `json:"timestamp"`
	Data      []byte      `json:"data"`
	Signature []byte      `json:"signature,omitempty"`
}

// HandshakeData represents handshake information
type HandshakeData struct {
	NodeID    string    `json:"node_id"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Challenge string    `json:"challenge"`
	Response  string    `json:"response,omitempty"`
}

// MessageHandler defines the signature for message handlers
type MessageHandler func(*TCPPeer, *TCPMessage) error

// NewTCPNetwork creates a new TCP-based P2P network
func NewTCPNetwork(address string, port int) *TCPNetwork {
	return &TCPNetwork{
		LocalNode: &Node{
			ID:       uuid.New().String(),
			Address:  address,
			Port:     port,
			LastSeen: time.Now(),
			Status:   "online",
			Files:    make([]string, 0),
			Chunks:   make([]string, 0),
		},
		Peers:           make(map[string]*TCPPeer),
		connections:     make(map[string]net.Conn),
		stopChan:        make(chan bool),
		messageHandlers: make(map[MessageType]MessageHandler),
		running:         false,
	}
}

// Start initializes the TCP P2P network
func (n *TCPNetwork) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return fmt.Errorf("network already running")
	}

	// Register default message handlers
	n.registerDefaultHandlers()

	// Start TCP listener
	addr := fmt.Sprintf("%s:%d", n.LocalNode.Address, n.LocalNode.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %v", err)
	}

	n.listener = listener
	n.running = true

	// Start accepting connections
	go n.acceptConnections()

	// Start connection monitor
	go n.monitorConnections()

	fmt.Printf("🌐 TCP P2P Network started - Node ID: %s, Address: %s\n", n.LocalNode.ID, addr)
	return nil
}

// Stop shuts down the TCP P2P network
func (n *TCPNetwork) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running {
		return nil
	}

	n.running = false
	close(n.stopChan)

	// Close listener
	if n.listener != nil {
		n.listener.Close()
	}

	// Close all peer connections
	for _, peer := range n.Peers {
		if peer.Connection != nil {
			peer.Connection.Close()
		}
	}

	// Close direct connections
	for _, conn := range n.connections {
		conn.Close()
	}

	fmt.Printf("🛑 TCP P2P Network stopped\n")
	return nil
}

// ConnectToPeer connects to a remote peer
func (n *TCPNetwork) ConnectToPeer(address string, port int) (*TCPPeer, error) {
	addr := fmt.Sprintf("%s:%d", address, port)
	
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %v", addr, err)
	}

	peer := &TCPPeer{
		Node: &Node{
			ID:       "", // Will be set during handshake
			Address:  address,
			Port:     port,
			LastSeen: time.Now(),
			Status:   "connecting",
			Files:    make([]string, 0),
			Chunks:   make([]string, 0),
		},
		Connection: conn,
		Connected:  true,
		Reader:     bufio.NewReader(conn),
		Writer:     bufio.NewWriter(conn),
	}

	// Perform handshake
	if err := n.performHandshake(peer); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed: %v", err)
	}

	n.mu.Lock()
	n.Peers[peer.ID] = peer
	n.connections[peer.ID] = conn
	n.mu.Unlock()

	// Start message handler for this peer
	go n.handlePeerConnection(peer)

	fmt.Printf("🤝 Connected to peer: %s (%s)\n", peer.ID, addr)
	return peer, nil
}

// acceptConnections accepts incoming TCP connections
func (n *TCPNetwork) acceptConnections() {
	for n.running {
		conn, err := n.listener.Accept()
		if err != nil {
			if n.running {
				fmt.Printf("❌ Error accepting connection: %v\n", err)
			}
			continue
		}

		go n.handleIncomingConnection(conn)
	}
}

// handleIncomingConnection handles a new incoming connection
func (n *TCPNetwork) handleIncomingConnection(conn net.Conn) {
	defer conn.Close()

	peer := &TCPPeer{
		Node: &Node{
			Address:  conn.RemoteAddr().(*net.TCPAddr).IP.String(),
			Port:     conn.RemoteAddr().(*net.TCPAddr).Port,
			LastSeen: time.Now(),
			Status:   "connecting",
			Files:    make([]string, 0),
			Chunks:   make([]string, 0),
		},
		Connection: conn,
		Connected:  true,
		Reader:     bufio.NewReader(conn),
		Writer:     bufio.NewWriter(conn),
	}

	// Wait for handshake from remote peer
	if err := n.handleHandshakeRequest(peer); err != nil {
		fmt.Printf("❌ Handshake failed for incoming connection: %v\n", err)
		return
	}

	n.mu.Lock()
	n.Peers[peer.ID] = peer
	n.connections[peer.ID] = conn
	n.mu.Unlock()

	// Handle messages from this peer
	n.handlePeerConnection(peer)
}

// performHandshake performs handshake with a remote peer
func (n *TCPNetwork) performHandshake(peer *TCPPeer) error {
	// Generate challenge
	challenge := make([]byte, 32)
	rand.Read(challenge)
	challengeHex := hex.EncodeToString(challenge)

	handshakeData := HandshakeData{
		NodeID:    n.LocalNode.ID,
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Challenge: challengeHex,
	}

	// Send handshake
	if err := n.sendMessageToPeer(peer, MessageTypeHandshake, handshakeData); err != nil {
		return fmt.Errorf("failed to send handshake: %v", err)
	}

	// Wait for handshake reply
	msg, err := n.readMessageFromPeer(peer)
	if err != nil {
		return fmt.Errorf("failed to read handshake reply: %v", err)
	}

	if msg.Type != MessageTypeHandshakeReply {
		return fmt.Errorf("expected handshake reply, got %d", msg.Type)
	}

	var replyData HandshakeData
	if err := json.Unmarshal(msg.Data, &replyData); err != nil {
		return fmt.Errorf("failed to unmarshal handshake reply: %v", err)
	}

	// Verify challenge response
	expectedResponse := n.generateChallengeResponse(challengeHex)
	if replyData.Response != expectedResponse {
		return fmt.Errorf("invalid challenge response")
	}

	// Set peer ID
	peer.ID = replyData.NodeID
	peer.Status = "online"

	fmt.Printf("🤝 Handshake completed with peer: %s\n", peer.ID)
	return nil
}

// handleHandshakeRequest handles incoming handshake request
func (n *TCPNetwork) handleHandshakeRequest(peer *TCPPeer) error {
	// Read handshake message
	msg, err := n.readMessageFromPeer(peer)
	if err != nil {
		return fmt.Errorf("failed to read handshake: %v", err)
	}

	if msg.Type != MessageTypeHandshake {
		return fmt.Errorf("expected handshake, got %d", msg.Type)
	}

	var handshakeData HandshakeData
	if err := json.Unmarshal(msg.Data, &handshakeData); err != nil {
		return fmt.Errorf("failed to unmarshal handshake: %v", err)
	}

	// Set peer ID
	peer.ID = handshakeData.NodeID

	// Generate challenge response
	response := n.generateChallengeResponse(handshakeData.Challenge)

	// Send handshake reply
	replyData := HandshakeData{
		NodeID:    n.LocalNode.ID,
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Response:  response,
	}

	if err := n.sendMessageToPeer(peer, MessageTypeHandshakeReply, replyData); err != nil {
		return fmt.Errorf("failed to send handshake reply: %v", err)
	}

	peer.Status = "online"
	fmt.Printf("🤝 Handshake completed with incoming peer: %s\n", peer.ID)
	return nil
}

// generateChallengeResponse generates a response to a handshake challenge
func (n *TCPNetwork) generateChallengeResponse(challenge string) string {
	hasher := sha256.New()
	hasher.Write([]byte(challenge + n.LocalNode.ID))
	return hex.EncodeToString(hasher.Sum(nil))
}

// sendMessageToPeer sends a message to a specific peer
func (n *TCPNetwork) sendMessageToPeer(peer *TCPPeer, msgType MessageType, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	msg := &TCPMessage{
		Type:      msgType,
		ID:        uuid.New().String(),
		From:      n.LocalNode.ID,
		To:        peer.ID,
		Timestamp: time.Now(),
		Data:      dataBytes,
	}

	return n.writeMessageToPeer(peer, msg)
}

// writeMessageToPeer writes a TCP message to a peer
func (n *TCPNetwork) writeMessageToPeer(peer *TCPPeer, msg *TCPMessage) error {
	peer.writeMutex.Lock()
	defer peer.writeMutex.Unlock()

	// Serialize message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// Write message length first (4 bytes)
	length := uint32(len(msgBytes))
	if err := binary.Write(peer.Writer, binary.BigEndian, length); err != nil {
		return fmt.Errorf("failed to write message length: %v", err)
	}

	// Write message data
	if _, err := peer.Writer.Write(msgBytes); err != nil {
		return fmt.Errorf("failed to write message data: %v", err)
	}

	// Flush buffer
	if err := peer.Writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush message: %v", err)
	}

	return nil
}

// readMessageFromPeer reads a TCP message from a peer
func (n *TCPNetwork) readMessageFromPeer(peer *TCPPeer) (*TCPMessage, error) {
	// Read message length first (4 bytes)
	var length uint32
	if err := binary.Read(peer.Reader, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("failed to read message length: %v", err)
	}

	// Validate length
	if length == 0 || length > 10*1024*1024 { // Max 10MB message
		return nil, fmt.Errorf("invalid message length: %d", length)
	}

	// Read message data
	msgBytes := make([]byte, length)
	if _, err := io.ReadFull(peer.Reader, msgBytes); err != nil {
		return nil, fmt.Errorf("failed to read message data: %v", err)
	}

	// Deserialize message
	var msg TCPMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v", err)
	}

	return &msg, nil
}

// handlePeerConnection handles messages from a connected peer
func (n *TCPNetwork) handlePeerConnection(peer *TCPPeer) {
	defer func() {
		peer.Connected = false
		peer.Connection.Close()
		n.mu.Lock()
		delete(n.Peers, peer.ID)
		delete(n.connections, peer.ID)
		n.mu.Unlock()
		fmt.Printf("🔌 Disconnected from peer: %s\n", peer.ID)
	}()

	for peer.Connected && n.running {
		// Set read timeout
		peer.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))

		msg, err := n.readMessageFromPeer(peer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout, keep trying
			}
			fmt.Printf("❌ Error reading from peer %s: %v\n", peer.ID, err)
			break
		}

		// Update peer last seen
		peer.LastSeen = time.Now()

		// Handle message
		if handler, exists := n.messageHandlers[msg.Type]; exists {
			go func() {
				if err := handler(peer, msg); err != nil {
					fmt.Printf("❌ Error handling message from %s: %v\n", peer.ID, err)
				}
			}()
		} else {
			fmt.Printf("⚠️ No handler for message type %d from %s\n", msg.Type, peer.ID)
		}
	}
}

// monitorConnections monitors peer connections and removes stale ones
func (n *TCPNetwork) monitorConnections() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.checkPeerHealth()
		case <-n.stopChan:
			return
		}
	}
}

// checkPeerHealth checks the health of all peers
func (n *TCPNetwork) checkPeerHealth() {
	n.mu.RLock()
	peers := make([]*TCPPeer, 0, len(n.Peers))
	for _, peer := range n.Peers {
		peers = append(peers, peer)
	}
	n.mu.RUnlock()

	for _, peer := range peers {
		if time.Since(peer.LastSeen) > 120*time.Second {
			peer.Status = "offline"
			continue
		}

		// Send ping
		go n.pingPeer(peer)
	}
}

// pingPeer sends a ping to a peer
func (n *TCPNetwork) pingPeer(peer *TCPPeer) {
	pingData := map[string]interface{}{
		"timestamp": time.Now(),
		"node_id":   n.LocalNode.ID,
	}

	if err := n.sendMessageToPeer(peer, MessageTypePing, pingData); err != nil {
		peer.Status = "offline"
		fmt.Printf("❌ Failed to ping peer %s: %v\n", peer.ID, err)
	}
}

// BroadcastMessage broadcasts a message to all connected peers
func (n *TCPNetwork) BroadcastMessage(msgType MessageType, data interface{}) {
	n.mu.RLock()
	peers := make([]*TCPPeer, 0, len(n.Peers))
	for _, peer := range n.Peers {
		if peer.Connected && peer.Status == "online" {
			peers = append(peers, peer)
		}
	}
	n.mu.RUnlock()

	for _, peer := range peers {
		go func(p *TCPPeer) {
			if err := n.sendMessageToPeer(p, msgType, data); err != nil {
				fmt.Printf("❌ Failed to broadcast to peer %s: %v\n", p.ID, err)
			}
		}(peer)
	}

	fmt.Printf("📡 Broadcasted message to %d peers\n", len(peers))
}

// registerDefaultHandlers registers default message handlers
func (n *TCPNetwork) registerDefaultHandlers() {
	n.messageHandlers[MessageTypePing] = n.handlePing
	n.messageHandlers[MessageTypePong] = n.handlePong
	n.messageHandlers[MessageTypeFileRequest] = n.handleFileRequest
	n.messageHandlers[MessageTypeChunkRequest] = n.handleChunkRequest
	n.messageHandlers[MessageTypeBroadcast] = n.handleBroadcast
	n.messageHandlers[MessageTypeNodeDiscovery] = n.handleNodeDiscovery
}

// Message handlers
func (n *TCPNetwork) handlePing(peer *TCPPeer, msg *TCPMessage) error {
	pongData := map[string]interface{}{
		"timestamp": time.Now(),
		"node_id":   n.LocalNode.ID,
	}
	return n.sendMessageToPeer(peer, MessageTypePong, pongData)
}

func (n *TCPNetwork) handlePong(peer *TCPPeer, msg *TCPMessage) error {
	peer.LastPing = time.Now()
	peer.Status = "online"
	return nil
}

func (n *TCPNetwork) handleFileRequest(peer *TCPPeer, msg *TCPMessage) error {
	// Handle file requests - to be implemented based on file distributor integration
	fmt.Printf("📁 File request from %s\n", peer.ID)
	return nil
}

func (n *TCPNetwork) handleChunkRequest(peer *TCPPeer, msg *TCPMessage) error {
	// Handle chunk requests - to be implemented based on chunk distributor integration
	fmt.Printf("🧩 Chunk request from %s\n", peer.ID)
	return nil
}

func (n *TCPNetwork) handleBroadcast(peer *TCPPeer, msg *TCPMessage) error {
	fmt.Printf("📡 Broadcast message from %s\n", peer.ID)
	return nil
}

func (n *TCPNetwork) handleNodeDiscovery(peer *TCPPeer, msg *TCPMessage) error {
	fmt.Printf("🔍 Node discovery from %s\n", peer.ID)
	return nil
}

// GetConnectedPeers returns all connected peers
func (n *TCPNetwork) GetConnectedPeers() []*TCPPeer {
	n.mu.RLock()
	defer n.mu.RUnlock()

	peers := make([]*TCPPeer, 0, len(n.Peers))
	for _, peer := range n.Peers {
		if peer.Connected && peer.Status == "online" {
			peers = append(peers, peer)
		}
	}
	return peers
}

// RegisterMessageHandler registers a custom message handler
func (n *TCPNetwork) RegisterMessageHandler(msgType MessageType, handler MessageHandler) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messageHandlers[msgType] = handler
}
