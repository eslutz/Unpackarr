package extract

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/xtractr"
)

type Watcher struct {
	config          *config.WatchConfig
	extract         *config.ExtractConfig
	timing          *config.TimingConfig
	queue           *Queue
	stop            chan struct{}
	cleanupStop     chan struct{}
	deleteOrig      bool
	cleanupInterval time.Duration
}

func NewWatcher(cfg *config.WatchConfig, extractCfg *config.ExtractConfig, timing *config.TimingConfig, queue *Queue) *Watcher {
	return &Watcher{
		config:          cfg,
		extract:         extractCfg,
		timing:          timing,
		queue:           queue,
		stop:            make(chan struct{}),
		cleanupStop:     make(chan struct{}),
		deleteOrig:      extractCfg.DeleteOrig,
		cleanupInterval: cfg.MarkerCleanup,
	}
}

func (w *Watcher) Start() {
	if !w.config.FolderWatchEnabled {
		return
	}

	// Clean orphaned markers on startup
	if !w.deleteOrig {
		w.cleanOrphanedMarkers()
	}

	go w.run()
	go w.runCleanup()
}

func (w *Watcher) Stop() {
	close(w.stop)
	close(w.cleanupStop)
}

func (w *Watcher) run() {
	ticker := time.NewTicker(w.timing.PollInterval)
	defer ticker.Stop()

	logger.Info("[Watcher] Started watching %d paths (poll interval: %s)", len(w.config.FolderWatchPaths), w.timing.PollInterval)

	for {
		select {
		case <-w.stop:
			logger.Info("[Watcher] Stopped")
			return
		case <-ticker.C:
			w.scan()
		}
	}
}

func (w *Watcher) scan() {
	for _, path := range w.config.FolderWatchPaths {
		w.scanPath(path)
	}
}

func (w *Watcher) scanPath(basePath string) {
	logger.Debug("[Watcher] Scanning path: %s", basePath)
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logger.Error("[Watcher] Error reading %s: %v", basePath, err)
		return
	}

	logger.Debug("[Watcher] Found %d entries in %s", len(entries), basePath)
	dirsScanned := 0
	archivesFound := 0
	markedDirs := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirsScanned++
		fullPath := filepath.Join(basePath, entry.Name())

		hasArchives := w.hasArchives(fullPath)
		hasMarker := w.hasMarker(fullPath)

		logger.Debug("[Watcher] Directory %s: hasArchives=%t, hasMarker=%t", entry.Name(), hasArchives, hasMarker)

		if hasArchives {
			archivesFound++
			if hasMarker {
				markedDirs++
				logger.Debug("[Watcher] Skipping %s (already marked)", entry.Name())
			} else {
				logger.Debug("[Watcher] Queueing %s (archives found, no marker)", entry.Name())
				w.queueExtraction(fullPath, entry.Name())
			}
		}
	}

	logger.Debug("[Watcher] Scan complete for %s: %d dirs scanned, %d with archives, %d already marked", basePath, dirsScanned, archivesFound, markedDirs)
}

func (w *Watcher) hasArchives(path string) bool {
	archives := xtractr.FindCompressedFiles(xtractr.Filter{Path: path})
	count := len(archives)
	if count > 0 {
		logger.Debug("[Watcher] Found %d archive(s) in %s", count, path)
	}
	return count > 0
}

func (w *Watcher) queueExtraction(path, name string) {
	_, added, err := w.queue.Add(&Request{
		Name:       name,
		Path:       path,
		Source:     "folder",
		DeleteOrig: w.extract.DeleteOrig,
		Passwords:  w.extract.Passwords,
	})

	if err != nil {
		logger.Error("[Watcher] Error queuing %s: %v", name, err)
	} else if added {
		logger.Info("[Watcher] Queued: %s", name)
	} else {
		logger.Debug("[Watcher] Skipped %s: already queued", name)
	}
}

// markerPath returns the path to the hidden marker file for an archive
func markerPath(archivePath string) string {
	dir := filepath.Dir(archivePath)
	base := filepath.Base(archivePath)
	return filepath.Join(dir, "."+base+".unpackarr")
}

