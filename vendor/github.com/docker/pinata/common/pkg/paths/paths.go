package paths

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/docker/pinata/common/pkg/logger"
	"github.com/docker/pinata/common/pkg/system"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	log                      = logger.Default.WithComponent("paths")
	location                 atomic.Int32
	integrationTestsRootPath string
	backupRootPath           string
)

type Location int32

const (
	Undefined Location = iota
	OnHost
	InsideLinuxkit
	InsideWslWorkspace
	IntegrationTests
	logsDirName    = "log"
	hostLogDirName = "host"
	vmLogDirName   = "vm"
)

func (l Location) String() string {
	switch l {
	case OnHost:
		return "OnHost"
	case InsideLinuxkit:
		return "InsideLinuxkit"
	case InsideWslWorkspace:
		return "InsideWslWorkspace"
	case IntegrationTests:
		return "IntegrationTests"
	default:
		return "Unknown"
	}
}

// Init must be called before any path is computed or else the code will deliberately panic().
// In particular this means paths cannot be computed in init().
// TODO: consider replacing this with a configuration file.
func Init(l Location) {
	SetLocation(l)
	if err := setCurrentDirectory(); err != nil {
		log.Fatal(err)
	}
}

// SetLocation configures the paths module but does not change the current directory so long
// usernames will not work. This is only exported for use by the e2e tests.
func SetLocation(l Location) {
	_ = location.CompareAndSwap(int32(Undefined), int32(l))
}

// ResetLocationForTests is used only for tests, to reset the location as if it hasn't been set.
func ResetLocationForTests() {
	location.Store(int32(Undefined))
	_ = os.Chdir(backupRootPath)
}

// SetIntegrationTestsRootPath ia used only for tests, to set root dir for all paths during tests
func SetIntegrationTestsRootPath(path string) {
	backupRootPath, _ = os.Getwd()
	var err error
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		panic(err)
	}
	integrationTestsRootPath = path
}

func setCurrentDirectory() error {
	l := whereami()
	if l == InsideLinuxkit {
		if err := os.Chdir("/"); err != nil {
			return errors.Wrapf(err, "chdir /")
		}
		return nil
	}
	if l == IntegrationTests {
		if integrationTestsRootPath == "" {
			panic("integration tests root path must be set before calling paths.Init")
		}
		return os.Chdir(integrationTestsRootPath)
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	// Set the current directory to allow the use of shortened relative socket paths on Mac and Linux.
	dir := Data()
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errors.Wrapf(err, "creating %s", dir)
	}
	if err := os.Chdir(dir); err != nil {
		return errors.Wrapf(err, "chdir %s", dir)
	}
	return nil
}

func whereami() Location {
	l := Location(location.Load())
	if l == Undefined {
		panic("paths have been accessed before the paths.Init(...) has been called. Decide which paths you want, and call paths.Init() from main()")
		// The stack trace will highlight the problem which we can then easily fix.
	}
	return l
}

func isDarwin() bool {
	return runtime.GOOS == "darwin"
}

func isLinux() bool {
	return runtime.GOOS == "linux"
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

// Home returns the user home folder path.
var homeDir = sync.OnceValue(func() string {
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		return devhome
	}
	if homeDir, ok := os.LookupEnv("HOME"); ok {
		return homeDir
	}
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
})

func Home(path ...string) string {
	if IsIntegrationTests() {
		return filepath.Join(append([]string{integrationTestsRootPath}, path...)...)
	}
	return filepath.Join(append([]string{homeDir()}, path...)...)
}

func IsIntegrationTests() bool {
	return whereami() == IntegrationTests
}

func IsInsideVM() bool {
	l := whereami()
	return l != OnHost && l != IntegrationTests
}

// VMLogDir is where we stream the VM logs to.
func VMLogDir() string {
	return filepath.Join(LogsDir(), vmLogDirName)
}

// HostLogDir is where we stream the host logs to.
func HostLogDir() string {
	return filepath.Join(LogsDir(), hostLogDirName)
}

// ConsoleLog is the path to the `console.log` file written by com.docker.virtualization.
func ConsoleLog() string {
	return filepath.Join(VMLogDir(), "console.log")
}

// DockerHome returns ~/.docker
func DockerHome(path ...string) string {
	return Home(append([]string{".docker"}, path...)...)
}

// Certd returns cert.d path
func Certd(path ...string) string {
	return DockerHome(append([]string{"certs.d"}, path...)...)
}

// UserID returns the UUID for the user (used for diagnostics and telemetry). Returns an empty string on error.
func UserID() string {
	fp := UserPaths.UserIDFile()
	if fp == "" {
		return "anonymousID"
	}
	outB, err := os.ReadFile(fp)
	if err != nil {
		// Create the ID if it doesn't exist.
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			return ""
		}
		id := uuid.NewString()
		if err := os.WriteFile(fp, []byte(id+"\n"), 0o644); err != nil {
			return ""
		}
		return id
	}
	outS := string(outB)
	tmp := strings.Split(outS, "\n")
	return strings.TrimSpace(tmp[0])
}

