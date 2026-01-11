package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HealthPort != 9092 {
		t.Errorf("HealthPort = %d, want 9092", cfg.HealthPort)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("LogLevel = %s, want INFO", cfg.LogLevel)
	}
	if cfg.Extract.Parallel != 1 {
		t.Errorf("Extract.Parallel = %d, want 1", cfg.Extract.Parallel)
	}
	if !cfg.Extract.DeleteOrig {
		t.Error("Extract.DeleteOrig should be true")
	}
}

func TestLoadWithEnv(t *testing.T) {
	os.Clearenv()
	if err := os.Setenv("HEALTH_PORT", "9090"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("LOG_LEVEL", "debug"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("EXTRACT_PARALLEL", "4"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("EXTRACT_DELETE_ORIG", "false"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HealthPort != 9090 {
		t.Errorf("HealthPort = %d, want 9090", cfg.HealthPort)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("LogLevel = %s, want DEBUG", cfg.LogLevel)
	}
	if cfg.Extract.Parallel != 4 {
		t.Errorf("Extract.Parallel = %d, want 4", cfg.Extract.Parallel)
	}
	if cfg.Extract.DeleteOrig {
		t.Error("Extract.DeleteOrig should be false")
	}

	os.Clearenv()
}

func TestEnabledApps(t *testing.T) {
	cfg := &Config{
		Sonarr: &StarrApp{URL: "http://sonarr:8989"},
		Radarr: &StarrApp{URL: "http://radarr:7878"},
	}

	apps := cfg.EnabledApps()
	if len(apps) != 2 {
		t.Errorf("EnabledApps() = %d apps, want 2", len(apps))
	}
}

func TestStarrAppHasPath(t *testing.T) {
	app := &StarrApp{
		Paths: "/downloads,/media",
	}

	if !app.HasPath("/downloads/movie") {
		t.Error("HasPath() should return true for /downloads/movie")
	}
	if app.HasPath("/other") {
		t.Error("HasPath() should return false for /other")
	}
}

func TestStarrAppHasProtocol(t *testing.T) {
	app := &StarrApp{
		Protocols: "torrent",
	}

	if !app.HasProtocol("torrent") {
		t.Error("HasProtocol() should return true for torrent")
	}
	if app.HasProtocol("usenet") {
		t.Error("HasProtocol() should return false for usenet")
	}

	appAll := &StarrApp{
		Protocols: "",
	}
	if !appAll.HasProtocol("anything") {
		t.Error("HasProtocol() with empty protocols should return true for any protocol")
	}
}

func TestStarrAppEnvironmentLoading(t *testing.T) {
	os.Clearenv()

	// Test Radarr configuration
	if err := os.Setenv("RADARR_URL", "http://radarr:7878"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("RADARR_API_KEY", "test-radarr-key"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("RADARR_PATHS", "/media,/downloads"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("RADARR_PROTOCOLS", "torrent,usenet"); err != nil {
		t.Fatal(err)
	}

	// Test Sonarr configuration
	if err := os.Setenv("SONARR_URL", "http://sonarr:8989"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("SONARR_API_KEY", "test-sonarr-key"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("SONARR_PATHS", "/tv"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("SONARR_PROTOCOLS", "torrent"); err != nil {
		t.Fatal(err)
	}

	// Test Lidarr configuration
	if err := os.Setenv("LIDARR_URL", "http://lidarr:8686"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("LIDARR_API_KEY", "test-lidarr-key"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("LIDARR_PATHS", "/music"); err != nil {
		t.Fatal(err)
	}

	// Test Readarr configuration
	if err := os.Setenv("READARR_URL", "http://readarr:8787"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("READARR_API_KEY", "test-readarr-key"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("READARR_PROTOCOLS", "usenet"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify Radarr
	if cfg.Radarr == nil {
		t.Fatal("Radarr should not be nil")
	}
	if cfg.Radarr.URL != "http://radarr:7878" {
		t.Errorf("Radarr.URL = %s, want http://radarr:7878", cfg.Radarr.URL)
	}
	if cfg.Radarr.APIKey != "test-radarr-key" {
		t.Errorf("Radarr.APIKey = %s, want test-radarr-key", cfg.Radarr.APIKey)
	}
	if cfg.Radarr.Paths != "/media,/downloads" {
		t.Errorf("Radarr.Paths = %s, want /media,/downloads", cfg.Radarr.Paths)
	}
	if cfg.Radarr.Protocols != "torrent,usenet" {
		t.Errorf("Radarr.Protocols = %s, want torrent,usenet", cfg.Radarr.Protocols)
	}
	radarrPaths := cfg.Radarr.GetPaths()
	if len(radarrPaths) != 2 || radarrPaths[0] != "/media" || radarrPaths[1] != "/downloads" {
		t.Errorf("Radarr.GetPaths() = %v, want [/media /downloads]", radarrPaths)
	}
	radarrProtocols := cfg.Radarr.GetProtocols()
	if len(radarrProtocols) != 2 || radarrProtocols[0] != "torrent" || radarrProtocols[1] != "usenet" {
		t.Errorf("Radarr.GetProtocols() = %v, want [torrent usenet]", radarrProtocols)
	}

	// Verify Sonarr
	if cfg.Sonarr == nil {
		t.Fatal("Sonarr should not be nil")
	}
	if cfg.Sonarr.URL != "http://sonarr:8989" {
		t.Errorf("Sonarr.URL = %s, want http://sonarr:8989", cfg.Sonarr.URL)
	}
	if cfg.Sonarr.APIKey != "test-sonarr-key" {
		t.Errorf("Sonarr.APIKey = %s, want test-sonarr-key", cfg.Sonarr.APIKey)
	}
	if cfg.Sonarr.Paths != "/tv" {
		t.Errorf("Sonarr.Paths = %s, want /tv", cfg.Sonarr.Paths)
	}
	sonarrPaths := cfg.Sonarr.GetPaths()
	if len(sonarrPaths) != 1 || sonarrPaths[0] != "/tv" {
		t.Errorf("Sonarr.GetPaths() = %v, want [/tv]", sonarrPaths)
	}

	// Verify Lidarr
	if cfg.Lidarr == nil {
		t.Fatal("Lidarr should not be nil")
	}
	if cfg.Lidarr.URL != "http://lidarr:8686" {
		t.Errorf("Lidarr.URL = %s, want http://lidarr:8686", cfg.Lidarr.URL)
	}
	if cfg.Lidarr.Paths != "/music" {
		t.Errorf("Lidarr.Paths = %s, want /music", cfg.Lidarr.Paths)
	}

	// Verify Readarr
	if cfg.Readarr == nil {
		t.Fatal("Readarr should not be nil")
	}
	if cfg.Readarr.URL != "http://readarr:8787" {
		t.Errorf("Readarr.URL = %s, want http://readarr:8787", cfg.Readarr.URL)
	}
	if cfg.Readarr.Protocols != "usenet" {
		t.Errorf("Readarr.Protocols = %s, want usenet", cfg.Readarr.Protocols)
	}

	os.Clearenv()
}

func TestMinimumParallel(t *testing.T) {
	os.Clearenv()
	if err := os.Setenv("EXTRACT_PARALLEL", "0"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Extract.Parallel != 1 {
		t.Errorf("Extract.Parallel = %d, want 1 (minimum)", cfg.Extract.Parallel)
	}

	os.Clearenv()
}

func TestDefaultTiming(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Timing.PollInterval != 5*time.Minute {
		t.Errorf("PollInterval = %v, want 5m", cfg.Timing.PollInterval)
	}
}
func TestWebhookEventValidation(t *testing.T) {
	os.Clearenv()
	if err := os.Setenv("WEBHOOK_URL", "http://webhook.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("WEBHOOK_EVENTS", "extracted,failed,invalid_event"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should only contain valid events
	if len(cfg.Webhook.Events) != 2 {
		t.Errorf("Webhook.Events = %d events, want 2", len(cfg.Webhook.Events))
	}

	validEvents := make(map[string]bool)
	for _, e := range cfg.Webhook.Events {
		validEvents[e] = true
	}

	if !validEvents["extracted"] {
		t.Error("Webhook.Events should contain 'extracted'")
	}
	if !validEvents["failed"] {
		t.Error("Webhook.Events should contain 'failed'")
	}
	if validEvents["invalid_event"] {
		t.Error("Webhook.Events should not contain 'invalid_event'")
	}

	os.Clearenv()
}

func TestWebhookEventValidationAllInvalid(t *testing.T) {
	os.Clearenv()
	if err := os.Setenv("WEBHOOK_URL", "http://webhook.example.com"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("WEBHOOK_EVENTS", "invalid1,invalid2"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should fall back to defaults
	if len(cfg.Webhook.Events) != 2 {
		t.Errorf("Webhook.Events = %d events, want 2 (defaults)", len(cfg.Webhook.Events))
	}

	os.Clearenv()
}

func TestWebhookEventValidationNoURL(t *testing.T) {
	os.Clearenv()
	// Don't set WEBHOOK_URL - webhook should be disabled

	if err := os.Setenv("WEBHOOK_EVENTS", "invalid1,invalid2"); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// With no URL, webhook events shouldn't be validated
	if len(cfg.Webhook.Events) != 2 {
		t.Errorf("Webhook.Events without URL = %d events, want 2", len(cfg.Webhook.Events))
	}

	os.Clearenv()
}
