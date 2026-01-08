package config

import (
	"fmt"
	"log"
	"strings"
	"time"

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
	Parallel   int      `xml:"parallel"`
	DeleteOrig bool     `xml:"delete_orig"`
	Passwords  []string `xml:"passwords"`
}

type WatchConfig struct {
	FolderWatchEnabled bool          `xml:"folder_watch_enabled"`
	FolderWatchPaths   []string      `xml:"folder_watch_paths"`
	MarkerCleanup      time.Duration `xml:"marker_cleanup_interval"`
}

type TimingConfig struct {
	PollInterval time.Duration `xml:"poll_interval"`
}

type WebhookConfig struct {
	URL      string        `xml:"url"`
	Template string        `xml:"template"`
	Events   []string      `xml:"events"`
	Timeout  time.Duration `xml:"timeout"`
}

type StarrApp struct {
	URL       string        `xml:"url"`
	APIKey    string        `xml:"api_key"`
	Paths     []string      `xml:"paths"`
	Protocols []string      `xml:"protocols"`
	Timeout   time.Duration `xml:"timeout"`
}

func Load() (*Config, error) {
	cfg := &Config{
		HealthPort: 8085,
		LogLevel:   "INFO",
		Extract: ExtractConfig{
			Parallel:   1,
			DeleteOrig: true,
		},
		Watch: WatchConfig{
			FolderWatchEnabled: false,
			FolderWatchPaths:   []string{"/downloads"},
			MarkerCleanup:      1 * time.Hour,
		},
		Timing: TimingConfig{
			PollInterval: 2 * time.Minute,
		},
		Webhook: WebhookConfig{
			Template: "discord",
			Events:   []string{"extracted", "failed"},
			Timeout:  10 * time.Second,
		},
	}

	if _, err := cnfg.UnmarshalENV(cfg, ""); err != nil {
		return nil, fmt.Errorf("unmarshal env: %w", err)
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
				log.Printf("[Config] Invalid webhook event configured: %s (ignoring)", evt)
			}
		}
		if len(filteredEvents) == 0 {
			if len(cfg.Webhook.Events) > 0 {
				log.Printf("[Config] All configured webhook events were invalid; falling back to defaults")
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
	for _, p := range s.Paths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func (s *StarrApp) HasProtocol(protocol string) bool {
	if len(s.Protocols) == 0 {
		return true
	}
	for _, p := range s.Protocols {
		if strings.EqualFold(p, protocol) {
			return true
		}
	}
	return false
}
