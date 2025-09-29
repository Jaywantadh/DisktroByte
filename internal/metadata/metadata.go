package metadata

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// FileMetadata represents metadata for a file.
type FileMetadata struct {
	FileName    string   `json:"file_name"`
	FileSize    int64    `json:"file_size"`
	NumChunks   int      `json:"num_chunks"`
	ChunkHashes []string `json:"chunk_hashes"`
	CreatedAt   int64    `json:"created_at"` // Unix timestamp
}

// ChunkMetadata represents metadata for a chunk with linked-list capabilities.
type ChunkMetadata struct {
	Index        int    `json:"index"`         // Position of this chunk in the sequence
	Hash         string `json:"hash"`          // SHA-256 hash of original chunk data
	Path         string `json:"path"`          // Storage path (hash) of encrypted chunk file
	Size         int64  `json:"size"`          // Encrypted size of this chunk
	Offset       int64  `json:"offset"`        // Byte offset in the original file
	PrevIndex    int    `json:"prev_index"`    // Index of previous chunk (-1 if first)
	NextIndex    int    `json:"next_index"`    // Index of next chunk (-1 if last)
	TotalChunks  int    `json:"total_chunks"`  // Total chunks in this file
	FileID       string `json:"file_id"`       // Unique file identifier (SHA-256 of full file)
	IsCompressed bool   `json:"is_compressed"` // Whether this chunk was compressed
}

// MetadataStore wraps BadgerDB for metadata operations.
type MetadataStore struct {
	db *badger.DB
}

// OpenMetadataStore opens (or creates) a BadgerDB at the given path.
func OpenMetadataStore(dbPath string) (*MetadataStore, error) {
	db, err := badger.Open(badger.DefaultOptions(dbPath).WithLogger(nil))
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %v", err)
	}
	return &MetadataStore{db: db}, nil
}

// Close closes the BadgerDB.
func (ms *MetadataStore) Close() error {
	return ms.db.Close()
}

// GetDB returns the underlying BadgerDB instance for advanced operations
func (ms *MetadataStore) GetDB() *badger.DB {
	return ms.db
}

// PutFileMetadata stores file metadata.
func (ms *MetadataStore) PutFileMetadata(meta FileMetadata) error {
	// Use filename as key for backward compatibility
	key := []byte("file:" + meta.FileName)
	val, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return ms.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

// PutFileMetadataByID stores file metadata by file ID.
func (ms *MetadataStore) PutFileMetadataByID(fileID string, meta FileMetadata) error {
	key := []byte("fileid:" + fileID)
	val, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return ms.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

// GetFileMetadata retrieves file metadata by file name.
func (ms *MetadataStore) GetFileMetadata(fileName string) (FileMetadata, error) {
	key := []byte("file:" + fileName)
	var meta FileMetadata
	err := ms.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &meta)
		})
	})
	return meta, err
}

// GetFileMetadataByID retrieves file metadata by file ID.
func (ms *MetadataStore) GetFileMetadataByID(fileID string) (FileMetadata, error) {
	key := []byte("fileid:" + fileID)
	var meta FileMetadata
	err := ms.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &meta)
		})
	})
	return meta, err
}

// PutChunkMetadata stores chunk metadata.
func (ms *MetadataStore) PutChunkMetadata(meta ChunkMetadata) error {
	key := []byte("chunk:" + meta.Hash)
	val, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return ms.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

// GetChunkMetadata retrieves chunk metadata by hash.
func (ms *MetadataStore) GetChunkMetadata(hash string) (ChunkMetadata, error) {
	key := []byte("chunk:" + hash)
	var meta ChunkMetadata
	err := ms.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &meta)
		})
	})
	return meta, err
}

// GetChunksByFileID retrieves all chunks for a specific FileID
func (ms *MetadataStore) GetChunksByFileID(fileID string) ([]ChunkMetadata, error) {
	var chunks []ChunkMetadata
	err := ms.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte("chunk:")
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var chunk ChunkMetadata
				if err := json.Unmarshal(val, &chunk); err != nil {
					return err
				}
				if chunk.FileID == fileID {
					chunks = append(chunks, chunk)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return chunks, err
}

// ValidateChunkChain validates the linked-list integrity of chunks for a file
func ValidateChunkChain(chunks []ChunkMetadata) error {
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks to validate")
	}

	// Check if all chunks have the same FileID and TotalChunks
	firstChunk := chunks[0]
	for _, chunk := range chunks {
		if chunk.FileID != firstChunk.FileID {
			return fmt.Errorf("chunk %d has different FileID: expected %s, got %s",
				chunk.Index, firstChunk.FileID, chunk.FileID)
		}
		if chunk.TotalChunks != firstChunk.TotalChunks {
			return fmt.Errorf("chunk %d has different TotalChunks: expected %d, got %d",
				chunk.Index, firstChunk.TotalChunks, chunk.TotalChunks)
		}
	}

	// Verify we have all chunks
	if len(chunks) != firstChunk.TotalChunks {
		return fmt.Errorf("missing chunks: expected %d, got %d", firstChunk.TotalChunks, len(chunks))
	}

	// Create map for easy lookup
	chunkMap := make(map[int]ChunkMetadata)
	for _, chunk := range chunks {
		chunkMap[chunk.Index] = chunk
	}

	// Validate linked-list chain
	for i := 0; i < firstChunk.TotalChunks; i++ {
		chunk, exists := chunkMap[i]
		if !exists {
			return fmt.Errorf("missing chunk at index %d", i)
		}

		// Check PrevIndex
		expectedPrevIndex := -1
		if i > 0 {
			expectedPrevIndex = i - 1
		}
		if chunk.PrevIndex != expectedPrevIndex {
			return fmt.Errorf("chunk %d has wrong PrevIndex: expected %d, got %d",
				i, expectedPrevIndex, chunk.PrevIndex)
		}

		// Check NextIndex
		expectedNextIndex := -1
		if i < firstChunk.TotalChunks-1 {
			expectedNextIndex = i + 1
		}
		if chunk.NextIndex != expectedNextIndex {
			return fmt.Errorf("chunk %d has wrong NextIndex: expected %d, got %d",
				i, expectedNextIndex, chunk.NextIndex)
		}

		// Check Offset
		// Note: Offset validation would need chunk size information from chunker
	}

	return nil
}

// Helper to create a new FileMetadata
func NewFileMetadata(fileName string, fileSize int64, chunkHashes []string) FileMetadata {
	return FileMetadata{
		FileName:    fileName,
		FileSize:    fileSize,
		NumChunks:   len(chunkHashes),
		ChunkHashes: chunkHashes,
		CreatedAt:   time.Now().Unix(),
	}
}
