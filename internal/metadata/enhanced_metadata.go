package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"
)

// EnhancedFileMetadata represents comprehensive file metadata
type EnhancedFileMetadata struct {
	// Basic file information
	FileID          string    `json:"file_id"`
	FileName        string    `json:"file_name"`
	OriginalName    string    `json:"original_name"`
	FileSize        int64     `json:"file_size"`
	MimeType        string    `json:"mime_type"`
	FileHash        string    `json:"file_hash"`
	
	// Chunking information
	ChunkCount      int       `json:"chunk_count"`
	ChunkSize       int64     `json:"chunk_size"`
	ChunkHashes     []string  `json:"chunk_hashes"`
	
	// Storage and replication
	StorageNodes    []string  `json:"storage_nodes"`
	ReplicaCount    int       `json:"replica_count"`
	StorageClass    string    `json:"storage_class"`    // "hot", "warm", "cold", "archive"
	
	// Encryption and security
	IsEncrypted     bool      `json:"is_encrypted"`
	EncryptionAlgo  string    `json:"encryption_algo"`
	KeyID           string    `json:"key_id"`
	AccessLevel     string    `json:"access_level"`     // "public", "private", "restricted"
	
	// Compression and optimization
	IsCompressed    bool      `json:"is_compressed"`
	CompressionAlgo string    `json:"compression_algo"`
	CompressionRatio float64  `json:"compression_ratio"`
	
	// Versioning
	Version         int       `json:"version"`
	PreviousVersions []string `json:"previous_versions"`
	IsDeleted       bool      `json:"is_deleted"`
	
	// Timestamps
	CreatedAt       time.Time `json:"created_at"`
	ModifiedAt      time.Time `json:"modified_at"`
	AccessedAt      time.Time `json:"accessed_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	
	// User and ownership
	OwnerID         string    `json:"owner_id"`
	CreatorID       string    `json:"creator_id"`
	ModifiedBy      string    `json:"modified_by"`
	
	// Classification and organization
	Tags            []string  `json:"tags"`
	Categories      []string  `json:"categories"`
	Description     string    `json:"description"`
	CustomMetadata  map[string]interface{} `json:"custom_metadata"`
	
	// Access tracking
	AccessCount     int64     `json:"access_count"`
	DownloadCount   int64     `json:"download_count"`
	ShareCount      int64     `json:"share_count"`
	
	// Relationships
	ParentFileID    string    `json:"parent_file_id,omitempty"`
	ChildFiles      []string  `json:"child_files,omitempty"`
	RelatedFiles    []string  `json:"related_files,omitempty"`
	
	// Integrity and health
	IntegrityHash   string    `json:"integrity_hash"`
	HealthStatus    string    `json:"health_status"`    // "healthy", "degraded", "corrupted"
	LastVerified    time.Time `json:"last_verified"`
}

// EnhancedChunkMetadata represents comprehensive chunk metadata
type EnhancedChunkMetadata struct {
	// Basic chunk information
	ChunkID         string    `json:"chunk_id"`
	FileID          string    `json:"file_id"`
	Index           int       `json:"index"`
	Hash            string    `json:"hash"`
	Size            int64     `json:"size"`
	Path            string    `json:"path"`
	
	// Storage optimization
	IsCompressed    bool      `json:"is_compressed"`
	CompressedSize  int64     `json:"compressed_size"`
	CompressionRatio float64  `json:"compression_ratio"`
	IsDeduplicated  bool      `json:"is_deduplicated"`
	ReferenceCount  int       `json:"reference_count"`
	
	// Replication and distribution
	StorageNodes    []string  `json:"storage_nodes"`
	ReplicaHealth   map[string]string `json:"replica_health"`
	PrimaryNode     string    `json:"primary_node"`
	
	// Performance metrics
	AccessCount     int64     `json:"access_count"`
	LastAccessTime  time.Time `json:"last_access_time"`
	AccessPattern   string    `json:"access_pattern"`   // "sequential", "random", "burst"
	
	// Quality and integrity
	IntegrityHash   string    `json:"integrity_hash"`
	HealthStatus    string    `json:"health_status"`
	LastVerified    time.Time `json:"last_verified"`
	ErrorCount      int       `json:"error_count"`
	
	// Timestamps
	CreatedAt       time.Time `json:"created_at"`
	ModifiedAt      time.Time `json:"modified_at"`
}

// FileVersion represents a file version
type FileVersion struct {
	VersionID       string    `json:"version_id"`
	FileID          string    `json:"file_id"`
	Version         int       `json:"version"`
	FileHash        string    `json:"file_hash"`
	FileSize        int64     `json:"file_size"`
	ChunkHashes     []string  `json:"chunk_hashes"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	ChangeLog       string    `json:"change_log"`
	ParentVersion   string    `json:"parent_version"`
}

