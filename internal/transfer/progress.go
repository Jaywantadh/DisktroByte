package transfer

import (
	"fmt"
	"sync"
	"time"
)

// ProgressTracker tracks the progress of file transfers
type ProgressTracker struct {
	transfers map[string]*TransferProgress
	mu        sync.RWMutex
}

// TransferProgress represents the progress of a single transfer
type TransferProgress struct {
	TransferID     string
	FileName       string
	Status         TransferStatus
	ChunksReceived int
	TotalChunks    int
	BytesReceived  int64
	TotalBytes     int64
	StartTime      time.Time
	LastUpdateTime time.Time
	Speed          float64 // bytes per second
	EstimatedTime  time.Duration
	mu             sync.RWMutex
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		transfers: make(map[string]*TransferProgress),
	}
}

// StartTracking starts tracking a new transfer
func (pt *ProgressTracker) StartTracking(transferID, fileName string, totalChunks int, totalBytes int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.transfers[transferID] = &TransferProgress{
		TransferID:     transferID,
		FileName:       fileName,
		Status:         StatusPending,
		TotalChunks:    totalChunks,
		TotalBytes:     totalBytes,
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
	}
}

// UpdateProgress updates the progress of a transfer
func (pt *ProgressTracker) UpdateProgress(transferID string, chunksReceived int, bytesReceived int64, status TransferStatus) {
	pt.mu.RLock()
	progress, exists := pt.transfers[transferID]
	pt.mu.RUnlock()

	if !exists {
		return
	}

	progress.mu.Lock()
	defer progress.mu.Unlock()

	now := time.Now()
	progress.ChunksReceived = chunksReceived
	progress.BytesReceived = bytesReceived
	progress.Status = status
	progress.LastUpdateTime = now

	// Calculate speed
	if !progress.StartTime.IsZero() {
		elapsed := now.Sub(progress.StartTime).Seconds()
		if elapsed > 0 {
			progress.Speed = float64(bytesReceived) / elapsed
		}
	}

	// Calculate estimated time remaining
	if progress.Speed > 0 && progress.TotalBytes > bytesReceived {
		remainingBytes := progress.TotalBytes - bytesReceived
		progress.EstimatedTime = time.Duration(remainingBytes/int64(progress.Speed)) * time.Second
	}
}

// GetProgress gets the current progress of a transfer
func (pt *ProgressTracker) GetProgress(transferID string) (*TransferProgress, bool) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress, exists := pt.transfers[transferID]
	return progress, exists
}

// RemoveTransfer removes a transfer from tracking
func (pt *ProgressTracker) RemoveTransfer(transferID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	delete(pt.transfers, transferID)
}

// GetAllProgress gets progress for all active transfers
func (pt *ProgressTracker) GetAllProgress() map[string]*TransferProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	result := make(map[string]*TransferProgress)
	for id, progress := range pt.transfers {
		result[id] = progress
	}
	return result
}

// PrintProgress prints the current progress of a transfer
func (pt *ProgressTracker) PrintProgress(transferID string) {
	progress, exists := pt.GetProgress(transferID)
	if !exists {
		fmt.Printf("Transfer %s not found\n", transferID)
		return
	}

	progress.mu.RLock()
	defer progress.mu.RUnlock()

	progressPercent := 0.0
	if progress.TotalChunks > 0 {
		progressPercent = float64(progress.ChunksReceived) / float64(progress.TotalChunks) * 100.0
	}

	fmt.Printf("\nTransfer Progress: %s\n", progress.FileName)
	fmt.Printf("  Status: %s\n", progress.Status)
	fmt.Printf("  Progress: %d/%d chunks (%.1f%%)\n",
		progress.ChunksReceived, progress.TotalChunks, progressPercent)
	fmt.Printf("  Bytes: %s/%s\n",
		formatBytes(progress.BytesReceived), formatBytes(progress.TotalBytes))

	if progress.Speed > 0 {
		fmt.Printf("  Speed: %s/s\n", formatBytes(int64(progress.Speed)))
	}

	if progress.EstimatedTime > 0 {
		fmt.Printf("  ETA: %s\n", formatDuration(progress.EstimatedTime))
	}

	fmt.Printf("  Last Update: %s\n", progress.LastUpdateTime.Format("15:04:05"))
}

// PrintAllProgress prints progress for all active transfers
func (pt *ProgressTracker) PrintAllProgress() {
	allProgress := pt.GetAllProgress()

	if len(allProgress) == 0 {
		fmt.Println("No active transfers")
		return
	}

	fmt.Printf("\n=== Active Transfers (%d) ===\n", len(allProgress))
	for transferID := range allProgress {
		pt.PrintProgress(transferID)
	}
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration into human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return fmt.Sprintf("%.0fh", d.Hours())
}

// MonitorProgress monitors a transfer and prints progress updates
func (pt *ProgressTracker) MonitorProgress(transferID string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			progress, exists := pt.GetProgress(transferID)
			if !exists {
				fmt.Printf("Transfer %s completed or not found\n", transferID)
				return
			}

			progress.mu.RLock()
			status := progress.Status
			progress.mu.RUnlock()

			if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
				pt.PrintProgress(transferID)
				return
			}

			pt.PrintProgress(transferID)
		}
	}
}
