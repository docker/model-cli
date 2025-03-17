package paths

import (
	"os"
	"path/filepath"
)

const (
	// vmsDir is the Data/ subdirectory where the VMs are.
	vmsDir = "vms"
	// VMDir is the Data/ subdirectory where the master VM is.
	vmDirName     = vmsDir + "/0"
	vmDataDirName = vmDirName + "/data"
)

// Place all files in ~/.docker/desktop.
func desktop(elems ...string) string {
	dir := Home(".docker", "desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Warnf("unable to create %s: %v", dir, err)
	}
	return filepath.Join(append([]string{dir}, elems...)...)
}

// Data returns the sandbox writable path.
func Data() string {
	return desktop()
}

// TasksDir returns the directory where the supervisor stores the active task metadata files.
func TasksDir() string {
	return desktop("tasks")
}

// EngineTasksDir returns the directory where the supervisor stores the active engine task metadata files.
func EngineTasksDir() string {
	return desktop("engine_tasks")
}

// LogsDir is the directory for log files.
func LogsDir() string {
	return desktop(logsDirName)
}

// ExtensionsRoot returns the root for Desktop extensions.
func ExtensionsRoot() string {
	return desktop("extensions")
}

/*---------.
| Per VM.  |
`---------*/

// VMDir is the path to the VM dir (vms/0).
func VMDir() (string, error) {
	return desktop(vmDirName), nil
}

// VMDefaultDiskDir is the default path to the VM disk dir (vms/0/data).
func VMDefaultDiskDir() (string, error) {
	return desktop(vmDataDirName), nil
}

// ConsoleRing is the path to the `console-ring` file written by hyperkit.
func ConsoleRing() string {
	return desktop(vmDirName, "console-ring")
}

// Macaddr is the path to where the persistent macaddr is stored by com.docker.virtualization
func Macaddr() string {
	return desktop(vmDirName, "macaddr")
}

// DockerCLIPlugins returns the path to the docker cli plugins directory
func DockerCLIPlugins() (string, error) {
	return desktop("cli-plugins"), nil
}

// WSL returns the path to the WSL executable.
func WSL() (string, error) {
	return "", nil
}