// FileRelationship represents relationships between files
type FileRelationship struct {
	RelationshipID  string    `json:"relationship_id"`
	SourceFileID    string    `json:"source_file_id"`
	TargetFileID    string    `json:"target_file_id"`
	RelationType    string    `json:"relation_type"`    // "parent", "child", "sibling", "reference", "dependency"
	Strength        float64   `json:"strength"`         // 0.0 to 1.0
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SearchQuery represents a metadata search query
type SearchQuery struct {
	// Text search
	Query           string    `json:"query"`
	Fields          []string  `json:"fields"`           // Fields to search in
	
	// Filters
	FileTypes       []string  `json:"file_types"`
	Tags            []string  `json:"tags"`
	Categories      []string  `json:"categories"`
	OwnerIDs        []string  `json:"owner_ids"`
	
	// Size filters
	MinSize         int64     `json:"min_size"`
	MaxSize         int64     `json:"max_size"`
	
	// Time filters
	CreatedAfter    *time.Time `json:"created_after"`
	CreatedBefore   *time.Time `json:"created_before"`
	ModifiedAfter   *time.Time `json:"modified_after"`
	ModifiedBefore  *time.Time `json:"modified_before"`
	
	// Status filters
	HealthStatus    []string  `json:"health_status"`
	StorageClass    []string  `json:"storage_class"`
	IsDeleted       *bool     `json:"is_deleted"`
	
	// Sorting and pagination
	SortBy          string    `json:"sort_by"`          // Field to sort by
	SortOrder       string    `json:"sort_order"`       // "asc" or "desc"
	Limit           int       `json:"limit"`
	Offset          int       `json:"offset"`
}

// SearchResult represents search results
type SearchResult struct {
	Files           []*EnhancedFileMetadata `json:"files"`
	TotalCount      int64                   `json:"total_count"`
	SearchDuration  time.Duration           `json:"search_duration"`
	Facets          map[string]map[string]int64 `json:"facets"`
}

// MetadataIndex represents different types of indices
type MetadataIndex struct {
	IndexType       string                  `json:"index_type"`     // "hash", "tag", "owner", "size", "time"
	IndexData       map[string][]string     `json:"index_data"`     // Key -> list of file IDs
	LastUpdated     time.Time               `json:"last_updated"`
}

// EnhancedMetadataStore provides advanced metadata management
type EnhancedMetadataStore struct {
	db              *badger.DB
	logger          *logrus.Logger
	
	// Indices for fast searching
	indices         map[string]*MetadataIndex
	indicesMu       sync.RWMutex
	
	// Background tasks
	indexUpdateChan chan string
	stopChan        chan bool
	wg              sync.WaitGroup
}

// NewEnhancedMetadataStore creates a new enhanced metadata store
func NewEnhancedMetadataStore(dbPath string) (*EnhancedMetadataStore, error) {
	db, err := badger.Open(badger.DefaultOptions(dbPath).WithLogger(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %v", err)
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	store := &EnhancedMetadataStore{
		db:              db,
		logger:          logger,
		indices:         make(map[string]*MetadataIndex),
		indexUpdateChan: make(chan string, 1000),
		stopChan:        make(chan bool),
	}
	
	// Load existing indices
	if err := store.loadIndices(); err != nil {
		store.logger.Warnf("⚠️ Failed to load indices: %v", err)
	}
	
	// Start background index updater
	store.wg.Add(1)
	go store.indexUpdater()
	
	store.logger.Info("✅ Enhanced Metadata Store initialized")
	return store, nil
}

// Close closes the metadata store
func (ems *EnhancedMetadataStore) Close() error {
	close(ems.stopChan)
	close(ems.indexUpdateChan)
	ems.wg.Wait()
	
	// Save indices
	if err := ems.saveIndices(); err != nil {
		ems.logger.Errorf("❌ Failed to save indices: %v", err)
	}
	
	return ems.db.Close()
}

// StoreFileMetadata stores enhanced file metadata
func (ems *EnhancedMetadataStore) StoreFileMetadata(meta *EnhancedFileMetadata) error {
	// Set timestamps
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.ModifiedAt = time.Now()
	
	// Calculate integrity hash
	meta.IntegrityHash = ems.calculateIntegrityHash(meta)
	
	// Serialize metadata
	key := []byte(fmt.Sprintf("file:%s", meta.FileID))
	value, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal file metadata: %v", err)
	}
	
	// Store in database
	err = ems.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	if err != nil {
		return fmt.Errorf("failed to store file metadata: %v", err)
	}
	
	// Update indices asynchronously
	select {
	case ems.indexUpdateChan <- meta.FileID:
	default:
		// Channel is full, skip this update
	}
	
	return nil
}

// GetFileMetadata retrieves enhanced file metadata
func (ems *EnhancedMetadataStore) GetFileMetadata(fileID string) (*EnhancedFileMetadata, error) {
	key := []byte(fmt.Sprintf("file:%s", fileID))
	var meta EnhancedFileMetadata
	
	err := ems.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &meta)
		})
	})
	
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("file metadata not found: %s", fileID)
		}
		return nil, fmt.Errorf("failed to get file metadata: %v", err)
	}
	
	// Update access time
	meta.AccessedAt = time.Now()
	meta.AccessCount++
	go ems.StoreFileMetadata(&meta)
	
	return &meta, nil
}

