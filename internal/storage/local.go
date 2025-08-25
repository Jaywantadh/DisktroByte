package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorage implements the Storage interface for the local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a new LocalStorage instance.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// Put stores a chunk on the local filesystem. The filename is the SHA-256 hash of the content.
func (s *LocalStorage) Put(chunkData io.Reader) (string, error) {
	// Read the data into memory to calculate the hash
	data, err := io.ReadAll(chunkData)
	if err != nil {
		return "", fmt.Errorf("failed to read chunk data: %w", err)
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	filePath := filepath.Join(s.basePath, hashStr)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write chunk to file: %w", err)
	}

	return hashStr, nil
}

// Get retrieves a chunk from the local filesystem.
func (s *LocalStorage) Get(id string) (io.ReadCloser, error) {
	filePath := filepath.Join(s.basePath, id)
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chunk not found: %s", id)
		}
		return nil, fmt.Errorf("failed to open chunk file: %w", err)
	}
	return file, nil
}

// GetPath returns the file path for a given chunk identifier.
func (s *LocalStorage) GetPath(id string) (string, error) {
	return filepath.Join(s.basePath, id), nil
}
