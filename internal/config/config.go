package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/eslutz/unpackarr/internal/logger"
	"golift.io/cnfg"
)

type Config struct {
	HealthPort int    `xml:"health_port"`
	LogLevel   string `xml:"log_level"`

	Extract ExtractConfig `xml:"extract"`
	Watch   WatchConfig   `xml:"watch"`
	Timing  TimingConfig  `xml:"timing"`
	Webhook WebhookConfig `xml:"webhook"`

	Sonarr  *StarrApp `xml:"sonarr"`
	Radarr  *StarrApp `xml:"radarr"`
	Lidarr  *StarrApp `xml:"lidarr"`
	Readarr *StarrApp `xml:"readarr"`
}

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
	Paths     string `xml:"paths"`      // Comma-separated path prefixes
	Protocols string `xml:"protocols"`  // Comma-separated protocols
}

func Load() (*Config, error) {
	cfg := &Config{
		HealthPort: 9092,
		LogLevel:   "INFO",
		Extract: ExtractConfig{
			Parallel:         1,
			DeleteOrig:       true,
			ProgressInterval: 30 * time.Second,
			StallTimeout:     5 * time.Minute,
		},
		Watch: WatchConfig{
			FolderWatchEnabled: false,
			FolderWatchPaths:   []string{"/downloads"},
			MarkerCleanup:      1 * time.Hour,
		},
		Timing: TimingConfig{
			PollInterval: 5 * time.Minute,
			StarrTimeout: 30 * time.Second,
		},
		Webhook: WebhookConfig{
			Template: "discord",
			Events:   []string{"extracted", "failed"},
			Timeout:  10 * time.Second,
		},
		Sonarr:  &StarrApp{},
		Radarr:  &StarrApp{},
		Lidarr:  &StarrApp{},
		Readarr: &StarrApp{},
	}

	if _, err := cnfg.UnmarshalENV(cfg, ""); err != nil {
		return nil, fmt.Errorf("unmarshal env: %w", err)
	}

	// Debug log config values after unmarshal
	logger.Debug("[Config] After unmarshal:")
	logger.Debug("[Config]   Timing.PollInterval: %v", cfg.Timing.PollInterval)
	logger.Debug("[Config]   Radarr != nil: %v", cfg.Radarr != nil)
	if cfg.Radarr != nil {
		logger.Debug("[Config]   Radarr.URL: %s", cfg.Radarr.URL)
		logger.Debug("[Config]   Radarr.Paths: %s → %v", cfg.Radarr.Paths, cfg.Radarr.GetPaths())
		logger.Debug("[Config]   Radarr.Protocols: %s → %v", cfg.Radarr.Protocols, cfg.Radarr.GetProtocols())
	}
	logger.Debug("[Config]   Sonarr != nil: %v", cfg.Sonarr != nil)
	if cfg.Sonarr != nil {
		logger.Debug("[Config]   Sonarr.URL: %s", cfg.Sonarr.URL)
		logger.Debug("[Config]   Sonarr.Paths: %s → %v", cfg.Sonarr.Paths, cfg.Sonarr.GetPaths())
		logger.Debug("[Config]   Sonarr.Protocols: %s → %v", cfg.Sonarr.Protocols, cfg.Sonarr.GetProtocols())
	}
	logger.Debug("[Config]   Lidarr != nil: %v", cfg.Lidarr != nil)
	if cfg.Lidarr != nil {
		logger.Debug("[Config]   Lidarr.URL: %s", cfg.Lidarr.URL)
		logger.Debug("[Config]   Lidarr.Paths: %s → %v", cfg.Lidarr.Paths, cfg.Lidarr.GetPaths())
		logger.Debug("[Config]   Lidarr.Protocols: %s → %v", cfg.Lidarr.Protocols, cfg.Lidarr.GetProtocols())
	}
	logger.Debug("[Config]   Readarr != nil: %v", cfg.Readarr != nil)
	if cfg.Readarr != nil {
		logger.Debug("[Config]   Readarr.URL: %s", cfg.Readarr.URL)
		logger.Debug("[Config]   Readarr.Paths: %s → %v", cfg.Readarr.Paths, cfg.Readarr.GetPaths())
		logger.Debug("[Config]   Readarr.Protocols: %s → %v", cfg.Readarr.Protocols, cfg.Readarr.GetProtocols())
	}

	if cfg.Extract.Parallel < 1 {
		cfg.Extract.Parallel = 1
	}

	cfg.LogLevel = strings.ToUpper(cfg.LogLevel)

	// Validate webhook events
	if cfg.Webhook.URL != "" {
		validEvents := map[string]struct{}{
			"extracted": {},
			"failed":    {},
		}
		filteredEvents := make([]string, 0, len(cfg.Webhook.Events))
		for _, evt := range cfg.Webhook.Events {
			if _, ok := validEvents[evt]; ok {
				filteredEvents = append(filteredEvents, evt)
			} else {
				logger.Warn("[Config] Invalid webhook event configured: %s (ignoring)", evt)
			}
		}
		if len(filteredEvents) == 0 {
			if len(cfg.Webhook.Events) > 0 {
				logger.Warn("[Config] All configured webhook events were invalid; falling back to defaults")
			}
			cfg.Webhook.Events = []string{"extracted", "failed"}
		} else {
			cfg.Webhook.Events = filteredEvents
		}
	}

	return cfg, nil
}

func (c *Config) EnabledApps() []string {
	apps := []string{}
	if c.Sonarr != nil && c.Sonarr.URL != "" {
		apps = append(apps, "sonarr")
	}
	if c.Radarr != nil && c.Radarr.URL != "" {
		apps = append(apps, "radarr")
	}
	if c.Lidarr != nil && c.Lidarr.URL != "" {
		apps = append(apps, "lidarr")
	}
	if c.Readarr != nil && c.Readarr.URL != "" {
		apps = append(apps, "readarr")
	}
	return apps
}

func (s *StarrApp) HasPath(path string) bool {
	if path == "" {
		return false
	}
	if s.Paths == "" {
		return true // Empty paths means process all
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

func (s *StarrApp) HasProtocol(protocol string) bool {
	if s.Protocols == "" {
		return true // Empty protocols means process all
	}
	protocols := strings.Split(s.Protocols, ",")
	for _, p := range protocols {
		if strings.EqualFold(strings.TrimSpace(p), protocol) {
			return true
		}
	}
	return false
}

// GetPaths returns the configured paths as a slice
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

// GetProtocols returns the configured protocols as a slice
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
