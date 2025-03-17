package paths

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	vmDirName = "vm-data"
)

// Data returns a writable path where we can store internal state.
func Data() string {
	appData, err := AppData()
	if err != nil {
		return ""
	}
	return appData
}

// TasksDir returns the directory where the supervisor stores the active task metadata files.
func TasksDir() string {
	tmp, err := LocalAppData()
	if err != nil {
		return ""
	}
	return filepath.Join(tmp, "tasks")
}

// EngineTasksDir returns the directory where the supervisor stores the active engine task metadata files.
func EngineTasksDir() string {
	tmp, err := LocalAppData()
	if err != nil {
		return ""
	}
	return filepath.Join(tmp, "engine_tasks")
}

// UnixSocketsDir contains AF_UNIX socket files for Desktop daemons.
func UnixSocketsDir() string {
	tmp, err := LocalAppData()
	if err != nil {
		return ""
	}
	return filepath.Join(tmp, "run")
}

// VMLogDir is where we stream the VM logs to.
func LogsDir() string {
	tmp, err := LocalAppData()
	if err != nil {
		return ""
	}
	return filepath.Join(tmp, logsDirName)
}

// ExtensionsRoot returns the root for Desktop extensions.
func ExtensionsRoot() string {
	tmp, err := AppData()
	if err != nil {
		return ""
	}
	return filepath.Join(tmp, "extensions")
}

// VMDir is the path where the VM is stored.
func VMDir() (string, error) {
	return DockerDesktopProgramData(vmDirName)
}

// VMDefaultDiskDir is the default path to the VM disk dir (vms/0/data).
func VMDefaultDiskDir() (string, error) {
	return VMDir()
}

// DockerCLIPlugins returns the path to the docker cli plugins directory.
func DockerCLIPlugins() (string, error) {
	programFiles := os.Getenv("PROGRAMFILES")
	if programFiles == "" {
		return "", errors.New("unable to get 'PROGRAMFILES'")
	}
	return filepath.Join(programFiles, "Docker", "cli-plugins"), nil
}

// WSL returns the path to the WSL executable.
func WSL() (string, error) {
	if wslpath := os.Getenv("E2E_TEST_WSL_PATH"); wslpath != "" {
		return wslpath, nil
	}
	windir := os.Getenv("WINDIR")
	if windir == "" {
		return "", errors.New("unable to get 'WINDIR'")
	}
	return filepath.Join(windir, "System32", "wsl.exe"), nil
}

// SC returns the path to the sc executable.
func SC() (string, error) {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		return "", errors.New("unable to get 'WINDIR'")
	}
	return filepath.Join(windir, "System32", "sc.exe"), nil
}

func Macaddr() string {
	return ""
}
