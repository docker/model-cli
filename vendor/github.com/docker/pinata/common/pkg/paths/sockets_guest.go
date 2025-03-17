package paths

import (
	"os"
	"path"
	"strings"
)

const (
	linuxKitBackendNativeSocketName                = "backend.native.sock"
	linuxKitBackendSocketName                      = "backend.sock"
	linuxKitContainerFilesystemSocketName          = "container-filesystem.sock"
	linuxKitImageInspectorSocketName               = "image-inspector.sock"
	linuxKitJfsSocketName                          = "jfs.sock"
	linuxKitDNSForwarderSocketName                 = "dns-forwarder.sock"
	linuxKitDNSGRPCInternalSocketName              = "dns.internal.grpc.sock"
	linuxKitDNSGRPCSystemSocketName                = "dns.system.grpc.sock"
	linuxKitDevEnvVolumesSocketName                = "devenv-volumes.sock"
	linuxKitDiagnosticdSocketName                  = "diagnosticd.sock"
	linuxKitDockerGuestProxySocketName             = "docker.proxy.sock"
	linuxKitDockerContainerProxySocketName         = "docker.container-proxy.sock"
	linuxKitDockerProxySocketName                  = "docker.proxy.sock"
	linuxKitCriDockerdProxySocketName              = "cri-dockerd-proxy.sock"
	linuxKitExtensionsSocketName                   = "extension-manager.sock"
	linuxKitModulesSocketName                      = "modules-manager.sock"
	linuxKitDockerAPIProxyControlSocketName        = "docker-api-proxy-control.sock"
	linuxKitFilesystemEventSocketName              = "filesystem-event.sock"
	linuxKitFilesystemSocketName                   = "filesystem.sock"
	linuxKitHTTPProxyRestrictedSocketName          = "http-proxy-restricted.sock"
	linuxKitHTTPProxySocketName                    = "http-proxy.sock"
	linuxKitHubProxySocketName                     = "hubproxy.sock"
	linuxKitLifecycleServerSocketName              = "lifecycle-server.sock"
	linuxKitStatsSocketName                        = "stats.sock"
	linuxKitMemlogdQSocketName                     = "memlogdq.sock"
	linuxKitMutagenConduitSocketName               = "mutagen-conduit.sock"
	linuxKitNetworkProxySocketName                 = "network-proxy.sock"
	linuxkitGvisorEthernetVPNkitProtocolSocketName = "gvisor-ethernet-vpnkit-protocol.sock"
	linuxKitOSXFSDataSocketName                    = "osxfs-data.sock"
	linuxKitProcdSocketName                        = "procd.sock"
	linuxKitSSHAuthSocketName                      = "ssh-auth.sock"
	linuxKitTransfusedSocketName                   = "transfused.sock"
	linuxKitVolumeContentsSocketName               = "volume-contents.sock"
	linuxKitVpnkitDataSocketName                   = "vpnkit-data.sock"
	linuxKitVpnkitSocketName                       = "vpnkit.sock"
	linuxKitWSL2BootstrapExposePortsSocketName     = "wsl2-bootstrap-expose-ports.sock"
	linuxKitWSLCrossDistroServiceSocketName        = "wsl-cross-distro.sock"
	linuxKitWSLSocketforwarderReceiveFDsSocketName = "socketforwarder-receive-fds.sock"
	linuxKitOtelSystemTelemetrySocketName          = "system-telemetry.otlp.grpc.sock"
	linuxKitWebGPUSocketName                       = "webgpu.sock"
	linuxKitBuildSocketName                        = "docker-desktop-build.sock" // https://github.com/docker/desktop-build
)

var (
	distro           = distroName()
	wslGuestServices = "/mnt/host/wsl/" + distro + "/shared-sockets/guest-services"
	wslHostServices  = "/mnt/host/wsl/" + distro + "/shared-sockets/host-services"
)

type (
	guestServiceSockets     struct{}
	hostServiceGuestSockets struct{}
)

var (
	GuestServiceSockets     = guestServiceSockets{}
	HostServiceGuestSockets = hostServiceGuestSockets{}
)

func distroName() string {
	if distro, ok := os.LookupEnv("WSL_DISTRO_NAME"); ok {
		// The e2e distro tests docker-desktop
		if !strings.Contains(distro, "e2e") {
			return distro
		}
	}
	return "docker-desktop"
}

func guestServiceSocket(name string) string {
	return path.Join("/run/guest-services", name)
}

func hostServiceSocket(name string) string {
	return path.Join("/run/host-services", name)
}

// DevEnvVolumes returns the path of the devenv volume socket in the LinuxKit VM.
func (s guestServiceSockets) DevEnvVolumes() string {
	return guestServiceSocket(linuxKitDevEnvVolumesSocketName)
}

