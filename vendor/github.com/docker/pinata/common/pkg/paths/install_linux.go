package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// installation directory for Docker Desktop files: /opt/docker-desktop
const (
	Backend        = "com.docker.backend"
	Frontend       = "Docker Desktop"
	DevEnvs        = "com.docker.dev-envs"
	HyperKit       = "com.docker.hyperkit"
	Qemu           = "qemu-system-x86_64"
	Virtualization = "com.docker.virtualization"
	Build          = "com.docker.build"
	UrlHandler     = "com.docker.url-handler"
	Harmonia       = "docker-harmonia"

	// electron binary
	frontendBinaryPath = Frontend

	// config files
	versionFilePath = "componentsVersion.json"

	// internal binaries
	binaryResourcePath   = "bin"
	backendBinaryPath    = "bin/" + Backend
	diagnoseBinaryPath   = "bin/com.docker.diagnose"
	askpassBinaryPath    = "bin/com.docker.askpass"
	virtiofsdBinaryPath  = "bin/virtiofsd"
	urlHandlerBinaryPath = "bin/" + UrlHandler

	// additional host services
	devEnvBinaryPath = "bin/" + DevEnvs
	buildBinaryPath  = "bin/" + Build

	// dependencies
	dockerBinaryPath = "/usr/local/bin/docker"
	qemuBinaryPath   = "/usr/bin/" + Qemu

	// linuxkit resources
	linuxkitKernelPath = "linuxkit/kernel"
	linuxkitBootPath   = "linuxkit/boot.img"

	hyperkitBinaryPath         = ""
	osxfsBinaryPath            = ""
	virtualizationBinaryPath   = ""
	krunBinaryPath             = ""
	wslBootstrappingDistroPath = ""
	wslCLIISO                  = ""
	wslDataDistroPath          = ""
	installerBinaryPath        = ""
	installationManifestPath   = ""
)

// InstallPaths groups path getters for installed resources
var InstallPaths = installPaths{}

// Dir returns the root installation directory when no arguments are given, if
// arguments are provided a path within the installation directory will be returned
func (p installPaths) Dir(elem ...string) (string, error) {
	path := filepath.Join(elem...)
	if DevRoot != "" {
		localBuildPath := filepath.Join(append([]string{DevRoot, "linux", "build", "docker-desktop"}, elem...)...)
		if _, err := os.Stat(localBuildPath); err == nil {
			return localBuildPath, nil
		}
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(append([]string{"/opt/docker-desktop"}, elem...)...), nil
}

func frontendBinaryDevPath() string {
	if DevRoot != "" {
		dir := "linux-unpacked"
		if runtime.GOARCH == "arm64" {
			dir = "linux-arm64-unpacked"
		}
		return filepath.Join(DevRoot, "client", "desktop", "dist", dir, "Docker Desktop")
	}
	return ""
}

func backendBinaryDevPath() string {
	if DevRoot != "" {
		return filepath.Join(DevRoot, "linux", "build", backendBinaryPath)
	}
	return ""
}

func (p installPaths) AppPath() (string, error) {
	return p.Dir()
}
