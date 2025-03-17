package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/docker/pinata/common/pkg/logger"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"
)

const (
	Backend    = "com.docker.backend.exe"
	Frontend   = "Docker Desktop.exe"
	DevEnvs    = "com.docker.dev-envs.exe"
	Build      = "com.docker.build.exe"
	UrlHandler = "com.docker.url-handler.exe"
	Harmonia   = "docker-harmonia.exe"

	backendBinaryPath          = `resources\` + Backend
	binaryResourcePath         = `resources\bin`
	diagnoseBinaryPath         = `resources\com.docker.diagnose.exe`
	askpassBinaryPath          = `resources\com.docker.askpass.exe`
	dockerBinaryPath           = binaryResourcePath + `\docker.exe`
	frontendBinaryPath         = `frontend\` + Frontend
	hyperkitBinaryPath         = ""
	linuxkitKernelPath         = ""
	linuxkitBootPath           = `resources\docker-desktop.iso`
	osxfsBinaryPath            = ""
	qemuBinaryPath             = ""
	versionFilePath            = `resources\componentsVersion.json`
	virtiofsdBinaryPath        = ""
	virtualizationBinaryPath   = ""
	krunBinaryPath             = ""
	adminBinaryPath            = `resources\com.docker.admin.exe`
	devEnvBinaryPath           = `resources\` + DevEnvs
	buildBinaryPath            = `resources\` + Build
	installerBinaryPath        = ""
	wslBootstrappingDistroPath = `resources\wsl\wsl-bootstrap.tar`
	wslCLIISO                  = `resources\wsl\docker-wsl-cli.iso`
	wslDataDistroPath          = `resources\wsl\wsl-data.tar`
	installationManifestPath   = `app.json`
	urlHandlerBinaryPath       = `resources\` + UrlHandler
)

// InstallPaths groups path getters for installed resources
var InstallPaths = installPaths{}

// Dir returns the root installation directory when no arguments are given, if
// arguments are provided a path within the installation directory will be returned
func (p installPaths) Dir(elem ...string) (string, error) {
	return app(elem...)
}

// app returns the full path to where the application is installed if
// invoked with no argument or the full path of a sub path provided as
// argument in the application directory.
func app(elem ...string) (string, error) {
	if IsIntegrationTests() {
		return filepath.Join(integrationTestsRootPath, "app"), nil
	}
	if DevRoot != "" {
		return makePath(filepath.Join(DevRoot, "win", "build", "win"), elem...)
	}
	return appFind(elem...)
}

// For MSI installations, we can no longer rely on a named registry key in UNINSTALL.
// MSI installations use a product code. In this case we must fallback to a known
// registry key for the application.
func appFindFallback(elem ...string) (string, error) {
	path := `SOFTWARE\Docker Inc.\Docker Desktop`
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return "", errors.Wrapf(err, "cannot find registry key %q", path)
	}
	defer logger.Close(log, key)
	name := "AppPath"
	root, _, err := key.GetStringValue(name)
	if err != nil {
		return "", errors.Wrapf(err, "cannot find registry value %q", name)
	}
	return makePath(root, elem...)
}

func appFind(elem ...string) (string, error) {
	path := `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop`
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return appFindFallback(elem...)
	}
	defer logger.Close(log, key)
	name := "InstallLocation"
	root, _, err := key.GetStringValue(name)
	if err != nil {
		return "", errors.Wrapf(err, "cannot find registry value %q", name)
	}
	return makePath(root, elem...)
}

func makePath(path string, elem ...string) (string, error) {
	f := filepath.Join(append([]string{path}, elem...)...)
	if _, err := os.Stat(f); err != nil {
		return "", fmt.Errorf("%s does not exist", f)
	}
	return f, nil
}

func frontendBinaryDevPath() string {
	buildDir := "win-unpacked"
	if runtime.GOARCH == "arm64" {
		buildDir = "win-arm64-unpacked"
	}
	if DevRoot != "" {
		return filepath.Join(DevRoot, InstallPaths.DevFrontendPath(), buildDir, "Docker Desktop.exe")
	}
	return ""
}

func backendBinaryDevPath() string {
	if DevRoot != "" {
		return filepath.Join(DevRoot, "win", "build", "win", backendBinaryPath)
	}
	return ""
}

func (p installPaths) AppPath() (string, error) {
	return p.Dir()
}

// DevFrontendPath returns electron frontend application path from project root.
func (p installPaths) DevFrontendPath() string {
	return filepath.Join("client", "desktop", "dist")
}

// AdminBinary returns the path of com.docker.admin binary.
func (p installPaths) AdminBinary() (string, error) {
	return p.Dir(adminBinaryPath)
}
