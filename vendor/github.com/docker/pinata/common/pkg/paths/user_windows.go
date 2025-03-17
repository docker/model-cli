package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// UserPaths groups path getters for user-specific runtime resources
var UserPaths = userPaths{}

// GroupDir returns the root of the user resource directory holding runtime data
// shared at developer level (think GroupContainer on OSX), or when arguments
// are given a path within the group user resource directory
func (p userPaths) GroupDir(elem ...string) string {
	path, err := AppData()
	if err != nil {
		return ""
	}
	return filepath.Join(append([]string{path}, elem...)...)
}

// LocalDir returns the root of the directory holding machine specific user
// runtime resources, or when arguments are given a path within the user's local
// runtime resource directory
func (p userPaths) LocalDir(elem ...string) string {
	path, err := LocalAppData()
	if err != nil {
		return ""
	}
	return filepath.Join(append([]string{path}, elem...)...)
}

// RoamingDir returns the root of the directory holding user runtime resources
// which can be shared within a domain, or when arguments are given a path
// within the user's roaming runtime resource directory
func (p userPaths) RoamingDir(elem ...string) string {
	path, err := AppData()
	if err != nil {
		return ""
	}
	return filepath.Join(append([]string{path}, elem...)...)
}

// AppData returns the path to the application data (roaming profile).
func AppData() (string, error) {
	if IsIntegrationTests() {
		return filepath.Join(integrationTestsRootPath, "AppData"), nil
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("unable to get 'APPDATA'")
	}
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		appData = filepath.Join(devhome, "Roaming")
	}
	return filepath.Join(appData, "Docker"), nil
}

// LocalAppData returns the path to the local application data.
func LocalAppData() (string, error) {
	if IsIntegrationTests() {
		return filepath.Join(integrationTestsRootPath, "LocalAppData"), nil
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", fmt.Errorf("unable to get 'LOCALAPPDATA'")
	}
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		localAppData = filepath.Join(devhome, "Local")
	}
	return filepath.Join(localAppData, "Docker"), nil
}

// FrontendUserData returns the equivalent to app.getPath('userData') in go
func FrontendUserData() string {
	if IsIntegrationTests() {
		return filepath.Join(integrationTestsRootPath, "FrontendUserData")
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return ""
	}
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		appData = filepath.Join(devhome, "Roaming")
	}
	return filepath.Join(appData, "Docker Desktop")
}

// ElectronDataPartition returns the host directory that holds the data persisted by the extension's webview.
// Partition directory: C:\Users\<user>\AppData\Roaming\Docker Desktop\Partitions\<partition>
func ElectronDataPartition(partition string) string {
	return filepath.Join(FrontendUserData(), "Partitions", partition)
}