// StoreChunkMetadata stores enhanced chunk metadata
func (ems *EnhancedMetadataStore) StoreChunkMetadata(meta *EnhancedChunkMetadata) error {
	// Set timestamps
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	meta.ModifiedAt = time.Now()
	
	// Calculate integrity hash
	meta.IntegrityHash = ems.calculateChunkIntegrityHash(meta)
	
	// Serialize metadata
	key := []byte(fmt.Sprintf("chunk:%s", meta.ChunkID))
	value, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk metadata: %v", err)
	}
	
	// Store in database
	return ems.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// GetChunkMetadata retrieves enhanced chunk metadata
func (ems *EnhancedMetadataStore) GetChunkMetadata(chunkID string) (*EnhancedChunkMetadata, error) {
	key := []byte(fmt.Sprintf("chunk:%s", chunkID))
	var meta EnhancedChunkMetadata
	
	err := ems.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &meta)
		})
	})
	
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("chunk metadata not found: %s", chunkID)
		}
		return nil, fmt.Errorf("failed to get chunk metadata: %v", err)
	}
	
	// Update access statistics
	meta.LastAccessTime = time.Now()
	meta.AccessCount++
	go ems.StoreChunkMetadata(&meta)
	
	return &meta, nil
}

// CreateFileVersion creates a new file version
func (ems *EnhancedMetadataStore) CreateFileVersion(fileID, createdBy, changeLog string) (*FileVersion, error) {
	// Get current file metadata
	fileMeta, err := ems.GetFileMetadata(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %v", err)
	}
	
	// Create version ID
	versionID := fmt.Sprintf("%s-v%d", fileID, fileMeta.Version+1)
	
	version := &FileVersion{
		VersionID:     versionID,
		FileID:        fileID,
		Version:       fileMeta.Version + 1,
		FileHash:      fileMeta.FileHash,
		FileSize:      fileMeta.FileSize,
		ChunkHashes:   fileMeta.ChunkHashes,
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
		ChangeLog:     changeLog,
		ParentVersion: fmt.Sprintf("%s-v%d", fileID, fileMeta.Version),
	}
	
	// Store version
	key := []byte(fmt.Sprintf("version:%s", versionID))
	value, err := json.Marshal(version)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version: %v", err)
	}
	
	err = ems.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store version: %v", err)
	}
	
	// Update file metadata with new version
	fileMeta.Version = version.Version
	fileMeta.PreviousVersions = append(fileMeta.PreviousVersions, versionID)
	if err := ems.StoreFileMetadata(fileMeta); err != nil {
		ems.logger.Errorf("❌ Failed to update file metadata with new version: %v", err)
	}
	
	return version, nil
}

