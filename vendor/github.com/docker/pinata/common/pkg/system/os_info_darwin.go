package system

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/docker/pinata/common/pkg/logger"
	"golang.org/x/mod/semver"
)

const (
	virtualizationFrameworkSupported         = "11.1"
	virtualizationFrameworkEnabledByDefault  = "12.5"
	virtualizationFrameworkVirtioFSSupported = "12.5"
	virtualizationFrameworkRosettaSupported  = "13.0"
	virtualizationRosettaEnabledByDefault    = "14.1"
)

var log = logger.Default.WithComponent("system")

type PlatformSpecific struct {
	IsVirtualizationSupported               bool
	IsVirtualizationEnabledByDefault        bool
	IsVirtualizationVirtioFSSupported       bool
	IsVirtualizationRosettaSupported        bool
	IsVirtualizationRosettaEnabledByDefault bool
}

func getSystemInfo() (OsInfo, error) {
	versions, err := exec.Command("sw_vers").Output()
	if err != nil {
		return OsInfo{}, fmt.Errorf("retrieving system version: %w", err)
	}
	name := parseProductName(versions)
	v := parseProductVersion(versions)
	build := parseBuildVersion(versions)

	lang, err := exec.Command("defaults", "read", "-g", "AppleLocale").Output()
	language := string(lang)
	if err != nil {
		// better not make this fatal
		log.Warn(fmt.Errorf("retrieving language: %w", err))
		language = "unknown"
	}

	version := parseVersion(strings.TrimSpace(string(v)))
	osInfo := OsInfo{
		Name:        strings.ToLower(strings.TrimSpace(string(name))),
		BuildNumber: string(build),
		Language:    LanguageInfo{ShortName: strings.Split(strings.TrimSpace(language), "_")[0]},
		Version:     version,
		PlatformSpecific: PlatformSpecific{
			IsVirtualizationSupported:         semver.Compare("v"+version.String(), "v"+virtualizationFrameworkSupported) >= 0,
			IsVirtualizationEnabledByDefault:  semver.Compare("v"+version.String(), "v"+virtualizationFrameworkEnabledByDefault) >= 0,
			IsVirtualizationVirtioFSSupported: semver.Compare("v"+version.String(), "v"+virtualizationFrameworkVirtioFSSupported) >= 0,
			IsVirtualizationRosettaSupported:  semver.Compare("v"+version.String(), "v"+virtualizationFrameworkRosettaSupported) >= 0 && runtime.GOARCH == "arm64",
		},
	}
	// assuming virtualizationFrameworkRosettaSupported >= virtualizationFrameworkEnabledByDefault
	osInfo.PlatformSpecific.IsVirtualizationRosettaEnabledByDefault = osInfo.PlatformSpecific.IsVirtualizationRosettaSupported &&
		semver.Compare("v"+osInfo.Version.String(), "v"+virtualizationRosettaEnabledByDefault) >= 0
	return osInfo, nil
}

func parseProductName(versions []byte) []byte {
	matches := regexp.MustCompile(`ProductName:\W*(.*)`).FindSubmatch(versions)
	if len(matches) != 2 {
		return []byte("unknown")
	}
	return matches[1]
}

func parseProductVersion(versions []byte) []byte {
	matches := regexp.MustCompile(`ProductVersion:\W*(.*)`).FindSubmatch(versions)
	if len(matches) != 2 {
		return []byte("unknown")
	}
	return matches[1]
}

func parseBuildVersion(versions []byte) []byte {
	matches := regexp.MustCompile(`BuildVersion:\W*(.*)`).FindSubmatch(versions)
	if len(matches) != 2 {
		return []byte("unknown")
	}
	return matches[1]
}
