package distributor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaywantadh/DisktroByte/internal/chunker"
	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/p2p"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

// FileInfo represents information about a distributed file
type FileInfo struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Chunks     []string  `json:"chunks"`
	Replicas   int       `json:"replicas"`
	CreatedAt  time.Time `json:"created_at"`
	Owner      string    `json:"owner"`
	Compressed bool      `json:"compressed"`
	Encrypted  bool      `json:"encrypted"`
	Nodes      []string  `json:"nodes"` // List of nodes that have this file
}

// ChunkInfo represents information about a chunk
type ChunkInfo struct {
	ID        string    `json:"id"`
	FileID    string    `json:"file_id"`
	Index     int       `json:"index"`
	Size      int64     `json:"size"`
	Hash      string    `json:"hash"`
	Nodes     []string  `json:"nodes"` // List of nodes that have this chunk
	Replicas  int       `json:"replicas"`
	CreatedAt time.Time `json:"created_at"`
}

// Distributor manages file distribution across the P2P network
type Distributor struct {
	network      *p2p.Network
	store        storage.Storage
	metaStore    *metadata.MetadataStore
	files        map[string]*FileInfo
	chunks       map[string]*ChunkInfo
	mu           sync.RWMutex
	replicaCount int
}

// NewDistributor creates a new file distributor
func NewDistributor(network *p2p.Network, store storage.Storage, metaStore *metadata.MetadataStore) *Distributor {
	return &Distributor{
		network:      network,
		store:        store,
		metaStore:    metaStore,
		files:        make(map[string]*FileInfo),
		chunks:       make(map[string]*ChunkInfo),
		replicaCount: 3, // Default replica count
	}
}

// SetReplicaCount sets the number of replicas for each chunk
func (d *Distributor) SetReplicaCount(count int) {
	d.replicaCount = count
}

// DistributeFile distributes a file across the P2P network
func (d *Distributor) DistributeFile(filePath, password string) (*FileInfo, error) {
	// Generate file ID
	fileID := uuid.New().String()
	fileName := filepath.Base(filePath)

	// Get file size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}

	// Create file record
	file := &FileInfo{
		ID:         fileID,
		Name:       fileName,
		Size:       fileInfo.Size(),
		Chunks:     make([]string, 0),
		Replicas:   d.replicaCount,
		CreatedAt:  time.Now(),
		Owner:      d.network.LocalNode.ID,
		Compressed: false,
		Encrypted:  true,
		Nodes:      []string{d.network.LocalNode.ID},
	}

	// Chunk the file
	chunkMetadata, err := chunker.ChunkAndStore(filePath, password, d.metaStore, d.store)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk file: %v", err)
	}

	// Check if file should be compressed
	shouldCompress := !compressor.ShouldSkipCompression(fileName)
	if shouldCompress {
		file.Compressed = true
		// Note: Compression would be applied during chunking
	}

	// Process each chunk
	for i, chunkMeta := range chunkMetadata {
		chunkID := uuid.New().String()

		chunk := &ChunkInfo{
			ID:        chunkID,
			FileID:    fileID,
			Index:     i,
			Size:      chunkMeta.Size,
			Hash:      chunkMeta.Hash,
			Nodes:     []string{d.network.LocalNode.ID},
			Replicas:  d.replicaCount,
			CreatedAt: time.Now(),
		}

		// Store chunk info
		d.mu.Lock()
		d.chunks[chunkID] = chunk
		file.Chunks = append(file.Chunks, chunkID)
		d.mu.Unlock()

		// Add chunk to local node
		d.network.AddChunkToNode(d.network.LocalNode.ID, chunkID)

		// Distribute chunk to other nodes
		go d.distributeChunk(chunk, &chunkMeta)
	}

	// Store file info
	d.mu.Lock()
	d.files[fileID] = file
	d.mu.Unlock()

	// Add file to local node
	d.network.AddFileToNode(d.network.LocalNode.ID, fileID)

	// Broadcast file availability
	d.broadcastFileAvailability(file)

	fmt.Printf("📦 File '%s' distributed with %d chunks\n", fileName, len(chunkMetadata))
	return file, nil
}

// distributeChunk distributes a chunk to multiple nodes for redundancy
func (d *Distributor) distributeChunk(chunk *ChunkInfo, chunkMeta *chunker.ChunkMetadata) {
	peers := d.network.GetPeers()

	// Sort peers by reliability (online status, last seen, etc.)
	reliablePeers := d.getReliablePeers(peers)

	// Distribute to reliable peers
	replicasCreated := 0
	for _, peer := range reliablePeers {
		if replicasCreated >= d.replicaCount-1 { // -1 because we already have it locally
			break
		}

		if d.sendChunkToPeer(chunk, chunkMeta, peer) {
			replicasCreated++
			chunk.Nodes = append(chunk.Nodes, peer.ID)
			d.network.AddChunkToNode(peer.ID, chunk.ID)
		}
	}

	fmt.Printf("🔄 Chunk %s distributed to %d nodes\n", chunk.ID, replicasCreated+1)
}

