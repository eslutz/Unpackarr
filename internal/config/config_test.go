package config

import (
	"os"
	"testing"
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
}

func TestLoadWithEnv(t *testing.T) {
	os.Clearenv()
	if err := os.Setenv("HEALTH_PORT", "9090"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("LOG_LEVEL", "debug"); err != nil {
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

	os.Clearenv()
}
