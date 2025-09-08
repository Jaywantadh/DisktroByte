package streaming

import (
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StreamProcessor handles large file streaming operations
type StreamProcessor struct {
	bufferSize    int
	maxConcurrent int
	mu            sync.RWMutex
	activeStreams map[string]*StreamSession
}

// StreamSession represents an active streaming session
type StreamSession struct {
	ID            string
	FilePath      string
	FileSize      int64
	BytesRead     int64
	StartTime     time.Time
	LastActivity  time.Time
	BufferSize    int
	Reader        io.ReadCloser
	Writer        io.WriteCloser
	Context       context.Context
	Cancel        context.CancelFunc
	ProgressChan  chan StreamProgress
	ErrorChan     chan error
	Complete      bool
	Paused        bool
	mu            sync.RWMutex
}

// StreamProgress represents streaming progress information
type StreamProgress struct {
	SessionID     string
	BytesRead     int64
	TotalBytes    int64
	Percentage    float64
	Speed         int64 // bytes per second
	ETA           time.Duration
	CurrentBuffer int
	TotalBuffers  int
}

// StreamChunk represents a chunk of data being streamed
type StreamChunk struct {
	SessionID string
	ChunkID   string
	Index     int
	Data      []byte
	Size      int64
	Hash      string
	Timestamp time.Time
	IsLast    bool
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(bufferSize, maxConcurrent int) *StreamProcessor {
	if bufferSize <= 0 {
		bufferSize = 64 * 1024 // Default 64KB
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	return &StreamProcessor{
		bufferSize:    bufferSize,
		maxConcurrent: maxConcurrent,
		activeStreams: make(map[string]*StreamSession),
	}
}

// StartFileStream starts streaming a file
func (sp *StreamProcessor) StartFileStream(filePath string) (*StreamSession, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if len(sp.activeStreams) >= sp.maxConcurrent {
		return nil, fmt.Errorf("maximum concurrent streams reached: %d", sp.maxConcurrent)
	}

	// Check if file exists and get size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	// Create stream session
	sessionID := uuid.New().String()
	ctx, cancel := context.WithCancel(context.Background())

	session := &StreamSession{
		ID:           sessionID,
		FilePath:     filePath,
		FileSize:     fileInfo.Size(),
		BytesRead:    0,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		BufferSize:   sp.bufferSize,
		Context:      ctx,
		Cancel:       cancel,
		ProgressChan: make(chan StreamProgress, 100),
		ErrorChan:    make(chan error, 10),
		Complete:     false,
		Paused:       false,
	}

	sp.activeStreams[sessionID] = session

	fmt.Printf("📡 Started file stream: %s (Session: %s, Size: %d bytes)\n", 
		filePath, sessionID, fileInfo.Size())

	return session, nil
}

// ReadStream reads data from a file stream in chunks
func (sp *StreamProcessor) ReadStream(sessionID string) (<-chan StreamChunk, error) {
	sp.mu.RLock()
	session, exists := sp.activeStreams[sessionID]
	sp.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream session not found: %s", sessionID)
	}

	chunkChan := make(chan StreamChunk, 10)

	go func() {
		defer close(chunkChan)
		defer func() {
			session.mu.Lock()
			session.Complete = true
			if session.Reader != nil {
				session.Reader.Close()
			}
			session.mu.Unlock()
		}()

		// Open file for reading
		file, err := os.Open(session.FilePath)
		if err != nil {
			session.ErrorChan <- fmt.Errorf("failed to open file: %v", err)
			return
		}
		defer file.Close()

		session.mu.Lock()
		session.Reader = file
		session.mu.Unlock()

		reader := bufio.NewReader(file)
		buffer := make([]byte, session.BufferSize)
		chunkIndex := 0
		hasher := md5.New()

		for {
			select {
			case <-session.Context.Done():
				return
			default:
				// Check if paused
				session.mu.RLock()
				paused := session.Paused
				session.mu.RUnlock()

				if paused {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				// Read chunk
				n, err := reader.Read(buffer)
				if n > 0 {
					// Create chunk copy (important for concurrent access)
					chunkData := make([]byte, n)
					copy(chunkData, buffer[:n])

					// Calculate hash
					hasher.Reset()
					hasher.Write(chunkData)
					chunkHash := fmt.Sprintf("%x", hasher.Sum(nil))

					chunk := StreamChunk{
						SessionID: sessionID,
						ChunkID:   fmt.Sprintf("%s-%d", sessionID, chunkIndex),
						Index:     chunkIndex,
						Data:      chunkData,
						Size:      int64(n),
						Hash:      chunkHash,
						Timestamp: time.Now(),
						IsLast:    false,
					}

					// Update session progress
					session.mu.Lock()
					session.BytesRead += int64(n)
					session.LastActivity = time.Now()
					bytesRead := session.BytesRead
					totalBytes := session.FileSize
					session.mu.Unlock()

					// Send progress update
					progress := sp.calculateProgress(session, bytesRead, totalBytes)
					select {
					case session.ProgressChan <- progress:
					default:
						// Channel full, skip this progress update
					}

					// Check if this is the last chunk
					if err == io.EOF {
						chunk.IsLast = true
					}

					// Send chunk
					select {
					case chunkChan <- chunk:
					case <-session.Context.Done():
						return
					}

					chunkIndex++
				}

				if err == io.EOF {
					break
				} else if err != nil {
					session.ErrorChan <- fmt.Errorf("read error: %v", err)
					return
				}
			}
		}

		fmt.Printf("✅ Completed streaming file: %s (Session: %s)\n", 
			session.FilePath, sessionID)
	}()

	return chunkChan, nil
}

// WriteStream writes streaming data to a file
func (sp *StreamProcessor) WriteStream(sessionID string, outputPath string, chunkChan <-chan StreamChunk) error {
	sp.mu.RLock()
	session, exists := sp.activeStreams[sessionID]
	sp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("stream session not found: %s", sessionID)
	}

	// Create/open output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	session.mu.Lock()
	session.Writer = file
	session.mu.Unlock()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	var totalBytesWritten int64
	expectedChunkIndex := 0

	fmt.Printf("📝 Writing stream to file: %s (Session: %s)\n", outputPath, sessionID)

	for chunk := range chunkChan {
		select {
		case <-session.Context.Done():
			return fmt.Errorf("stream cancelled")
		default:
			// Verify chunk order
			if chunk.Index != expectedChunkIndex {
				return fmt.Errorf("chunk out of order: expected %d, got %d", 
					expectedChunkIndex, chunk.Index)
			}

			// Write chunk data
			n, err := writer.Write(chunk.Data)
			if err != nil {
				return fmt.Errorf("failed to write chunk %d: %v", chunk.Index, err)
			}

			if n != len(chunk.Data) {
				return fmt.Errorf("partial write for chunk %d: wrote %d, expected %d", 
					chunk.Index, n, len(chunk.Data))
			}

			totalBytesWritten += int64(n)
			expectedChunkIndex++

			// Update progress
			session.mu.Lock()
			session.BytesRead = totalBytesWritten
			session.LastActivity = time.Now()
			session.mu.Unlock()

			// Flush periodically
			if expectedChunkIndex%10 == 0 {
				if err := writer.Flush(); err != nil {
					return fmt.Errorf("failed to flush writer: %v", err)
				}
			}

			// Check if this was the last chunk
			if chunk.IsLast {
				break
			}
		}
	}

	// Final flush
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to final flush: %v", err)
	}

	session.mu.Lock()
	session.Complete = true
	session.mu.Unlock()

	fmt.Printf("✅ Completed writing stream: %s (%d bytes, Session: %s)\n", 
		outputPath, totalBytesWritten, sessionID)

	return nil
}

// PauseStream pauses a streaming session
func (sp *StreamProcessor) PauseStream(sessionID string) error {
	sp.mu.RLock()
	session, exists := sp.activeStreams[sessionID]
	sp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("stream session not found: %s", sessionID)
	}

	session.mu.Lock()
	session.Paused = true
	session.mu.Unlock()

	fmt.Printf("⏸️ Paused stream session: %s\n", sessionID)
	return nil
}

