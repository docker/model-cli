package paths

import (
	"os"
	"path/filepath"
)

// UserPaths groups path getters for user-specific runtime resources
var UserPaths = userPaths{}

// GroupDir returns the root of the user resource directory holding runtime data
// shared at developer level (think GroupContainer on OSX), or when arguments
// are given a path within the group user resource directory
func (p userPaths) GroupDir(elem ...string) string {
	return desktop(elem...)
}

// LocalDir returns the root of the directory holding machine specific user
// runtime resources, or when arguments are given a path within the user's local
// runtime resource directory
func (p userPaths) LocalDir(elem ...string) string {
	return desktop(elem...)
}

// FrontendUserData returns the equivalent to app.getPath('userData') in go
func FrontendUserData() string {
	if os.Getenv("XDG_CONFIG_HOME") != "" {
		return filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "Docker Desktop")
	}
	return Home(".config", "Docker Desktop")
}

// ElectronDataPartition returns the host directory that holds the data persisted by the extension's webview.
func ElectronDataPartition(partition string) string {
	return desktop("Partitions", partition)
}
