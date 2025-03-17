package paths

import (
	"io/fs"
	"path/filepath"
)

const (
	// TODO: could we have a common root here?
	adminSettingsJSONPath   = `com.docker.docker/admin-settings.json`
	installSettingsJSONPath = `com.docker.docker/install-settings.json`
	registryJSONPath        = "com.docker.docker/registry.json"
	accessJSONPath          = "com.docker.docker/access.json"
	allowedOrgsPlistPath    = "com.docker.docker/desktop.plist"
	licensePubPath          = "com.docker.docker/license.pub"
	licenseEncPath          = "com.docker.docker/license.enc"
	vmnetdPlist             = "com.docker.vmnetd.plist"
	socketPlist             = "com.docker.socket.plist"
)

// SharedPaths groups path getters for resources shared between users of a given
// machine
var SharedPaths = sharedPaths{}

// Dir returns the root of the shared resource directory, or when arguments are
// given a path within the shared resource directory
func (p sharedPaths) Dir(elem ...string) string {
	if IsIntegrationTests() {
		return filepath.Join(append([]string{integrationTestsRootPath, "share"}, elem...)...)
	}
	return filepath.Join(append([]string{"/Library", "Application Support"}, elem...)...)
}

func (p sharedPaths) DirLaunchDaemons(elem ...string) string {
	if IsIntegrationTests() {
		return filepath.Join(append([]string{integrationTestsRootPath, "share"}, elem...)...)
	}
	return filepath.Join(append([]string{"/Library", "LaunchDaemons"}, elem...)...)
}

// ApplicationSupport returns the global library's application support folder.
func ApplicationSupport() string {
	return SharedPaths.Dir("Docker", "DockerDesktop")
}

// ProgramData returns the path relative to the program data directory.
func ProgramData(_ ...string) (string, error) {
	return "", fs.ErrNotExist
}

func DockerDesktopProgramData(_ ...string) (string, error) {
	return "", fs.ErrNotExist
}

// AllowedOrgsPlist returns the path to desktop.plist for access management allowed organizations.
func (p sharedPaths) AllowedOrgsPlist() string {
	return p.Dir(allowedOrgsPlistPath)
}

func (p sharedPaths) VmnetdPlist() string {
	return p.DirLaunchDaemons(vmnetdPlist)
}

func (p sharedPaths) SocketPlist() string {
	return p.DirLaunchDaemons(socketPlist)
}
