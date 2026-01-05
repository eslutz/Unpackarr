package version

import (
	"fmt"
	"runtime"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	return fmt.Sprintf("unpackarr %s (commit: %s, built: %s, go: %s)",
		Version, Commit, Date, runtime.Version())
}

func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     Commit,
		"date":       Date,
		"go_version": runtime.Version(),
	}
}
