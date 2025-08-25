package transfer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/metadata"
)

// API version and base path
const (
	APIVersion = "v1"
	BasePath   = "/api/" + APIVersion + "/transfer"
)

// TransferStatus represents the current status of a transfer
type TransferStatus string

const (
	StatusPending    TransferStatus = "pending"
	StatusInProgress TransferStatus = "in_progress"
	StatusCompleted  TransferStatus = "completed"
	StatusFailed     TransferStatus = "failed"
	StatusCancelled  TransferStatus = "cancelled"
)

// InitiateTransferRequest represents a request to start a file transfer
type InitiateTransferRequest struct {
	FileName   string `json:"file_name"`
	ChunkCount int    `json:"chunk_count"`
	TotalSize  int64  `json:"total_size"`
	Password   string `json:"password,omitempty"` // Optional, for encrypted transfers
}

// InitiateTransferResponse represents the response to a transfer initiation
type InitiateTransferResponse struct {
	TransferID string         `json:"transfer_id"`
	Status     TransferStatus `json:"status"`
	Message    string         `json:"message,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// TransferStatusResponse represents the current status of a transfer
type TransferStatusResponse struct {
	TransferID      string         `json:"transfer_id"`
	Status          TransferStatus `json:"status"`
	ChunksReceived  int            `json:"chunks_received"`
	TotalChunks     int            `json:"total_chunks"`
	BytesReceived   int64          `json:"bytes_received"`
	TotalBytes      int64          `json:"total_bytes"`
	ProgressPercent float64        `json:"progress_percent"`
	Message         string         `json:"message,omitempty"`
	LastUpdated     time.Time      `json:"last_updated"`
}

// ChunkUploadRequest represents a chunk upload (binary data in body)
type ChunkUploadRequest struct {
	ChunkIndex int    `json:"chunk_index"`
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
}

// ChunkUploadResponse represents the response to a chunk upload
type ChunkUploadResponse struct {
	ChunkIndex int            `json:"chunk_index"`
	Status     TransferStatus `json:"status"`
	Hash       string         `json:"hash"`
	Message    string         `json:"message,omitempty"`
}

// CompleteTransferRequest represents the final transfer completion
type CompleteTransferRequest struct {
	FileMetadata metadata.FileMetadata `json:"file_metadata"`
	Password     string                `json:"password,omitempty"`
}

// CompleteTransferResponse represents the response to transfer completion
type CompleteTransferResponse struct {
	TransferID  string         `json:"transfer_id"`
	Status      TransferStatus `json:"status"`
	Message     string         `json:"message,omitempty"`
	CompletedAt time.Time      `json:"completed_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// Transfer represents an active transfer session
type Transfer struct {
	ID           string
	FileName     string
	ChunkCount   int
	TotalSize    int64
	Status       TransferStatus
	Chunks       map[int]*ChunkInfo
	FileMetadata *metadata.FileMetadata
	Password     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ChunkInfo represents information about a specific chunk
type ChunkInfo struct {
	Index    int
	Hash     string
	Size     int64
	Received bool
	Path     string
	Data     []byte
}

// API Endpoints
var (
	EndpointInitiateTransfer = BasePath + "/initiate"
	EndpointTransferStatus   = BasePath + "/{transfer_id}/status"
	EndpointUploadChunk      = BasePath + "/{transfer_id}/chunk/{chunk_index}"
	EndpointDownloadChunk    = BasePath + "/{transfer_id}/chunk/{chunk_index}"
	EndpointCompleteTransfer = BasePath + "/{transfer_id}/complete"
)

// HTTP Methods
const (
	MethodPOST = "POST"
	MethodGET  = "GET"
)

// Response helpers
func WriteJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: errorMsg,
		Code:    statusCode,
	}
	WriteJSONResponse(w, statusCode, response)
}

// Validation helpers
func (req *InitiateTransferRequest) Validate() error {
	if req.FileName == "" {
		return fmt.Errorf("file_name is required")
	}
	if req.ChunkCount <= 0 {
		return fmt.Errorf("chunk_count must be positive")
	}
	if req.TotalSize <= 0 {
		return fmt.Errorf("total_size must be positive")
	}
	return nil
}

func (req *ChunkUploadRequest) Validate() error {
	if req.ChunkIndex < 0 {
		return fmt.Errorf("chunk_index must be non-negative")
	}
	if req.Hash == "" {
		return fmt.Errorf("hash is required")
	}
	if req.Size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	return nil
}

// Progress calculation
func (t *Transfer) CalculateProgress() (int, int64, float64) {
	receivedChunks := 0
	receivedBytes := int64(0)

	for _, chunk := range t.Chunks {
		if chunk.Received {
			receivedChunks++
			receivedBytes += chunk.Size
		}
	}

	progressPercent := 0.0
	if t.ChunkCount > 0 {
		progressPercent = float64(receivedChunks) / float64(t.ChunkCount) * 100.0
	}

	return receivedChunks, receivedBytes, progressPercent
}