// hasMarker checks if a marker file exists for the given path
// Only returns true when deleteOrig is false
func (w *Watcher) hasMarker(path string) bool {
	if w.deleteOrig {
		logger.Debug("[Watcher] Marker check skipped for %s (deleteOrig=true)", path)
		return false
	}

	// Find first archive in the path
	archives := xtractr.FindCompressedFiles(xtractr.Filter{Path: path})
	if len(archives) == 0 {
		return false
	}

	// Get first archive from map
	for archivePath := range archives {
		marker := markerPath(archivePath)
		_, err := os.Stat(marker)
		hasMarker := err == nil
		logger.Debug("[Watcher] Marker check for %s: marker=%s, exists=%t", path, marker, hasMarker)
		return hasMarker
	}

	return false
}

// WriteMarkerForPath creates a marker file for the first archive found in the given path
func WriteMarkerForPath(path string) error {
	logger.Debug("[Watcher] WriteMarkerForPath called for: %s", path)
	// Find first archive in the path
	archives := xtractr.FindCompressedFiles(xtractr.Filter{Path: path})
	if len(archives) == 0 {
		logger.Debug("[Watcher] No archives found in %s, skipping marker creation", path)
		return nil // No archives found, nothing to mark
	}

	// Get first archive from map
	for archivePath := range archives {
		logger.Debug("[Watcher] Creating marker for archive: %s", archivePath)
		err := writeMarker(archivePath)
		if err != nil {
			logger.Debug("[Watcher] Failed to create marker: %v", err)
		} else {
			logger.Debug("[Watcher] Marker created successfully", )
		}
		return err
	}

	return nil
}

// writeMarker creates a marker file for the given archive path
func writeMarker(archivePath string) error {
	marker := markerPath(archivePath)
	f, err := os.Create(marker)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Write timestamp to marker file
	_, err = f.WriteString(time.Now().Format(time.RFC3339) + "\n")
	return err
}

// runCleanup periodically cleans orphaned marker files
func (w *Watcher) runCleanup() {
	if w.deleteOrig {
		return
	}

	ticker := time.NewTicker(w.cleanupInterval)
	defer ticker.Stop()

	logger.Info("[Watcher] Started marker cleanup with interval %s", w.cleanupInterval)

	for {
		select {
		case <-w.cleanupStop:
			logger.Info("[Watcher] Cleanup stopped")
			return
		case <-ticker.C:
			w.cleanOrphanedMarkers()
		}
	}
}

// cleanOrphanedMarkers removes marker files where the corresponding archive no longer exists
func (w *Watcher) cleanOrphanedMarkers() {
	logger.Debug("[Watcher] Starting orphaned marker cleanup")
	totalMarkers := 0
	orphanedMarkers := 0

	for _, basePath := range w.config.FolderWatchPaths {
		logger.Debug("[Watcher] Cleaning markers in: %s", basePath)
		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors and continue
			}

			if info.IsDir() {
				return nil
			}

			// Check if this is a marker file
			name := filepath.Base(path)
			if !strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".unpackarr") {
				return nil
			}

			totalMarkers++

			// Extract the original archive name
			archiveName := strings.TrimSuffix(strings.TrimPrefix(name, "."), ".unpackarr")
			archivePath := filepath.Join(filepath.Dir(path), archiveName)

			logger.Debug("[Watcher] Checking marker %s for archive %s", name, archiveName)

			// Check if archive still exists
			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				orphanedMarkers++
				logger.Debug("[Watcher] Archive %s not found, removing marker", archiveName)
				// Archive doesn't exist, remove the marker
				if err := os.Remove(path); err != nil {
					logger.Error("[Watcher] Error removing orphaned marker %s: %v", path, err)
				} else {
					logger.Info("[Watcher] Removed orphaned marker: %s", name)
				}
			} else {
				logger.Debug("[Watcher] Archive %s still exists, keeping marker", archiveName)
			}

			return nil
		})

		if err != nil {
			logger.Error("[Watcher] Error cleaning markers in %s: %v", basePath, err)
		}
	}

	logger.Debug("[Watcher] Cleanup complete: %d markers checked, %d orphaned markers removed", totalMarkers, orphanedMarkers)
}

func (w *Watcher) Paths() []string {
	paths := make([]string, len(w.config.FolderWatchPaths))
	copy(paths, w.config.FolderWatchPaths)
	return paths
}

func archiveInPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".rar") ||
		strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".7z") ||
		strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".gz") ||
		strings.HasSuffix(lower, ".bz2")
}
