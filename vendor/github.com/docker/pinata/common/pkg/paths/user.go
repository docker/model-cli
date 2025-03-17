package paths

import (
	"errors"
)

// userPaths groups path getters for user-specific runtime resources
type userPaths struct{}

// CNIDir returns the path to the host directory containing Kubernetes CNI config files.
func (p userPaths) CNIDir() string {
	return p.GroupDir("cni")
}

// DefaultPkiFolder returns the default pki folder path.
func (p userPaths) DefaultPkiFolder() string {
	return p.LocalDir("pki")
}

// DHCPFile returns the DHCP options (search domain, domain) file path.
func (p userPaths) DHCPFile() string {
	return p.GroupDir("dhcp.json")
}

// DaemonLock returns path to a file used to protect from running multiple backend instances.
func (p userPaths) DaemonLock() string {
	return p.LocalDir("backend.lock")
}

// DefaultWSLDataDistroDir returns path to a the default directory used for holding the WSL data
func (p userPaths) DefaultWSLDir() string {
	if isWindows() {
		return p.LocalDir("wsl")
	}
	return ""
}

// FeaturesOverridesFile is the path to the feature overrides file.
func (p userPaths) FeaturesOverridesFile() string {
	return p.GroupDir("features-overrides.json")
}

// FeaturesFile is the path to the feature file.
func (p userPaths) FeaturesFile() string {
	return DockerHome("features.json")
}

// CustomFlagsFile is the path to custom flags file.
func (p userPaths) CustomFlagsFile() string {
	return p.GroupDir("flags.json")
}

// UnleashBackupFile is the path to Unleash local backup file.
func (p userPaths) UnleashBackupFile() string {
	return p.GroupDir("unleash-repo-schema-v1-Docker Desktop.json")
}

// VPNKitForwardsFile returns the vpnkit forwards configuration file path.
func (p userPaths) VPNKitForwardsFile() string {
	return p.GroupDir("forwards.json")
}

// LinuxKitCaCertificatesFile is where the Csharp writes the ca-certificates.crt.
func (p userPaths) LinuxKitCaCertificatesFile() (string, error) {
	if isWindows() {
		path := p.LocalDir("vm-config", "ca-certificates.crt")
		if path == "" {
			return "", errors.New("failed to get certificate file path")
		}
		return path, nil
	}
	return "", errors.New("certificate file path getter is not implemented")
}

// MutagenDataDirectory returns the path of the Docker-specific Mutagen data
// directory on the host.
func (p userPaths) MutagenDataDirectory() string {
	return DockerHome("mutagen")
}

// HarmoniaDataDirectory returns the path of the Docker-specific Harmonia data
// directory on the host.
func (p userPaths) HarmoniaDataDirectory() string {
	return DockerHome("harmonia")
}

// CloudDataDirectory returns the path of the Docker-specific Cloud data
// directory on the host.
func (p userPaths) CloudDataDirectory() string {
	return DockerHome("cloud")
}

// DockerContexts returns the path where the CLI stores docker contexts.
func (p userPaths) DockerContexts() string {
	return DockerHome("contexts")
}

// SettingsFile returns the settings file path.
func (p userPaths) SettingsFile() string {
	return p.GroupDir("settings-store.json")
}

// RegistryAccessManagementState returns the path to the state of the refreshing code.
func (p userPaths) RegistryAccessManagementState() string {
	return p.GroupDir("registry-access-managment-state.json")
}

// LinuxDaemonConfigFile returns the linux daemon.json file path.
func (p userPaths) LinuxDaemonConfigFile() string {
	return Home(".docker", "daemon.json")
}

// WindowsDaemonConfigFile returns the windows daemon.json file path.
func (p userPaths) WindowsDaemonConfigFile() string {
	if isWindows() {
		return Home(".docker", "windows-daemon.json")
	}
	return ""
}

// UserIDFile returns the path to the userID file
func (p userPaths) UserIDFile() string {
	if isWindows() {
		return p.GroupDir(".trackid")
	}
	return p.GroupDir("userId")
}

// LicenseFiles returns the paths of the license and certificate files
func (p userPaths) LicenseFiles() (lic, cert string) {
	return p.GroupDir("license.enc"), p.GroupDir("license.pub")
}

// AuthTokenFamilyCreationTime returns the path to a file that, if it exists, contains the time
// at which the current auth token family was created, in RFC 3339 format.
func (p userPaths) AuthTokenFamilyCreationTime() string {
	return p.GroupDir("auth-token-family-creation-time.txt")
}

// BackendErrorFile returns the path to the error file
func (p userPaths) BackendErrorFile() string {
	return p.LocalDir("backend.error.json")
}

func (p userPaths) InstallerErrorFile() string {
	return p.LocalDir("installer.error.json")
}

// HypervisorErrorFile returns the path to the error file for the hypervisor
func (p userPaths) HypervisorErrorFile() string {
	return p.LocalDir("hypervisor.error.json")
}

// BuildDataRoot returns the desktop-build data directory.
// https://github.com/docker/desktop-build/blob/b99ba214c4490240370d0ee8a768639cb83d128a/pkg/paths/paths.go#LL18C43-L18C56
func (p userPaths) BuildDataRoot() string {
	return DockerHome("desktop-build")
}

func (p userPaths) ReportCache() string {
	return p.GroupDir("reports.log")
}

func (p userPaths) VolumesBackupScheduleFile() string {
	return p.LocalDir("volume-backup-schedule.json")
}

func (p userPaths) ExportLogFile() string {
	return p.LocalDir("volume-backup-exports.json")
}
