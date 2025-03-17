//go:build darwin && cgo

package logger

import "io"

func considerAddingASLHook(res *Logger) {
	res.AddHook(NewLogrusASLHook())
	// In addition to the default destination, our logs are sent
	// to ASL via the previous hook.  But since our stderr is also
	// redirected to ASL, each entry would appear twice: don't
	// send logs to stderr.
	res.Out = io.Discard
}

func rootMustUseASLOnly() {}
