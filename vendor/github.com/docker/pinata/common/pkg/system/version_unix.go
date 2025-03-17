//go:build !windows

package system

import (
	"strings"
)

func parseVersion(v string) Version {
	version := Version{
		Major: "0",
		Minor: "0",
		Patch: "0",
	}
	parts := strings.Split(v, ".")
	version.Major = parts[0]
	if len(parts) >= 2 {
		version.Minor = parts[1]
	}
	if len(parts) >= 3 {
		version.Patch = parts[2]
	}
	return version
}
