package logging

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Logger is a bridging interface between logrus and Docker Desktop's internal
// logging types.
type Logger interface {
	logrus.FieldLogger
	Writer() *io.PipeWriter
}
