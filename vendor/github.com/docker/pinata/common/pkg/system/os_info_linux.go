package system

import (
	"os"
	"runtime"
	"strings"
)

type PlatformSpecific struct{}

var osRelease = "/etc/os-release"

// os-release docs: https://man.archlinux.org/man/os-release.5.en
// os-release samples: https://github.com/chef/os_release
func getSystemInfo() (OsInfo, error) {
	lang, ok := os.LookupEnv("LANG")
	if !ok {
		lang = "unknown"
	}
	lang = strings.Split(strings.Split(lang, ".")[0], "_")[0]
	info := readOsReleaseFile(osRelease)
	return OsInfo{
		Name:         runtime.GOOS,
		ReleaseId:    info["VERSION_ID"],
		Version:      parseVersion(info["VERSION"]),
		Edition:      info["NAME"],
		BuildLabName: info["PRETTY_NAME"],
		Language:     LanguageInfo{ShortName: lang},
		Vendor:       info["ID"],
		VendorSuper:  info["ID_LIKE"],
	}, nil
}

func readOsReleaseFile(filename string) map[string]string {
	m := map[string]string{}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := strings.Trim(parts[1], `"`)
		m[key] = value
	}
	return m
}
