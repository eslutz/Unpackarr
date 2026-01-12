package extract

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressManager_BasicFlow(t *testing.T) {
	t.Parallel()

	cfg := &config.ExtractConfig{
		ProgressInterval: 100 * time.Millisecond,
		StallTimeout:     500 * time.Millisecond,
	}

	pm := NewProgressManager(cfg)
	require.NotNil(t, pm)
	assert.Equal(t, 0, len(pm.trackers))

	// Create a temporary output directory for testing
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_output")
	err := os.MkdirAll(outputPath, 0755)
	require.NoError(t, err)

	// Start tracking
	pm.StartTracking("test-archive", "/path/to/archive", outputPath, 1, 1024*1024)
	assert.Equal(t, 1, len(pm.trackers))

	// Stop tracking
	pm.StopTracking("/path/to/archive")

	// Give goroutine time to clean up
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, len(pm.trackers))
}

func TestProgressManager_MultipleConcurrent(t *testing.T) {
	t.Parallel()

	cfg := &config.ExtractConfig{
		ProgressInterval: 100 * time.Millisecond,
		StallTimeout:     5 * time.Minute,
	}

	pm := NewProgressManager(cfg)

	tmpDir := t.TempDir()

	// Start multiple trackers
	pm.StartTracking("archive1", "/path/1", filepath.Join(tmpDir, "out1"), 1, 100)
	pm.StartTracking("archive2", "/path/2", filepath.Join(tmpDir, "out2"), 2, 200)
	pm.StartTracking("archive3", "/path/3", filepath.Join(tmpDir, "out3"), 3, 300)

	assert.Equal(t, 3, len(pm.trackers))

	// Stop one
	pm.StopTracking("/path/2")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, len(pm.trackers))

	// Clean up
	pm.StopTracking("/path/1")
	pm.StopTracking("/path/3")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, len(pm.trackers))
}

func TestProgressTracker_GetOutputSize(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_output")
	err := os.MkdirAll(outputPath, 0755)
	require.NoError(t, err)

	// Create test files
	file1 := filepath.Join(outputPath, "file1.txt")
	err = os.WriteFile(file1, make([]byte, 1024), 0644)
	require.NoError(t, err)

	file2 := filepath.Join(outputPath, "file2.txt")
	err = os.WriteFile(file2, make([]byte, 2048), 0644)
	require.NoError(t, err)

	tracker := &ProgressTracker{
		name:         "test",
		sourcePath:   "/source",
		outputPath:   outputPath,
		archiveBytes: 5000,
		startTime:    time.Now(),
		lastCheck:    time.Now(),
		done:         make(chan struct{}),
	}

	size := tracker.getOutputSize()
	assert.Equal(t, int64(3072), size) // 1024 + 2048
}

func TestProgressTracker_GetOutputSize_NonExistent(t *testing.T) {
	t.Parallel()

	tracker := &ProgressTracker{
		name:         "test",
		sourcePath:   "/source",
		outputPath:   "/nonexistent/path",
		archiveBytes: 1000,
		startTime:    time.Now(),
		lastCheck:    time.Now(),
		done:         make(chan struct{}),
	}

	size := tracker.getOutputSize()
	assert.Equal(t, int64(0), size)
}

func TestProgressTracker_IsStalled(t *testing.T) {
	t.Parallel()

	tracker := &ProgressTracker{
		name:         "test",
		sourcePath:   "/source",
		outputPath:   "/output",
		archiveBytes: 1000,
		startTime:    time.Now(),
		lastCheck:    time.Now(),
		stallWarned:  false,
		done:         make(chan struct{}),
	}

	assert.False(t, tracker.IsStalled())

	tracker.stallWarned = true
	assert.True(t, tracker.IsStalled())
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 500, "500.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 2, "2.00 GB"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatBytes(tc.bytes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		duration time.Duration
		expected string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{125 * time.Second, "2m5s"},
		{3665 * time.Second, "1h1m5s"},
		{7325 * time.Second, "2h2m5s"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := formatDuration(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestProgressTracker_CheckAndReport(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_output")
	err := os.MkdirAll(outputPath, 0755)
	require.NoError(t, err)

	cfg := &config.ExtractConfig{
		ProgressInterval: 100 * time.Millisecond,
		StallTimeout:     200 * time.Millisecond,
	}

	tracker := &ProgressTracker{
		name:         "test",
		sourcePath:   "/source",
		outputPath:   outputPath,
		archiveBytes: 10000,
		startTime:    time.Now(),
		lastCheck:    time.Now().Add(-150 * time.Millisecond),
		lastSize:     0,
		stallWarned:  false,
		done:         make(chan struct{}),
	}

	// Create a file to simulate extraction progress
	testFile := filepath.Join(outputPath, "test.bin")
	err = os.WriteFile(testFile, make([]byte, 5000), 0644)
	require.NoError(t, err)

	// First check - should not be stalled (has data)
	tracker.checkAndReport(cfg)
	assert.False(t, tracker.stallWarned)
	assert.Equal(t, int64(5000), tracker.lastSize)

	// Wait and check again with no change - should trigger stall
	time.Sleep(250 * time.Millisecond)
	tracker.checkAndReport(cfg)
	assert.True(t, tracker.stallWarned)

	// Add more data - should reset stall warning
	err = os.WriteFile(testFile, make([]byte, 8000), 0644)
	require.NoError(t, err)
	tracker.checkAndReport(cfg)
	assert.False(t, tracker.stallWarned)
	assert.Equal(t, int64(8000), tracker.lastSize)
}
