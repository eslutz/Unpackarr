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
	if cfg.Timing.StartDelay != 1*time.Minute {
		t.Errorf("StartDelay = %v, want 1m", cfg.Timing.StartDelay)
	}
	if cfg.Timing.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.Timing.MaxRetries)
	}
}