// getReliablePeers returns peers sorted by reliability
func (d *Distributor) getReliablePeers(peers []*p2p.Node) []*p2p.Node {
	// Simple reliability scoring based on status and last seen time
	var reliablePeers []*p2p.Node

	for _, peer := range peers {
		if peer.Status == "online" {
			// Check if peer was seen recently (within last 5 minutes)
			if time.Since(peer.LastSeen) < 5*time.Minute {
				reliablePeers = append(reliablePeers, peer)
			}
		}
	}

	return reliablePeers
}

// sendChunkToPeer sends a chunk to a specific peer
func (d *Distributor) sendChunkToPeer(chunk *ChunkInfo, chunkMeta *chunker.ChunkMetadata, peer *p2p.Node) bool {
	client := &http.Client{Timeout: 30 * time.Second}

	// Create chunk transfer request
	transferReq := map[string]interface{}{
		"chunk_id":  chunk.ID,
		"file_id":   chunk.FileID,
		"index":     chunk.Index,
		"size":      chunk.Size,
		"hash":      chunk.Hash,
		"from_node": d.network.LocalNode.ID,
	}

	reqData, err := json.Marshal(transferReq)
	if err != nil {
		fmt.Printf("❌ Failed to marshal chunk transfer request: %v\n", err)
		return false
	}

	// Send chunk data
	url := fmt.Sprintf("http://%s:%d/chunk-transfer", peer.Address, peer.Port)

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(reqData))
	if err != nil {
		fmt.Printf("❌ Failed to send chunk to %s: %v\n", peer.ID, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("✅ Chunk %s sent to peer %s\n", chunk.ID, peer.ID)
		return true
	}

	fmt.Printf("⚠️ Failed to send chunk to %s: status %d\n", peer.ID, resp.StatusCode)
	return false
}

// ReassembleFile reassembles a file from distributed chunks
func (d *Distributor) ReassembleFile(fileID, outputPath, password string) error {
	d.mu.RLock()
	file, exists := d.files[fileID]
	d.mu.RUnlock()

	if !exists {
		return fmt.Errorf("file %s not found", fileID)
	}

	// Check if we have all chunks locally
	missingChunks := d.findMissingChunks(file.Chunks)
	if len(missingChunks) > 0 {
		// Download missing chunks from peers
		if err := d.downloadMissingChunks(missingChunks); err != nil {
			return fmt.Errorf("failed to download missing chunks: %v", err)
		}
	}

	// Reassemble the file
	err := chunker.ReassembleFile(file.Name, outputPath, password, d.metaStore, d.store)
	if err != nil {
		return fmt.Errorf("failed to reassemble file: %v", err)
	}

	fmt.Printf("🔧 File '%s' reassembled successfully\n", file.Name)
	return nil
}

// findMissingChunks finds chunks that are not available locally
func (d *Distributor) findMissingChunks(chunkIDs []string) []string {
	var missingChunks []string

	for _, chunkID := range chunkIDs {
		d.mu.RLock()
		chunk, exists := d.chunks[chunkID]
		d.mu.RUnlock()

		if !exists {
			missingChunks = append(missingChunks, chunkID)
			continue
		}

		// Check if we have this chunk locally
		hasChunk := false
		for _, nodeID := range chunk.Nodes {
			if nodeID == d.network.LocalNode.ID {
				hasChunk = true
				break
			}
		}

		if !hasChunk {
			missingChunks = append(missingChunks, chunkID)
		}
	}

	return missingChunks
}

// downloadMissingChunks downloads missing chunks from peers
func (d *Distributor) downloadMissingChunks(chunkIDs []string) error {
	for _, chunkID := range chunkIDs {
		// Find nodes that have this chunk
		nodes := d.network.FindNodesWithChunk(chunkID)
		if len(nodes) == 0 {
			return fmt.Errorf("no nodes have chunk %s", chunkID)
		}

		// Try to download from the first available node
		if err := d.downloadChunkFromNode(chunkID, nodes[0]); err != nil {
			fmt.Printf("⚠️ Failed to download chunk %s from %s: %v\n", chunkID, nodes[0].ID, err)
			// Try next node if available
			if len(nodes) > 1 {
				if err := d.downloadChunkFromNode(chunkID, nodes[1]); err != nil {
					return fmt.Errorf("failed to download chunk %s from any node", chunkID)
				}
			} else {
				return fmt.Errorf("failed to download chunk %s", chunkID)
			}
		}
	}

	return nil
}

