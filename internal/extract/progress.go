package extract

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/logger"
)

// ProgressTracker monitors extraction progress by scanning the output directory
type ProgressTracker struct {
	name         string
	sourcePath   string
	outputPath   string
	archiveBytes int64
	startTime    time.Time
	lastSize     int64
	lastCheck    time.Time
	stallWarned  bool
	done         chan struct{}
	mu           sync.RWMutex
}

// ProgressManager coordinates progress tracking across all extractions
type ProgressManager struct {
	cfg      *config.ExtractConfig
	trackers map[string]*ProgressTracker
	mu       sync.RWMutex
}

// NewProgressManager creates a new progress manager
func NewProgressManager(cfg *config.ExtractConfig) *ProgressManager {
	return &ProgressManager{
		cfg:      cfg,
		trackers: make(map[string]*ProgressTracker),
	}
}

// StartTracking begins monitoring progress for an extraction
func (m *ProgressManager) StartTracking(name, sourcePath, outputPath string, totalFiles int, archiveBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tracker := &ProgressTracker{
		name:         name,
		sourcePath:   sourcePath,
		outputPath:   outputPath,
		archiveBytes: archiveBytes,
		startTime:    time.Now(),
		lastCheck:    time.Now(),
		done:         make(chan struct{}),
	}

	m.trackers[sourcePath] = tracker

	logger.Info("Starting extraction: %s (archives: %s compressed)", name, formatBytes(archiveBytes))

	go tracker.monitorProgress(m.cfg)
}

// StopTracking ends monitoring for an extraction
func (m *ProgressManager) StopTracking(sourcePath string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tracker, exists := m.trackers[sourcePath]; exists {
		close(tracker.done)
		delete(m.trackers, sourcePath)
	}
}

// monitorProgress periodically checks output directory size and reports progress
func (t *ProgressTracker) monitorProgress(cfg *config.ExtractConfig) {
	interval := cfg.ProgressInterval
	if interval == 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			t.checkAndReport(cfg)
		}
	}
}

// checkAndReport walks the output directory, calculates progress, and logs
func (t *ProgressTracker) checkAndReport(cfg *config.ExtractConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()

	currentSize := t.getOutputSize()
	now := time.Now()
	elapsed := now.Sub(t.startTime)

	// Calculate speed based on time since last check
	var speed int64
	if !t.lastCheck.IsZero() {
		timeDelta := now.Sub(t.lastCheck).Seconds()
		if timeDelta > 0 {
			byteDelta := currentSize - t.lastSize
			speed = int64(float64(byteDelta) / timeDelta)
		}
	}

	// Check for stalls
	stallTimeout := cfg.StallTimeout
	if stallTimeout == 0 {
		stallTimeout = 5 * time.Minute
	}

	timeSinceActivity := now.Sub(t.lastCheck)
	isStalled := currentSize == t.lastSize && timeSinceActivity >= stallTimeout
	hasProgress := currentSize > t.lastSize

	// Update state
	t.lastSize = currentSize
	t.lastCheck = now

	// Report progress
	if currentSize > 0 {
		speedStr := ""
		if speed > 0 {
			speedStr = fmt.Sprintf(" | Speed: %s/s", formatBytes(speed))
		}

		logger.Info("[Progress] %s extracted (archives: %s compressed) | Running: %v%s",
			formatBytes(currentSize),
			formatBytes(t.archiveBytes),
			formatDuration(elapsed),
			speedStr)

		// Reset stall warning if we're making progress
		if hasProgress {
			t.stallWarned = false
		}
	}

	// Warn about stalls
	if isStalled && !t.stallWarned {
		logger.Warn("[Progress] WARNING: Extraction appears stalled: %s (no activity for %v)",
			t.name, timeSinceActivity.Round(time.Second))
		t.stallWarned = true
	}
}

// getOutputSize walks the output directory and sums all file sizes
func (t *ProgressTracker) getOutputSize() int64 {
	var totalSize int64

	// Check if output directory exists
	if _, err := os.Stat(t.outputPath); os.IsNotExist(err) {
		return 0
	}

	// Walk the directory tree
	_ = filepath.Walk(t.outputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize
}

// IsStalled returns whether this extraction has triggered a stall warning
func (t *ProgressTracker) IsStalled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stallWarned
}

// formatBytes converts bytes to human-readable format
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
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration converts duration to short human-readable format
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