// LinuxKitMutagenDataStorage returns the path to the Mutagen data directory
// within the VM.
func LinuxKitMutagenDataStorage() string {
	return "/var/lib/mutagen/data"
}

// LinuxKitMutagenFileShareStorage returns the path to the Mutagen file share
// storage directory within the VM.
func LinuxKitMutagenFileShareStorage() string {
	return "/var/lib/mutagen/file-shares"
}

// LinuxKitMutagenFileShareStorageSelfownerMarkMount returns the path to the
// selfowner mark mount point for the Mutagen file share storage directory
// within the VM.
func LinuxKitMutagenFileShareStorageSelfownerMarkMount() string {
	return "/run/mutagen-file-shares-mark"
}

// LinuxKitMutagenFileShareStorageSelfownerMount returns the path to the
// selfowner mount point for the Mutagen file share storage directory within the
// VM.
func LinuxKitMutagenFileShareStorageSelfownerMount() string {
	return "/run/mutagen-file-shares"
}

// LinuxKitTelemetryDir is the path within the VM to the directory
// holding telemetry information.
//
// Currently only used for WSL2 trace context propagation. However we need to
// create the (empty) dir in all platforms to simplify setup of bind mounts
func LinuxKitTelemetryDir() string {
	return "/run/telemetry"
}

// LinuxKitWSLStartupTraceContextCmdline is the path within the WSL2 VM
// to the file that is used to propagate the startup trace context.
//
// Currently only used for WSL2 trace context propagation.
func LinuxKitWSLStartupTraceContextCmdline() string {
	return filepath.Join(LinuxKitTelemetryDir(), "startup_tracecontext")
}

// LinuxKitStartupTraceContextCmdline is the path within the VM
// to the file that is used to propagate the startup trace context.
//
// For platforms other than WSL2 and HyperV this is the kernel cmdline.
// For WSL2 see [LinuxKitWSLStartupTraceContextCmdline]. Not supported yet in HyperV
func LinuxKitStartupTraceContextCmdline() string {
	return "/proc/cmdline"
}

// DockerCLIConfig is the path to the CLI config file.
func DockerCLIConfig() string {
	return DockerHome("config.json")
}

// PlaintextPasswordsFile is where the credential helper stores the plaintext passwords.
// This cannot be the CLI config because the CLI will erase the entry and write a bare URL
// because it expects the credential helper to store the passwords elsewhere.
func PlaintextPasswordsFile() string {
	return DockerHome("plaintext-passwords.json")
}

// DevRoot holds the root folder of the project if running a locally built application.
var DevRoot = func() string {
	pathBinary, err := executable()
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn(err)
		}
		return ""
	}
	pinataSources := findFirstInParent("pinata.sh")
	if strings.HasPrefix(pathBinary, pinataSources+string(filepath.Separator)) {
		return pinataSources
	}
	return ""
}()

func executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func findFirstInParent(name string) string {
	cd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if _, err := os.Stat(filepath.Join(cd, name)); err == nil {
		return cd
	}
	exe, err := executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	for {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if dir == parent {
			return ""
		}
		dir = parent
	}
}

func LastOSVersion() (string, func(string)) {
	if !isDarwin() {
		return "", func(string) {}
	}
	lastOSVersionFile := UserPaths.GroupDir(".os_version")
	updateLastOSVersion := func(lastOSVersion string) {
		systemInfo, err := system.GetSystemInfo()
		if err != nil {
			log.Warnln("getting system info:", err)
			_ = os.Remove(lastOSVersionFile)
			return
		}
		if lastOSVersion != systemInfo.Version.String() {
			if err := os.WriteFile(lastOSVersionFile, []byte(systemInfo.Version.String()+"\n"), 0o644); err != nil {
				log.Warnln("writing "+lastOSVersionFile+":", err)
				_ = os.Remove(lastOSVersionFile)
			}
		}
	}
	lastOSVersion, err := os.ReadFile(lastOSVersionFile)
	if err != nil {
		return "", updateLastOSVersion
	}
	return strings.TrimSpace(string(lastOSVersion)), updateLastOSVersion
}

func ToUNCPath(path string) string {
	if !isWindows() {
		return path
	}

	volumeName := filepath.VolumeName(path)
	if strings.HasPrefix(volumeName, `\\`) {
		// Already a UNC path.
		return path
	} else if strings.HasSuffix(volumeName, ":") {
		// Windows drive.
		// TODO(jsternberg): This is a hack to enable the docker/cli.
		// Fixed in https://github.com/docker/cli/pull/5445.
		// We send the volume to lowercase because Windows doesn't care and
		// the logic to convert the path to a WSL path cares.
		// This can be removed when the CLI and all CLI plugins contain the above fix.
		return `\\.\` + strings.ToLower(volumeName) + path[len(volumeName):]
	}
	return path
}
