package paths

const (
	backendNativeSocketName                = "dockerBackendNativeApiServer"
	backendSocketName                      = "dockerBackendApiServer"
	virtualizationSocketName               = "dockerVirtualization"
	cliAPISocketName                       = "dockerCliApi"
	devEnvSocketName                       = "dockerDevEnvApiServer"
	dockerWindowsAPIProxyControlSocketName = "dockerWindowsAPIProxyControl"
	dockerCLISocketName                    = "docker_cli"
	dockerDesktopLinuxEngineSocketName     = "dockerDesktopLinuxEngine"
	dockerDesktopWindowsEngineSocketName   = "dockerDesktopWindowsEngine"
	dockerSocketName                       = "" // not used on Windows
	dockerVolumesSocketName                = "dockerVolumeContents"
	extensionsSocketName                   = "dockerExtensionManagerAPI"
	modulesSocketName                      = "dockerModulesManagerAPI"
	filesystemAddressDPassing              = "" // not used on Windows
	filesystemVolumeSocketName             = "dockerVolume"
	filesystemVolumeLibKrunSocketName      = "" // not used on Windows
	mountpointsLibKrunSocketName           = "" // not used on Windows
	httpProxySocketName                    = "dockerHTTPProxy"
	networkProxySocketName                 = "dockerNetworkProxy"
	gvisorEthernetSocketName               = "dockerEthernet"
	gvisorEthernetVPNkitProtocolSocketName = "dockerEthernetVpnkit"
	gvisorEthernetVfkitProtocolSocketName  = "dockerEthernetVfkit"
	gvisorEthernetQemuProtocolSocketName   = "dockerEthernetQemu"
	osxfsControlSocketName                 = "" // not used on Windows
	osxfsDataSocketName                    = "" // not used on Windows
	serviceSocketName                      = "dockerBackendV2"
	virtiofsSocketName                     = "" // not used on Windows
	vpnkitBridgeFDPassing                  = "" // not used on Windows
	vpnkitDiagnosticsSocketName            = "dockerVpnKitDiagnostics"
	vpnkitHTTPControl                      = "dockerVpnKitHTTPControl"
	vpnkitPcapSocketName                   = "dockerVpnKitPcap"
	vpnkitSocketName                       = "dockerVpnkit"
	errorReportingSocketName               = "errorReporter"
	otelSystemTelemetrySocketName          = "systemTelemetryOtlpHttp.sock"
	otelUserAnalyticsSocketName            = "userAnalyticsOtlpHttp.sock"
	webGPUSocketName                       = "webgpu.sock"
	inferenceSocketName                    = "dockerInference"
	inferenceBackendSocketNameTemplate     = "inference-%d.sock"
	buildSocketName                        = "dockerDesktopBuildServer" // https://github.com/docker/desktop-build
)

func HostServiceSockets() hostServiceSockets {
	return hostServiceSockets{
		root:        `\\.\pipe`,
		winUnixPath: UnixSocketsDir(),
		uriPrefix:   "npipe:",
	}
}
