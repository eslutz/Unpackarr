package extract

import (
	"fmt"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/logger"
)

// ProgressConfig holds configuration for progress tracking
type ProgressConfig struct {
	// ReportInterval is how often to log progress updates (default: 30s)
	ReportInterval time.Duration
	// StallTimeout is how long without activity before considering stalled (default: 5m)
	StallTimeout time.Duration
}

// DefaultProgressConfig returns sensible defaults for progress tracking
func DefaultProgressConfig() ProgressConfig {
	return ProgressConfig{
		ReportInterval: 30 * time.Second,
		StallTimeout:   5 * time.Minute,
	}
}

// ProgressTracker tracks extraction progress for a single extraction job
type ProgressTracker struct {
	name           string
	path           string
	config         ProgressConfig
	mu             sync.RWMutex
	startTime      time.Time
	lastActivity   time.Time
	lastReport     time.Time
	totalFiles     int
	currentFile    int
	totalBytes     int64
	bytesExtracted int64
	currentArchive string
	isStarted      bool
	isDone         bool
	stallWarned    bool
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

// NewProgressTracker creates a new progress tracker for an extraction
func NewProgressTracker(name, path string, cfg ProgressConfig) *ProgressTracker {
	return &ProgressTracker{
		name:   name,
		path:   path,
		config: cfg,
		stopCh: make(chan struct{}),
	}
}

// Start begins tracking progress. Call this when extraction starts.
func (p *ProgressTracker) Start(totalFiles int, totalBytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.startTime = time.Now()
	p.lastActivity = p.startTime
	p.lastReport = p.startTime
	p.totalFiles = totalFiles
	p.totalBytes = totalBytes
	p.isStarted = true
	p.isDone = false

	logger.Info("[Progress] Started extraction: %s (%d archives, %s total)",
		p.name, totalFiles, formatBytes(totalBytes))

	// Start the stall detection goroutine
	p.wg.Add(1)
	go p.watchStall()
}

// UpdateFile updates progress when starting a new file
func (p *ProgressTracker) UpdateFile(fileIndex int, archiveName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.currentFile = fileIndex
	p.currentArchive = archiveName
	p.lastActivity = time.Now()

	// Log at debug level for each file
	logger.Debug("[Progress] %s: processing file %d/%d: %s",
		p.name, fileIndex, p.totalFiles, archiveName)
}

// UpdateBytes adds bytes extracted and reports progress if interval elapsed
func (p *ProgressTracker) UpdateBytes(bytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.bytesExtracted += bytes
	p.lastActivity = time.Now()
	p.stallWarned = false // Reset stall warning on activity

	// Check if it's time to report progress
	if time.Since(p.lastReport) >= p.config.ReportInterval {
		p.reportProgressLocked()
		p.lastReport = time.Now()
	}
}

// SetBytesExtracted sets the total bytes extracted so far (for use when we can't track incrementally)
func (p *ProgressTracker) SetBytesExtracted(bytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.bytesExtracted = bytes
	p.lastActivity = time.Now()
	p.stallWarned = false

	if time.Since(p.lastReport) >= p.config.ReportInterval {
		p.reportProgressLocked()
		p.lastReport = time.Now()
	}
}

// RecordActivity marks activity without updating bytes (for file writes, etc)
func (p *ProgressTracker) RecordActivity() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastActivity = time.Now()
	p.stallWarned = false
}

// Done marks the extraction as complete and logs final stats
func (p *ProgressTracker) Done(success bool, finalBytes int64, finalFiles int, err error) {
	p.mu.Lock()
	p.isDone = true
	p.bytesExtracted = finalBytes
	elapsed := time.Since(p.startTime)
	p.mu.Unlock()

	// Stop the stall watcher
	close(p.stopCh)
	p.wg.Wait()

	if success {
		rate := float64(finalBytes) / elapsed.Seconds()
		logger.Info("[Progress] Completed extraction: %s (%d files, %s in %v, %s/s)",
			p.name, finalFiles, formatBytes(finalBytes), elapsed.Round(time.Second), formatBytes(int64(rate)))
	} else {
		logger.Info("[Progress] Failed extraction: %s after %v: %v",
			p.name, elapsed.Round(time.Second), err)
	}
}

