package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	result := String()
	if !strings.Contains(result, "unpackarr") {
		t.Errorf("String() should contain 'unpackarr', got: %s", result)
	}
	if !strings.Contains(result, Version) {
		t.Errorf("String() should contain version %s, got: %s", Version, result)
	}
}

func TestInfo(t *testing.T) {
	info := Info()

	if info["version"] != Version {
		t.Errorf("Info() version = %s, want %s", info["version"], Version)
	}
	if info["commit"] != Commit {
		t.Errorf("Info() commit = %s, want %s", info["commit"], Commit)
	}
	if info["date"] != Date {
		t.Errorf("Info() date = %s, want %s", info["date"], Date)
	}
	if info["go_version"] == "" {
		t.Error("Info() go_version should not be empty")
	}
}
