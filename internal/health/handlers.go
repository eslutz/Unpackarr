package health

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/extract"
	"github.com/eslutz/unpackarr/internal/starr"
)

type Server struct {
	queue      *extract.Queue
	watcher    *extract.Watcher
	clients    map[string]*starr.Client
	startTime  time.Time
	watcherCfg *config.WatchConfig
}

func NewServer(queue *extract.Queue, watcher *extract.Watcher, watcherCfg *config.WatchConfig) *Server {
	return &Server{
		queue:      queue,
		watcher:    watcher,
		clients:    make(map[string]*starr.Client),
		startTime:  time.Now(),
		watcherCfg: watcherCfg,
	}
}

func (s *Server) RegisterClient(name string, client *starr.Client) {
	s.clients[name] = client
}

func (s *Server) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", s.handlePing)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/metrics", s.handleMetrics)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[Health] Starting server on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"healthy": true})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	ready := true
	reasons := []string{}

	if s.queue == nil {
		ready = false
		reasons = append(reasons, "queue not initialized")
	}

	for name, client := range s.clients {
		connected, _ := client.Status()
		if !connected {
			ready = false
			reasons = append(reasons, fmt.Sprintf("%s disconnected", name))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ready":   false,
			"reasons": reasons,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ready": true})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	stats := s.queue.Stats()

	apps := make(map[string]interface{})
	for name, client := range s.clients {
		connected, queueSize := client.Status()
		apps[name] = map[string]interface{}{
			"connected":   connected,
			"queue_items": queueSize,
		}
	}

	status := map[string]interface{}{
		"queue": map[string]int{
			"waiting":    stats.Waiting,
			"extracting": stats.Extracting,
		},
		"folder_watcher": map[string]interface{}{
			"enabled": s.watcherCfg.Enabled,
			"paths":   s.watcherCfg.Paths,
		},
		"apps":           apps,
		"uptime_seconds": int(time.Since(s.startTime).Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	stats := s.queue.Stats()

	_, _ = fmt.Fprintf(w, "# HELP unpackarr_queue_size Current queue size by state\n")
	_, _ = fmt.Fprintf(w, "# TYPE unpackarr_queue_size gauge\n")
	_, _ = fmt.Fprintf(w, "unpackarr_queue_size{state=\"waiting\"} %d\n", stats.Waiting)
	_, _ = fmt.Fprintf(w, "unpackarr_queue_size{state=\"extracting\"} %d\n", stats.Extracting)

	for name, client := range s.clients {
		connected, queueSize := client.Status()
		connectedValue := 0
		if connected {
			connectedValue = 1
		}

		_, _ = fmt.Fprintf(w, "# HELP unpackarr_starr_connected Connection status (1=connected, 0=disconnected)\n")
		_, _ = fmt.Fprintf(w, "# TYPE unpackarr_starr_connected gauge\n")
		_, _ = fmt.Fprintf(w, "unpackarr_starr_connected{app=\"%s\"} %d\n", name, connectedValue)

		_, _ = fmt.Fprintf(w, "# HELP unpackarr_starr_queue_items Number of items in starr queue\n")
		_, _ = fmt.Fprintf(w, "# TYPE unpackarr_starr_queue_items gauge\n")
		_, _ = fmt.Fprintf(w, "unpackarr_starr_queue_items{app=\"%s\"} %d\n", name, queueSize)
	}

	_, _ = fmt.Fprintf(w, "# HELP unpackarr_start_time_seconds Start time of the application\n")
	_, _ = fmt.Fprintf(w, "# TYPE unpackarr_start_time_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "unpackarr_start_time_seconds %d\n", s.startTime.Unix())
}
