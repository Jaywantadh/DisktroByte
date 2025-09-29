package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"sync"

	"bytes"

	"github.com/jaywantadh/DisktroByte/config"
	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/jaywantadh/DisktroByte/internal/encryptor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

type ChunkMetadata struct {
	Index        int    // Position of this chunk in the sequence
	Hash         string // SHA-256 hash of original chunk data
	Path         string // Storage path (hash) of encrypted chunk file
	Size         int64  // Encrypted size of this chunk
	Offset       int64  // Byte offset in the original file
	PrevIndex    int    // Index of previous chunk (-1 if first)
	NextIndex    int    // Index of next chunk (-1 if last)
	TotalChunks  int    // Total chunks in this file
	FileID       string // Unique file identifier (SHA-256 of full file)
	IsCompressed bool   // Whether this chunk was compressed
}

type chunkTask struct {
	Index int
	Data  []byte
}

// ChunkAndStore splits, compresses, encrypts, and stores file chunks, and writes metadata to the provided MetadataStore
func ChunkAndStore(filePath, password string, metaStore *metadata.MetadataStore, store storage.Storage) ([]ChunkMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}
	fileSize := fileInfo.Size()
	chunkSize := determineChunkSize(fileSize)

	// Calculate FileID (SHA-256 hash of entire file)
	fileID, err := CalculateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate file ID: %v", err)
	}

	parallelismRatio := config.Config.ParallelismRatio
	if parallelismRatio <= 0 {
		parallelismRatio = 2 // Default to 2 if config value is invalid
	}
	numWorkers := runtime.NumCPU() / parallelismRatio
	if numWorkers < 1 {
		numWorkers = 1
	}

	taskChan := make(chan chunkTask, numWorkers*2)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var metadataList []ChunkMetadata
	var errOnce sync.Once
	var processErr error
	var chunkHashes []string

	enc := encryptor.NewEncryptor()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				// Calculate hash of original data for integrity verification
				originalHash := sha256.Sum256(task.Data)
				originalHashStr := hex.EncodeToString(originalHash[:])

				// Process data (compression)
				var processedData []byte
				isCompressed := false
				if compressor.ShouldSkipCompression(filePath) {
					processedData = task.Data
				} else {
					compressed, err := compressor.CompressChunk(task.Data)
					if err != nil {
						setErrOnce(&errOnce, &processErr, fmt.Errorf("compression failed: %v", err))
						return
					}
					processedData = compressed
					isCompressed = true
				}

				// Encrypt processed data
				encrypted, err := enc.Encrypt(processedData, password)
				if err != nil {
					setErrOnce(&errOnce, &processErr, fmt.Errorf("encryption failed: %v", err))
					return
				}

				// Store encrypted chunk (returns storage path/hash)
				chunkPath, err := store.Put(bytes.NewReader(encrypted))
				if err != nil {
					setErrOnce(&errOnce, &processErr, fmt.Errorf("failed to store chunk: %v", err))
					return
				}

				// Calculate offset based on chunk index and size
				offset := int64(task.Index) * chunkSize

				// Create preliminary chunk metadata (linked-list info will be filled later)
				info := ChunkMetadata{
					Index:        task.Index,
					Hash:         originalHashStr, // Hash of original data for verification
					Path:         chunkPath,       // Storage path (encrypted hash)
					Size:         int64(len(encrypted)),
					Offset:       offset,
					PrevIndex:    -1, // Will be set later
					NextIndex:    -1, // Will be set later
					TotalChunks:  0,  // Will be set later
					FileID:       fileID,
					IsCompressed: isCompressed,
				}

				mu.Lock()
				metadataList = append(metadataList, info)
				chunkHashes = append(chunkHashes, originalHashStr) // Use original hash
				mu.Unlock()
			}
		}()
	}

	buf := make([]byte, chunkSize)
	index := 0
	for {
		n, err := io.ReadFull(file, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			close(taskChan)
			wg.Wait()
			return nil, fmt.Errorf("failed to read chunk: %v", err)
		}
		if n == 0 {
			break
		}

		taskCopy := make([]byte, n)
		copy(taskCopy, buf[:n])
		taskChan <- chunkTask{Index: index, Data: taskCopy}
		index++

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
	}

	close(taskChan)
	wg.Wait()

	if processErr != nil {
		return nil, processErr
	}

	// Sort metadata by index to ensure correct order
	sortMetadataByIndex(metadataList)
	totalChunks := len(metadataList)

	// Update all chunks with linked-list information and TotalChunks
	for i := range metadataList {
		metadataList[i].TotalChunks = totalChunks

		// Set PrevIndex
		if i > 0 {
			metadataList[i].PrevIndex = metadataList[i-1].Index
		} else {
			metadataList[i].PrevIndex = -1
		}

		// Set NextIndex
		if i < totalChunks-1 {
			metadataList[i].NextIndex = metadataList[i+1].Index
		} else {
			metadataList[i].NextIndex = -1
		}
	}

	// Store enhanced chunk metadata in BadgerDB
	if metaStore != nil {
		for _, chunk := range metadataList {
			chunkMeta := metadata.ChunkMetadata{
				Index:        chunk.Index,
				Hash:         chunk.Hash,
				Path:         chunk.Path,
				Size:         chunk.Size,
				Offset:       chunk.Offset,
				PrevIndex:    chunk.PrevIndex,
				NextIndex:    chunk.NextIndex,
				TotalChunks:  chunk.TotalChunks,
				FileID:       chunk.FileID,
				IsCompressed: chunk.IsCompressed,
			}
			if err := metaStore.PutChunkMetadata(chunkMeta); err != nil {
				return nil, fmt.Errorf("failed to store chunk metadata: %v", err)
			}
		}

		// Store file metadata in BadgerDB by both filename and FileID
		fileMeta := metadata.NewFileMetadata(fileInfo.Name(), fileSize, chunkHashes)
		if err := metaStore.PutFileMetadata(fileMeta); err != nil {
			return nil, fmt.Errorf("failed to store file metadata: %v", err)
		}
		if err := metaStore.PutFileMetadataByID(fileID, fileMeta); err != nil {
			return nil, fmt.Errorf("failed to store file metadata by ID: %v", err)
		}
	}

	return metadataList, nil
}

