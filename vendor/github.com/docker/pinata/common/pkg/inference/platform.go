package inference

import (
	"runtime"
)

// Supported returns whether or not the current host platform is supported for
// inference operations. This support should expand over time.
func Supported() bool {
	// This should mirror the logic in
	// client/desktop-ui/src/settings/features-control/BetaSettings.tsx.
	return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64"
}
