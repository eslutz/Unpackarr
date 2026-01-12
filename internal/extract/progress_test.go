package extract

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressTracker_BasicFlow(t *testing.T) {
	t.Parallel()

	cfg := ProgressConfig{
		ReportInterval: 100 * time.Millisecond,
		StallTimeout:   500 * time.Millisecond,
	}

	tracker := NewProgressTracker("test-archive", "/path/to/archive", cfg)
	require.NotNil(t, tracker)

	// Start tracking
	tracker.Start(5, 1024*1024) // 5 files, 1MB total

	// Simulate progress
	tracker.UpdateFile(1, "file1.rar")
	tracker.UpdateBytes(200 * 1024) // 200KB

	tracker.UpdateFile(2, "file2.rar")
	tracker.UpdateBytes(300 * 1024) // 300KB more

	// Check progress
	percent, eta, rate := tracker.GetProgress()
	assert.Greater(t, percent, 0.0)
	assert.Less(t, percent, 100.0)
	assert.Greater(t, rate, 0.0)
	_ = eta // ETA may be 0 if very fast

	// Mark done
	tracker.Done(true, 1024*1024, 10, nil)
}

func TestProgressTracker_IsStalled(t *testing.T) {
	t.Parallel()

	cfg := ProgressConfig{
		ReportInterval: 50 * time.Millisecond,
		StallTimeout:   100 * time.Millisecond,
	}

	tracker := NewProgressTracker("test-stall", "/path/to/archive", cfg)
	tracker.Start(1, 1024)

	// Initially not stalled
	assert.False(t, tracker.IsStalled())

	// Wait for stall timeout
	time.Sleep(150 * time.Millisecond)
	assert.True(t, tracker.IsStalled())

	// Activity resets stall
	tracker.RecordActivity()
	assert.False(t, tracker.IsStalled())

	tracker.Done(true, 1024, 1, nil)
}

func TestProgressTracker_SetBytesExtracted(t *testing.T) {
	t.Parallel()

	cfg := DefaultProgressConfig()
	tracker := NewProgressTracker("test-set", "/path", cfg)
	tracker.Start(1, 1000)

	tracker.SetBytesExtracted(500)

	percent, _, _ := tracker.GetProgress()
	assert.InDelta(t, 50.0, percent, 0.1)

	tracker.Done(true, 1000, 1, nil)
}

func TestProgressTracker_NotStarted(t *testing.T) {
	t.Parallel()

	cfg := DefaultProgressConfig()
	tracker := NewProgressTracker("test-not-started", "/path", cfg)

	// Not stalled if not started
	assert.False(t, tracker.IsStalled())

	// GetProgress should return zeros
	percent, eta, rate := tracker.GetProgress()
	assert.Equal(t, 0.0, percent)
	assert.Equal(t, time.Duration(0), eta)
	assert.Equal(t, 0.0, rate)
}

func TestProgressManager_BasicFlow(t *testing.T) {
	t.Parallel()

	cfg := ProgressConfig{
		ReportInterval: 100 * time.Millisecond,
		StallTimeout:   5 * time.Minute,
	}

	pm := NewProgressManager(cfg)
	require.NotNil(t, pm)

	assert.Equal(t, 0, pm.ActiveCount())

	// Start tracking
	tracker := pm.StartTracking("archive1", "/path/1", 3, 1024)
	require.NotNil(t, tracker)
	assert.Equal(t, 1, pm.ActiveCount())

	// Get tracker
	got := pm.GetTracker("/path/1")
	assert.Equal(t, tracker, got)

	// Unknown path returns nil
	assert.Nil(t, pm.GetTracker("/unknown"))

	// Stop tracking
	pm.StopTracking("/path/1", true, 1024, 5, nil)
	assert.Equal(t, 0, pm.ActiveCount())
	assert.Nil(t, pm.GetTracker("/path/1"))
}

func TestProgressManager_MultipleConcurrent(t *testing.T) {
	t.Parallel()

	cfg := DefaultProgressConfig()
	pm := NewProgressManager(cfg)

	// Start multiple trackers
	pm.StartTracking("archive1", "/path/1", 1, 100)
	pm.StartTracking("archive2", "/path/2", 2, 200)
	pm.StartTracking("archive3", "/path/3", 3, 300)

	assert.Equal(t, 3, pm.ActiveCount())

	// Stop one
	pm.StopTracking("/path/2", true, 200, 5, nil)
	assert.Equal(t, 2, pm.ActiveCount())

	// Clean up
	pm.StopTracking("/path/1", true, 100, 3, nil)
	pm.StopTracking("/path/3", false, 0, 0, assert.AnError)
	assert.Equal(t, 0, pm.ActiveCount())
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

func TestDefaultProgressConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultProgressConfig()
	assert.Equal(t, 30*time.Second, cfg.ReportInterval)
	assert.Equal(t, 5*time.Minute, cfg.StallTimeout)
}

func TestProgressTracker_ZeroTotalBytes(t *testing.T) {
	t.Parallel()

	cfg := DefaultProgressConfig()
	tracker := NewProgressTracker("test-zero", "/path", cfg)
	tracker.Start(1, 0) // Zero total bytes

	// Should not panic, return zeros
	percent, eta, rate := tracker.GetProgress()
	assert.Equal(t, 0.0, percent)
	assert.Equal(t, time.Duration(0), eta)
	assert.Equal(t, 0.0, rate)

	tracker.Done(true, 0, 1, nil)
}
