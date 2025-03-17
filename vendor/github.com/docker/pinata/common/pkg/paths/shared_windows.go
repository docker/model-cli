package paths

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	adminSettingsJSONPath   = `DockerDesktop\admin-settings.json`
	installSettingsJSONPath = `DockerDesktop\install-settings.json`
	registryJSONPath        = `DockerDesktop\registry.json`
	accessJSONPath          = `DockerDesktop\access.json`
	allowedOrgsRegistryKey  = `SOFTWARE\Policies\Docker\Docker Desktop`
	licensePubPath          = `DockerDesktop\license.pub`
	licenseEncPath          = `DockerDesktop\license.enc`
)

// SharedPaths groups path getters for resources shared between users of a given
// machine
var SharedPaths = sharedPaths{}

// Dir returns the root of the shared resource directory, or when arguments are
// given a path within the shared resource directory
func (p sharedPaths) Dir(elem ...string) string {
	path, err := ProgramData(elem...)
	if err != nil {
		log.Warn(err)
		return ""
	}
	return path
}

// ProgramData returns the path relative to the program data directory.
func ProgramData(path ...string) (string, error) {
	if IsIntegrationTests() {
		return filepath.Join(integrationTestsRootPath, "ProgramData"), nil
	}
	appData := os.Getenv("ProgramData")
	if appData == "" {
		return "", errors.New("unable to get 'ProgramData'")
	}
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		appData = filepath.Join(devhome, "ProgramData")
	}
	return filepath.Join(append([]string{appData}, path...)...), nil
}

// DockerDesktopProgramData returns a path within the docker desktop program data directory.
func DockerDesktopProgramData(path ...string) (string, error) {
	return ProgramData(append([]string{"DockerDesktop"}, path...)...)
}

// AllowedOrgsRegistryKey returns the registry key for allowed organizations configuration.
func (p sharedPaths) AllowedOrgsRegistryKey() (registry.Key, string) {
	return registry.LOCAL_MACHINE, allowedOrgsRegistryKey
}
