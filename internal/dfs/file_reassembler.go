package dfs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/jaywantadh/DisktroByte/internal/distributor"
	"github.com/jaywantadh/DisktroByte/internal/encryptor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/p2p"
	"github.com/jaywantadh/DisktroByte/internal/storage"
	"github.com/sirupsen/logrus"
)

// ReassemblyJob represents a file reassembly job
type ReassemblyJob struct {
	ID              string                    `json:"id"`
	FileID          string                    `json:"file_id"`
	FileName        string                    `json:"file_name"`
	OutputPath      string                    `json:"output_path"`
	TotalChunks     int                       `json:"total_chunks"`
	ChunksObtained  int                       `json:"chunks_obtained"`
	Status          string                    `json:"status"`          // "pending", "downloading", "assembling", "verifying", "completed", "failed"
	Progress        float64                   `json:"progress"`        // 0.0 to 100.0
	StartTime       time.Time                 `json:"start_time"`
	CompletionTime  time.Time                 `json:"completion_time"`
	ChunkStatus     map[string]string         `json:"chunk_status"`    // chunk_id -> status
	IntegrityCheck  *IntegrityCheckResult     `json:"integrity_check"`
	ErrorMessage    string                    `json:"error_message"`
}

// IntegrityCheckResult represents the result of integrity verification
type IntegrityCheckResult struct {
	FileHash        string    `json:"file_hash"`
	ExpectedHash    string    `json:"expected_hash"`
	ChunkHashes     map[string]string `json:"chunk_hashes"`
	CorruptedChunks []string  `json:"corrupted_chunks"`
	IsValid         bool      `json:"is_valid"`
	CheckTime       time.Time `json:"check_time"`
}

// ChunkDownloadResult represents the result of downloading a chunk
type ChunkDownloadResult struct {
	ChunkID   string
	Success   bool
	Data      []byte
	Hash      string
	Error     error
	Source    string // Node ID that provided the chunk
}

// FileReassembler handles lossless file reconstruction from distributed chunks
type FileReassembler struct {
	dfsCore      *DFSCore
	distributor  *distributor.Distributor
	storage      storage.Storage
	metaStore    *metadata.MetadataStore
	network      *p2p.Network
	logger       *logrus.Logger
	
	// Job management
	activeJobs   map[string]*ReassemblyJob
	jobHistory   []*ReassemblyJob
	maxHistory   int
}

// NewFileReassembler creates a new file reassembler
func NewFileReassembler(dfsCore *DFSCore, distributor *distributor.Distributor, 
	storage storage.Storage, metaStore *metadata.MetadataStore, network *p2p.Network) *FileReassembler {
	
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &FileReassembler{
		dfsCore:     dfsCore,
		distributor: distributor,
		storage:     storage,
		metaStore:   metaStore,
		network:     network,
		logger:      logger,
		activeJobs:  make(map[string]*ReassemblyJob),
		jobHistory:  make([]*ReassemblyJob, 0),
		maxHistory:  100,
	}
}

// ReassembleFile starts the process of reassembling a distributed file
func (fr *FileReassembler) ReassembleFile(fileID, outputPath, password string) (*ReassemblyJob, error) {
	fr.logger.Infof("🔧 Starting reassembly of file %s", fileID)
	
	// Get file information
	fileInfo, err := fr.distributor.GetFileInfo(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}
	
	// Create reassembly job
	job := &ReassemblyJob{
		ID:             fmt.Sprintf("reassemble-%d", time.Now().UnixNano()),
		FileID:         fileID,
		FileName:       fileInfo.Name,
		OutputPath:     outputPath,
		TotalChunks:    len(fileInfo.Chunks),
		ChunksObtained: 0,
		Status:         "pending",
		Progress:       0.0,
		StartTime:      time.Now(),
		ChunkStatus:    make(map[string]string),
		IntegrityCheck: &IntegrityCheckResult{
			ChunkHashes:     make(map[string]string),
			CorruptedChunks: make([]string, 0),
		},
	}
	
	// Initialize chunk status
	for _, chunkID := range fileInfo.Chunks {
		job.ChunkStatus[chunkID] = "pending"
	}
	
	fr.activeJobs[job.ID] = job
	
	// Start reassembly process asynchronously
	go fr.executeReassembly(job, fileInfo, password)
	
	return job, nil
}

