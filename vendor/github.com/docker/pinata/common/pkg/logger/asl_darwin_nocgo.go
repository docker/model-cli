//go:build darwin && !cgo

package logger

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Fire sends a log entry to ASL. Or at least it would if we had any CGO
func (t *LogrusASLHook) Fire(entry *log.Entry) error {
	return errors.New("ASL logging requires CGO")
}
