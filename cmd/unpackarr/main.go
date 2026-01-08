package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/health"
	"github.com/eslutz/unpackarr/internal/notify"
	"github.com/eslutz/unpackarr/internal/starr"
	"github.com/eslutz/unpackarr/pkg/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println(version.String())

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	setLogLevel(cfg.LogLevel)

	metrics := health.NewMetrics()
	webhook := notify.NewWebhook(&cfg.Webhook)

	queue := extract.NewQueue(&cfg.Extract, func(result *extract.Result) {
		log.Printf("[Extract] Completed: %s (source: %s, success: %t, duration: %s)",
			result.Name, result.Source, result.Success, result.Elapsed)

		// Write marker file when not deleting originals to prevent re-extraction
		if result.Success && !result.DeleteOrig {
			if err := extract.WriteMarkerForPath(result.Path); err != nil {
				log.Printf("[Extract] Warning: failed to write marker for %s: %v", result.Name, err)
			}
		}

		metrics.RecordExtraction(result)

		if webhook != nil {
			go func() {
				// Recover from panics to prevent crashing the application
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[Webhook] Panic recovered: %v", r)
					}
				}()

				event := "extracted"
				if !result.Success {
					event = "failed"
				}

				start := time.Now()
				webhook.Notify(result)
				duration := time.Since(start)

				if metrics != nil {
					metrics.RecordWebhook(event, true, duration)
				}
			}()
		}
	})

	watcher := extract.NewWatcher(&cfg.Watch, &cfg.Extract, &cfg.Timing, queue)
	watcher.Start()

	healthServer := health.NewServer(queue, watcher, &cfg.Watch)

	clients := initStarrClients(cfg, queue, healthServer)

	log.Printf("Started %d starr clients", len(clients))
	if cfg.Watch.FolderWatchEnabled {
		log.Printf("Folder watcher enabled for %d paths", len(cfg.Watch.FolderWatchPaths))
	}

	go func() {
		if err := healthServer.Start(cfg.HealthPort); err != nil {
			log.Fatalf("Health server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	for _, client := range clients {
		client.Stop()
	}
	watcher.Stop()
	queue.Stop()

	log.Println("Shutdown complete")
}

func initStarrClients(cfg *config.Config, queue *extract.Queue, server *health.Server) []*starr.Client {
	clients := []*starr.Client{}

	if cfg.Sonarr != nil && cfg.Sonarr.URL != "" {
		client := starr.NewSonarr(cfg.Sonarr, queue, &cfg.Timing)
		server.RegisterClient("sonarr", client.Client)
		clients = append(clients, client.Client)
	}

	if cfg.Radarr != nil && cfg.Radarr.URL != "" {
		client := starr.NewRadarr(cfg.Radarr, queue, &cfg.Timing)
		server.RegisterClient("radarr", client.Client)
		clients = append(clients, client.Client)
	}

	if cfg.Lidarr != nil && cfg.Lidarr.URL != "" {
		client := starr.NewLidarr(cfg.Lidarr, queue, &cfg.Timing)
		server.RegisterClient("lidarr", client.Client)
		clients = append(clients, client.Client)
	}

	if cfg.Readarr != nil && cfg.Readarr.URL != "" {
		client := starr.NewReadarr(cfg.Readarr, queue, &cfg.Timing)
		server.RegisterClient("readarr", client.Client)
		clients = append(clients, client.Client)
	}

	return clients
}

func setLogLevel(level string) {
	switch level {
	case "DEBUG":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "INFO", "WARN", "ERROR":
		log.SetFlags(log.LstdFlags)
	default:
		log.SetFlags(log.LstdFlags)
	}
}