// downloadChunkFromNode downloads a chunk from a specific node
func (d *Distributor) downloadChunkFromNode(chunkID string, node *p2p.Node) error {
	client := &http.Client{Timeout: 30 * time.Second}

	url := fmt.Sprintf("http://%s:%d/chunk/%s", node.Address, node.Port, chunkID)

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to request chunk: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download chunk: status %d", resp.StatusCode)
	}

	// Read chunk data
	chunkData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read chunk data: %v", err)
	}

	// Store chunk locally using the storage interface
	chunkReader := bytes.NewReader(chunkData)
	if _, err := d.store.Put(chunkReader); err != nil {
		return fmt.Errorf("failed to store chunk: %v", err)
	}

	// Update chunk info
	d.mu.Lock()
	if chunk, exists := d.chunks[chunkID]; exists {
		chunk.Nodes = append(chunk.Nodes, d.network.LocalNode.ID)
	}
	d.mu.Unlock()

	// Add chunk to local node
	d.network.AddChunkToNode(d.network.LocalNode.ID, chunkID)

	fmt.Printf("📥 Downloaded chunk %s from %s\n", chunkID, node.ID)
	return nil
}

// GetFileInfo returns information about a file
func (d *Distributor) GetFileInfo(fileID string) (*FileInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	file, exists := d.files[fileID]
	if !exists {
		return nil, fmt.Errorf("file %s not found", fileID)
	}

	return file, nil
}

// GetAllFiles returns all files in the network
func (d *Distributor) GetAllFiles() []*FileInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	files := make([]*FileInfo, 0, len(d.files))
	for _, file := range d.files {
		files = append(files, file)
	}

	return files
}

// GetChunkInfo returns information about a chunk
func (d *Distributor) GetChunkInfo(chunkID string) (*ChunkInfo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	chunk, exists := d.chunks[chunkID]
	if !exists {
		return nil, fmt.Errorf("chunk %s not found", chunkID)
	}

	return chunk, nil
}

// broadcastFileAvailability broadcasts file availability to all peers
func (d *Distributor) broadcastFileAvailability(file *FileInfo) {
	msg := &p2p.NetworkMessage{
		Type:      "file_available",
		From:      d.network.LocalNode.ID,
		To:        "",
		Data:      file,
		Timestamp: time.Now(),
	}

	d.network.BroadcastMessage(msg)
}

// HandleChunkTransfer handles incoming chunk transfer requests
func (d *Distributor) HandleChunkTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var transferReq map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&transferReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	chunkID := transferReq["chunk_id"].(string)
	fileID := transferReq["file_id"].(string)
	index := int(transferReq["index"].(float64))
	size := int64(transferReq["size"].(float64))
	hash := transferReq["hash"].(string)
	fromNode := transferReq["from_node"].(string)

	// Create chunk info
	chunk := &ChunkInfo{
		ID:        chunkID,
		FileID:    fileID,
		Index:     index,
		Size:      size,
		Hash:      hash,
		Nodes:     []string{d.network.LocalNode.ID, fromNode},
		Replicas:  d.replicaCount,
		CreatedAt: time.Now(),
	}

	// Store chunk info
	d.mu.Lock()
	d.chunks[chunkID] = chunk
	d.mu.Unlock()

	// Add chunk to local node
	d.network.AddChunkToNode(d.network.LocalNode.ID, chunkID)

	w.WriteHeader(http.StatusOK)
	fmt.Printf("📥 Received chunk %s from %s\n", chunkID, fromNode)
}

// HandleChunkRequest handles requests for chunk data
func (d *Distributor) HandleChunkRequest(w http.ResponseWriter, r *http.Request) {
	chunkID := r.URL.Query().Get("id")
	if chunkID == "" {
		http.Error(w, "Chunk ID required", http.StatusBadRequest)
		return
	}

	// Check if we have this chunk
	d.mu.RLock()
	chunk, exists := d.chunks[chunkID]
	d.mu.RUnlock()

	if !exists {
		http.Error(w, "Chunk not found", http.StatusNotFound)
		return
	}

	// Check if we have the chunk locally
	hasChunk := false
	for _, nodeID := range chunk.Nodes {
		if nodeID == d.network.LocalNode.ID {
			hasChunk = true
			break
		}
	}

	if !hasChunk {
		http.Error(w, "Chunk not available locally", http.StatusNotFound)
		return
	}

	// Read chunk data
	chunkReader, err := d.store.Get(chunkID)
	if err != nil {
		http.Error(w, "Failed to read chunk", http.StatusInternalServerError)
		return
	}
	defer chunkReader.Close()

	chunkData, err := io.ReadAll(chunkReader)
	if err != nil {
		http.Error(w, "Failed to read chunk data", http.StatusInternalServerError)
		return
	}

	// Send chunk data
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(chunkData)
}
