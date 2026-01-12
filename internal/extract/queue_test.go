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

func TestQueueDeduplication(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}

	queue := NewQueue(cfg, nil)

	// First add should succeed
	req := &Request{
		Name:       "test-archive",
		Path:       "/downloads/test",
		Source:     "radarr",
		DeleteOrig: false,
	}

	_, added, err := queue.Add(req)
	if err != nil {
		t.Fatalf("First Add() should succeed: %v", err)
	}
	if !added {
		t.Error("First Add() should return added=true")
	}

	stats := queue.Stats()
	if stats.Waiting != 1 {
		t.Errorf("After first add, waiting = %d, want 1", stats.Waiting)
	}

	// Second add with same path should be skipped (no error, but not added)
	_, added, err = queue.Add(req)
	if err != nil {
		t.Fatalf("Second Add() should not error: %v", err)
	}
	if added {
		t.Error("Second Add() should return added=false for duplicate")
	}

	stats = queue.Stats()
	if stats.Waiting != 1 {
		t.Errorf("After duplicate add, waiting = %d, want 1 (should not add duplicate)", stats.Waiting)
	}

	// Different path should be added
	req2 := &Request{
		Name:       "another-archive",
		Path:       "/downloads/another",
		Source:     "sonarr",
		DeleteOrig: false,
	}

	_, added, err = queue.Add(req2)
	if err != nil {
		t.Fatalf("Third Add() with different path should succeed: %v", err)
	}
	if !added {
		t.Error("Third Add() with different path should return added=true")
	}

	stats = queue.Stats()
	if stats.Waiting != 2 {
		t.Errorf("After adding different path, waiting = %d, want 2", stats.Waiting)
	}
}

func TestQueueIsActive(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}

	queue := NewQueue(cfg, nil)

	// Initially no paths should be active
	if queue.IsActive("/downloads/test") {
		t.Error("Path should not be active before adding")
	}
	if queue.ActiveCount() != 0 {
		t.Errorf("ActiveCount should be 0, got %d", queue.ActiveCount())
	}

	// Add a request
	req := &Request{
		Name:       "test-archive",
		Path:       "/downloads/test",
		Source:     "radarr",
		DeleteOrig: false,
	}

	_, added, err := queue.Add(req)
	if err != nil {
		t.Fatalf("Add() should succeed: %v", err)
	}
	if !added {
		t.Error("Add() should return added=true")
	}

	// Path should now be active
	if !queue.IsActive("/downloads/test") {
		t.Error("Path should be active after adding")
	}
	if queue.ActiveCount() != 1 {
		t.Errorf("ActiveCount should be 1, got %d", queue.ActiveCount())
	}

	// Different path should not be active
	if queue.IsActive("/downloads/other") {
		t.Error("Different path should not be active")
	}
}

func TestQueueDifferentNamesSamePath(t *testing.T) {
	cfg := &config.ExtractConfig{
		Parallel: 1,
	}

	queue := NewQueue(cfg, nil)

	// Add first request
	req1 := &Request{
		Name:       "archive-v1",
		Path:       "/downloads/test",
		Source:     "radarr",
		DeleteOrig: false,
	}

	_, added, err := queue.Add(req1)
	if err != nil {
		t.Fatalf("First Add() should succeed: %v", err)
	}
	if !added {
		t.Error("First Add() should return added=true")
	}

	// Add second request with different name but same path - should be deduplicated
	req2 := &Request{
		Name:       "archive-v2",
		Path:       "/downloads/test",
		Source:     "sonarr",
		DeleteOrig: false,
	}

	_, added, err = queue.Add(req2)
	if err != nil {
		t.Fatalf("Second Add() should not error: %v", err)
	}
	if added {
		t.Error("Second Add() with same path should return added=false")
	}

	stats := queue.Stats()
	if stats.Waiting != 1 {
		t.Errorf("Waiting should be 1, got %d", stats.Waiting)
	}
}

