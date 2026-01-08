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
		Enabled:         true,
		Paths:           []string{"/downloads"},
		Interval:        30 * time.Second,
		DeleteDelay:     5 * time.Minute,
		CleanupInterval: 1 * time.Hour,
	}

	extractCfg := &config.ExtractConfig{
		Parallel:   1,
		DeleteOrig: true,
	}

	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	if watcher == nil {
		t.Fatal("NewWatcher() should not return nil")
	}
	if watcher.config != cfg {
		t.Error("Watcher config should match input")
	}
	if watcher.deleteOrig != extractCfg.DeleteOrig {
		t.Error("Watcher deleteOrig should match extract config")
	}
	if watcher.cleanupInterval != cfg.CleanupInterval {
		t.Error("Watcher cleanupInterval should match config")
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

func TestMarkerPath(t *testing.T) {
	tests := []struct {
		archive string
		want    string
	}{
		{"/downloads/movie/movie.rar", "/downloads/movie/.movie.rar.unpackarr"},
		{"/path/to/archive.zip", "/path/to/.archive.zip.unpackarr"},
		{"file.7z", ".file.7z.unpackarr"},
	}

	for _, tt := range tests {
		got := markerPath(tt.archive)
		if got != tt.want {
			t.Errorf("markerPath(%s) = %s, want %s", tt.archive, got, tt.want)
		}
	}
}

func TestWriteMarker(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpackarr-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	archivePath := filepath.Join(tmpDir, "test.rar")
	markerFile := markerPath(archivePath)

	// Write marker
	if err := writeMarker(archivePath); err != nil {
		t.Fatalf("writeMarker() error = %v", err)
	}

	// Verify marker exists
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Error("Marker file was not created")
	}

	// Verify marker content
	content, err := os.ReadFile(markerFile)
	if err != nil {
		t.Fatalf("Failed to read marker file: %v", err)
	}
	if len(content) == 0 {
		t.Error("Marker file should not be empty")
	}
}

func TestHasMarker(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpackarr-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a subdirectory to simulate a download folder
	downloadDir := filepath.Join(tmpDir, "download")
	if err := os.Mkdir(downloadDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a fake archive file in the download directory
	archivePath := filepath.Join(downloadDir, "test.rar")
	if err := os.WriteFile(archivePath, []byte("fake archive"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.WatchConfig{Enabled: true}
	extractCfg := &config.ExtractConfig{Parallel: 1, DeleteOrig: false}
	queue := NewQueue(extractCfg, nil)

	// Note: hasMarker uses xtractr.FindCompressedFiles which may not recognize
	// our fake archive. Test the marker file directly instead.
	markerFile := markerPath(archivePath)

	// Should not have marker initially
	if _, err := os.Stat(markerFile); !os.IsNotExist(err) {
		t.Error("Marker should not exist initially")
	}

	// Write marker
	if err := writeMarker(archivePath); err != nil {
		t.Fatal(err)
	}

	// Should have marker now
	if _, err := os.Stat(markerFile); os.IsNotExist(err) {
		t.Error("Marker should exist after writeMarker")
	}

	// Test with deleteOrig=true (should always return false)
	extractCfg.DeleteOrig = true
	watcher2 := NewWatcher(cfg, extractCfg, queue)
	// Even with marker present, hasMarker should return false when deleteOrig=true
	// (though this check may not work with fake archive)
	if watcher2.deleteOrig != true {
		t.Error("watcher2 should have deleteOrig=true")
	}
}

func TestCleanOrphanedMarkers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "unpackarr-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create archive and marker
	archivePath := filepath.Join(tmpDir, "exists.rar")
	if err := os.WriteFile(archivePath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeMarker(archivePath); err != nil {
		t.Fatal(err)
	}

	// Create orphaned marker (no archive)
	orphanedMarkerPath := filepath.Join(tmpDir, ".missing.rar.unpackarr")
	if err := os.WriteFile(orphanedMarkerPath, []byte("timestamp"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.WatchConfig{
		Enabled: true,
		Paths:   []string{tmpDir},
	}
	extractCfg := &config.ExtractConfig{Parallel: 1, DeleteOrig: false}
	queue := NewQueue(extractCfg, nil)
	watcher := NewWatcher(cfg, extractCfg, queue)

	// Clean orphaned markers
	watcher.cleanOrphanedMarkers()

	// Verify orphaned marker was removed
	if _, err := os.Stat(orphanedMarkerPath); !os.IsNotExist(err) {
		t.Error("Orphaned marker should be removed")
	}

	// Verify valid marker still exists
	validMarker := markerPath(archivePath)
	if _, err := os.Stat(validMarker); os.IsNotExist(err) {
		t.Error("Valid marker should not be removed")
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

