package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/starr"
)

func TestNewServer(t *testing.T) {
	cfg := &config.WatchConfig{
		FolderWatchEnabled: false,
	}

	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)

	server := NewServer(queue, watcher, cfg)
	if server == nil {
		t.Fatal("NewServer() should not return nil")
	}
}

func TestHandlePing(t *testing.T) {
	cfg := &config.WatchConfig{FolderWatchEnabled: false}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	server.handlePing(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handlePing() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]string
	_ = json.NewDecoder(w.Body).Decode(&response)
	if response["status"] != "ok" {
		t.Errorf("handlePing() status = %s, want ok", response["status"])
	}
}

func TestHandleHealth(t *testing.T) {
	cfg := &config.WatchConfig{FolderWatchEnabled: false}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]bool
	_ = json.NewDecoder(w.Body).Decode(&response)
	if !response["healthy"] {
		t.Error("handleHealth() healthy should be true")
	}
}

func TestHandleStatus(t *testing.T) {
	cfg := &config.WatchConfig{
		FolderWatchEnabled: true,
		FolderWatchPaths:   []string{"/downloads"},
	}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()

	server.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleStatus() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&response)

	if response["queue"] == nil {
		t.Error("handleStatus() should include queue")
	}
	if response["folder_watcher"] == nil {
		t.Error("handleStatus() should include folder_watcher")
	}
}

func TestHandleMetrics(t *testing.T) {
	cfg := &config.WatchConfig{FolderWatchEnabled: false}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleMetrics() status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "unpackarr_queue_size") {
		t.Error("handleMetrics() should contain unpackarr_queue_size")
	}
}

func TestRegisterClient(t *testing.T) {
	cfg := &config.WatchConfig{FolderWatchEnabled: false}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	appCfg := &config.StarrApp{
		URL:    "http://test:8989",
		APIKey: "test",
	}
	timing := &config.TimingConfig{}
	client := starr.NewClient("test", appCfg, queue, timing, 30*time.Second)

	server.RegisterClient("test", client)

	if len(server.clients) != 1 {
		t.Errorf("RegisterClient() clients count = %d, want 1", len(server.clients))
	}
}

func TestHandleReady(t *testing.T) {
	cfg := &config.WatchConfig{FolderWatchEnabled: false}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleReady() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleReadyWithDisconnectedClient(t *testing.T) {
	cfg := &config.WatchConfig{FolderWatchEnabled: false}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	appCfg := &config.StarrApp{
		URL:    "http://test:8989",
		APIKey: "test",
	}
	timing := &config.TimingConfig{}
	client := starr.NewClient("test", appCfg, queue, timing, 30*time.Second)
	server.RegisterClient("test", client)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.handleReady(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("handleReady() with disconnected client status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleStatusWithClients(t *testing.T) {
	cfg := &config.WatchConfig{
		FolderWatchEnabled: false,
	}
	extractCfg := &config.ExtractConfig{Parallel: 1}
	queue := extract.NewQueue(extractCfg, nil)
	watcher := extract.NewWatcher(cfg, extractCfg, &config.TimingConfig{PollInterval: 2 * time.Minute}, queue)
	server := NewServer(queue, watcher, cfg)

	appCfg := &config.StarrApp{
		URL:    "http://sonarr:8989",
		APIKey: "test",
	}
	timing := &config.TimingConfig{}
	client := starr.NewClient("sonarr", appCfg, queue, timing, 30*time.Second)
	server.RegisterClient("sonarr", client)

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()

	server.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleStatus() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&response)

	apps := response["apps"].(map[string]interface{})
	if len(apps) != 1 {
		t.Errorf("handleStatus() apps count = %d, want 1", len(apps))
	}
}

