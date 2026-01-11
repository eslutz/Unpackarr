package logger

import (
	"testing"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected Level
	}{
		{"DEBUG", "DEBUG", DEBUG},
		{"INFO", "INFO", INFO},
		{"WARN", "WARN", WARN},
		{"ERROR", "ERROR", ERROR},
		{"default", "INVALID", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			if got := GetLevel(); got != tt.expected {
				t.Errorf("SetLevel(%q) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestShouldLog(t *testing.T) {
	SetLevel("INFO")

	tests := []struct {
		name     string
		level    Level
		expected bool
	}{
		{"DEBUG below INFO", DEBUG, false},
		{"INFO at INFO", INFO, true},
		{"WARN above INFO", WARN, true},
		{"ERROR above INFO", ERROR, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldLog(tt.level); got != tt.expected {
				t.Errorf("shouldLog(%v) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestLogLevelFiltering(t *testing.T) {
	// Test that DEBUG messages are filtered at INFO level
	SetLevel("INFO")
	if shouldLog(DEBUG) {
		t.Error("DEBUG messages should be filtered at INFO level")
	}

	// Test that INFO messages pass at INFO level
	if !shouldLog(INFO) {
		t.Error("INFO messages should pass at INFO level")
	}

	// Test that ERROR messages pass at INFO level
	if !shouldLog(ERROR) {
		t.Error("ERROR messages should pass at INFO level")
	}
}