// CreateFileRelationship creates a relationship between files
func (ems *EnhancedMetadataStore) CreateFileRelationship(sourceFileID, targetFileID, relationType, createdBy string, strength float64) (*FileRelationship, error) {
	relationshipID := fmt.Sprintf("%s-%s-%s", sourceFileID, targetFileID, relationType)
	
	relationship := &FileRelationship{
		RelationshipID: relationshipID,
		SourceFileID:   sourceFileID,
		TargetFileID:   targetFileID,
		RelationType:   relationType,
		Strength:       strength,
		CreatedAt:      time.Now(),
		CreatedBy:      createdBy,
		Metadata:       make(map[string]interface{}),
	}
	
	// Store relationship
	key := []byte(fmt.Sprintf("relationship:%s", relationshipID))
	value, err := json.Marshal(relationship)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal relationship: %v", err)
	}
	
	err = ems.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store relationship: %v", err)
	}
	
	return relationship, nil
}

// SearchFiles performs advanced file search
func (ems *EnhancedMetadataStore) SearchFiles(query *SearchQuery) (*SearchResult, error) {
	start := time.Now()
	
	result := &SearchResult{
		Files:   make([]*EnhancedFileMetadata, 0),
		Facets:  make(map[string]map[string]int64),
	}
	
	// Use indices for efficient searching
	candidateFileIDs := ems.findCandidateFiles(query)
	
	// Filter and score candidates
	var matches []*EnhancedFileMetadata
	for _, fileID := range candidateFileIDs {
		if fileMeta, err := ems.GetFileMetadata(fileID); err == nil {
			if ems.matchesQuery(fileMeta, query) {
				matches = append(matches, fileMeta)
			}
		}
	}
	
	// Sort results
	ems.sortResults(matches, query)
	
	// Apply pagination
	result.TotalCount = int64(len(matches))
	startIdx := query.Offset
	endIdx := query.Offset + query.Limit
	if startIdx < len(matches) {
		if endIdx > len(matches) {
			endIdx = len(matches)
		}
		result.Files = matches[startIdx:endIdx]
	}
	
	// Calculate facets
	result.Facets = ems.calculateFacets(matches)
	
	result.SearchDuration = time.Since(start)
	return result, nil
}

// calculateIntegrityHash calculates an integrity hash for file metadata
func (ems *EnhancedMetadataStore) calculateIntegrityHash(meta *EnhancedFileMetadata) string {
	hash := sha256.New()
	hash.Write([]byte(meta.FileID))
	hash.Write([]byte(meta.FileHash))
	hash.Write([]byte(fmt.Sprintf("%d", meta.FileSize)))
	hash.Write([]byte(strings.Join(meta.ChunkHashes, ",")))
	return hex.EncodeToString(hash.Sum(nil))
}

// calculateChunkIntegrityHash calculates an integrity hash for chunk metadata
func (ems *EnhancedMetadataStore) calculateChunkIntegrityHash(meta *EnhancedChunkMetadata) string {
	hash := sha256.New()
	hash.Write([]byte(meta.ChunkID))
	hash.Write([]byte(meta.Hash))
	hash.Write([]byte(fmt.Sprintf("%d", meta.Size)))
	hash.Write([]byte(meta.Path))
	return hex.EncodeToString(hash.Sum(nil))
}

// indexUpdater runs background index updates
func (ems *EnhancedMetadataStore) indexUpdater() {
	defer ems.wg.Done()
	
	for {
		select {
		case fileID := <-ems.indexUpdateChan:
			ems.updateIndicesForFile(fileID)
		case <-ems.stopChan:
			return
		}
	}
}

