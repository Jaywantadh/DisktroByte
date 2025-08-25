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

// ChunkMetadata represents metadata for a chunk.
type ChunkMetadata struct {
	FileName string `json:"file_name"`
	Index    int    `json:"index"`
	Hash     string `json:"hash"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
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

// PutFileMetadata stores file metadata.
func (ms *MetadataStore) PutFileMetadata(meta FileMetadata) error {
	key := []byte("file:" + meta.FileName)
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