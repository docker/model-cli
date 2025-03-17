package paths

const (
	containerFilesystemSocketName      = "dockerContainerFilesystem"
	imageInspectorSocketName           = "imageInspector"
	jfsSocketName                      = "dockerJfs"
	debugShellSocketName               = "dockerDebugShell"
	debugSocketforwarderSocketName     = "dockerDebugSocketforwarder"
	perfSocketforwarderSocketName      = "dockerPerfSocketforwarder"
	vmControlInitSocketName            = "dockerControlInit"
	devEnvVolumesSocketName            = "dockerDevEnvVolumes"
	diagnosticdDirectSocketName        = "dockerDiagnosticdDirect"
	diagnosticdSocketName              = "dockerDiagnosticd"
	dnsForwarderSocketName             = "dockerDnsForwarder"
	dockerRawSocketName                = "docker_engine_linux"
	dockerRawWindowsSocketName         = "docker_engine_windows"
	filesystemEventSocketName          = "dockerFilesystemEvent"
	filesystemTestSocketName           = "dockerFilesystemTest"
	httpProxyControlSocketName         = "dockerHttpProxyControl"
	dockerAPIProxyControlSocketName    = "dockerAPIProxyControl"
	lifecycleServerSocketName          = "dockerLifecycleServer"
	statsSocketName                    = "dockerStats"
	mutagenConduitSocketName           = "dockerMutagenConduit"
	transfusedSocketName               = "" // not used on Windows
	volumeContentsSocketName           = "dockerVolumeContents"
	wsl2BootstrapExposePortsSocketName = "dockerWsl2BootstrapExposePorts"
	wslCrossDistroServiceSocketName    = "dockerWSLCrossDistroService"
)

func GuestServiceHostSockets() guestServiceHostSockets {
	return guestServiceHostSockets{
		uriPrefix: "npipe:",
		root:      func() string { return `\\.\pipe` },
	}
}
