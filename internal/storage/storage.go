package storage

import (
	"io"
)

// Storage defines the interface for storing and retrieving file chunks.
type Storage interface {
	// Put stores a chunk and returns its unique identifier (e.g., a hash or a path).
	Put(chunkData io.Reader) (string, error)
	// Get retrieves a chunk by its identifier.
	Get(id string) (io.ReadCloser, error)
	// GetPath returns the file path for a given chunk identifier.
	GetPath(id string) (string, error)
}
