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

	if cfg.HealthPort != 8085 {
		t.Errorf("HealthPort = %d, want 8085", cfg.HealthPort)
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
		Paths: []string{"/downloads", "/media"},
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
		Protocols: []string{"torrent"},
	}

	if !app.HasProtocol("torrent") {
		t.Error("HasProtocol() should return true for torrent")
	}
	if app.HasProtocol("usenet") {
		t.Error("HasProtocol() should return false for usenet")
	}

	appAll := &StarrApp{
		Protocols: []string{},
	}
	if !appAll.HasProtocol("anything") {
		t.Error("HasProtocol() with empty protocols should return true for any protocol")
	}
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

	if cfg.Timing.PollInterval != 2*time.Minute {
		t.Errorf("PollInterval = %v, want 2m", cfg.Timing.PollInterval)
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