// Diagnosticd returns the path of the Diagnosticd socket in the LinuxKit VM.
func (s guestServiceSockets) Diagnosticd() string {
	return guestServiceSocket(linuxKitDiagnosticdSocketName)
}

// DNSForwarder returns the path of the dns-forwarder control socket in the LinuxKit VM.
func (s guestServiceSockets) DNSForwarder() string {
	return guestServiceSocket(linuxKitDNSForwarderSocketName)
}

// DockerProxy returns the path of the docker daemon API proxy in the LinuxKit VM
func (s guestServiceSockets) DockerProxy() string {
	return guestServiceSocket(linuxKitDockerGuestProxySocketName)
}

// DockerContainerProxy returns the path of the containers docker daemon API proxy in the LinuxKit VM
func (s guestServiceSockets) DockerContainerProxy() string {
	return guestServiceSocket(linuxKitDockerContainerProxySocketName)
}

func (s guestServiceSockets) DockerContainerWslProxy(wslDistro string) string {
	return guestServiceSocket(wslDistro + "." + linuxKitDockerContainerProxySocketName)
}

// DockerContainerProxy returns the path of the containers docker daemon API proxy in the LinuxKit VM
func (s guestServiceSockets) CriDockerdProxy() string {
	return guestServiceSocket(linuxKitCriDockerdProxySocketName)
}

// DockerAPIProxyControl returns the path of the control socket for the Docker API guest proxy.
func (s guestServiceSockets) DockerAPIProxyControl() string {
	return guestServiceSocket(linuxKitDockerAPIProxyControlSocketName)
}

// FilesystemEvent returns the path of the filesystem event Unix domain socket in the LinuxKit VM
func (s guestServiceSockets) FilesystemEvent() string {
	return guestServiceSocket(linuxKitFilesystemEventSocketName)
}

// Lifecycle returns the path of the lifecycle-server Unix domain socket in the LinuxKit VM
func (s guestServiceSockets) Lifecycle() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslGuestServices, linuxKitLifecycleServerSocketName)
	}
	return guestServiceSocket(linuxKitLifecycleServerSocketName)
}

// Stats returns the path of the stats Unix domain socket in the LinuxKit VM
func (s guestServiceSockets) Stats() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslGuestServices, linuxKitStatsSocketName)
	}
	return guestServiceSocket(linuxKitStatsSocketName)
}

// Log returns the path of the memlogd socket in the LinuxKit VM.
func (s guestServiceSockets) Log() string {
	return guestServiceSocket(linuxKitMemlogdQSocketName)
}

// MutagenConduit returns the path of the Mutagen conduit server socket in the
// LinuxKit VM.
func (s guestServiceSockets) MutagenConduit() string {
	return guestServiceSocket(linuxKitMutagenConduitSocketName)
}

// Procd returns the path of the Procd socket in the LinuxKit VM.
func (s guestServiceSockets) Procd() string {
	return guestServiceSocket(linuxKitProcdSocketName)
}

// Transfused returns the path of the Transfused socket in the LinuxKit VM.
func (s guestServiceSockets) Transfused() string {
	return guestServiceSocket(linuxKitTransfusedSocketName)
}

// VolumeContents returns the path of the volume contents socket in the LinuxKit VM.
func (s guestServiceSockets) VolumeContents() string {
	return guestServiceSocket(linuxKitVolumeContentsSocketName)
}

// ContainerFilesystem returns the path of the container filesystem socket in the LinuxKit VM.
func (s guestServiceSockets) ContainerFilesystem() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslGuestServices, linuxKitContainerFilesystemSocketName)
	}
	return guestServiceSocket(linuxKitContainerFilesystemSocketName)
}

// ImageInspector returns the path of the image inspector socket in the LinuxKit VM.
func (s guestServiceSockets) ImageInspector() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslGuestServices, linuxKitImageInspectorSocketName)
	}
	return guestServiceSocket(linuxKitImageInspectorSocketName)
}

// JFS returns the path of the jfs control server socket in the LinuxKit VM.
func (s guestServiceSockets) JFS() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslGuestServices, linuxKitJfsSocketName)
	}
	return guestServiceSocket(linuxKitJfsSocketName)
}

// WSL2BootstrapExposePorts returns the path of the port forwarding
// service's bootstrap socket in the LinuxKit VM.
func (s guestServiceSockets) WSL2BootstrapExposePorts() string {
	return guestServiceSocket(linuxKitWSL2BootstrapExposePortsSocketName)
}

// WSLCrossDistroService is the socket path to cross distro services
func (s guestServiceSockets) WSLCrossDistroService() string {
	return guestServiceSocket(linuxKitWSLCrossDistroServiceSocketName)
}

func (s guestServiceSockets) WSLSocketforwarderReceiveFDs() string {
	return guestServiceSocket(linuxKitWSLSocketforwarderReceiveFDsSocketName)
}

