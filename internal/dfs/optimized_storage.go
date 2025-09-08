package dfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
	"github.com/sirupsen/logrus"
)

// OptimizedStorage integrates the storage optimization engine with DFS
type OptimizedStorage struct {
	// Core components
	basePath         string
	optimizationEngine *storage.OptimizationEngine
	enhancedMetadata *metadata.EnhancedMetadataStore
	logger           *logrus.Logger
	
	// Statistics
	statsMu          sync.RWMutex
	stats            map[string]interface{}
	lastStatsUpdate  time.Time
	statsUpdateChan  chan bool
	stopChan         chan bool
	wg               sync.WaitGroup
}

// NewOptimizedStorage creates a new optimized storage layer
func NewOptimizedStorage(basePath string) (*OptimizedStorage, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %v", err)
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	// Create optimization engine
	optimizationEngine := storage.NewOptimizationEngine(
		filepath.Join(basePath, "optimized_chunks"), 
		nil, // Use default config
	)
	
	// Start optimization engine
	if err := optimizationEngine.Start(); err != nil {
		return nil, fmt.Errorf("failed to start optimization engine: %v", err)
	}
	
	// Create enhanced metadata store
	enhancedMetadata, err := metadata.NewEnhancedMetadataStore(
		filepath.Join(basePath, "enhanced_metadata"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create enhanced metadata store: %v", err)
	}
	
	optimizedStorage := &OptimizedStorage{
		basePath:          basePath,
		optimizationEngine: optimizationEngine,
		enhancedMetadata:  enhancedMetadata,
		logger:            logger,
		stats:             make(map[string]interface{}),
		statsUpdateChan:   make(chan bool, 10),
		stopChan:          make(chan bool),
	}
	
	// Start background statistics update
	optimizedStorage.wg.Add(1)
	go optimizedStorage.backgroundStatsUpdater()
	
	optimizedStorage.logger.Info("🚀 Optimized Storage initialized")
	return optimizedStorage, nil
}

// Close shuts down the optimized storage
func (os *OptimizedStorage) Close() error {
	// Signal background workers to stop
	close(os.stopChan)
	os.wg.Wait()
	
	// Close components
	if err := os.optimizationEngine.Stop(); err != nil {
		os.logger.Errorf("❌ Failed to stop optimization engine: %v", err)
	}
	
	if err := os.enhancedMetadata.Close(); err != nil {
		os.logger.Errorf("❌ Failed to close enhanced metadata store: %v", err)
	}
	
	return nil
}

// StoreChunk stores a chunk with optimization
func (os *OptimizedStorage) StoreChunk(chunkData io.Reader, chunkID, fileID string, index int, storageNodes []string) (string, error) {
	// Use optimization engine to store chunk
	hashID, chunkInfo, err := os.optimizationEngine.OptimizedPut(chunkData)
	if err != nil {
		return "", fmt.Errorf("failed to store chunk: %v", err)
	}
	
	// Create enhanced chunk metadata
	chunkMeta := &metadata.EnhancedChunkMetadata{
		ChunkID:         chunkID,
		FileID:          fileID,
		Index:           index,
		Hash:            hashID,
		Size:            chunkInfo.Size,
		Path:            chunkInfo.StoragePath,
		IsCompressed:    chunkInfo.IsCompressed,
		CompressedSize:  chunkInfo.CompressedSize,
		CompressionRatio: chunkInfo.CompressionRatio,
		IsDeduplicated:  chunkInfo.IsDeduplicated,
		ReferenceCount:  chunkInfo.ReferenceCount,
		StorageNodes:    storageNodes,
		ReplicaHealth:   make(map[string]string),
		PrimaryNode:     storageNodes[0],
		AccessCount:     0,
		LastAccessTime:  time.Now(),
		AccessPattern:   "sequential",
		HealthStatus:    "healthy",
		LastVerified:    time.Now(),
		ErrorCount:      0,
		CreatedAt:       time.Now(),
		ModifiedAt:      time.Now(),
	}
	
	// Initialize replica health status
	for _, node := range storageNodes {
		chunkMeta.ReplicaHealth[node] = "healthy"
	}
	
	// Store enhanced metadata
	if err := os.enhancedMetadata.StoreChunkMetadata(chunkMeta); err != nil {
		os.logger.Warnf("⚠️ Failed to store enhanced chunk metadata: %v", err)
	}
	
	// Trigger stats update
	select {
	case os.statsUpdateChan <- true:
	default:
	}
	
	return hashID, nil
}

// RetrieveChunk retrieves a chunk with optimization
func (os *OptimizedStorage) RetrieveChunk(chunkID string) (io.ReadCloser, error) {
	// Get chunk using optimization engine
	reader, _, err := os.optimizationEngine.OptimizedGet(chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunk: %v", err)
	}
	
	// Try to update enhanced metadata
	go func() {
		if chunkMeta, err := os.enhancedMetadata.GetChunkMetadata(chunkID); err == nil {
			// Metadata exists, update access info
			chunkMeta.LastAccessTime = time.Now()
			chunkMeta.AccessCount++
			chunkMeta.ModifiedAt = time.Now()
			
			if err := os.enhancedMetadata.StoreChunkMetadata(chunkMeta); err != nil {
				os.logger.Warnf("⚠️ Failed to update chunk metadata: %v", err)
			}
		}
	}()
	
	return reader, nil
}

// StoreFileMetadata stores comprehensive file metadata
func (os *OptimizedStorage) StoreFileMetadata(meta *metadata.EnhancedFileMetadata) error {
	return os.enhancedMetadata.StoreFileMetadata(meta)
}

// GetFileMetadata retrieves comprehensive file metadata
func (os *OptimizedStorage) GetFileMetadata(fileID string) (*metadata.EnhancedFileMetadata, error) {
	return os.enhancedMetadata.GetFileMetadata(fileID)
}

// VersionFile creates a new version of a file
func (os *OptimizedStorage) VersionFile(fileID, createdBy, changeLog string) (*metadata.FileVersion, error) {
	return os.enhancedMetadata.CreateFileVersion(fileID, createdBy, changeLog)
}

// CreateFileRelationship creates a relationship between files
func (os *OptimizedStorage) CreateFileRelationship(sourceFileID, targetFileID, relationType, createdBy string) error {
	_, err := os.enhancedMetadata.CreateFileRelationship(sourceFileID, targetFileID, relationType, createdBy, 1.0)
	return err
}

// SearchFiles performs advanced file search
func (os *OptimizedStorage) SearchFiles(query *metadata.SearchQuery) (*metadata.SearchResult, error) {
	return os.enhancedMetadata.SearchFiles(query)
}

// GetFileVersions gets all versions of a file
func (os *OptimizedStorage) GetFileVersions(fileID string) ([]*metadata.FileVersion, error) {
	// TODO: Implement this method in enhanced metadata store
	// For now, return empty list
	return []*metadata.FileVersion{}, nil
}

// GetFileRelationships gets all relationships for a file
func (os *OptimizedStorage) GetFileRelationships(fileID string) ([]*metadata.FileRelationship, error) {
	// TODO: Implement this method in enhanced metadata store
	// For now, return empty list
	return []*metadata.FileRelationship{}, nil
}

// GetStorageStats returns comprehensive storage statistics
func (os *OptimizedStorage) GetStorageStats() map[string]interface{} {
	os.statsMu.RLock()
	defer os.statsMu.RUnlock()
	
	// Create a copy to avoid race conditions
	stats := make(map[string]interface{})
	for k, v := range os.stats {
		stats[k] = v
	}
	
	return stats
}

// UpdateStorageStats updates storage statistics
func (os *OptimizedStorage) UpdateStorageStats() {
	os.statsMu.Lock()
	defer os.statsMu.Unlock()
	
	// Get optimization engine stats
	optimizationReport := os.optimizationEngine.GetOptimizationReport()
	
	// Get file and chunk counts
	fileCount := 0
	chunkCount := 0
	
	// Construct storage statistics
	os.stats = map[string]interface{}{
		"optimization":   optimizationReport,
		"file_count":     fileCount,
		"chunk_count":    chunkCount,
		"last_updated":   time.Now(),
	}
	
	os.lastStatsUpdate = time.Now()
}

// backgroundStatsUpdater periodically updates storage statistics
func (os *OptimizedStorage) backgroundStatsUpdater() {
	defer os.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			os.UpdateStorageStats()
		case <-os.statsUpdateChan:
			// Only update if it's been at least 1 minute since last update
			if time.Since(os.lastStatsUpdate) > time.Minute {
				os.UpdateStorageStats()
			}
		case <-os.stopChan:
			return
		}
	}
}
