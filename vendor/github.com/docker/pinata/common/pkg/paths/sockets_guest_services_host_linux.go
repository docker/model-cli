package paths

import "sync"

const (
	containerFilesystemSocketName      = "container-filesystem.sock"
	imageInspectorSocketName           = "image-inspector.sock"
	jfsSocketName                      = "jfs.sock"
	debugSocketforwarderSocketName     = "debug-socketforwarder.sock"
	perfSocketforwarderSocketName      = "perf-socketforwarder.sock"
	vmControlInitSocketName            = "control-init.sock"
	devEnvVolumesSocketName            = "devenv-volumes.sock"
	diagnosticdSocketName              = "diagnosticd.sock"
	dnsForwarderSocketName             = "dns-forwarder.sock"
	dockerRawSocketName                = "docker.raw.sock"
	dockerRawWindowsSocketName         = ""
	filesystemEventSocketName          = "filesystem-event.sock"
	httpProxyControlSocketName         = "http-proxy-control.sock"
	dockerAPIProxyControlSocketName    = "docker-api-proxy-control.sock"
	lifecycleServerSocketName          = "lifecycle-server.sock"
	statsSocketName                    = "stats.sock"
	mutagenConduitSocketName           = "mutagen-conduit.sock"
	transfusedSocketName               = ""
	volumeContentsSocketName           = "volume-contents.sock"
	wsl2BootstrapExposePortsSocketName = ""
	wslCrossDistroServiceSocketName    = ""
)

func GuestServiceHostSockets() guestServiceHostSockets {
	return guestServiceHostSockets{
		uriPrefix: "unix:",
		root: sync.OnceValue[string](func() string {
			return desktop()
		}),
	}
}
