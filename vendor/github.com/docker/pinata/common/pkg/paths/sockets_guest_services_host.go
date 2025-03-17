package paths

import (
	"path"
)

type guestServiceHostSockets struct {
	root      func() string // directory in which sockets are placed
	uriPrefix string        // socket address prefix
}

func (s guestServiceHostSockets) path(name string) string {
	if isWindows() {
		// filepath.Join does not do the right thing for pipe paths for some reason
		return s.root() + `\` + IntegrationTestPrefix() + name
	}
	// There is a maximum size of Unix domain socket path which fits in the kernel's
	// struct sockaddr. We work around this by changing the current directory in main()
	// to the directory containing the sockets, and using relative paths.
	short, err := ShortenUnixSocketPath(path.Join(s.root(), name))
	if err != nil {
		// This can only happen if we've created a path inside the socket directory which
		// is still too long. We will catch this in CI, so fail immediately here with a
		// readable error (instead of the kernel's default "Invalid argument", see
		// https://docker.slack.com/archives/D035DAECJH2/p1648222173757319 )
		log.Fatal(err)
	}
	return short
}

// DebugSocketForwarder returns the path of the VM debug socketforwarder socket on the host.
func (s guestServiceHostSockets) DebugSocketforwarder() string {
	return s.path(debugSocketforwarderSocketName)
}

// PerfSocketForwarder returns the path of the performance socketforwarder socket on the host.
func (s guestServiceHostSockets) PerfSocketforwarder() string {
	return s.path(perfSocketforwarderSocketName)
}

// VMControlInit returns the path of the VM init control API socket on the host.
func (s guestServiceHostSockets) VMControlInit() string {
	return s.path(vmControlInitSocketName)
}

// DevEnvVolumes returns the pathe to the Dev environments volume socket.
func (s guestServiceHostSockets) DevEnvVolumes() string {
	return s.path(devEnvVolumesSocketName)
}

// Diagnosticd returns the path to the diagnosticd endpoint socket.
func (s guestServiceHostSockets) Diagnosticd() string {
	return s.path(diagnosticdSocketName)
}

// DNSForwarder returns the path of the dns-forwarder control socket in the LinuxKit VM.
func (s guestServiceHostSockets) DNSForwarder() string {
	return s.path(dnsForwarderSocketName)
}

// DockerRaw returns the path to the raw (unproxied) docker API endpoint socket
func (s guestServiceHostSockets) DockerRaw() string {
	return s.path(dockerRawSocketName)
}

func (s guestServiceHostSockets) DockerRawWindows() string {
	return s.path(dockerRawWindowsSocketName)
}

// FilesystemEventAddress returns the path to the grpcfuse event injection socket.
func (s guestServiceHostSockets) FilesystemEventAddress(withUriPrefix bool) string {
	path := s.path(filesystemEventSocketName)
	if withUriPrefix {
		path = s.uriPrefix + path
	}
	return path
}

// HttpProxyControl returns the path of the HTTP proxy control socket.
func (s guestServiceHostSockets) HttpProxyControl() string {
	return s.path(httpProxyControlSocketName)
}

// DockerAPIProxyControl returns the path of the control socket for the Docker API proxy.
func (s guestServiceHostSockets) DockerAPIProxyControl() string {
	return s.path(dockerAPIProxyControlSocketName)
}

// LifecycleServer returns the directory of the host's lifecycle-server.sock.
func (s guestServiceHostSockets) LifecycleServer() string {
	return s.path(lifecycleServerSocketName)
}

// Stats returns the directory of the host's stats.sock.
func (s guestServiceHostSockets) Stats() string {
	return s.path(statsSocketName)
}

// MutagenConduit returns the path to the Mutagen conduit server socket.
func (s guestServiceHostSockets) MutagenConduit() string {
	return s.path(mutagenConduitSocketName)
}

// Transfused returns the path to the transfused socket.
func (s guestServiceHostSockets) Transfused() string {
	if isDarwin() {
		return s.path(transfusedSocketName)
	}
	return ""
}

// VolumeContents returns the path to the volume contents socket.
func (s guestServiceHostSockets) VolumeContents() string {
	return s.path(volumeContentsSocketName)
}

// ContainerFilesystem returns the path to the container filesystem socket.
func (s guestServiceHostSockets) ContainerFilesystem() string {
	return s.path(containerFilesystemSocketName)
}

// ImageInspector returns the path to the image inspector socket.
func (s guestServiceHostSockets) ImageInspector() string {
	return s.path(imageInspectorSocketName)
}

// JFS returns the path to the jfs control server socket.
func (s guestServiceHostSockets) JFS() string {
	return s.path(jfsSocketName)
}

// WindowsWSLCrossDistroService is the named pipe to cross distro services
func (s guestServiceHostSockets) WindowsWSLCrossDistroService() string {
	if isWindows() {
		return s.path(wslCrossDistroServiceSocketName)
	}
	return ""
}

// Wsl2BootstrapExposePorts is not used on OSx.
func (s guestServiceHostSockets) Wsl2BootstrapExposePorts() string {
	if isWindows() {
		return s.path(wsl2BootstrapExposePortsSocketName)
	}
	return ""
}
