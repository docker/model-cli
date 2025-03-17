package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// installPaths groups path getters for installed resources
type installPaths struct{}

func (p installPaths) BackendBinary() (string, error) {
	if path := backendBinaryDevPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return p.Dir(backendBinaryPath)
}

// CredentialsHelperBinary returns the path to a docker credentials binary.
func (p installPaths) CredentialsHelperBinary(helper string) (string, error) {
	return p.Dir(binaryResourcePath, helper)
}

func (p installPaths) WebgpuRuntimeBinary() (string, error) {
	return p.Dir(binaryResourcePath, "com.docker.webgpu-runtime")
}

// DiagnoseExecutable is the path to com.docker.diagnose
func (p installPaths) DiagnoseExecutable() string {
	path, err := p.Dir(diagnoseBinaryPath)
	if err != nil {
		log.Warn(err)
		return ""
	}
	return path
}

func (p installPaths) AskpassBinary() string {
	path, err := p.Dir(askpassBinaryPath)
	if err != nil {
		log.Warn(err)
		return ""
	}
	return path
}

// DockerBinary returns the path to the docker binary.
func (p installPaths) DockerBinary() (string, error) {
	return p.Dir(dockerBinaryPath)
}

// FrontendExecutable returns electron frontend executable full path
func (p installPaths) FrontendExecutable() string {
	if foundDevBuild := p.devFrontendExecutable(); foundDevBuild != "" {
		return foundDevBuild
	}
	frontendPath, err := p.Dir(frontendBinaryPath)
	if err != nil {
		log.Warnf("finding installed frontend executable: %v", err)
		return ""
	}
	return frontendPath
}

func (p installPaths) devFrontendExecutable() string {
	if DevRoot != "" {
		localBuildPath := frontendBinaryDevPath()
		if _, err := os.Stat(localBuildPath); err == nil {
			return localBuildPath
		}
	}
	return ""
}

// HyperKitBinary returns the path to the binary of HyperKit shipped within the application package.
func (p installPaths) HyperKitBinary() (string, error) {
	if isDarwin() {
		return p.Dir(hyperkitBinaryPath)
	}
	return "", nil
}

// LinuxKitKernel returns the path to the linuxkit kernel within the application package.
func (p installPaths) LinuxKitKernel() (string, error) {
	return p.Dir(linuxkitKernelPath)
}

// LinuxKitBoot returns the path to the default (.iso|.raw) shipped within the application package.
func (p installPaths) LinuxKitBoot() (string, error) {
	return p.Dir(linuxkitBootPath)
}

// LinuxKitBootDigest returns the path to the default (.iso|.raw) digest shipped within the application package.
func (p installPaths) LinuxKitBootDigest() (string, error) {
	return p.Dir(linuxkitBootPath + ".sha256")
}

// OsxfsBinary returns the path to com.docker.osxfs
func (p installPaths) OsxfsBinary() (string, error) {
	if isDarwin() {
		return p.Dir(osxfsBinaryPath)
	}
	return "", nil
}

// QemuBinary returns the path to the binary of qemu shipped within the application package.
func (p installPaths) QemuBinary() (string, error) {
	return p.Dir(qemuBinaryPath)
}

func (p installPaths) VersionFile() (string, error) {
	return p.Dir(versionFilePath)
}

func (p installPaths) Virtiofsd() (string, error) {
	if isLinux() {
		return p.Dir(virtiofsdBinaryPath)
	}
	return "", nil
}

// VirtualizationBinary returns the path of the virtualization.framework tool.
func (p installPaths) VirtualizationBinary() (string, error) {
	if isDarwin() {
		return p.Dir(virtualizationBinaryPath)
	}
	return "", nil
}

// KrunBinary returns the path of the libkrun runner.
func (p installPaths) KrunBinary() (string, error) {
	if isDarwin() {
		return p.Dir(krunBinaryPath)
	}
	return "", nil
}

// WSLBootstrappingDistroPath returns path to the WSL main distro tarball.
func (p installPaths) WSLBootstrappingDistroPath() (string, error) {
	return p.Dir(wslBootstrappingDistroPath)
}

// WSLCLIISO returns the path to the WSL image containing CLI tools.
func (p installPaths) WSLCLIISO() (string, error) {
	return p.Dir(wslCLIISO)
}

// WSLDataDistroPath returns path to the WSL data distro tarball.
func (p installPaths) WSLDataDistroPath() (string, error) {
	return p.Dir(wslDataDistroPath)
}

// DevEnvsBinary returns the path to the driver binary.
func (p installPaths) DevEnvsBinary() (string, error) {
	return p.Dir(devEnvBinaryPath)
}

// InstallerBinary returns the path to the installer.
func (p installPaths) InstallerBinary() (string, error) {
	if isDarwin() {
		return p.Dir(installerBinaryPath)
	}
	return "", nil
}

// BuildBinary returns the path to desktop build binary.
func (p installPaths) BuildBinary() (string, error) {
	return p.Dir(buildBinaryPath)
}

func (p installPaths) UrlHandlerBinary() (string, error) {
	return p.Dir(urlHandlerBinaryPath)
}

// BinResourcesPath returns the path to the docker binaries directory
func (p installPaths) BinResourcesPath() (string, error) {
	if runtime.GOOS == "linux" {
		// Docker CLI is not included in Docker Desktop package
		// it is declared as a dependency, installed usually in /usr/bin
		return filepath.Dir(dockerBinaryPath), nil
	}
	return p.Dir(binaryResourcePath)
}

// InstallationManifest returns the path of application manifest
func (p installPaths) InstallationManifest() (string, error) {
	return p.Dir(installationManifestPath)
}