// ResumeStream resumes a paused streaming session
func (sp *StreamProcessor) ResumeStream(sessionID string) error {
	sp.mu.RLock()
	session, exists := sp.activeStreams[sessionID]
	sp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("stream session not found: %s", sessionID)
	}

	session.mu.Lock()
	session.Paused = false
	session.LastActivity = time.Now()
	session.mu.Unlock()

	fmt.Printf("▶️ Resumed stream session: %s\n", sessionID)
	return nil
}

// CancelStream cancels a streaming session
func (sp *StreamProcessor) CancelStream(sessionID string) error {
	sp.mu.Lock()
	session, exists := sp.activeStreams[sessionID]
	if exists {
		delete(sp.activeStreams, sessionID)
	}
	sp.mu.Unlock()

	if !exists {
		return fmt.Errorf("stream session not found: %s", sessionID)
	}

	session.Cancel()

	session.mu.Lock()
	if session.Reader != nil {
		session.Reader.Close()
	}
	if session.Writer != nil {
		session.Writer.Close()
	}
	session.mu.Unlock()

	fmt.Printf("❌ Cancelled stream session: %s\n", sessionID)
	return nil
}

// GetStreamProgress returns the current progress of a stream
func (sp *StreamProcessor) GetStreamProgress(sessionID string) (*StreamProgress, error) {
	sp.mu.RLock()
	session, exists := sp.activeStreams[sessionID]
	sp.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("stream session not found: %s", sessionID)
	}

	session.mu.RLock()
	bytesRead := session.BytesRead
	totalBytes := session.FileSize
	session.mu.RUnlock()

	progress := sp.calculateProgress(session, bytesRead, totalBytes)
	return &progress, nil
}

