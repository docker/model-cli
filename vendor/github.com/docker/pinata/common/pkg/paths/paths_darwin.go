package paths

import (
	"path"
	"path/filepath"
)

// Directories.
const (
	// vmsDir is the Data/ subdirectory where the VMs are.
	vmsDir = "vms"
	// VMDir is the Data/ subdirectory where the master VM is.
	// Go does not allow to use `filepath.Join(VmsDir, "0")` for a constant.
	vmDirName     = vmsDir + "/0"
	vmDataDirName = vmDirName + "/data"
)

// Data returns the sandbox writable path.
func Data() string {
	return filepath.Join(Container(), "Data")
}

// TasksDir returns the directory where the supervisor stores the active task metadata files.
func TasksDir() string {
	return filepath.Join(Data(), "tasks")
}

// EngineTasksDir returns the directory where the supervisor stores the active engine task metadata files.
func EngineTasksDir() string {
	return filepath.Join(Data(), "engine_tasks")
}

// data returns the concatenation of the path elements `elem` in the
// `Data` directory.  If the returned path is under CWD, then the
// result is a relative path, otherwise absolute.
func data(elem ...string) string {
	return filepath.Join(append([]string{Data()}, elem...)...)
}

func LogsDir() string {
	return data(logsDirName)
}

// ExtensionsRoot returns the root for Desktop extensions.
func ExtensionsRoot() string {
	return data("extensions")
}

/*---------.
| Per VM.  |
`---------*/

// VMDir is the path to the VM dir (vms/0).
func VMDir() (string, error) {
	return data(vmDirName), nil
}

// VMDefaultDiskDir is the default path to the VM disk dir (vms/0/data).
func VMDefaultDiskDir() (string, error) {
	return data(vmDataDirName), nil
}

// ConsoleRing is the path to the `console-ring` file written by hyperkit.
func ConsoleRing() string {
	return path.Join(data(vmDirName), "console-ring")
}

// Macaddr is the path to where the persistent macaddr is stored by com.docker.virtualization.
func Macaddr() string {
	return path.Join(data(vmDirName), "macaddr")
}

// DockerCLIPlugins returns the path to the docker cli plugins directory
func DockerCLIPlugins() (string, error) {
	return InstallPaths.Dir("Resources", "cli-plugins")
}

// GroupContainerPreferences returns group.com.docker.plist path
func GroupContainerPreferences() string {
	return Home("Library", "Preferences", "group.com.docker.plist")
}

func ElectronData() string {
	return Home("Library", "Application Support", "Docker Desktop")
}

// ElectronDataPartition returns the host directory that holds the data persisted by the extension's webview.
// Partition directory: ~/Library/Application Support/Docker Desktop/Partitions/<partition>
func ElectronDataPartition(partition string) string {
	return filepath.Join(ElectronData(), "Partitions", partition)
}

// InfoPlist returns the path to the Info.plist
func InfoPlist() (string, error) {
	return InstallPaths.Dir("Info.plist")
}

// VmnetdBinary returns the path to the vmnetd binary
func VmnetdBinary() (string, error) {
	return InstallPaths.Dir("Library", "LaunchServices", "com.docker.vmnetd")
}

// WSL returns the path to the WSL executable.
func WSL() (string, error) {
	return "", nil
}
