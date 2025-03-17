//go:build darwin && !cgo

package logger

import (
	"log"
)

func considerAddingASLHook(_ *Logger) {}

func rootMustUseASLOnly() {
	log.Fatal("must use ASL logging (which requires CGO) if running as root")
}