// executeReassembly performs the actual file reassembly
func (fr *FileReassembler) executeReassembly(job *ReassemblyJob, fileInfo *distributor.FileInfo, password string) {
	defer fr.moveJobToHistory(job)
	
	job.Status = "downloading"
	fr.logger.Infof("📥 Downloading %d chunks for file %s", job.TotalChunks, job.FileName)
	
	// Download all chunks with parallel processing
	chunkData, err := fr.downloadAllChunks(job, fileInfo.Chunks)
	if err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to download chunks: %v", err)
		fr.logger.Errorf("❌ Failed to download chunks for %s: %v", job.FileName, err)
		return
	}
	
	job.Status = "assembling"
	job.Progress = 50.0
	fr.logger.Infof("🔨 Assembling file %s from %d chunks", job.FileName, len(chunkData))
	
	// Assemble the file from chunks
	if err := fr.assembleFile(job, chunkData, password); err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Failed to assemble file: %v", err)
		fr.logger.Errorf("❌ Failed to assemble file %s: %v", job.FileName, err)
		return
	}
	
	job.Status = "verifying"
	job.Progress = 85.0
	fr.logger.Infof("🔍 Verifying integrity of reassembled file %s", job.FileName)
	
	// Verify file integrity
	if err := fr.verifyFileIntegrity(job, fileInfo); err != nil {
		job.Status = "failed"
		job.ErrorMessage = fmt.Sprintf("Integrity verification failed: %v", err)
		fr.logger.Errorf("❌ Integrity verification failed for %s: %v", job.FileName, err)
		return
	}
	
	job.Status = "completed"
	job.Progress = 100.0
	job.CompletionTime = time.Now()
	
	duration := job.CompletionTime.Sub(job.StartTime)
	fr.logger.Infof("✅ Successfully reassembled file %s in %v", job.FileName, duration)
}

// downloadAllChunks downloads all chunks for a file with parallel processing
func (fr *FileReassembler) downloadAllChunks(job *ReassemblyJob, chunkIDs []string) (map[int][]byte, error) {
	chunkData := make(map[int][]byte)
	resultChan := make(chan *ChunkDownloadResult, len(chunkIDs))
	
	// Start download workers
	for i, chunkID := range chunkIDs {
		go fr.downloadChunk(chunkID, i, resultChan)
	}
	
	// Collect results
	completedChunks := 0
	for i := 0; i < len(chunkIDs); i++ {
		result := <-resultChan
		
		if result.Success {
			chunkData[i] = result.Data
			job.ChunkStatus[result.ChunkID] = "downloaded"
			job.IntegrityCheck.ChunkHashes[result.ChunkID] = result.Hash
			completedChunks++
			
			fr.logger.Infof("📦 Downloaded chunk %s from node %s", result.ChunkID, result.Source)
		} else {
			job.ChunkStatus[result.ChunkID] = "failed"
			fr.logger.Errorf("❌ Failed to download chunk %s: %v", result.ChunkID, result.Error)
			
			// Try to recover the chunk from other replicas
			if recoveredData, err := fr.recoverChunkFromReplicas(result.ChunkID); err == nil {
				chunkData[i] = recoveredData
				job.ChunkStatus[result.ChunkID] = "recovered"
				completedChunks++
				fr.logger.Infof("🔄 Recovered chunk %s from replicas", result.ChunkID)
			}
		}
		
		job.ChunksObtained = completedChunks
		job.Progress = float64(completedChunks) / float64(job.TotalChunks) * 50.0 // First 50% is downloading
	}
	
	if completedChunks < job.TotalChunks {
		missingChunks := job.TotalChunks - completedChunks
		return nil, fmt.Errorf("failed to download %d chunks", missingChunks)
	}
	
	return chunkData, nil
}

