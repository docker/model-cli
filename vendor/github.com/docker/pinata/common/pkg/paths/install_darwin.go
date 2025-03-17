package paths

import (
	"errors"
	"path/filepath"
	"runtime"
)

const (
	Backend        = "com.docker.backend"
	Frontend       = "Docker Desktop"
	DevEnvs        = "com.docker.dev-envs"
	HyperKit       = "com.docker.hyperkit"
	Qemu           = "qemu-system-aarch64"
	Virtualization = "com.docker.virtualization"
	Krun           = "com.docker.krun"
	Installer      = "install"
	Build          = "com.docker.build"
	UrlHandler     = "com.docker.url-handler"
	Harmonia       = "docker-harmonia"

	backendBinaryPath          = "MacOS/" + Backend
	binaryResourcePath         = "Resources/bin"
	diagnoseBinaryPath         = "MacOS/com.docker.diagnose"
	askpassBinaryPath          = "MacOS/com.docker.askpass"
	dockerBinaryPath           = binaryResourcePath + "/docker"
	frontendBinaryPath         = "MacOS/Docker Desktop.app/Contents/MacOS/" + Frontend
	hyperkitBinaryPath         = binaryResourcePath + "/" + HyperKit
	linuxkitKernelPath         = "Resources/linuxkit/kernel"
	linuxkitBootPath           = "Resources/linuxkit/boot.img"
	osxfsBinaryPath            = "MacOS/com.docker.osxfs"
	qemuBinaryPath             = "MacOS/" + Qemu
	versionFilePath            = "Resources/componentsVersion.json"
	virtiofsdBinaryPath        = ""
	virtualizationBinaryPath   = "MacOS/" + Virtualization
	krunBinaryPath             = "MacOS/" + Krun
	wslBootstrappingDistroPath = ""
	wslCLIISO                  = ""
	wslDataDistroPath          = ""
	devEnvBinaryPath           = "MacOS/" + DevEnvs
	installerBinaryPath        = "MacOS/" + Installer
	buildBinaryPath            = "MacOS/" + Build
	urlHandlerBinaryPath       = "MacOS/" + UrlHandler
	installationManifestPath   = ""
)

// InstallPaths groups path getters for installed resources
var InstallPaths = installPaths{}

// Dir returns the root installation directory when no arguments are given, if
// arguments are provided a path within the installation directory will be returned
func (p installPaths) Dir(elem ...string) (string, error) {
	if IsIntegrationTests() {
		return filepath.Join(append([]string{"."}, elem...)...), nil
	}
	if DevRoot != "" {
		// in a build tree we have mac/gui/MacOS but in a Docker.app we have Contents/MacOS
		// The `install` command when run as root from Applescript does not have pinata.sh as the parent
		// so the DevRoot detection doesn't work.
		return filepath.Join(append([]string{DevRoot, "mac", "build", "Docker.app", "Contents"}, elem...)...), nil
	}

	f, err := executable()
	if err != nil {
		return "", err
	}
	for {
		f = filepath.Dir(f)
		if f == "/" || f == "." {
			return "", errors.New("cannot find resource Mac bundle root")
		}
		if filepath.Base(f) == "MacOS" {
			return filepath.Join(append([]string{filepath.Dir(f)}, elem...)...), nil
		}
	}
}

func frontendBinaryDevPath() string {
	dir := "mac"
	if runtime.GOARCH == "arm64" {
		dir += "-arm64" // or dir = "mac-arm64"
	}
	if DevRoot != "" {
		return filepath.Join(DevRoot, "client", "desktop", "dist", dir, "Docker Desktop.app", "Contents", "MacOS", "Docker Desktop")
	}
	return ""
}

// DevFrontendPath returns electron frontend application path from project root.
func (p installPaths) DevFrontendPath() string {
	dir := "mac"
	if runtime.GOARCH == "arm64" {
		dir += "-arm64" // or dir = "mac-arm64"
	}
	return filepath.Join("client", "desktop", "dist", dir, "Docker Desktop.app")
}

func backendBinaryDevPath() string {
	if DevRoot != "" {
		return filepath.Join(DevRoot, "mac", "build", "Docker.app", "Contents", backendBinaryPath)
	}
	return ""
}

// AppPath is returning root of the application
func (p installPaths) AppPath() (string, error) {
	path, err := p.Dir()
	if err == nil {
		return filepath.Dir(path), nil
	}
	return "", err
}
