package extract

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
)

func TestNewWatcher(t *testing.T) {
	cfg := &config.WatchConfig{
		Enabled:     true,
		Paths:       []string{"/downloads"},
		Interval:    30 * time.Second,
		DeleteDelay: 5 * time.Minute,
	}

	extractCfg := &config.ExtractConfig{
		Parallel: 1,
	}

	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	if watcher == nil {
		t.Fatal("NewWatcher() should not return nil")
	}
	if watcher.config != cfg {
		t.Error("Watcher config should match input")
	}
}

func TestWatcherPaths(t *testing.T) {
	cfg := &config.WatchConfig{
		Enabled: true,
		Paths:   []string{"/downloads", "/media"},
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	paths := watcher.Paths()
	if len(paths) != 2 {
		t.Errorf("Paths() = %d paths, want 2", len(paths))
	}
}

func TestArchiveInPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/downloads/movie.rar", true},
		{"/downloads/movie.zip", true},
		{"/downloads/movie.7z", true},
		{"/downloads/movie.tar", true},
		{"/downloads/movie.gz", true},
		{"/downloads/movie.bz2", true},
		{"/downloads/movie.mkv", false},
		{"/downloads/movie.txt", false},
	}

	for _, tt := range tests {
		got := archiveInPath(tt.path)
		if got != tt.want {
			t.Errorf("archiveInPath(%s) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestWatcherDisabled(t *testing.T) {
	cfg := &config.WatchConfig{
		Enabled: false,
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	watcher.Start()
	time.Sleep(100 * time.Millisecond)
	watcher.Stop()
}

func TestIsTracked(t *testing.T) {
	cfg := &config.WatchConfig{
		Enabled: true,
		Paths:   []string{"/downloads"},
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	if watcher.isTracked("/downloads/test") {
		t.Error("isTracked() should return false for untracked path")
	}

	watcher.tracked["/downloads/test"] = time.Now()

	if !watcher.isTracked("/downloads/test") {
		t.Error("isTracked() should return true for tracked path")
	}
}

func TestCleanTracked(t *testing.T) {
	cfg := &config.WatchConfig{
		Enabled: true,
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	watcher.tracked["/old"] = time.Now().Add(-25 * time.Hour)
	watcher.tracked["/new"] = time.Now()

	watcher.cleanTracked()

	if watcher.isTracked("/old") {
		t.Error("cleanTracked() should remove old entries")
	}
	if !watcher.isTracked("/new") {
		t.Error("cleanTracked() should keep recent entries")
	}
}

func TestScanPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpackarr-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	testDir := filepath.Join(tmpDir, "test-folder")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.WatchConfig{
		Enabled: true,
		Paths:   []string{tmpDir},
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	watcher.scanPath(tmpDir)
}

func TestHasArchives(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpackarr-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	cfg := &config.WatchConfig{
		Enabled: true,
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	if watcher.hasArchives(tmpDir) {
		t.Error("hasArchives() should return false for empty directory")
	}
}