// GetActiveStreams returns all active streaming sessions
func (sp *StreamProcessor) GetActiveStreams() map[string]*StreamSession {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	activeStreams := make(map[string]*StreamSession)
	for id, session := range sp.activeStreams {
		activeStreams[id] = session
	}

	return activeStreams
}

// CleanupCompletedStreams removes completed streaming sessions
func (sp *StreamProcessor) CleanupCompletedStreams() int {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	cleaned := 0
	for id, session := range sp.activeStreams {
		session.mu.RLock()
		complete := session.Complete
		lastActivity := session.LastActivity
		session.mu.RUnlock()

		// Clean up sessions that are complete or inactive for more than 1 hour
		if complete || time.Since(lastActivity) > time.Hour {
			session.Cancel()
			delete(sp.activeStreams, id)
			cleaned++
			fmt.Printf("🧹 Cleaned up stream session: %s\n", id)
		}
	}

	return cleaned
}

// StreamFile provides a convenient method to stream an entire file
func (sp *StreamProcessor) StreamFile(inputPath, outputPath string) error {
	session, err := sp.StartFileStream(inputPath)
	if err != nil {
		return fmt.Errorf("failed to start stream: %v", err)
	}

	chunkChan, err := sp.ReadStream(session.ID)
	if err != nil {
		return fmt.Errorf("failed to read stream: %v", err)
	}

	if err := sp.WriteStream(session.ID, outputPath, chunkChan); err != nil {
		return fmt.Errorf("failed to write stream: %v", err)
	}

	return nil
}

// calculateProgress calculates streaming progress
func (sp *StreamProcessor) calculateProgress(session *StreamSession, bytesRead, totalBytes int64) StreamProgress {
	var percentage float64
	var speed int64
	var eta time.Duration

	if totalBytes > 0 {
		percentage = float64(bytesRead) / float64(totalBytes) * 100
	}

	session.mu.RLock()
	elapsed := time.Since(session.StartTime)
	session.mu.RUnlock()

	if elapsed.Seconds() > 0 {
		speed = int64(float64(bytesRead) / elapsed.Seconds())
		
		if speed > 0 && bytesRead < totalBytes {
			remainingBytes := totalBytes - bytesRead
			eta = time.Duration(float64(remainingBytes)/float64(speed)) * time.Second
		}
	}

	bufferSize := int64(sp.bufferSize)
	totalBuffers := (totalBytes + bufferSize - 1) / bufferSize
	currentBuffer := (bytesRead + bufferSize - 1) / bufferSize

	return StreamProgress{
		SessionID:     session.ID,
		BytesRead:     bytesRead,
		TotalBytes:    totalBytes,
		Percentage:    percentage,
		Speed:         speed,
		ETA:           eta,
		CurrentBuffer: int(currentBuffer),
		TotalBuffers:  int(totalBuffers),
	}
}

// StreamHasher provides streaming hash calculation
type StreamHasher struct {
	hasher hash.Hash
	mu     sync.Mutex
}

// NewStreamHasher creates a new streaming hasher
func NewStreamHasher() *StreamHasher {
	return &StreamHasher{
		hasher: md5.New(),
	}
}

// Update adds data to the hash calculation
func (sh *StreamHasher) Update(data []byte) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.hasher.Write(data)
}

// Finalize returns the final hash
func (sh *StreamHasher) Finalize() string {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	return fmt.Sprintf("%x", sh.hasher.Sum(nil))
}

// Reset resets the hasher
func (sh *StreamHasher) Reset() {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.hasher.Reset()
}
