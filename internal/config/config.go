package config

import (
	"fmt"
	"strings"

	"golift.io/cnfg"
)

// Config holds the wrapper configuration
type Config struct {
	HealthPort int    `xml:"health_port"`
	LogLevel   string `xml:"log_level"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		HealthPort: 9092,
		LogLevel:   "INFO",
	}

	if _, err := cnfg.UnmarshalENV(cfg, ""); err != nil {
		return nil, fmt.Errorf("unmarshal env: %w", err)
	}

	cfg.LogLevel = strings.ToUpper(cfg.LogLevel)

	return cfg, nil
}
