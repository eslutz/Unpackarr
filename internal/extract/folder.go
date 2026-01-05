package extract

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"golift.io/xtractr"
)

type Watcher struct {
	config   *config.WatchConfig
	extract  *config.ExtractConfig
	queue    *Queue
	stop     chan struct{}
	mu       sync.RWMutex
	tracked  map[string]time.Time
}

func NewWatcher(cfg *config.WatchConfig, extractCfg *config.ExtractConfig, queue *Queue) *Watcher {
	return &Watcher{
		config:  cfg,
		extract: extractCfg,
		queue:   queue,
		stop:    make(chan struct{}),
		tracked: make(map[string]time.Time),
	}
}

func (w *Watcher) Start() {
	if !w.config.Enabled {
		return
	}

	go w.run()
}

func (w *Watcher) Stop() {
	close(w.stop)
}

func (w *Watcher) run() {
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	log.Printf("[Watcher] Started watching %d paths", len(w.config.Paths))

	for {
		select {
		case <-w.stop:
			log.Println("[Watcher] Stopped")
			return
		case <-ticker.C:
			w.scan()
		}
	}
}

func (w *Watcher) scan() {
	for _, path := range w.config.Paths {
		w.scanPath(path)
	}

	w.cleanTracked()
}

func (w *Watcher) scanPath(basePath string) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		log.Printf("[Watcher] Error reading %s: %v", basePath, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(basePath, entry.Name())
		
		if w.hasArchives(fullPath) && !w.isTracked(fullPath) {
			w.queueExtraction(fullPath, entry.Name())
		}
	}
}

func (w *Watcher) hasArchives(path string) bool {
	archives := xtractr.FindCompressedFiles(xtractr.Filter{Path: path})
	return len(archives) > 0
}

func (w *Watcher) isTracked(path string) bool {
	w.mu.RLock()
	_, exists := w.tracked[path]
	w.mu.RUnlock()
	return exists
}

func (w *Watcher) queueExtraction(path, name string) {
	w.mu.Lock()
	w.tracked[path] = time.Now()
	w.mu.Unlock()

	_, err := w.queue.Add(&Request{
		Name:       name,
		Path:       path,
		Source:     "folder",
		DeleteOrig: w.extract.DeleteOrig,
		Passwords:  w.extract.Passwords,
	})

	if err != nil {
		log.Printf("[Watcher] Error queuing %s: %v", name, err)
		w.mu.Lock()
		delete(w.tracked, path)
		w.mu.Unlock()
	} else {
		log.Printf("[Watcher] Queued: %s", name)
	}
}

func (w *Watcher) cleanTracked() {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for path, tracked := range w.tracked {
		if tracked.Before(cutoff) {
			delete(w.tracked, path)
		}
	}
}

func (w *Watcher) Paths() []string {
	paths := make([]string, len(w.config.Paths))
	copy(paths, w.config.Paths)
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
