package transfer

import (
	"io"
)

// TransferManager handles the uploading and downloading of file chunks.
type TransferManager interface {
	// Upload sends a chunk to a remote peer.
	Upload(chunkID string, chunkData io.Reader) error
	// Download retrieves a chunk from a remote peer.
	Download(chunkID string) (io.ReadCloser, error)
}
