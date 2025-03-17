//go:build darwin && cgo

package logger

import (
	"github.com/docker/pinata/common/pkg/obfuscate"
	log "github.com/sirupsen/logrus"
)

/*
#cgo CFLAGS: -I .

#include <asl.h>
#include "asl_darwin.h"
*/
import "C"

func aslLevel(l log.Level) C.int {
	switch l {
	case log.PanicLevel:
		return C.ASL_LEVEL_ALERT
	case log.FatalLevel:
		return C.ASL_LEVEL_CRIT
	case log.ErrorLevel:
		return C.ASL_LEVEL_ERR
	case log.WarnLevel:
		return C.ASL_LEVEL_WARNING
	case log.InfoLevel:
		return C.ASL_LEVEL_NOTICE
	case log.DebugLevel:
		return C.ASL_LEVEL_DEBUG
	}
	return C.ASL_LEVEL_DEBUG
}

// Fire sends a log entry to ASL
func (t *LogrusASLHook) Fire(entry *log.Entry) error {
	C.apple_asl_logger_log(aslLevel(entry.Level), C.CString(obfuscate.ObfuscateString(entry.Message)))
	return nil
}