// updateIndicesForFile updates all indices for a specific file
func (ems *EnhancedMetadataStore) updateIndicesForFile(fileID string) {
	fileMeta, err := ems.GetFileMetadata(fileID)
	if err != nil {
		return
	}
	
	ems.indicesMu.Lock()
	defer ems.indicesMu.Unlock()
	
	// Update tag index
	ems.updateTagIndex(fileID, fileMeta.Tags)
	
	// Update owner index
	ems.updateOwnerIndex(fileID, fileMeta.OwnerID)
	
	// Update size index
	ems.updateSizeIndex(fileID, fileMeta.FileSize)
	
	// Update time index
	ems.updateTimeIndex(fileID, fileMeta.CreatedAt, fileMeta.ModifiedAt)
	
	// Update category index
	ems.updateCategoryIndex(fileID, fileMeta.Categories)
	
	// Update health status index
	ems.updateHealthStatusIndex(fileID, fileMeta.HealthStatus)
}

// updateTagIndex updates the tag index
func (ems *EnhancedMetadataStore) updateTagIndex(fileID string, tags []string) {
	if ems.indices["tag"] == nil {
		ems.indices["tag"] = &MetadataIndex{
			IndexType:   "tag",
			IndexData:   make(map[string][]string),
			LastUpdated: time.Now(),
		}
	}
	
	for _, tag := range tags {
		if !ems.contains(ems.indices["tag"].IndexData[tag], fileID) {
			ems.indices["tag"].IndexData[tag] = append(ems.indices["tag"].IndexData[tag], fileID)
		}
	}
	ems.indices["tag"].LastUpdated = time.Now()
}

// updateOwnerIndex updates the owner index
func (ems *EnhancedMetadataStore) updateOwnerIndex(fileID, ownerID string) {
	if ems.indices["owner"] == nil {
		ems.indices["owner"] = &MetadataIndex{
			IndexType:   "owner",
			IndexData:   make(map[string][]string),
			LastUpdated: time.Now(),
		}
	}
	
	if !ems.contains(ems.indices["owner"].IndexData[ownerID], fileID) {
		ems.indices["owner"].IndexData[ownerID] = append(ems.indices["owner"].IndexData[ownerID], fileID)
	}
	ems.indices["owner"].LastUpdated = time.Now()
}

// updateSizeIndex updates the size index
func (ems *EnhancedMetadataStore) updateSizeIndex(fileID string, size int64) {
	if ems.indices["size"] == nil {
		ems.indices["size"] = &MetadataIndex{
			IndexType:   "size",
			IndexData:   make(map[string][]string),
			LastUpdated: time.Now(),
		}
	}
	
	// Create size buckets
	sizeBucket := ems.getSizeBucket(size)
	if !ems.contains(ems.indices["size"].IndexData[sizeBucket], fileID) {
		ems.indices["size"].IndexData[sizeBucket] = append(ems.indices["size"].IndexData[sizeBucket], fileID)
	}
	ems.indices["size"].LastUpdated = time.Now()
}

// updateTimeIndex updates the time index
func (ems *EnhancedMetadataStore) updateTimeIndex(fileID string, createdAt, modifiedAt time.Time) {
	if ems.indices["time"] == nil {
		ems.indices["time"] = &MetadataIndex{
			IndexType:   "time",
			IndexData:   make(map[string][]string),
			LastUpdated: time.Now(),
		}
	}
	
	// Create time buckets (by day)
	createdDay := createdAt.Format("2006-01-02")
	modifiedDay := modifiedAt.Format("2006-01-02")
	
	if !ems.contains(ems.indices["time"].IndexData[createdDay], fileID) {
		ems.indices["time"].IndexData[createdDay] = append(ems.indices["time"].IndexData[createdDay], fileID)
	}
	
	if createdDay != modifiedDay {
		if !ems.contains(ems.indices["time"].IndexData[modifiedDay], fileID) {
			ems.indices["time"].IndexData[modifiedDay] = append(ems.indices["time"].IndexData[modifiedDay], fileID)
		}
	}
	
	ems.indices["time"].LastUpdated = time.Now()
}

