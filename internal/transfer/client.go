package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

// Client represents the HTTP client for sending file transfers
type Client struct {
	baseURL    string
	httpClient *http.Client
	metaStore  *metadata.MetadataStore
	store      storage.Storage
}

// NewClient creates a new transfer client
func NewClient(baseURL string, metaStore *metadata.MetadataStore, store storage.Storage) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		metaStore: metaStore,
		store:     store,
	}
}

// SendFile sends a file to the specified server
func (c *Client) SendFile(filePath, password string) error {
	// Get file metadata
	fileMeta, err := c.metaStore.GetFileMetadata(filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to get file metadata: %v", err)
	}

	// Initiate transfer
	transferID, err := c.initiateTransfer(fileMeta, password)
	if err != nil {
		return fmt.Errorf("failed to initiate transfer: %v", err)
	}

	fmt.Printf("Transfer initiated: %s\n", transferID)

	// Send chunks
	err = c.sendChunks(transferID, fileMeta, password)
	if err != nil {
		return fmt.Errorf("failed to send chunks: %v", err)
	}

	// Complete transfer
	err = c.completeTransfer(transferID, fileMeta, password)
	if err != nil {
		return fmt.Errorf("failed to complete transfer: %v", err)
	}

	fmt.Printf("Transfer completed successfully: %s\n", transferID)
	return nil
}

// initiateTransfer initiates a new transfer
func (c *Client) initiateTransfer(fileMeta metadata.FileMetadata, password string) (string, error) {
	req := InitiateTransferRequest{
		FileName:   fileMeta.FileName,
		ChunkCount: fileMeta.NumChunks,
		TotalSize:  fileMeta.FileSize,
		Password:   password,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Post(
		c.baseURL+EndpointInitiateTransfer,
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("initiate transfer failed: %s - %s", resp.Status, string(body))
	}

	var response InitiateTransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.TransferID, nil
}

// sendChunks sends all chunks for a transfer
func (c *Client) sendChunks(transferID string, fileMeta metadata.FileMetadata, password string) error {
	for i, chunkHash := range fileMeta.ChunkHashes {
		// Get chunk metadata
		chunkMeta, err := c.metaStore.GetChunkMetadata(chunkHash)
		if err != nil {
			return fmt.Errorf("failed to get chunk metadata for hash %s: %v", chunkHash, err)
		}

		// Read chunk file
		chunkData, err := c.readChunkFile(chunkMeta.Path)
		if err != nil {
			return fmt.Errorf("failed to read chunk file %s: %v", chunkMeta.Path, err)
		}

		// Send chunk
		err = c.sendChunk(transferID, chunkMeta, chunkData)
		if err != nil {
			return fmt.Errorf("failed to send chunk %d: %v", i, err)
		}

		fmt.Printf("Sent chunk %d/%d (%s)\n", i+1, fileMeta.NumChunks, chunkHash[:8])
	}

	return nil
}

// sendChunk sends a single chunk
func (c *Client) sendChunk(transferID string, chunkMeta metadata.ChunkMetadata, chunkData []byte) error {
	url := fmt.Sprintf("%s%s/%s/chunk/%d", c.baseURL, BasePath, transferID, chunkMeta.Index)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(chunkData))
	if err != nil {
		return err
	}

	// Set chunk metadata headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Chunk-Hash", chunkMeta.Hash)
	req.Header.Set("X-Chunk-Size", strconv.FormatInt(chunkMeta.Size, 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chunk upload failed: %s - %s", resp.Status, string(body))
	}

	return nil
}

// completeTransfer completes the transfer
func (c *Client) completeTransfer(transferID string, fileMeta metadata.FileMetadata, password string) error {
	req := CompleteTransferRequest{
		FileMetadata: fileMeta,
		Password:     password,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s/%s/complete", c.baseURL, BasePath, transferID)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("complete transfer failed: %s - %s", resp.Status, string(body))
	}

	return nil
}

// GetTransferStatus gets the current status of a transfer
func (c *Client) GetTransferStatus(transferID string) (*TransferStatusResponse, error) {
	url := fmt.Sprintf("%s%s/%s/status", c.baseURL, BasePath, transferID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get status failed: %s - %s", resp.Status, string(body))
	}

	var status TransferStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// readChunkFile reads a chunk file from disk
func (c *Client) readChunkFile(chunkPath string) ([]byte, error) {
	chunkReader, err := c.store.Get(chunkPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chunk %s: %v", chunkPath, err)
	}
	defer chunkReader.Close()

	return io.ReadAll(chunkReader)
}

// SendFileWithChunker sends a file using the chunker to create chunks on-the-fly
func (c *Client) SendFileWithChunker(filePath, password string) error {
	// This method would use the chunker to create chunks and send them immediately
	// without storing them to disk first

	// TODO: Implement streaming chunk creation and sending
	// This would involve:
	// 1. Creating chunks on-the-fly using chunker
	// 2. Sending each chunk immediately
	// 3. Tracking progress

	return fmt.Errorf("streaming chunk sending not implemented yet")
}