// ChunkAndProcess processes each chunk in memory (no file output)
// The callback now receives enhanced chunk metadata including FileID and linked-list information
func ChunkAndProcess(filePath string, password string, callback func(chunkMeta ChunkMetadata, encrypted []byte)) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %v", err)
	}
	fileSize := fileInfo.Size()
	chunkSize := determineChunkSize(fileSize)

	// Calculate FileID (SHA-256 hash of entire file)
	fileID, err := CalculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file ID: %v", err)
	}

	parallelismRatio := config.Config.ParallelismRatio
	if parallelismRatio <= 0 {
		parallelismRatio = 2 // Default to 2 if config value is invalid
	}
	numWorkers := runtime.NumCPU() / parallelismRatio
	if numWorkers < 1 {
		numWorkers = 1
	}

	taskChan := make(chan chunkTask, numWorkers*2)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errOnce sync.Once
	var processErr error
	var metadataList []ChunkMetadata
	var encryptedDataList [][]byte

	enc := encryptor.NewEncryptor()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
				hash := sha256.Sum256(task.Data)
				hashStr := hex.EncodeToString(hash[:])

				var processedData []byte
				if compressor.ShouldSkipCompression(filePath) {
					processedData = task.Data
				} else {
					compressed, err := compressor.CompressChunk(task.Data)
					if err != nil {
						setErrOnce(&errOnce, &processErr, fmt.Errorf("compression failed: %v", err))
						return
					}
					processedData = compressed
				}

				encrypted, err := enc.Encrypt(processedData, password)
				if err != nil {
					setErrOnce(&errOnce, &processErr, fmt.Errorf("encryption failed: %v", err))
					return
				}

				// Calculate offset and create chunk metadata
				offset := int64(task.Index) * chunkSize
				chunkMeta := ChunkMetadata{
					Index:       task.Index,
					Hash:        hashStr,
					Path:        "", // No path for in-memory processing
					Size:        int64(len(encrypted)),
					Offset:      offset,
					PrevIndex:   -1, // Will be set later
					NextIndex:   -1, // Will be set later
					TotalChunks: 0,  // Will be set later
					FileID:      fileID,
				}

				mu.Lock()
				metadataList = append(metadataList, chunkMeta)
				encryptedDataList = append(encryptedDataList, encrypted)
				mu.Unlock()
			}
		}()
	}

	buf := make([]byte, chunkSize)
	index := 0
	for {
		n, err := io.ReadFull(file, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			close(taskChan)
			wg.Wait()
			return fmt.Errorf("failed to read chunk: %v", err)
		}
		if n == 0 {
			break
		}

		taskCopy := make([]byte, n)
		copy(taskCopy, buf[:n])
		taskChan <- chunkTask{Index: index, Data: taskCopy}
		index++

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
	}

	close(taskChan)
	wg.Wait()

	if processErr != nil {
		return processErr
	}

	// Sort metadata by index to ensure correct order
	sortMetadataByIndex(metadataList)
	totalChunks := len(metadataList)

	// Update all chunks with linked-list information and TotalChunks
	for i := range metadataList {
		metadataList[i].TotalChunks = totalChunks

		// Set PrevIndex
		if i > 0 {
			metadataList[i].PrevIndex = metadataList[i-1].Index
		} else {
			metadataList[i].PrevIndex = -1
		}

		// Set NextIndex
		if i < totalChunks-1 {
			metadataList[i].NextIndex = metadataList[i+1].Index
		} else {
			metadataList[i].NextIndex = -1
		}
	}

	// Call callback for each chunk with enhanced metadata and encrypted data
	for i, chunkMeta := range metadataList {
		callback(chunkMeta, encryptedDataList[i])
	}

	return nil
}

func determineChunkSize(fileSize int64) int64 {
	switch {
	case fileSize <= 1*1024*1024:
		return 256 * 1024
	case fileSize <= 10*1024*1024:
		return 512 * 1024
	case fileSize <= 100*1024*1024:
		return 1 * 1024 * 1024
	case fileSize <= 1024*1024*1024:
		return 4 * 1024 * 1024
	default:
		return 8 * 1024 * 1024
	}
}

// CalculateFileHash computes SHA-256 hash of the entire file
func CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %v", err)
	}
	defer file.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to calculate file hash: %v", err)
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

// sortMetadataByIndex sorts chunk metadata by index to ensure proper ordering
func sortMetadataByIndex(chunks []ChunkMetadata) {
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Index < chunks[j].Index
	})
}

func setErrOnce(once *sync.Once, target *error, err error) {
	once.Do(func() {
		*target = err
	})
}
