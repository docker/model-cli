package paths

const (
	backendNativeSocketName                = "backend.native.sock"
	backendSocketName                      = "backend.sock"
	virtualizationSocketName               = "virtualization.sock"
	cliAPISocketName                       = "docker-cli-api.sock"
	devEnvSocketName                       = "docker-dev-env-api.sock"
	dockerWindowsAPIProxyControlSocketName = ""
	dockerCLISocketName                    = "docker-cli.sock"
	dockerDesktopLinuxEngineSocketName     = "" // not used on Linux
	dockerDesktopWindowsEngineSocketName   = "" // not used on Linux
	dockerSocketName                       = "docker.sock"
	dockerVolumesSocketName                = "volume-contents.sock"
	extensionsSocketName                   = "extension-manager.sock"
	modulesSocketName                      = "modules-manager.sock"
	filesystemAddressDPassing              = "filesystem-fd.sock"
	filesystemVolumeSocketName             = "filesystem-volume.sock"
	filesystemVolumeLibKrunSocketName      = "filesystem-volume-libkrun.sock"
	mountpointsLibKrunSocketName           = "mountpoints-libkrun.sock"
	httpProxySocketName                    = "httpproxy.sock"
	networkProxySocketName                 = "networkproxy.sock"
	gvisorEthernetSocketName               = "ethernet-fd.sock"
	gvisorEthernetVPNkitProtocolSocketName = "ethernet-vpnkit.sock"
	gvisorEthernetVfkitProtocolSocketName  = "ethernet-vfkit.sock"
	gvisorEthernetQemuProtocolSocketName   = "ethernet-qemu.sock"
	osxfsControlSocketName                 = ""
	osxfsDataSocketName                    = ""
	serviceSocketName                      = "" // not used on Linux
	virtiofsSocketName                     = "virtiofs.sock"
	vpnkitBridgeFDPassing                  = "vpnkit-bridge-fd.sock"
	vpnkitDiagnosticsSocketName            = "vpnkit.diag.sock"
	vpnkitHTTPControl                      = "vpnkit.http.sock"
	vpnkitPcapSocketName                   = "vpnkit.pcap.sock"
	vpnkitSocketName                       = "vpnkit.eth.sock"
	errorReportingSocketName               = "error.reporter.sock"
	otelSystemTelemetrySocketName          = "system-telemetry.otlp.grpc.sock"
	otelUserAnalyticsSocketName            = "user-analytics.otlp.grpc.sock"
	webGPUSocketName                       = "webgpu.sock"
	inferenceSocketName                    = "inference.sock"
	inferenceBackendSocketNameTemplate     = "inference-%d.sock"
	buildSocketName                        = "docker-desktop-build.sock" // https://github.com/docker/desktop-build
)

func HostServiceSockets() hostServiceSockets {
	return hostServiceSockets{
		root:      desktop(),
		uriPrefix: "unix:",
	}
}