// updateCategoryIndex updates the category index
func (ems *EnhancedMetadataStore) updateCategoryIndex(fileID string, categories []string) {
	if ems.indices["category"] == nil {
		ems.indices["category"] = &MetadataIndex{
			IndexType:   "category",
			IndexData:   make(map[string][]string),
			LastUpdated: time.Now(),
		}
	}
	
	for _, category := range categories {
		if !ems.contains(ems.indices["category"].IndexData[category], fileID) {
			ems.indices["category"].IndexData[category] = append(ems.indices["category"].IndexData[category], fileID)
		}
	}
	ems.indices["category"].LastUpdated = time.Now()
}

// updateHealthStatusIndex updates the health status index
func (ems *EnhancedMetadataStore) updateHealthStatusIndex(fileID, healthStatus string) {
	if ems.indices["health"] == nil {
		ems.indices["health"] = &MetadataIndex{
			IndexType:   "health",
			IndexData:   make(map[string][]string),
			LastUpdated: time.Now(),
		}
	}
	
	if !ems.contains(ems.indices["health"].IndexData[healthStatus], fileID) {
		ems.indices["health"].IndexData[healthStatus] = append(ems.indices["health"].IndexData[healthStatus], fileID)
	}
	ems.indices["health"].LastUpdated = time.Now()
}

// findCandidateFiles finds candidate files based on search query
func (ems *EnhancedMetadataStore) findCandidateFiles(query *SearchQuery) []string {
	ems.indicesMu.RLock()
	defer ems.indicesMu.RUnlock()
	
	candidateSet := make(map[string]bool)
	
	// Search by tags
	if len(query.Tags) > 0 && ems.indices["tag"] != nil {
		for _, tag := range query.Tags {
			if fileIDs, exists := ems.indices["tag"].IndexData[tag]; exists {
				for _, fileID := range fileIDs {
					candidateSet[fileID] = true
				}
			}
		}
	}
	
	// Search by categories
	if len(query.Categories) > 0 && ems.indices["category"] != nil {
		for _, category := range query.Categories {
			if fileIDs, exists := ems.indices["category"].IndexData[category]; exists {
				for _, fileID := range fileIDs {
					candidateSet[fileID] = true
				}
			}
		}
	}
	
	// Search by owners
	if len(query.OwnerIDs) > 0 && ems.indices["owner"] != nil {
		for _, ownerID := range query.OwnerIDs {
			if fileIDs, exists := ems.indices["owner"].IndexData[ownerID]; exists {
				for _, fileID := range fileIDs {
					candidateSet[fileID] = true
				}
			}
		}
	}
	
	// If no specific filters, get all files
	if len(candidateSet) == 0 {
		// Scan all files - this is expensive but necessary for full-text search
		ems.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false
			it := txn.NewIterator(opts)
			defer it.Close()
			
			prefix := []byte("file:")
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				key := it.Item().Key()
				fileID := strings.TrimPrefix(string(key), "file:")
				candidateSet[fileID] = true
			}
			return nil
		})
	}
	
	// Convert to slice
	candidates := make([]string, 0, len(candidateSet))
	for fileID := range candidateSet {
		candidates = append(candidates, fileID)
	}
	
	return candidates
}