// GetProgress returns current progress information
func (p *ProgressTracker) GetProgress() (percent float64, eta time.Duration, rate float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.totalBytes == 0 {
		return 0, 0, 0
	}

	percent = float64(p.bytesExtracted) / float64(p.totalBytes) * 100
	elapsed := time.Since(p.startTime)

	if elapsed.Seconds() > 0 && p.bytesExtracted > 0 {
		rate = float64(p.bytesExtracted) / elapsed.Seconds()
		remaining := p.totalBytes - p.bytesExtracted
		if rate > 0 {
			eta = time.Duration(float64(remaining)/rate) * time.Second
		}
	}

	return percent, eta, rate
}

// IsStalled returns true if extraction appears to be stalled
func (p *ProgressTracker) IsStalled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.isDone || !p.isStarted {
		return false
	}

	return time.Since(p.lastActivity) > p.config.StallTimeout
}

// watchStall monitors for stalled extractions
func (p *ProgressTracker) watchStall() {
	defer p.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.checkStall()
		}
	}
}

// checkStall checks if extraction is stalled and warns
func (p *ProgressTracker) checkStall() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isDone || !p.isStarted {
		return
	}

	inactive := time.Since(p.lastActivity)
	if inactive > p.config.StallTimeout && !p.stallWarned {
		logger.Info("[Progress] WARNING: Extraction appears stalled: %s (no activity for %v, last file: %s)",
			p.name, inactive.Round(time.Second), p.currentArchive)
		p.stallWarned = true
	}
}

// reportProgressLocked logs current progress (must be called with lock held)
func (p *ProgressTracker) reportProgressLocked() {
	if p.totalBytes == 0 {
		logger.Info("[Progress] %s: processing file %d/%d (%s)",
			p.name, p.currentFile, p.totalFiles, p.currentArchive)
		return
	}

	percent := float64(p.bytesExtracted) / float64(p.totalBytes) * 100
	elapsed := time.Since(p.startTime)

	var etaStr string
	if elapsed.Seconds() > 0 && p.bytesExtracted > 0 {
		rate := float64(p.bytesExtracted) / elapsed.Seconds()
		remaining := p.totalBytes - p.bytesExtracted
		if rate > 0 {
			eta := time.Duration(float64(remaining)/rate) * time.Second
			etaStr = fmt.Sprintf(", ETA: %v", eta.Round(time.Second))
		}
	}

	logger.Info("[Progress] %s: %.1f%% (%s / %s, file %d/%d%s)",
		p.name, percent, formatBytes(p.bytesExtracted), formatBytes(p.totalBytes),
		p.currentFile, p.totalFiles, etaStr)
}

// formatBytes formats bytes in human readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ProgressManager manages multiple concurrent progress trackers
type ProgressManager struct {
	config   ProgressConfig
	mu       sync.RWMutex
	trackers map[string]*ProgressTracker
}

// NewProgressManager creates a new progress manager
func NewProgressManager(cfg ProgressConfig) *ProgressManager {
	return &ProgressManager{
		config:   cfg,
		trackers: make(map[string]*ProgressTracker),
	}
}

// StartTracking begins tracking progress for a path
func (pm *ProgressManager) StartTracking(name, path string, totalFiles int, totalBytes int64) *ProgressTracker {
	tracker := NewProgressTracker(name, path, pm.config)

	pm.mu.Lock()
	pm.trackers[path] = tracker
	pm.mu.Unlock()

	tracker.Start(totalFiles, totalBytes)
	return tracker
}

// GetTracker returns the tracker for a path, or nil if not found
func (pm *ProgressManager) GetTracker(path string) *ProgressTracker {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.trackers[path]
}

// StopTracking removes a tracker and marks it done
func (pm *ProgressManager) StopTracking(path string, success bool, finalBytes int64, finalFiles int, err error) {
	pm.mu.Lock()
	tracker := pm.trackers[path]
	delete(pm.trackers, path)
	pm.mu.Unlock()

	if tracker != nil {
		tracker.Done(success, finalBytes, finalFiles, err)
	}
}

// ActiveCount returns the number of active trackers
func (pm *ProgressManager) ActiveCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.trackers)
}
