package paths

import (
	"io/fs"
	"os"
	"path/filepath"
)

const (
	adminSettingsJSONPath   = `admin-settings.json`
	installSettingsJSONPath = `install-settings.json`
	registryJSONPath        = "registry/registry.json"
	accessJSONPath          = "registry/access.json"
	licensePubPath          = "license.pub"
	licenseEncPath          = "license.enc"
)

// SharedPaths groups path getters for resources shared between users of a given machine.
var SharedPaths = sharedPaths{}

// Dir returns the root of the shared resource directory, or when arguments are
// given a path within the shared resource directory.
func (p sharedPaths) Dir(elem ...string) string {
	if IsIntegrationTests() {
		return filepath.Join(append([]string{integrationTestsRootPath, "share"}, elem...)...)
	}
	if DevRoot != "" {
		localBuildPath := filepath.Join(append([]string{DevRoot, "linux", "build"}, elem...)...)
		if _, err := os.Stat(localBuildPath); err == nil {
			return localBuildPath
		}
	}
	return filepath.Join(append([]string{"/usr", "share", "docker-desktop"}, elem...)...)
}

// ProgramData returns the path relative to the program data directory.
func ProgramData(path ...string) (string, error) {
	return "", fs.ErrNotExist
}

func DockerDesktopProgramData(path ...string) (string, error) {
	return "", fs.ErrNotExist
}