// matchesQuery checks if a file matches the search query
func (ems *EnhancedMetadataStore) matchesQuery(fileMeta *EnhancedFileMetadata, query *SearchQuery) bool {
	// Text search
	if query.Query != "" {
		if !ems.matchesTextQuery(fileMeta, query.Query, query.Fields) {
			return false
		}
	}
	
	// Size filters
	if query.MinSize > 0 && fileMeta.FileSize < query.MinSize {
		return false
	}
	if query.MaxSize > 0 && fileMeta.FileSize > query.MaxSize {
		return false
	}
	
	// Time filters
	if query.CreatedAfter != nil && fileMeta.CreatedAt.Before(*query.CreatedAfter) {
		return false
	}
	if query.CreatedBefore != nil && fileMeta.CreatedAt.After(*query.CreatedBefore) {
		return false
	}
	if query.ModifiedAfter != nil && fileMeta.ModifiedAt.Before(*query.ModifiedAfter) {
		return false
	}
	if query.ModifiedBefore != nil && fileMeta.ModifiedAt.After(*query.ModifiedBefore) {
		return false
	}
	
	// Health status filter
	if len(query.HealthStatus) > 0 {
		found := false
		for _, status := range query.HealthStatus {
			if fileMeta.HealthStatus == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Deleted filter
	if query.IsDeleted != nil && fileMeta.IsDeleted != *query.IsDeleted {
		return false
	}
	
	return true
}

// matchesTextQuery performs text search across specified fields
func (ems *EnhancedMetadataStore) matchesTextQuery(fileMeta *EnhancedFileMetadata, queryText string, fields []string) bool {
	queryLower := strings.ToLower(queryText)
	
	// If no fields specified, search all text fields
	if len(fields) == 0 {
		fields = []string{"file_name", "original_name", "description", "tags", "categories"}
	}
	
	for _, field := range fields {
		switch field {
		case "file_name":
			if strings.Contains(strings.ToLower(fileMeta.FileName), queryLower) {
				return true
			}
		case "original_name":
			if strings.Contains(strings.ToLower(fileMeta.OriginalName), queryLower) {
				return true
			}
		case "description":
			if strings.Contains(strings.ToLower(fileMeta.Description), queryLower) {
				return true
			}
		case "tags":
			for _, tag := range fileMeta.Tags {
				if strings.Contains(strings.ToLower(tag), queryLower) {
					return true
				}
			}
		case "categories":
			for _, category := range fileMeta.Categories {
				if strings.Contains(strings.ToLower(category), queryLower) {
					return true
				}
			}
		}
	}
	
	return false
}

// sortResults sorts search results based on query parameters
func (ems *EnhancedMetadataStore) sortResults(files []*EnhancedFileMetadata, query *SearchQuery) {
	if query.SortBy == "" {
		query.SortBy = "modified_at"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}
	
	sort.Slice(files, func(i, j int) bool {
		var less bool
		
		switch query.SortBy {
		case "file_name":
			less = files[i].FileName < files[j].FileName
		case "file_size":
			less = files[i].FileSize < files[j].FileSize
		case "created_at":
			less = files[i].CreatedAt.Before(files[j].CreatedAt)
		case "modified_at":
			less = files[i].ModifiedAt.Before(files[j].ModifiedAt)
		case "accessed_at":
			less = files[i].AccessedAt.Before(files[j].AccessedAt)
		case "access_count":
			less = files[i].AccessCount < files[j].AccessCount
		default:
			less = files[i].ModifiedAt.Before(files[j].ModifiedAt)
		}
		
		if query.SortOrder == "desc" {
			return !less
		}
		return less
	})
}

// calculateFacets calculates facets for search results
func (ems *EnhancedMetadataStore) calculateFacets(files []*EnhancedFileMetadata) map[string]map[string]int64 {
	facets := make(map[string]map[string]int64)
	
	// Initialize facet maps
	facets["tags"] = make(map[string]int64)
	facets["categories"] = make(map[string]int64)
	facets["owners"] = make(map[string]int64)
	facets["health_status"] = make(map[string]int64)
	facets["storage_class"] = make(map[string]int64)
	
	for _, file := range files {
		// Count tags
		for _, tag := range file.Tags {
			facets["tags"][tag]++
		}
		
		// Count categories
		for _, category := range file.Categories {
			facets["categories"][category]++
		}
		
		// Count owners
		facets["owners"][file.OwnerID]++
		
		// Count health status
		facets["health_status"][file.HealthStatus]++
		
		// Count storage class
		facets["storage_class"][file.StorageClass]++
	}
	
	return facets
}

// Utility functions

func (ems *EnhancedMetadataStore) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (ems *EnhancedMetadataStore) getSizeBucket(size int64) string {
	if size < 1024 {
		return "tiny"       // < 1KB
	} else if size < 1024*1024 {
		return "small"      // < 1MB
	} else if size < 1024*1024*10 {
		return "medium"     // < 10MB
	} else if size < 1024*1024*100 {
		return "large"      // < 100MB
	} else {
		return "huge"       // >= 100MB
	}
}

func (ems *EnhancedMetadataStore) loadIndices() error {
	// Load indices from database - simplified implementation
	return nil
}

func (ems *EnhancedMetadataStore) saveIndices() error {
	// Save indices to database - simplified implementation
	return nil
}
