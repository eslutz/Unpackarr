package logger

import (
	"fmt"
	"log"
	"sync"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel Level
	mu           sync.RWMutex
)

func init() {
	currentLevel = INFO
}

// SetLevel sets the minimum log level
func SetLevel(level string) {
	mu.Lock()
	defer mu.Unlock()

	switch level {
	case "DEBUG":
		currentLevel = DEBUG
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "INFO":
		currentLevel = INFO
		log.SetFlags(log.LstdFlags)
	case "WARN":
		currentLevel = WARN
		log.SetFlags(log.LstdFlags)
	case "ERROR":
		currentLevel = ERROR
		log.SetFlags(log.LstdFlags)
	default:
		currentLevel = INFO
		log.SetFlags(log.LstdFlags)
	}
}

// GetLevel returns the current log level
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

func shouldLog(level Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return level >= currentLevel
}

// Debug logs a debug message
func Debug(format string, v ...any) {
	if shouldLog(DEBUG) {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Info logs an info message
func Info(format string, v ...any) {
	if shouldLog(INFO) {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warn logs a warning message
func Warn(format string, v ...any) {
	if shouldLog(WARN) {
		log.Printf("[WARN] "+format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...any) {
	if shouldLog(ERROR) {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Errorf logs an error message and returns a formatted error
func Errorf(format string, v ...any) error {
	if shouldLog(ERROR) {
		log.Printf("[ERROR] "+format, v...)
	}
	return fmt.Errorf(format, v...)
}
