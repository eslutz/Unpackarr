package config

import (
	"fmt"
	"strings"
	"time"

	"golift.io/cnfg"
)

// Config holds the wrapper configuration
type Config struct {
	HealthPort int    `xml:"health_port"`
	LogLevel   string `xml:"log_level"`
}

// Legacy types kept for compatibility with unused packages
// These are not used by the wrapper but kept to avoid breaking builds
type ExtractConfig struct {
	Parallel         int           `xml:"parallel"`
	DeleteOrig       bool          `xml:"delete_orig"`
	Passwords        []string      `xml:"passwords"`
	ProgressInterval time.Duration `xml:"progress_interval"`
	StallTimeout     time.Duration `xml:"stall_timeout"`
}

type WatchConfig struct {
	FolderWatchEnabled bool          `xml:"folder_watch_enabled"`
	FolderWatchPaths   []string      `xml:"folder_watch_paths"`
	MarkerCleanup      time.Duration `xml:"marker_cleanup_interval"`
}

type TimingConfig struct {
	PollInterval time.Duration `xml:"poll_interval"`
	StarrTimeout time.Duration `xml:"starr_timeout"`
}

type WebhookConfig struct {
	URL      string        `xml:"url"`
	Template string        `xml:"template"`
	Events   []string      `xml:"events"`
	Timeout  time.Duration `xml:"timeout"`
}

type StarrApp struct {
	URL       string `xml:"url"`
	APIKey    string `xml:"api_key"`
	Paths     string `xml:"paths"`
	Protocols string `xml:"protocols"`
}

// HasPath checks if a path matches the configured paths (legacy, unused in wrapper)
func (s *StarrApp) HasPath(path string) bool {
	if path == "" {
		return false
	}
	if s.Paths == "" {
		return true
	}
	paths := strings.Split(s.Paths, ",")
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// HasProtocol checks if a protocol matches the configured protocols (legacy, unused in wrapper)
func (s *StarrApp) HasProtocol(protocol string) bool {
	if s.Protocols == "" {
		return true
	}
	protocols := strings.Split(s.Protocols, ",")
	for _, p := range protocols {
		if strings.EqualFold(strings.TrimSpace(p), protocol) {
			return true
		}
	}
	return false
}

// GetPaths returns the configured paths as a slice (legacy, unused in wrapper)
func (s *StarrApp) GetPaths() []string {
	if s.Paths == "" {
		return []string{}
	}
	paths := strings.Split(s.Paths, ",")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// GetProtocols returns the configured protocols as a slice (legacy, unused in wrapper)
func (s *StarrApp) GetProtocols() []string {
	if s.Protocols == "" {
		return []string{}
	}
	protocols := strings.Split(s.Protocols, ",")
	result := make([]string, 0, len(protocols))
	for _, p := range protocols {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
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
