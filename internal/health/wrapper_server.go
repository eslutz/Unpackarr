package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/eslutz/unpackarr/internal/logger"
)

// WrapperServer provides health checks for the Unpackerr wrapper
type WrapperServer struct {
	startTime time.Time
	process   *os.Process
	mu        sync.RWMutex
}

// NewWrapperServer creates a new health server for the wrapper
func NewWrapperServer() *WrapperServer {
	return &WrapperServer{
		startTime: time.Now(),
	}
}

// SetUnpackerrProcess sets the Unpackerr subprocess reference
func (s *WrapperServer) SetUnpackerrProcess(p *os.Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.process = p
}

// Start starts the HTTP health server
func (s *WrapperServer) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", s.handlePing)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/metrics", s.handleMetrics)

	addr := fmt.Sprintf(":%d", port)
	logger.Info("[Health] Starting wrapper health server on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *WrapperServer) handlePing(w http.ResponseWriter, r *http.Request) {
	logger.Debug("[Health] Ping request from %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *WrapperServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	logger.Debug("[Health] Health check request from %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"healthy": true})
}

func (s *WrapperServer) handleReady(w http.ResponseWriter, r *http.Request) {
	logger.Debug("[Health] Readiness check request from %s", r.RemoteAddr)
	s.mu.RLock()
	process := s.process
	s.mu.RUnlock()

	ready := true
	reasons := []string{}

	if process == nil {
		ready = false
		reasons = append(reasons, "unpackerr process not started")
		logger.Debug("[Health] Readiness check failed: process not started")
	} else {
		// Check if process is still running
		// On Unix, sending signal 0 checks if process exists without affecting it
		if err := process.Signal(syscall.Signal(0)); err != nil {
			ready = false
			reasons = append(reasons, "unpackerr process not running")
			logger.Debug("[Health] Readiness check failed: process not running")
		}
	}

	if ready {
		logger.Debug("[Health] Readiness check passed")
	} else {
		logger.Debug("[Health] Readiness check failed: %v", reasons)
	}

	w.Header().Set("Content-Type", "application/json")
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ready":   false,
			"reasons": reasons,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]bool{"ready": true})
}

func (s *WrapperServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	logger.Debug("[Health] Status request from %s", r.RemoteAddr)
	
	s.mu.RLock()
	process := s.process
	s.mu.RUnlock()

	processStatus := "not started"
	var pid int
	if process != nil {
		pid = process.Pid
		if err := process.Signal(syscall.Signal(0)); err == nil {
			processStatus = "running"
		} else {
			processStatus = "stopped"
		}
	}

	status := map[string]any{
		"wrapper": map[string]any{
			"uptime_seconds": int(time.Since(s.startTime).Seconds()),
		},
		"unpackerr": map[string]any{
			"status": processStatus,
			"pid":    pid,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func (s *WrapperServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	s.mu.RLock()
	process := s.process
	s.mu.RUnlock()

	processRunning := 0
	if process != nil {
		if err := process.Signal(syscall.Signal(0)); err == nil {
			processRunning = 1
		}
	}

	_, _ = fmt.Fprintf(w, "# HELP unpackarr_wrapper_start_time_seconds Start time of the wrapper\n")
	_, _ = fmt.Fprintf(w, "# TYPE unpackarr_wrapper_start_time_seconds gauge\n")
	_, _ = fmt.Fprintf(w, "unpackarr_wrapper_start_time_seconds %d\n", s.startTime.Unix())

	_, _ = fmt.Fprintf(w, "# HELP unpackarr_process_running Whether the Unpackerr process is running (1=yes, 0=no)\n")
	_, _ = fmt.Fprintf(w, "# TYPE unpackarr_process_running gauge\n")
	_, _ = fmt.Fprintf(w, "unpackarr_process_running %d\n", processRunning)
}
