package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

// Server represents the HTTP server for receiving file transfers
type Server struct {
	transfers map[string]*Transfer
	mu        sync.RWMutex
	metaStore *metadata.MetadataStore
	store     storage.Storage
	port      int
}

// NewServer creates a new transfer server
func NewServer(metaStore *metadata.MetadataStore, store storage.Storage, port int) *Server {
	return &Server{
		transfers: make(map[string]*Transfer),
		metaStore: metaStore,
		store:     store,
		port:      port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc(BasePath+"/initiate", s.handleInitiateTransfer)
	mux.HandleFunc(BasePath+"/", s.handleTransferRoutes)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	fmt.Printf("Transfer server starting on port %d\n", s.port)
	return server.ListenAndServe()
}

// handleInitiateTransfer handles POST /api/v1/transfer/initiate
func (s *Server) handleInitiateTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req InitiateTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create new transfer
	transferID := uuid.New().String()
	transfer := &Transfer{
		ID:         transferID,
		FileName:   req.FileName,
		ChunkCount: req.ChunkCount,
		TotalSize:  req.TotalSize,
		Status:     StatusPending,
		Chunks:     make(map[int]*ChunkInfo),
		Password:   req.Password,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Store transfer
	s.mu.Lock()
	s.transfers[transferID] = transfer
	s.mu.Unlock()

	response := InitiateTransferResponse{
		TransferID: transferID,
		Status:     StatusPending,
		Message:    "Transfer initiated successfully",
		CreatedAt:  time.Now(),
	}

	WriteJSONResponse(w, http.StatusCreated, response)
}

// handleTransferRoutes handles transfer-specific routes
func (s *Server) handleTransferRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse URL to extract transfer_id and action
	path := strings.TrimPrefix(r.URL.Path, BasePath+"/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		WriteErrorResponse(w, http.StatusNotFound, "Invalid transfer route")
		return
	}

	transferID := parts[0]
	action := parts[1]

	// Get transfer
	s.mu.RLock()
	transfer, exists := s.transfers[transferID]
	s.mu.RUnlock()

	if !exists {
		WriteErrorResponse(w, http.StatusNotFound, "Transfer not found")
		return
	}

	switch action {
	case "status":
		s.handleTransferStatus(w, r, transfer)
	case "chunk":
		if len(parts) >= 3 {
			chunkIndex, err := strconv.Atoi(parts[2])
			if err != nil {
				WriteErrorResponse(w, http.StatusBadRequest, "Invalid chunk index")
				return
			}
			s.handleChunkUpload(w, r, transfer, chunkIndex)
		} else {
			WriteErrorResponse(w, http.StatusBadRequest, "Chunk index required")
		}
	case "complete":
		s.handleCompleteTransfer(w, r, transfer)
	default:
		WriteErrorResponse(w, http.StatusNotFound, "Invalid action")
	}
}

// handleTransferStatus handles GET /api/v1/transfer/{transfer_id}/status
func (s *Server) handleTransferStatus(w http.ResponseWriter, r *http.Request, transfer *Transfer) {
	if r.Method != http.MethodGet {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chunksReceived, bytesReceived, progressPercent := transfer.CalculateProgress()

	response := TransferStatusResponse{
		TransferID:      transfer.ID,
		Status:          transfer.Status,
		ChunksReceived:  chunksReceived,
		TotalChunks:     transfer.ChunkCount,
		BytesReceived:   bytesReceived,
		TotalBytes:      transfer.TotalSize,
		ProgressPercent: progressPercent,
		LastUpdated:     time.Now(),
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// handleChunkUpload handles POST /api/v1/transfer/{transfer_id}/chunk/{chunk_index}
func (s *Server) handleChunkUpload(w http.ResponseWriter, r *http.Request, transfer *Transfer, chunkIndex int) {
	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Read chunk data
	chunkData, err := io.ReadAll(r.Body)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Failed to read chunk data")
		return
	}

	// Get chunk metadata from headers
	hash := r.Header.Get("X-Chunk-Hash")
	sizeStr := r.Header.Get("X-Chunk-Size")

	if hash == "" || sizeStr == "" {
		WriteErrorResponse(w, http.StatusBadRequest, "Missing chunk metadata headers")
		return
	}

	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid chunk size")
		return
	}

	// Validate chunk
	if int64(len(chunkData)) != size {
		WriteErrorResponse(w, http.StatusBadRequest, "Chunk size mismatch")
		return
	}

	// Store chunk
	chunkPath, err := s.store.Put(bytes.NewReader(chunkData))
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store chunk")
		return
	}

	chunk := &ChunkInfo{
		Index:    chunkIndex,
		Hash:     hash,
		Size:     size,
		Received: true,
		Path:     chunkPath,
	}

	s.mu.Lock()
	transfer.Chunks[chunkIndex] = chunk
	transfer.Status = StatusInProgress
	transfer.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := ChunkUploadResponse{
		ChunkIndex: chunkIndex,
		Status:     StatusInProgress,
		Hash:       hash,
		Message:    "Chunk received successfully",
	}

	WriteJSONResponse(w, http.StatusOK, response)
}

// handleCompleteTransfer handles POST /api/v1/transfer/{transfer_id}/complete
func (s *Server) handleCompleteTransfer(w http.ResponseWriter, r *http.Request, transfer *Transfer) {
	if r.Method != http.MethodPost {
		WriteErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req CompleteTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Validate all chunks received
	chunksReceived, _, _ := transfer.CalculateProgress()
	if chunksReceived != transfer.ChunkCount {
		WriteErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Not all chunks received. Got %d/%d", chunksReceived, transfer.ChunkCount))
		return
	}

	// Store file metadata
	if s.metaStore != nil {
		err := s.metaStore.PutFileMetadata(req.FileMetadata)
		if err != nil {
			WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store file metadata")
			return
		}

		// Store chunk metadata
		for _, chunk := range transfer.Chunks {
			chunkMeta := metadata.ChunkMetadata{
				FileName: transfer.FileName,
				Index:    chunk.Index,
				Hash:     chunk.Hash,
				Path:     chunk.Path,
				Size:     chunk.Size,
			}
			err := s.metaStore.PutChunkMetadata(chunkMeta)
			if err != nil {
				WriteErrorResponse(w, http.StatusInternalServerError, "Failed to store chunk metadata")
				return
			}
		}
	}

	// Update transfer status
	s.mu.Lock()
	transfer.Status = StatusCompleted
	transfer.FileMetadata = &req.FileMetadata
	transfer.UpdatedAt = time.Now()
	s.mu.Unlock()

	response := CompleteTransferResponse{
		TransferID:  transfer.ID,
		Status:      StatusCompleted,
		Message:     "Transfer completed successfully",
		CompletedAt: time.Now(),
	}

	WriteJSONResponse(w, http.StatusOK, response)
}
