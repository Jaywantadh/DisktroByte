package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"bytes"

	"github.com/jaywantadh/DisktroByte/config"
	"github.com/jaywantadh/DisktroByte/internal/compressor"
	"github.com/jaywantadh/DisktroByte/internal/encryptor"
	"github.com/jaywantadh/DisktroByte/internal/metadata"
	"github.com/jaywantadh/DisktroByte/internal/storage"
)

type ChunkMetadata struct {
	Index int
	Hash  string
	Path  string
	Size  int64
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

				chunkPath, err := store.Put(bytes.NewReader(encrypted))
				if err != nil {
					setErrOnce(&errOnce, &processErr, fmt.Errorf("failed to store chunk: %v", err))
					return
				}

				info := ChunkMetadata{
					Index: task.Index,
					Hash:  hashStr,
					Path:  chunkPath,
					Size:  int64(len(encrypted)),
				}

				// Store chunk metadata in BadgerDB
				if metaStore != nil {
					chunkMeta := metadata.ChunkMetadata{
						FileName: fileInfo.Name(),
						Index:    task.Index,
						Hash:     hashStr,
						Path:     chunkPath,
						Size:     int64(len(encrypted)),
					}
					if err := metaStore.PutChunkMetadata(chunkMeta); err != nil {
						setErrOnce(&errOnce, &processErr, fmt.Errorf("failed to store chunk metadata: %v", err))
						return
					}
				}

				mu.Lock()
				metadataList = append(metadataList, info)
				chunkHashes = append(chunkHashes, hashStr)
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

	// Store file metadata in BadgerDB
	if metaStore != nil {
		fileMeta := metadata.NewFileMetadata(fileInfo.Name(), fileSize, chunkHashes)
		if err := metaStore.PutFileMetadata(fileMeta); err != nil {
			return nil, fmt.Errorf("failed to store file metadata: %v", err)
		}
	}

	return metadataList, nil
}

// ChunkAndProcess processes each chunk in memory (no file output)
func ChunkAndProcess(filePath string, password string, callback func(index int, encrypted []byte)) error {
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
	var errOnce sync.Once
	var processErr error

	enc := encryptor.NewEncryptor()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range taskChan {
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

				callback(task.Index, encrypted)
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

	return processErr
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

func setErrOnce(once *sync.Once, target *error, err error) {
	once.Do(func() {
		*target = err
	})
}
