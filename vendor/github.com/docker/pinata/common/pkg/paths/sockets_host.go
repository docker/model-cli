package paths

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/pinata/common/pkg/engine"
)

type hostServiceSockets struct {
	root        string // directory in which sockets are placed
	winUnixPath string // directory in which AF_UNIX sockets are stored in Windows
	uriPrefix   string // socket address prefix
}

// path returns a path for a named pipe (on Windows) or a Unix socket (on macOS/Linux).
func (s hostServiceSockets) path(elem ...string) string {
	if isWindows() {
		return s.namedPipePath(elem...)
	}
	return s.unixSocketPath(elem...)
}

// namedPipePath returns a path for a named pipe on Windows.
//
// If this function is called on a non-Windows platform, it will panic.
func (s hostServiceSockets) namedPipePath(elem ...string) string {
	if !isWindows() {
		log.Fatalf("cannot create a named pipe on os: %s", runtime.GOOS)
	}
	// manually concat - filepath.Join does not handle pipe paths correctly
	return s.root + `\` + IntegrationTestPrefix() + filepath.Join(elem...)
}

func IntegrationTestPrefix() string {
	if whereami() == IntegrationTests {
		return filepath.Base(integrationTestsRootPath)
	}
	return ""
}

// unixSocketPath returns a path for an AF_UNIX socket across all platforms (Windows/macOS/Linux).
func (s hostServiceSockets) unixSocketPath(elem ...string) string {
	if isWindows() {
		// windows does not have special path length requirements for AF_UNIX,
		// and we need to use a different field for the base path
		return filepath.Join(append([]string{s.winUnixPath}, elem...)...)
	}

	// On Linux/macOS, there is a maximum size of Unix domain socket paths which
	// fits in the kernel's struct sockaddr. We work around this by changing
	// the current directory in main() to the directory containing the sockets,
	// and using relative paths.
	short, err := ShortenUnixSocketPath(path.Join(append([]string{s.root}, elem...)...))
	if err != nil {
		// This can only happen if we've created a path inside the socket directory which
		// is still too long. We will catch this in CI, so fail immediately here with a
		// readable error (instead of the kernel's default "Invalid argument")
		log.Fatal(err)
	}
	return short
}

// Backend returns the path to the backend socket used by the GUI and VM.
func (s hostServiceSockets) Backend() string {
	if IsInsideVM() {
		return HostServiceGuestSockets.Backend()
	}
	return s.path(backendSocketName)
}

// BackendNative returns the path to legacy (native - c#/swift) backend socket.
func (s hostServiceSockets) BackendNative() string {
	if IsInsideVM() {
		return HostServiceGuestSockets.BackendNative()
	}
	return s.path(backendNativeSocketName)
}

// DockerCLI returns the path to the local HTTP service for CLI tooling to use
// for Docker Desktop integration.
func (s hostServiceSockets) DockerCLI(includeScheme bool) string {
	scheme := "unix://"
	address := s.path(dockerCLISocketName)

	if runtime.GOOS == "windows" {
		scheme = "npipe://"
	} else if runtime.GOOS == "linux" && whereami() == InsideWslWorkspace {
		address = filepath.Join("/var/run", dockerCLISocketName)
	}

	if includeScheme {
		address = scheme + address
	}
	return address
}

// ComposeCloudAPI returns the path to the local gRPC service hosted by the
// compose-cli `serve` command.
func (s hostServiceSockets) ComposeCloudAPI(includeScheme bool) string {
	var address, scheme string
	if isDarwin() {
		scheme = "unix://"
		address = Home(".docker", "run", cliAPISocketName)
	}
	if isWindows() {
		scheme = "npipe://"
		address = s.path(cliAPISocketName)
	}
	if includeScheme && address != "" {
		address = scheme + address
	}
	return address
}

// Console is the path to the console Unix domain socket used by com.docker.virtualization.
func (s hostServiceSockets) Console() string {
	if isWindows() {
		return ""
	}
	if isLinux() {
		return s.path("console.sock")
	}
	return s.path(vmDirName, "console.sock")
}

// DockerWindowsAPIProxyControl returns the path of the control socket for the Docker API proxy.
// Eventually this should be folded into the Go backend.
func (s hostServiceSockets) DockerWindowsAPIProxyControl() string {
	return s.path(dockerWindowsAPIProxyControlSocketName)
}

// DockerHost returns a string suitable for `docker -H`
func (s hostServiceSockets) DockerHost(e engine.Engine) string {
	if isWindows() {
		return s.uriPrefix + "//" + filepath.ToSlash(s.Docker(e))
	}
	return s.uriPrefix + "//" + s.Docker(e)
}

// DockerRawHost returns a string suitable for `docker -H`
func (s hostServiceSockets) DockerRawHost() string {
	return s.uriPrefix + "//" + filepath.ToSlash(s.path(dockerRawSocketName))
}

// DockerRawHostSocket is similar to DockerRawHost but without the uriPrefix
func (s hostServiceSockets) DockerRawHostSocket() string {
	return s.path(dockerRawSocketName)
}

func (s hostServiceSockets) Proxy() string {
	if runtime.GOOS == "windows" {
		return s.path("docker_engine")
	}
	return ""
}

func (s hostServiceSockets) ProxyPrivate() string {
	if runtime.GOOS == "windows" {
		return s.path("dockerDesktopEngine")
	}
	return ""
}

// Docker returns the path to docker API endpoint socket.
func (s hostServiceSockets) Docker(e engine.Engine) string {
	switch runtime.GOOS {
	case "darwin":
		// The Library/Containers path is too long, and often hits the 104 character limit
		// (108 on Linux). For most sockets we can work around this with `chdir` and using
		// relative paths, but the docker socket is used by the Docker CLI from any directory
		// and so we need the absolute path to be short.
		return Home(".docker", "run", dockerSocketName)
	case "windows":
		switch e {
		case engine.Linux:
			return s.path(dockerDesktopLinuxEngineSocketName)
		case engine.Windows:
			return s.path(dockerDesktopWindowsEngineSocketName)
		}
		return ""
	case "linux":
		if _, err := os.Stat("/run/host-services"); err == nil {
			return HostServiceGuestSockets.DockerProxy()
		}
		if whereami() == InsideLinuxkit {
			return HostServiceGuestSockets.DockerProxy()
		}
		if whereami() == InsideWslWorkspace {
			return "/var/run/docker.sock"
		}
		// We require an absolute path
		return filepath.Join(s.root, dockerSocketName)
	}
	return ""
}

// HTTPProxy is the address of the internal HTTP proxy on the host. This proxy forwards to the configured upstream proxy.
func (s hostServiceSockets) HTTPProxy() string {
	return s.path(httpProxySocketName)
}

// NetworkProxy is the address of the TCP network proxy on the host.
func (s hostServiceSockets) NetworkProxy() string {
	return s.path(networkProxySocketName)
}

// GvisorEthernet receives fds with ethernet frames.
func (s hostServiceSockets) GvisorEthernet() string {
	return s.path(gvisorEthernetSocketName)
}

// GvisorEthernetVPNkitProtocol receives a stream of ethernet frames using the vpnkit protocol.
func (s hostServiceSockets) GvisorEthernetVPNkitProtocol() string {
	return s.path(gvisorEthernetVPNkitProtocolSocketName)
}

// GvisorEthernetVfkitProtocol receives a stream of ethernet frames using the vfkit protocol.
func (s hostServiceSockets) GvisorEthernetVfkitProtocol() string {
	// libkrun adds an extra -krun.sock suffix to the socket name, which can easily push the path over the limit.
	// https://github.com/containers/libkrun/blob/0bea04816f4dc414a947aa7675e169cbbfbd45dc/src/devices/src/virtio/net/gvproxy.rs#L30C64-L30C76
	suffix := "-krun.sock"
	tmp := s.path(gvisorEthernetVfkitProtocolSocketName + suffix)
	return strings.TrimSuffix(tmp, suffix)
}

// GvisorEthernetQemuProtocol receives a stream of ethernet frames using the qemu protocol.
func (s hostServiceSockets) GvisorEthernetQemuProtocol() string {
	return s.path(gvisorEthernetQemuProtocolSocketName)
}

// FilesystemAddressFDPassing is sent file descriptors from AF_VSOCK connections received by virtualization.framework.
func (s hostServiceSockets) FilesystemAddressFDPassing() string {
	if isWindows() {
		return ""
	}
	return s.path(filesystemAddressDPassing)
}

// FilesystemVolumeAddress is the local address of the volume sharer on the host.
func (s hostServiceSockets) FilesystemVolumeAddress() string {
	return s.uriPrefix + s.path(filesystemVolumeSocketName)
}

// FilesystemVolumeAddressLibKrun is the local address of the volume sharer on the host in the com.docker.libkrun process.
// This is only needed so the file watching can be in the same process as the virtiofs backend, so the IgnoreSelf flag works as expected..
func (s hostServiceSockets) FilesystemVolumeAddressLibKrun() string {
	return s.uriPrefix + s.path(filesystemVolumeLibKrunSocketName)
}

// MountpointsLibKrun returns the path to the mountpoints socket in the com.docker.krun process.
// This is used to inform libkrun about docker run -v mountpoints, so it can disable the attribute cache.
func (s hostServiceSockets) MountpointsLibKrun() string {
	return s.uriPrefix + s.path(mountpointsLibKrunSocketName)
}

// Mutagen returns the path of the Mutagen daemon socket (which exists within
// the Mutagen data directory). On Windows, this path points to a file
// containing the path of a Windows Named Pipe. As such, the path returned by
// this function should only be dialed using Mutagen's IPC dialers (which
// understand this convention).
func (s hostServiceSockets) Mutagen() string {
	return filepath.Join(UserPaths.MutagenDataDirectory(), "daemon", "daemon.sock")
}

// OsxfsControl returns the path to the osx control socket
func (s hostServiceSockets) OsxfsControl() string {
	if isDarwin() {
		return s.path(osxfsControlSocketName)
	}
	return ""
}

// OSXFSData returns the path to the OSx FS data socket
func (s hostServiceSockets) OSXFSData() string {
	if isDarwin() {
		return s.path(osxfsDataSocketName)
	}
	return ""
}

// Service returns the path of the Administrator service pipe.
func (s hostServiceSockets) Service() string {
	if isWindows() {
		return s.path(serviceSocketName)
	}
	return ""
}

// ServiceDev returns the path of the Administrator service pipe
// for local development builds (pinata).
func (s hostServiceSockets) ServiceDev() string {
	if isWindows() {
		return s.path(serviceSocketName + "Dev")
	}
	return ""
}

// SSHAuth returns the path to the SSH auth socket
func (s hostServiceSockets) SSHAuth() (string, error) {
	if isWindows() {
		return "", errors.New("ssh auth socket forwarding not available on Windows")
	}
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		return sock, nil
	}
	return "", errors.New("no SSH_AUTH_SOCK environment variable defined")
}

// Vpnkit returns the path to vpnkit unix socket
func (s hostServiceSockets) Vpnkit() string {
	return s.path(vpnkitSocketName)
}

// VpnkitHTTPControl returns the path of the vpnkit HTTP control API
func (s hostServiceSockets) VpnkitHTTPControl() string {
	return s.path(vpnkitHTTPControl)
}

// VpnkitBridgeFDPassing is sent file descriptors from AF_VSOCK connections received by virtualization.framework.
func (s hostServiceSockets) VpnkitBridgeFDPassing() string {
	if isWindows() {
		return ""
	}
	return s.path(vpnkitBridgeFDPassing)
}

// VpnkitDiagnostics returns the path to vpnkit diagnostics socket.
// It is used by docker-diagnose to get... diagnostics.
func (s hostServiceSockets) VpnkitDiagnostics() string {
	return s.path(vpnkitDiagnosticsSocketName)
}

// VpnkitPcap returns the path to vpnkit pcap socket.
// This can be used to debug networking problems.
func (s hostServiceSockets) VpnkitPcap() string {
	return s.path(vpnkitPcapSocketName)
}

// VirtioFSDaemon returns the path to virtiofsd.
func (s hostServiceSockets) VirtioFSDaemon() string {
	if isWindows() {
		return ""
	}
	return s.path(virtiofsSocketName)
}

// ExtensionsHost returns the path of the Unix domain socket / named pipe running the Extension manager on the host.
func (s hostServiceSockets) ExtensionsHost() string {
	return s.path(extensionsSocketName)
}

// ModulesHost returns the path of the Unix domain socket / named pipe running the Modules manager on the host.
func (s hostServiceSockets) ModulesHost() string {
	return s.path(modulesSocketName)
}

// DevEnvsSocket returns the path to dev environments Unix domain socket / named pipe.
func (s hostServiceSockets) DevEnvsSocket() string {
	return s.path(devEnvSocketName)
}

// DockerVolumeSocket returns the path to docker volumes Unix domain socket / named pipe.
func (s hostServiceSockets) DockerVolumeSocket() string {
	return s.path(dockerVolumesSocketName)
}

// Virtualization returns the path to the com.docker.virtualization API socket.
func (s hostServiceSockets) Virtualization() string {
	return s.path(virtualizationSocketName)
}

// ErrorReportingSocket returns the path to the error reporting socket.
func (s hostServiceSockets) ErrorReportingSocket() string {
	return s.path(errorReportingSocketName)
}

func (s hostServiceSockets) OTelUserAnalytics() string {
	if isDarwin() {
		// CLI tooling uses this socket, so we need to keep it < 108 characters
		return Home(".docker", "run", otelUserAnalyticsSocketName)
	}
	return s.unixSocketPath(otelUserAnalyticsSocketName)
}

func (s hostServiceSockets) OTelSystemTelemetry() string {
	if IsInsideVM() {
		return HostServiceGuestSockets.OTelSystemTelemetry()
	}
	// no need for path shortening here, we're the only ones writing to it
	return s.unixSocketPath(otelSystemTelemetrySocketName)
}

func (s hostServiceSockets) WebGPU() string {
	if IsInsideVM() {
		return HostServiceGuestSockets.WebGPU()
	}
	return s.path(webGPUSocketName)
}

func (s hostServiceSockets) Inference() string {
	return s.path(inferenceSocketName)
}

func (s hostServiceSockets) InferenceBackend(slot int) string {
	return s.unixSocketPath(fmt.Sprintf(inferenceBackendSocketNameTemplate, slot))
}

func (s hostServiceSockets) Build() string {
	if IsInsideVM() {
		return HostServiceGuestSockets.Build()
	}
	return s.path(buildSocketName)
}
