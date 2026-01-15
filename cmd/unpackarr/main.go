package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/eslutz/unpackarr/internal/config"
	"github.com/eslutz/unpackarr/internal/health"
	"github.com/eslutz/unpackarr/internal/logger"
	"github.com/eslutz/unpackarr/pkg/version"
)

const unpackerrBinary = "/usr/local/bin/unpackerr"

func main() {
	logger.Info("Unpackarr Wrapper %s", version.String())

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	logger.SetLevel(cfg.LogLevel)
	logger.Debug("[Wrapper] Log level set to: %s", cfg.LogLevel)

	// Create context for managing subprocess lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the health server
	healthServer := health.NewWrapperServer()
	go func() {
		if err := healthServer.Start(cfg.HealthPort); err != nil {
			logger.Error("[Wrapper] Health server error: %v", err)
			os.Exit(1)
		}
	}()

	logger.Info("[Wrapper] Health server started on port %d", cfg.HealthPort)

	// Start Unpackerr as a subprocess
	logger.Info("[Wrapper] Starting Unpackerr subprocess...")
	cmd := exec.CommandContext(ctx, unpackerrBinary)
	cmd.Env = os.Environ()

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("[Wrapper] Failed to create stdout pipe: %v", err)
		os.Exit(1)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("[Wrapper] Failed to create stderr pipe: %v", err)
		os.Exit(1)
	}

	// Start the subprocess
	if err := cmd.Start(); err != nil {
		logger.Error("[Wrapper] Failed to start Unpackerr: %v", err)
		os.Exit(1)
	}

	healthServer.SetUnpackerrProcess(cmd.Process)
	logger.Info("[Wrapper] Unpackerr started with PID %d", cmd.Process.Pid)

	// Stream subprocess logs
	go streamLogs(stdout, "[Unpackerr]")
	go streamLogs(stderr, "[Unpackerr ERR]")

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Monitor subprocess exit
	exitChan := make(chan error, 1)
	go func() {
		exitChan <- cmd.Wait()
	}()

	select {
	case sig := <-sigChan:
		logger.Info("[Wrapper] Received signal %s, shutting down...", sig)
		cancel() // This will terminate the subprocess via context
	case err := <-exitChan:
		if err != nil {
			logger.Error("[Wrapper] Unpackerr exited with error: %v", err)
		} else {
			logger.Info("[Wrapper] Unpackerr exited normally")
		}
	}

	logger.Info("[Wrapper] Shutdown complete")
}

func streamLogs(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		logAtParsedLevel(prefix, line)
	}
	if err := scanner.Err(); err != nil {
		logger.Error("%s Error reading logs: %v", prefix, err)
	}
}

// logAtParsedLevel parses the log level from Unpackerr output and logs at the appropriate level
func logAtParsedLevel(prefix, line string) {
	// Unpackerr log format typically includes [DEBUG], [INFO], [WARN], [ERROR] in the message
	// Look for these patterns and log at the corresponding level
	switch {
	case containsLogLevel(line, "[DEBUG]"):
		logger.Debug("%s %s", prefix, line)
	case containsLogLevel(line, "[WARN]"):
		logger.Warn("%s %s", prefix, line)
	case containsLogLevel(line, "[ERROR]"):
		logger.Error("%s %s", prefix, line)
	default:
		// Default to INFO for lines without explicit level or with [INFO]
		logger.Info("%s %s", prefix, line)
	}
}

// containsLogLevel checks if a line contains a specific log level marker
func containsLogLevel(line, level string) bool {
	// Case-insensitive check for log level markers
	return len(line) >= len(level) && 
		(line[:len(level)] == level || 
		 findSubstring(line, level))
}

// findSubstring does a simple substring search
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
