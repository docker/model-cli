package paths

import (
	"path/filepath"
)

// UserPaths groups path getters for user-specific runtime resources
var UserPaths = userPaths{}

// GroupDir returns the root of the user resource directory holding runtime data
// shared at developer level (think GroupContainer on OSX), or when arguments
// are given a path within the group user resource directory
func (p userPaths) GroupDir(elem ...string) string {
	return filepath.Join(append([]string{GroupContainer()}, elem...)...)
}

// LocalDir returns the root of the directory holding machine specific user
// runtime resources, or when arguments are given a path within the user's local
// runtime resource directory
func (p userPaths) LocalDir(elem ...string) string {
	return filepath.Join(append([]string{Container()}, elem...)...)
}

// Container returns the sandbox path.
func Container() string {
	return Home("Library", "Containers", "com.docker.docker")
}

// Caches returns the caches sandbox path.
func Caches() string {
	return Home("Library", "Caches", "com.docker.docker")
}

// GroupContainer returns the group container directory path.
func GroupContainer() string {
	return Home("Library", "Group Containers", "group.com.docker")
}

// FrontendUserData returns the equivalent to app.getPath('userData') in go
func FrontendUserData() string {
	return Home("Library", "Application Support", "Docker Desktop")
}

// DevEnvsData returns the dev environments data directory.
func (p userPaths) DevEnvsData() string {
	return DockerHome("devenvironments")
}