// downloadChunk downloads a specific chunk
func (fr *FileReassembler) downloadChunk(chunkID string, index int, resultChan chan *ChunkDownloadResult) {
	result := &ChunkDownloadResult{
		ChunkID: chunkID,
		Success: false,
	}
	
	// First try to get from local storage
	if data, hash, err := fr.getChunkFromLocalStorage(chunkID); err == nil {
		result.Success = true
		result.Data = data
		result.Hash = hash
		result.Source = fr.network.LocalNode.ID
		resultChan <- result
		return
	}
	
	// Find nodes that have this chunk
	nodesWithChunk := fr.network.FindNodesWithChunk(chunkID)
	if len(nodesWithChunk) == 0 {
		result.Error = fmt.Errorf("no nodes have chunk %s", chunkID)
		resultChan <- result
		return
	}
	
	// Try to download from each node until successful
	for _, node := range nodesWithChunk {
		if data, hash, err := fr.downloadChunkFromNode(chunkID, node); err == nil {
			result.Success = true
			result.Data = data
			result.Hash = hash
			result.Source = node.ID
			resultChan <- result
			return
		} else {
			fr.logger.Warnf("⚠️ Failed to download chunk %s from node %s: %v", chunkID, node.ID, err)
		}
	}
	
	result.Error = fmt.Errorf("failed to download chunk from any node")
	resultChan <- result
}

// getChunkFromLocalStorage retrieves a chunk from local storage
func (fr *FileReassembler) getChunkFromLocalStorage(chunkID string) ([]byte, string, error) {
	reader, err := fr.storage.Get(chunkID)
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	
	// Calculate hash for verification
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	
	return data, hashStr, nil
}

// downloadChunkFromNode downloads a chunk from a specific node
func (fr *FileReassembler) downloadChunkFromNode(chunkID string, node *p2p.Node) ([]byte, string, error) {
	// First check if this is the local node - use local storage
	if node.ID == fr.network.LocalNode.ID {
		// Try to get chunk from local storage using chunkID (UUID)
		if data, hash, err := fr.getChunkFromLocalStorage(chunkID); err == nil {
			return data, hash, nil
		}
		
		// If that fails, try to get from metadata store to find the storage path
		if fr.metaStore != nil {
			if chunkMeta, err := fr.metaStore.GetChunkMetadata(chunkID); err == nil {
				// Use the path from metadata to retrieve chunk
				if reader, err := fr.storage.Get(chunkMeta.Path); err == nil {
					defer reader.Close()
					data, err := io.ReadAll(reader)
					if err != nil {
						return nil, "", fmt.Errorf("failed to read chunk data from path %s: %v", chunkMeta.Path, err)
					}
					
					// Calculate hash for verification
					hash := sha256.Sum256(data)
					hashStr := hex.EncodeToString(hash[:])
					
					return data, hashStr, nil
				}
			}
		}
		
		return nil, "", fmt.Errorf("chunk %s not found locally", chunkID)
	}
		
	// For remote nodes, try to download via HTTP API
	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("http://%s:%d/chunk-request?id=%s", node.Address, node.Port, chunkID)
	
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to request chunk from node %s: %v", node.ID, err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("chunk download failed from node %s: status %d", node.ID, resp.StatusCode)
	}
	
	// Read chunk data
	chunkData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read chunk data from node %s: %v", node.ID, err)
	}
	
	// Calculate hash for verification
	hash := sha256.Sum256(chunkData)
	hashStr := hex.EncodeToString(hash[:])
	
	return chunkData, hashStr, nil
}

