package extract

import (
	"errors"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
)

func TestNewQueue(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel:   2,
		DeleteOrig: true,
		Passwords:  []string{"test"},
		Timeout:    10 * time.Minute,
	}
	
	callback := func(r *Result) {
		// callback implementation
	}
	
	queue := NewQueue(cfg, callback)
	if queue == nil {
		t.Fatal("NewQueue() should not return nil")
	}
	if queue.config != cfg {
		t.Error("Queue config should match input")
	}
	if queue.callback == nil {
		t.Error("Queue callback should be set")
	}
}

func TestQueueStats(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}
	
	queue := NewQueue(cfg, nil)
	stats := queue.Stats()
	
	if stats.Waiting != 0 {
		t.Errorf("Initial waiting = %d, want 0", stats.Waiting)
	}
	if stats.Extracting != 0 {
		t.Errorf("Initial extracting = %d, want 0", stats.Extracting)
	}
}

func TestResult(t *testing.T) {
	result := &Result{
		Name:     "test",
		Source:   "folder",
		Started:  time.Now(),
		Elapsed:  30 * time.Second,
		Archives: 1,
		Files:    10,
		Size:     1024,
		Success:  true,
		Error:    nil,
	}
	
	if result.Name != "test" {
		t.Errorf("Name = %s, want test", result.Name)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestRequest(t *testing.T) {
	req := &Request{
		Name:       "test-archive",
		Path:       "/downloads/test",
		Source:     "sonarr",
		DeleteOrig: true,
		Passwords:  []string{"pass1", "pass2"},
	}
	
	if req.Name != "test-archive" {
		t.Errorf("Name = %s, want test-archive", req.Name)
	}
	if len(req.Passwords) != 2 {
		t.Errorf("Passwords length = %d, want 2", len(req.Passwords))
	}
}

func TestQueueStop(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}
	
	queue := NewQueue(cfg, nil)
	queue.Stop()
}

func TestQueuePrintf(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}
	
	queue := NewQueue(cfg, nil)
	queue.Printf("test %s", "message")
}

func TestQueueDebugf(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}
	
	queue := NewQueue(cfg, nil)
	queue.Debugf("test %s", "debug")
}

func TestResultWithError(t *testing.T) {
	result := &Result{
		Name:     "failed-extraction",
		Source:   "radarr",
		Started:  time.Now(),
		Elapsed:  5 * time.Second,
		Archives: 0,
		Files:    0,
		Size:     0,
		Success:  false,
		Error:    errors.New("extraction failed"),
	}
	
	if result.Success {
		t.Error("Success should be false for failed extraction")
	}
	if result.Error == nil {
		t.Error("Error should not be nil for failed extraction")
	}
}