// Backend returns the path of the backend API socket in the LinuxKit VM.
func (s hostServiceGuestSockets) Backend() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslHostServices, linuxKitBackendSocketName)
	}
	return hostServiceSocket(linuxKitBackendSocketName)
}

// BackendNative returns the path of the legacy (native) backend API within the LinuxKit VM.
func (s hostServiceGuestSockets) BackendNative() string {
	if whereami() == InsideWslWorkspace {
		return path.Join(wslHostServices, linuxKitBackendNativeSocketName)
	}
	return hostServiceSocket(linuxKitBackendNativeSocketName)
}

// DNSGRPCInternal returns the path of the DNS over GRPC socket handling .docker.internal queries in the LinuxKit VM.
func (s hostServiceGuestSockets) DNSGRPCInternal() string {
	return hostServiceSocket(linuxKitDNSGRPCInternalSocketName)
}

// DNSGRPCSystem returns the path of the DNS over GRPC socket handling system queries in the LinuxKit VM.
func (s hostServiceGuestSockets) DNSGRPCSystem() string {
	return hostServiceSocket(linuxKitDNSGRPCSystemSocketName)
}

// DockerProxy returns the path of the host docker proxy in the LinuxKit VM
func (s hostServiceGuestSockets) DockerProxy() string {
	return hostServiceSocket(linuxKitDockerProxySocketName)
}

// HubProxy returns the path of the Docker Hub pull proxy in the LinuxKit VM
func (s hostServiceGuestSockets) HubProxy() string {
	return hostServiceSocket(linuxKitHubProxySocketName)
}

// HTTPProxy returns the path of the backend HTTP proxy in the LinuxKit VM
func (s hostServiceGuestSockets) HTTPProxy() string {
	return hostServiceSocket(linuxKitHTTPProxySocketName)
}

// HTTPProxyRestricted returns the path of the backend restriced HTTP proxy in the LinuxKit VM. This applies the RAM config.
func (s hostServiceGuestSockets) HTTPProxyRestricted() string {
	return hostServiceSocket(linuxKitHTTPProxyRestrictedSocketName)
}

// NetworkProxy returns the path of the backend network proxy in the LinuxKit VM
func (s hostServiceGuestSockets) NetworkProxy() string {
	return hostServiceSocket(linuxKitNetworkProxySocketName)
}

// GvisorEthernetVPNkitProtocol returns the path of the gVisor ethernet VPNKit protocol socket in the LinuxKit VM
func (s hostServiceGuestSockets) GvisorEthernetVPNkitProtocol() string {
	return hostServiceSocket(linuxkitGvisorEthernetVPNkitProtocolSocketName)
}

// Filesystem returns the path of the filesystem server Unix domain socket in the LinuxKit VM
func (s hostServiceGuestSockets) Filesystem() string {
	return hostServiceSocket(linuxKitFilesystemSocketName)
}

// FilesystemAddressVsock is the address of the main fileserver on the host over vsock.
func (s hostServiceGuestSockets) FilesystemAddressVsock() string {
	return `vsock:4099` // 0x1003
}

// OSXFSData returns the path of the OSXFS data socket in the LinuxKit VM.
func (s hostServiceGuestSockets) OSXFSData() string {
	return hostServiceSocket(linuxKitOSXFSDataSocketName)
}

// SSHAuth returns the path of the SSH authentication socket in the LinuxKit VM.
func (s hostServiceGuestSockets) SSHAuth() string {
	return hostServiceSocket(linuxKitSSHAuthSocketName)
}

// Vpnkit returns the path of the vpnkit socket in the LinuxKit VM.
func (s hostServiceGuestSockets) Vpnkit() string {
	return hostServiceSocket(linuxKitVpnkitSocketName)
}

// VpnkitData returns the path of the vpnkit data socket in the LinuxKit VM.
func (s hostServiceGuestSockets) VpnkitData() string {
	return hostServiceSocket(linuxKitVpnkitDataSocketName)
}

// Extensions returns the path of the Extension Manager socket in the LinuxKit VM.
func (s hostServiceGuestSockets) Extensions() string {
	return hostServiceSocket(linuxKitExtensionsSocketName)
}

// Modules returns the path of the Modules Manager socket in the LinuxKit VM.
func (s hostServiceGuestSockets) Modules() string {
	return hostServiceSocket(linuxKitModulesSocketName)
}

func (s hostServiceGuestSockets) OTelSystemTelemetry() string {
	return hostServiceSocket(linuxKitOtelSystemTelemetrySocketName)
}

func (s hostServiceGuestSockets) WebGPU() string {
	return hostServiceSocket(linuxKitWebGPUSocketName)
}

func (s hostServiceGuestSockets) Build() string {
	return hostServiceSocket(linuxKitBuildSocketName)
}
