package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/sirupsen/logrus"
)

// OptimizationConfig holds configuration for storage optimization
type OptimizationConfig struct {
	EnableDeduplication  bool          `json:"enable_deduplication"`
	EnableCompression    bool          `json:"enable_compression"`
	EnableIntelligentCache bool        `json:"enable_intelligent_cache"`
	MaxCacheSize         int64         `json:"max_cache_size"`          // bytes
	CacheEvictionPolicy  string        `json:"cache_eviction_policy"`   // "lru", "lfu", "fifo"
	CompressionThreshold int64         `json:"compression_threshold"`   // minimum size to compress
	DeduplicationWindow  time.Duration `json:"deduplication_window"`    // time window for dedup analysis
	OptimizationInterval time.Duration `json:"optimization_interval"`   // how often to run optimization
	AnalyticsRetention   time.Duration `json:"analytics_retention"`     // how long to keep analytics data
}

// DefaultOptimizationConfig returns a default configuration
func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		EnableDeduplication:  true,
		EnableCompression:    true,
		EnableIntelligentCache: true,
		MaxCacheSize:         1024 * 1024 * 1024, // 1GB
		CacheEvictionPolicy:  "lru",
		CompressionThreshold: 1024,               // 1KB minimum
		DeduplicationWindow:  24 * time.Hour,     // 24 hours
		OptimizationInterval: 30 * time.Minute,   // 30 minutes
		AnalyticsRetention:   7 * 24 * time.Hour, // 7 days
	}
}

// ChunkInfo represents detailed information about a stored chunk
type ChunkInfo struct {
	Hash            string    `json:"hash"`
	Size            int64     `json:"size"`
	CompressedSize  int64     `json:"compressed_size"`
	IsCompressed    bool      `json:"is_compressed"`
	IsDeduplicated  bool      `json:"is_deduplicated"`
	ReferenceCount  int       `json:"reference_count"`
	CreatedAt       time.Time `json:"created_at"`
	LastAccessedAt  time.Time `json:"last_accessed_at"`
	AccessCount     int64     `json:"access_count"`
	StoragePath     string    `json:"storage_path"`
	DeduplicationKey string   `json:"deduplication_key"`
	CompressionRatio float64  `json:"compression_ratio"`
}

// CacheEntry represents a cache entry
type CacheEntry struct {
	Data           []byte    `json:"data"`
	Hash           string    `json:"hash"`
	Size           int64     `json:"size"`
	CreatedAt      time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
	AccessCount    int64     `json:"access_count"`
	Priority       float64   `json:"priority"`
}

