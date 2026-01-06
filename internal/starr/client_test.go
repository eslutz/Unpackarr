package starr

import (
	"context"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
)

func TestNewClient(t *testing.T) {
	cfg := &config.StarrApp{
		URL:       "http://test:8989",
		APIKey:    "test-key",
		Paths:     []string{"/downloads"},
		Protocols: []string{"torrent"},
		Timeout:   30 * time.Second,
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)

	timing := &config.TimingConfig{
		PollInterval: 2 * time.Minute,
	}

	client := NewClient("test", cfg, queue, timing)
	if client == nil {
		t.Fatal("NewClient() should not return nil")
	}
	if client.Name() != "test" {
		t.Errorf("Name() = %s, want test", client.Name())
	}
}

func TestClientConfig(t *testing.T) {
	cfg := &config.StarrApp{
		URL:    "http://test:8989",
		APIKey: "test-key",
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	timing := &config.TimingConfig{}

	client := NewClient("test", cfg, queue, timing)
	starrCfg := client.Config()

	if starrCfg.URL != cfg.URL {
		t.Errorf("Config().URL = %s, want %s", starrCfg.URL, cfg.URL)
	}
	if starrCfg.APIKey != cfg.APIKey {
		t.Errorf("Config().APIKey = %s, want %s", starrCfg.APIKey, cfg.APIKey)
	}
}

func TestShouldProcess(t *testing.T) {
	cfg := &config.StarrApp{
		URL:       "http://test:8989",
		APIKey:    "test-key",
		Paths:     []string{"/downloads"},
		Protocols: []string{"torrent"},
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	timing := &config.TimingConfig{}

	client := NewClient("test", cfg, queue, timing)

	tests := []struct {
		name     string
		item     *QueueItem
		expected bool
	}{
		{
			name: "valid path and protocol",
			item: &QueueItem{
				Path:     "/downloads/movie",
				Protocol: "torrent",
			},
			expected: true,
		},
		{
			name: "invalid path",
			item: &QueueItem{
				Path:     "/other/movie",
				Protocol: "torrent",
			},
			expected: false,
		},
		{
			name: "invalid protocol",
			item: &QueueItem{
				Path:     "/downloads/movie",
				Protocol: "usenet",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.ShouldProcess(tt.item)
			if result != tt.expected {
				t.Errorf("ShouldProcess() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClientStatus(t *testing.T) {
	cfg := &config.StarrApp{
		URL:    "http://test:8989",
		APIKey: "test-key",
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	timing := &config.TimingConfig{}

	client := NewClient("test", cfg, queue, timing)

	connected, queueSize := client.Status()
	if connected {
		t.Error("Status() connected should be false initially")
	}
	if queueSize != 0 {
		t.Errorf("Status() queueSize = %d, want 0", queueSize)
	}

	client.SetQueueSize(5)
	_, queueSize = client.Status()
	if queueSize != 5 {
		t.Errorf("Status() queueSize after SetQueueSize(5) = %d, want 5", queueSize)
	}
}

func TestQueueItem(t *testing.T) {
	item := &QueueItem{
		ID:         123,
		Path:       "/downloads/movie",
		Protocol:   "torrent",
		Status:     "completed",
		Name:       "Movie.Name.2024",
		Size:       1024.0,
		DownloadID: "abc123",
	}

	if item.ID != 123 {
		t.Errorf("ID = %d, want 123", item.ID)
	}
	if item.Protocol != "torrent" {
		t.Errorf("Protocol = %s, want torrent", item.Protocol)
	}
}

func TestFormatError(t *testing.T) {
	err := formatError("Sonarr", "get queue", nil)
	if err == nil {
		t.Error("formatError() should return error even with nil input")
	}
}

func TestClientStop(t *testing.T) {
	cfg := &config.StarrApp{
		URL:    "http://test:8989",
		APIKey: "test",
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	timing := &config.TimingConfig{}

	client := NewClient("test", cfg, queue, timing)
	client.Stop()
}

func TestClientStart(t *testing.T) {
	cfg := &config.StarrApp{
		URL:     "http://test:8989",
		APIKey:  "test",
		Timeout: 1 * time.Second,
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	timing := &config.TimingConfig{
		PollInterval: 100 * time.Millisecond,
	}

	client := NewClient("test", cfg, queue, timing)

	testPoller := func(ctx context.Context, c *Client) error {
		return nil
	}

	client.Start(testPoller)
	time.Sleep(200 * time.Millisecond)
	client.Stop()
}