// recoverChunkFromReplicas attempts to recover a chunk from alternative replicas
func (fr *FileReassembler) recoverChunkFromReplicas(chunkID string) ([]byte, error) {
	replicaInfo := fr.dfsCore.GetReplicaInfo(chunkID)
	if replicaInfo == nil {
		return nil, fmt.Errorf("no replica information for chunk %s", chunkID)
	}
	
	// Try each replica node
	for _, nodeID := range replicaInfo.CurrentReplicas {
		if nodeID == fr.network.LocalNode.ID {
			// Try local storage again
			if data, _, err := fr.getChunkFromLocalStorage(chunkID); err == nil {
				return data, nil
			}
		} else {
			// Try remote node
			node := fr.network.GetPeerByID(nodeID)
			if node != nil {
				if data, _, err := fr.downloadChunkFromNode(chunkID, node); err == nil {
					return data, nil
				}
			}
		}
	}
	
	return nil, fmt.Errorf("failed to recover chunk from any replica")
}

// assembleFile assembles the final file from downloaded chunks with enhanced metadata validation
func (fr *FileReassembler) assembleFile(job *ReassemblyJob, chunkData map[int][]byte, password string) error {
	// Get chunk metadata for validation and proper ordering
	chunks, err := fr.metaStore.GetChunksByFileID(job.FileID)
	if err != nil {
		return fmt.Errorf("failed to get chunks for FileID %s: %v", job.FileID, err)
	}

	// Validate chunk chain integrity
	if err := metadata.ValidateChunkChain(chunks); err != nil {
		return fmt.Errorf("chunk chain validation failed: %v", err)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(job.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create output file
	outputFile, err := os.Create(job.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Sort chunks by offset to ensure correct order
	sortChunksByOffset(chunks)

	// Initialize encryptor and compressor
	enc := encryptor.NewEncryptor()

	totalBytesWritten := int64(0)
	for i, chunk := range chunks {
		data, exists := chunkData[chunk.Index]
		if !exists {
			return fmt.Errorf("missing chunk data for index %d", chunk.Index)
		}

		// Decrypt chunk
		decrypted := data
		if password != "" {
			decrypted, err = enc.Decrypt(data, password)
			if err != nil {
				return fmt.Errorf("failed to decrypt chunk %d: %v", chunk.Index, err)
			}
		}

		// Decompress chunk if needed
		decompressed := decrypted
		// Try decompression, fallback to original data if it fails
		if decompData, err := compressor.DecompressData(decrypted); err == nil {
			decompressed = decompData
		}

		// Validate chunk hash
		hash := sha256.Sum256(decompressed)
		calculatedHash := hex.EncodeToString(hash[:])
		if calculatedHash != chunk.Hash {
			return fmt.Errorf("hash mismatch for chunk %d: expected %s, got %s", 
				chunk.Index, chunk.Hash, calculatedHash)
		}

		// Write decompressed data to output file
		bytesWritten, err := outputFile.Write(decompressed)
		if err != nil {
			return fmt.Errorf("failed to write chunk %d: %v", chunk.Index, err)
		}

		totalBytesWritten += int64(bytesWritten)

		// Update progress
		progress := 50.0 + (float64(i+1)/float64(len(chunks)))*35.0
		job.Progress = progress
	}
	
	fr.logger.Infof("📝 Wrote %d bytes to %s", totalBytesWritten, job.OutputPath)
	return nil
}

// verifyFileIntegrity verifies the integrity of the reassembled file using FileID
func (fr *FileReassembler) verifyFileIntegrity(job *ReassemblyJob, fileInfo *distributor.FileInfo) error {
	// Calculate hash of reassembled file
	fileHash, err := fr.calculateFileHash(job.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %v", err)
	}

	job.IntegrityCheck.FileHash = fileHash
	job.IntegrityCheck.CheckTime = time.Now()

	// The expected hash should match the FileID (which is the original file's SHA-256)
	job.IntegrityCheck.ExpectedHash = job.FileID
	job.IntegrityCheck.IsValid = (fileHash == job.FileID)

	if !job.IntegrityCheck.IsValid {
		return fmt.Errorf("file hash mismatch: expected %s (FileID), got %s", 
			job.FileID, fileHash)
	}

	// Verify individual chunk hashes using enhanced metadata
	chunks, err := fr.metaStore.GetChunksByFileID(job.FileID)
	if err != nil {
		fr.logger.Warnf("⚠️ Could not get chunks for FileID %s: %v", job.FileID, err)
	} else {
		corruptedChunks := make([]string, 0)
		for _, chunk := range chunks {
			if actualHash, exists := job.IntegrityCheck.ChunkHashes[chunk.Hash]; exists {
				if chunk.Hash != actualHash {
					corruptedChunks = append(corruptedChunks, chunk.Hash)
					fr.logger.Warnf("❌ Chunk %s is corrupted (expected: %s, actual: %s)", 
						chunk.Hash, chunk.Hash, actualHash)
				}
			}
		}

		job.IntegrityCheck.CorruptedChunks = corruptedChunks

		if len(corruptedChunks) > 0 {
			job.IntegrityCheck.IsValid = false
			return fmt.Errorf("%d chunks are corrupted", len(corruptedChunks))
		}
	}

	fr.logger.Infof("✅ File integrity verification passed for %s (FileID: %s)", 
		job.FileName, job.FileID)
	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func (fr *FileReassembler) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

// GetJob returns information about a reassembly job
func (fr *FileReassembler) GetJob(jobID string) *ReassemblyJob {
	if job, exists := fr.activeJobs[jobID]; exists {
		return job
	}
	
	// Search in job history
	for _, job := range fr.jobHistory {
		if job.ID == jobID {
			return job
		}
	}
	
	return nil
}

// GetActiveJobs returns all active reassembly jobs
func (fr *FileReassembler) GetActiveJobs() []*ReassemblyJob {
	jobs := make([]*ReassemblyJob, 0, len(fr.activeJobs))
	for _, job := range fr.activeJobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJobHistory returns completed job history
func (fr *FileReassembler) GetJobHistory() []*ReassemblyJob {
	return fr.jobHistory
}

// CancelJob cancels an active reassembly job
func (fr *FileReassembler) CancelJob(jobID string) error {
	job, exists := fr.activeJobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found or already completed", jobID)
	}
	
	job.Status = "cancelled"
	job.ErrorMessage = "Job cancelled by user"
	job.CompletionTime = time.Now()
	
	fr.moveJobToHistory(job)
	fr.logger.Infof("🚫 Cancelled reassembly job %s", jobID)
	
	return nil
}

// moveJobToHistory moves a job from active to history
func (fr *FileReassembler) moveJobToHistory(job *ReassemblyJob) {
	delete(fr.activeJobs, job.ID)
	
	fr.jobHistory = append(fr.jobHistory, job)
	
	// Maintain history size limit
	if len(fr.jobHistory) > fr.maxHistory {
		fr.jobHistory = fr.jobHistory[len(fr.jobHistory)-fr.maxHistory:]
	}
}

// GetReassemblyStats returns statistics about file reassembly operations
func (fr *FileReassembler) GetReassemblyStats() map[string]interface{} {
	activeCount := len(fr.activeJobs)
	totalJobs := activeCount + len(fr.jobHistory)
	
	completedJobs := 0
	failedJobs := 0
	totalDuration := time.Duration(0)
	
	for _, job := range fr.jobHistory {
		switch job.Status {
		case "completed":
			completedJobs++
			if !job.CompletionTime.IsZero() {
				totalDuration += job.CompletionTime.Sub(job.StartTime)
			}
		case "failed", "cancelled":
			failedJobs++
		}
	}
	
	stats := map[string]interface{}{
		"active_jobs":    activeCount,
		"total_jobs":     totalJobs,
		"completed_jobs": completedJobs,
		"failed_jobs":    failedJobs,
		"success_rate":   0.0,
		"avg_duration":   "0s",
	}
	
	if totalJobs > 0 {
		stats["success_rate"] = float64(completedJobs) / float64(totalJobs) * 100.0
	}
	
	if completedJobs > 0 {
		avgDuration := totalDuration / time.Duration(completedJobs)
		stats["avg_duration"] = avgDuration.String()
	}
	
	return stats
}

// sortChunksByOffset sorts chunks by their offset in the original file
func sortChunksByOffset(chunks []metadata.ChunkMetadata) {
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Offset < chunks[j].Offset
	})
}