// DeduplicationIndex tracks chunk deduplication
type DeduplicationIndex struct {
	ContentHash   string   `json:"content_hash"`
	ChunkHashes   []string `json:"chunk_hashes"`
	Size          int64    `json:"size"`
	Count         int      `json:"count"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
}

// StorageAnalytics tracks storage performance and usage
type StorageAnalytics struct {
	TotalStorageUsed       int64                    `json:"total_storage_used"`
	CompressedStorageUsed  int64                    `json:"compressed_storage_used"`
	DeduplicationSavings   int64                    `json:"deduplication_savings"`
	CompressionSavings     int64                    `json:"compression_savings"`
	CacheHitRate          float64                  `json:"cache_hit_rate"`
	CacheSize             int64                    `json:"cache_size"`
	TotalChunks           int64                    `json:"total_chunks"`
	UniqueChunks          int64                    `json:"unique_chunks"`
	DuplicateChunks       int64                    `json:"duplicate_chunks"`
	AverageChunkSize      float64                  `json:"average_chunk_size"`
	AverageCompressionRatio float64                `json:"average_compression_ratio"`
	AccessPatterns        map[string]int64         `json:"access_patterns"`
	HourlyStats           map[string]*HourlyStats  `json:"hourly_stats"`
	LastOptimization      time.Time                `json:"last_optimization"`
}

// HourlyStats tracks hourly performance metrics
type HourlyStats struct {
	Hour            string  `json:"hour"`
	Reads           int64   `json:"reads"`
	Writes          int64   `json:"writes"`
	CacheHits       int64   `json:"cache_hits"`
	CacheMisses     int64   `json:"cache_misses"`
	CompressionOps  int64   `json:"compression_ops"`
	DecompressionOps int64  `json:"decompression_ops"`
	DeduplicationOps int64  `json:"deduplication_ops"`
	AverageLatency  float64 `json:"average_latency"`
}

// OptimizationEngine manages storage optimization
type OptimizationEngine struct {
	config           *OptimizationConfig
	basePath         string
	logger           *logrus.Logger
	
	// Chunk tracking
	chunks           map[string]*ChunkInfo
	chunksMu         sync.RWMutex
	
	// Deduplication
	deduplicationIndex map[string]*DeduplicationIndex
	dedupMu            sync.RWMutex
	
	// Intelligent cache
	cache             map[string]*CacheEntry
	cacheSize         int64
	cacheMu           sync.RWMutex
	
	// Analytics
	analytics         *StorageAnalytics
	analyticsMu       sync.RWMutex
	
	// Background tasks
	optimizationTicker *time.Ticker
	stopChan          chan bool
}

// NewOptimizationEngine creates a new storage optimization engine
func NewOptimizationEngine(basePath string, config *OptimizationConfig) *OptimizationEngine {
	if config == nil {
		config = DefaultOptimizationConfig()
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &OptimizationEngine{
		config:             config,
		basePath:           basePath,
		logger:             logger,
		chunks:             make(map[string]*ChunkInfo),
		deduplicationIndex: make(map[string]*DeduplicationIndex),
		cache:              make(map[string]*CacheEntry),
		cacheSize:          0,
		analytics:          &StorageAnalytics{
			AccessPatterns: make(map[string]int64),
			HourlyStats:    make(map[string]*HourlyStats),
		},
		stopChan: make(chan bool),
	}
}

// Start initializes and starts the optimization engine
func (oe *OptimizationEngine) Start() error {
	oe.logger.Info("🚀 Starting Storage Optimization Engine")
	
	// Ensure storage directories exist
	if err := os.MkdirAll(oe.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %v", err)
	}
	
	// Load existing chunk index
	if err := oe.loadChunkIndex(); err != nil {
		oe.logger.Warnf("⚠️ Failed to load chunk index: %v", err)
	}
	
	// Load deduplication index
	if err := oe.loadDeduplicationIndex(); err != nil {
		oe.logger.Warnf("⚠️ Failed to load deduplication index: %v", err)
	}
	
	// Initialize analytics
	oe.initializeAnalytics()
	
	// Start background optimization
	if oe.config.OptimizationInterval > 0 {
		oe.optimizationTicker = time.NewTicker(oe.config.OptimizationInterval)
		go oe.backgroundOptimization()
	}
	
	oe.logger.Info("✅ Storage Optimization Engine started successfully")
	return nil
}

// Stop shuts down the optimization engine
func (oe *OptimizationEngine) Stop() error {
	oe.logger.Info("🛑 Stopping Storage Optimization Engine")
	
	if oe.optimizationTicker != nil {
		oe.optimizationTicker.Stop()
	}
	close(oe.stopChan)
	
	// Save indices
	if err := oe.saveChunkIndex(); err != nil {
		oe.logger.Errorf("❌ Failed to save chunk index: %v", err)
	}
	
	if err := oe.saveDeduplicationIndex(); err != nil {
		oe.logger.Errorf("❌ Failed to save deduplication index: %v", err)
	}
	
	oe.logger.Info("✅ Storage Optimization Engine stopped")
	return nil
}

// OptimizedPut stores a chunk with optimization (compression, deduplication)
func (oe *OptimizationEngine) OptimizedPut(chunkData io.Reader) (string, *ChunkInfo, error) {
	start := time.Now()
	
	// Read data into memory
	data, err := io.ReadAll(chunkData)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read chunk data: %v", err)
	}
	
	// Calculate hash for deduplication
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	
	// Check for deduplication
	if oe.config.EnableDeduplication {
		if existing := oe.findDuplicateChunk(hashStr, data); existing != nil {
			existing.ReferenceCount++
			existing.LastAccessedAt = time.Now()
			existing.AccessCount++
			
			oe.updateAnalytics("deduplication", len(data), time.Since(start))
			oe.logger.Debugf("🔄 Deduplicated chunk %s (refs: %d)", hashStr, existing.ReferenceCount)
			return hashStr, existing, nil
		}
	}
	
	chunkInfo := &ChunkInfo{
		Hash:           hashStr,
		Size:           int64(len(data)),
		IsCompressed:   false,
		IsDeduplicated: false,
		ReferenceCount: 1,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		AccessCount:    1,
	}
	
	// Apply compression if enabled and beneficial
	finalData := data
	if oe.config.EnableCompression && int64(len(data)) >= oe.config.CompressionThreshold {
		if compressed, err := compressor.CompressChunk(data); err == nil {
			compressionRatio := float64(len(data)) / float64(len(compressed))
			if compressionRatio > 1.1 { // Only compress if we save at least 10%
				finalData = compressed
				chunkInfo.IsCompressed = true
				chunkInfo.CompressedSize = int64(len(compressed))
				chunkInfo.CompressionRatio = compressionRatio
			}
		}
	}
	
	// Store to filesystem
	storagePath := filepath.Join(oe.basePath, hashStr)
	if err := os.WriteFile(storagePath, finalData, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write chunk: %v", err)
	}
	
	chunkInfo.StoragePath = storagePath
	
	// Update indices
	oe.chunksMu.Lock()
	oe.chunks[hashStr] = chunkInfo
	oe.chunksMu.Unlock()
	
	// Update deduplication index
	if oe.config.EnableDeduplication {
		oe.updateDeduplicationIndex(hashStr, data)
	}
	
	// Add to cache if enabled
	if oe.config.EnableIntelligentCache {
		oe.addToCache(hashStr, data)
	}
	
	oe.updateAnalytics("write", len(data), time.Since(start))
	
	oe.logger.Debugf("📦 Stored optimized chunk %s (size: %d, compressed: %v)", 
		hashStr, chunkInfo.Size, chunkInfo.IsCompressed)
	
	return hashStr, chunkInfo, nil
}

// OptimizedGet retrieves a chunk with optimization (caching, decompression)
func (oe *OptimizationEngine) OptimizedGet(hash string) (io.ReadCloser, *ChunkInfo, error) {
	start := time.Now()
	
	// Check cache first
	if oe.config.EnableIntelligentCache {
		if cached := oe.getFromCache(hash); cached != nil {
			oe.updateAnalytics("cache_hit", len(cached), time.Since(start))
			return io.NopCloser(bytes.NewReader(cached)), oe.getChunkInfo(hash), nil
		}
		oe.updateAnalytics("cache_miss", 0, time.Since(start))
	}
	
	// Get chunk info
	chunkInfo := oe.getChunkInfo(hash)
	if chunkInfo == nil {
		return nil, nil, fmt.Errorf("chunk %s not found", hash)
	}
	
	// Read from storage
	data, err := os.ReadFile(chunkInfo.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read chunk: %v", err)
	}
	
	// Decompress if needed
	if chunkInfo.IsCompressed {
		decompressed, err := compressor.DecompressData(data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decompress chunk: %v", err)
		}
		data = decompressed
	}
	
	// Update access statistics
	oe.chunksMu.Lock()
	chunkInfo.LastAccessedAt = time.Now()
	chunkInfo.AccessCount++
	oe.chunksMu.Unlock()
	
	// Add to cache
	if oe.config.EnableIntelligentCache {
		oe.addToCache(hash, data)
	}
	
	oe.updateAnalytics("read", len(data), time.Since(start))
	
	return io.NopCloser(bytes.NewReader(data)), chunkInfo, nil
}

// findDuplicateChunk checks if a chunk already exists
func (oe *OptimizationEngine) findDuplicateChunk(hash string, data []byte) *ChunkInfo {
	oe.chunksMu.RLock()
	defer oe.chunksMu.RUnlock()
	
	if chunk, exists := oe.chunks[hash]; exists {
		return chunk
	}
	
	// Advanced content-based deduplication
	contentHash := oe.calculateContentHash(data)
	oe.dedupMu.RLock()
	if dedupEntry, exists := oe.deduplicationIndex[contentHash]; exists {
		for _, chunkHash := range dedupEntry.ChunkHashes {
			if chunk, exists := oe.chunks[chunkHash]; exists {
				oe.dedupMu.RUnlock()
				return chunk
			}
		}
	}
	oe.dedupMu.RUnlock()
	
	return nil
}

// calculateContentHash calculates a content-based hash for deduplication
func (oe *OptimizationEngine) calculateContentHash(data []byte) string {
	// Use rolling hash for better deduplication of similar content
	hash := sha256.New()
	
	// Add data in chunks to create a rolling hash effect
	chunkSize := 1024
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		hash.Write(data[i:end])
	}
	
	return hex.EncodeToString(hash.Sum(nil))
}

// updateDeduplicationIndex updates the deduplication index
func (oe *OptimizationEngine) updateDeduplicationIndex(hash string, data []byte) {
	contentHash := oe.calculateContentHash(data)
	
	oe.dedupMu.Lock()
	defer oe.dedupMu.Unlock()
	
	if entry, exists := oe.deduplicationIndex[contentHash]; exists {
		entry.ChunkHashes = append(entry.ChunkHashes, hash)
		entry.Count++
		entry.LastSeen = time.Now()
	} else {
		oe.deduplicationIndex[contentHash] = &DeduplicationIndex{
			ContentHash: contentHash,
			ChunkHashes: []string{hash},
			Size:        int64(len(data)),
			Count:       1,
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}
	}
}

// addToCache adds data to the intelligent cache
func (oe *OptimizationEngine) addToCache(hash string, data []byte) {
	if int64(len(data)) > oe.config.MaxCacheSize/10 {
		return // Don't cache very large chunks
	}
	
	oe.cacheMu.Lock()
	defer oe.cacheMu.Unlock()
	
	// Check if already cached
	if entry, exists := oe.cache[hash]; exists {
		entry.LastAccessedAt = time.Now()
		entry.AccessCount++
		entry.Priority = oe.calculateCachePriority(entry)
		return
	}
	
	// Make space if needed
	for oe.cacheSize+int64(len(data)) > oe.config.MaxCacheSize {
		oe.evictFromCache()
	}
	
	// Add to cache
	entry := &CacheEntry{
		Data:           make([]byte, len(data)),
		Hash:           hash,
		Size:           int64(len(data)),
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		AccessCount:    1,
	}
	copy(entry.Data, data)
	entry.Priority = oe.calculateCachePriority(entry)
	
	oe.cache[hash] = entry
	oe.cacheSize += entry.Size
}

// getFromCache retrieves data from cache
func (oe *OptimizationEngine) getFromCache(hash string) []byte {
	oe.cacheMu.Lock()
	defer oe.cacheMu.Unlock()
	
	if entry, exists := oe.cache[hash]; exists {
		entry.LastAccessedAt = time.Now()
		entry.AccessCount++
		entry.Priority = oe.calculateCachePriority(entry)
		
		result := make([]byte, len(entry.Data))
		copy(result, entry.Data)
		return result
	}
	
	return nil
}

// calculateCachePriority calculates priority for cache entries
func (oe *OptimizationEngine) calculateCachePriority(entry *CacheEntry) float64 {
	switch oe.config.CacheEvictionPolicy {
	case "lru":
		return float64(entry.LastAccessedAt.Unix())
	case "lfu":
		return float64(entry.AccessCount)
	case "fifo":
		return float64(entry.CreatedAt.Unix())
	default:
		// Balanced approach: LRU + LFU
		timeFactor := float64(entry.LastAccessedAt.Unix()) / 100000.0
		accessFactor := float64(entry.AccessCount)
		return timeFactor + accessFactor
	}
}

// evictFromCache evicts the lowest priority item from cache
func (oe *OptimizationEngine) evictFromCache() {
	if len(oe.cache) == 0 {
		return
	}
	
	var lowestHash string
	var lowestPriority float64 = -1
	
	for hash, entry := range oe.cache {
		priority := oe.calculateCachePriority(entry)
		if lowestPriority == -1 || priority < lowestPriority {
			lowestPriority = priority
			lowestHash = hash
		}
	}
	
	if entry, exists := oe.cache[lowestHash]; exists {
		oe.cacheSize -= entry.Size
		delete(oe.cache, lowestHash)
	}
}

// getChunkInfo retrieves chunk information
func (oe *OptimizationEngine) getChunkInfo(hash string) *ChunkInfo {
	oe.chunksMu.RLock()
	defer oe.chunksMu.RUnlock()
	
	if chunk, exists := oe.chunks[hash]; exists {
		return chunk
	}
	return nil
}

// updateAnalytics updates performance analytics
func (oe *OptimizationEngine) updateAnalytics(operation string, dataSize int, duration time.Duration) {
	oe.analyticsMu.Lock()
	defer oe.analyticsMu.Unlock()
	
	hour := time.Now().Format("2006-01-02-15")
	if oe.analytics.HourlyStats[hour] == nil {
		oe.analytics.HourlyStats[hour] = &HourlyStats{
			Hour: hour,
		}
	}
	
	stats := oe.analytics.HourlyStats[hour]
	latency := float64(duration.Nanoseconds()) / 1000000.0 // convert to milliseconds
	
	switch operation {
	case "read":
		stats.Reads++
		oe.analytics.AccessPatterns["reads"]++
	case "write":
		stats.Writes++
		oe.analytics.AccessPatterns["writes"]++
	case "cache_hit":
		stats.CacheHits++
		oe.analytics.AccessPatterns["cache_hits"]++
	case "cache_miss":
		stats.CacheMisses++
		oe.analytics.AccessPatterns["cache_misses"]++
	case "compression":
		stats.CompressionOps++
	case "decompression":
		stats.DecompressionOps++
	case "deduplication":
		stats.DeduplicationOps++
	}
	
	// Update average latency
	if stats.Reads+stats.Writes > 0 {
		stats.AverageLatency = (stats.AverageLatency + latency) / 2.0
	}
}

// backgroundOptimization runs periodic optimization tasks
func (oe *OptimizationEngine) backgroundOptimization() {
	for {
		select {
		case <-oe.optimizationTicker.C:
			oe.performOptimization()
		case <-oe.stopChan:
			return
		}
	}
}

// performOptimization runs comprehensive storage optimization
func (oe *OptimizationEngine) performOptimization() {
	oe.logger.Info("🔧 Performing storage optimization...")
	start := time.Now()
	
	// Clean up old analytics data
	oe.cleanupAnalytics()
	
	// Optimize cache
	oe.optimizeCache()
	
	// Cleanup orphaned chunks
	oe.cleanupOrphanedChunks()
	
	// Update comprehensive analytics
	oe.updateComprehensiveAnalytics()
	
	oe.analyticsMu.Lock()
	oe.analytics.LastOptimization = time.Now()
	oe.analyticsMu.Unlock()
	
	duration := time.Since(start)
	oe.logger.Infof("✅ Storage optimization completed in %v", duration)
}

// optimizeCache optimizes the intelligent cache
func (oe *OptimizationEngine) optimizeCache() {
	oe.cacheMu.Lock()
	defer oe.cacheMu.Unlock()
	
	// Remove stale entries
	cutoff := time.Now().Add(-24 * time.Hour)
	for hash, entry := range oe.cache {
		if entry.LastAccessedAt.Before(cutoff) && entry.AccessCount < 2 {
			oe.cacheSize -= entry.Size
			delete(oe.cache, hash)
		}
	}
	
	// Rebalance cache priorities
	if len(oe.cache) > 0 {
		for _, entry := range oe.cache {
			entry.Priority = oe.calculateCachePriority(entry)
		}
	}
}

// cleanupOrphanedChunks removes chunks that are no longer referenced
func (oe *OptimizationEngine) cleanupOrphanedChunks() {
	oe.chunksMu.Lock()
	defer oe.chunksMu.Unlock()
	
	orphaned := 0
	for hash, chunk := range oe.chunks {
		if chunk.ReferenceCount <= 0 {
			// Remove from filesystem
			if err := os.Remove(chunk.StoragePath); err != nil {
				oe.logger.Warnf("⚠️ Failed to remove orphaned chunk %s: %v", hash, err)
			}
			
			// Remove from index
			delete(oe.chunks, hash)
			orphaned++
		}
	}
	
	if orphaned > 0 {
		oe.logger.Infof("🗑️ Cleaned up %d orphaned chunks", orphaned)
	}
}

// cleanupAnalytics removes old analytics data
func (oe *OptimizationEngine) cleanupAnalytics() {
	oe.analyticsMu.Lock()
	defer oe.analyticsMu.Unlock()
	
	cutoff := time.Now().Add(-oe.config.AnalyticsRetention)
	cleaned := 0
	
	for hour := range oe.analytics.HourlyStats {
		if hourTime, err := time.Parse("2006-01-02-15", hour); err == nil {
			if hourTime.Before(cutoff) {
				delete(oe.analytics.HourlyStats, hour)
				cleaned++
			}
		}
	}
	
	if cleaned > 0 {
		oe.logger.Debugf("🧹 Cleaned up %d old analytics entries", cleaned)
	}
}

// updateComprehensiveAnalytics calculates comprehensive storage statistics
func (oe *OptimizationEngine) updateComprehensiveAnalytics() {
	oe.analyticsMu.Lock()
	oe.chunksMu.RLock()
	oe.cacheMu.RLock()
	oe.dedupMu.RLock()
	
	defer func() {
		oe.dedupMu.RUnlock()
		oe.cacheMu.RUnlock()
		oe.chunksMu.RUnlock()
		oe.analyticsMu.Unlock()
	}()
	
	// Calculate storage statistics
	var totalStorage, compressedStorage int64
	var totalCompressionRatio float64
	chunksWithCompression := 0
	
	for _, chunk := range oe.chunks {
		totalStorage += chunk.Size
		if chunk.IsCompressed {
			compressedStorage += chunk.CompressedSize
			totalCompressionRatio += chunk.CompressionRatio
			chunksWithCompression++
		} else {
			compressedStorage += chunk.Size
		}
	}
	
	// Calculate deduplication statistics
	var deduplicationSavings int64
	uniqueChunks := int64(len(oe.chunks))
	totalChunks := int64(0)
	
	for _, dedupEntry := range oe.deduplicationIndex {
		if dedupEntry.Count > 1 {
			deduplicationSavings += dedupEntry.Size * int64(dedupEntry.Count-1)
			totalChunks += int64(dedupEntry.Count)
		} else {
			totalChunks++
		}
	}
	
	// Calculate cache statistics
	cacheHits := oe.analytics.AccessPatterns["cache_hits"]
	cacheMisses := oe.analytics.AccessPatterns["cache_misses"]
	cacheHitRate := 0.0
	if cacheHits+cacheMisses > 0 {
		cacheHitRate = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100.0
	}
	
	// Update analytics
	oe.analytics.TotalStorageUsed = totalStorage
	oe.analytics.CompressedStorageUsed = compressedStorage
	oe.analytics.DeduplicationSavings = deduplicationSavings
	oe.analytics.CompressionSavings = totalStorage - compressedStorage
	oe.analytics.CacheHitRate = cacheHitRate
	oe.analytics.CacheSize = oe.cacheSize
	oe.analytics.TotalChunks = totalChunks
	oe.analytics.UniqueChunks = uniqueChunks
	oe.analytics.DuplicateChunks = totalChunks - uniqueChunks
	
	if uniqueChunks > 0 {
		oe.analytics.AverageChunkSize = float64(totalStorage) / float64(uniqueChunks)
	}
	
	if chunksWithCompression > 0 {
		oe.analytics.AverageCompressionRatio = totalCompressionRatio / float64(chunksWithCompression)
	}
}

// initializeAnalytics initializes the analytics system
func (oe *OptimizationEngine) initializeAnalytics() {
	oe.analytics.AccessPatterns = map[string]int64{
		"reads":       0,
		"writes":      0,
		"cache_hits":  0,
		"cache_misses": 0,
	}
}

// loadChunkIndex loads the chunk index from disk
func (oe *OptimizationEngine) loadChunkIndex() error {
	indexPath := filepath.Join(oe.basePath, "chunk_index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing index
		}
		return err
	}
	
	oe.chunksMu.Lock()
	defer oe.chunksMu.Unlock()
	
	return json.Unmarshal(data, &oe.chunks)
}

// saveChunkIndex saves the chunk index to disk
func (oe *OptimizationEngine) saveChunkIndex() error {
	indexPath := filepath.Join(oe.basePath, "chunk_index.json")
	
	oe.chunksMu.RLock()
	data, err := json.MarshalIndent(oe.chunks, "", "  ")
	oe.chunksMu.RUnlock()
	
	if err != nil {
		return err
	}
	
	return os.WriteFile(indexPath, data, 0644)
}

// loadDeduplicationIndex loads the deduplication index from disk
func (oe *OptimizationEngine) loadDeduplicationIndex() error {
	indexPath := filepath.Join(oe.basePath, "dedup_index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing index
		}
		return err
	}
	
	oe.dedupMu.Lock()
	defer oe.dedupMu.Unlock()
	
	return json.Unmarshal(data, &oe.deduplicationIndex)
}

// saveDeduplicationIndex saves the deduplication index to disk
func (oe *OptimizationEngine) saveDeduplicationIndex() error {
	indexPath := filepath.Join(oe.basePath, "dedup_index.json")
	
	oe.dedupMu.RLock()
	data, err := json.MarshalIndent(oe.deduplicationIndex, "", "  ")
	oe.dedupMu.RUnlock()
	
	if err != nil {
		return err
	}
	
	return os.WriteFile(indexPath, data, 0644)
}

// GetAnalytics returns current storage analytics
func (oe *OptimizationEngine) GetAnalytics() *StorageAnalytics {
	oe.analyticsMu.RLock()
	defer oe.analyticsMu.RUnlock()
	
	// Create a copy to avoid race conditions
	analytics := *oe.analytics
	analytics.AccessPatterns = make(map[string]int64)
	for k, v := range oe.analytics.AccessPatterns {
		analytics.AccessPatterns[k] = v
	}
	
	analytics.HourlyStats = make(map[string]*HourlyStats)
	for k, v := range oe.analytics.HourlyStats {
		analytics.HourlyStats[k] = v
	}
	
	return &analytics
}

// GetOptimizationReport generates a comprehensive optimization report
func (oe *OptimizationEngine) GetOptimizationReport() map[string]interface{} {
	analytics := oe.GetAnalytics()
	
	// Calculate savings percentages
	compressionSavingsPercent := 0.0
	if analytics.TotalStorageUsed > 0 {
		compressionSavingsPercent = float64(analytics.CompressionSavings) / float64(analytics.TotalStorageUsed) * 100.0
	}
	
	deduplicationSavingsPercent := 0.0
	if analytics.TotalStorageUsed > 0 {
		deduplicationSavingsPercent = float64(analytics.DeduplicationSavings) / float64(analytics.TotalStorageUsed) * 100.0
	}
	
	totalSavings := analytics.CompressionSavings + analytics.DeduplicationSavings
	totalSavingsPercent := 0.0
	if analytics.TotalStorageUsed > 0 {
		totalSavingsPercent = float64(totalSavings) / float64(analytics.TotalStorageUsed) * 100.0
	}
	
	return map[string]interface{}{
		"storage_optimization": map[string]interface{}{
			"total_storage_used":         analytics.TotalStorageUsed,
			"compressed_storage_used":    analytics.CompressedStorageUsed,
			"compression_savings":        analytics.CompressionSavings,
			"compression_savings_percent": compressionSavingsPercent,
			"deduplication_savings":      analytics.DeduplicationSavings,
			"deduplication_savings_percent": deduplicationSavingsPercent,
			"total_savings":              totalSavings,
			"total_savings_percent":      totalSavingsPercent,
			"average_compression_ratio":  analytics.AverageCompressionRatio,
		},
		"cache_performance": map[string]interface{}{
			"hit_rate":    analytics.CacheHitRate,
			"cache_size":  analytics.CacheSize,
			"max_size":    oe.config.MaxCacheSize,
			"utilization": float64(analytics.CacheSize) / float64(oe.config.MaxCacheSize) * 100.0,
		},
		"deduplication_stats": map[string]interface{}{
			"total_chunks":     analytics.TotalChunks,
			"unique_chunks":    analytics.UniqueChunks,
			"duplicate_chunks": analytics.DuplicateChunks,
			"dedup_ratio":      float64(analytics.DuplicateChunks) / float64(analytics.TotalChunks) * 100.0,
		},
		"performance_metrics": map[string]interface{}{
			"average_chunk_size": analytics.AverageChunkSize,
			"access_patterns":    analytics.AccessPatterns,
			"last_optimization": analytics.LastOptimization,
		},
		"config": oe.config,
	}
}
