package version

import (
	"runtime/debug"
)

var (
	// Version is set via ldflags at build time
	Version = "dev"
	// Commit is set via ldflags at build time
	Commit = "none"
	// Date is set via ldflags at build time
	Date = "unknown"
)

// Get returns the version string
func Get() string {
	if Version != "dev" {
		return Version
	}

	// Try to get version from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
	}

	return Version
}

// GetFull returns full version information
func GetFull() string {
	return "gotion version " + Get() + " (commit: " + Commit + ", built: " + Date + ")"
}
