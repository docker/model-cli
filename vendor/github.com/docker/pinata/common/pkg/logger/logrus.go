package logger

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type logrusHook struct {
	formatter logrus.Formatter
	logF      *LogFile
	m         sync.Mutex
	logDir    string
	logName   string
}

// NewLocalFileHook returns a hook for Logrus that redirects logs to a file with automatic rotation.
func NewLocalFileHook(component string) (logrus.Hook, error) {
	return &logrusHook{
		formatter: NewDesktopFormatter(component, false, DateFormat),
	}, nil
}

func NewLogFileHook(dir, name string) logrus.Hook {
	return &logrusHook{
		formatter: NewDesktopFormatter("", false, DateFormat),
		logDir:    dir,
		logName:   name,
	}
}

func (h *logrusHook) open() error {
	if h.logF != nil {
		return nil
	}
	if h.logDir == "" {
		var err error
		h.logDir, err = hostLogDir()
		if err != nil {
			return errors.Wrap(err, "while creating logrus local file hook")
		}
	}
	if h.logName == "" {
		h.logName = filepath.Base(os.Args[0])
	}
	logF, err := Open(h.logDir, h.logName)
	if err != nil {
		return errors.Wrapf(err, "unable to create a log file for %s in directory %s", h.logName, h.logDir)
	}
	h.logF = logF
	return nil
}

// Levels returns the available log levels
func (h *logrusHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

// Fire writes a log entry
func (h *logrusHook) Fire(entry *logrus.Entry) error {
	h.m.Lock()
	defer h.m.Unlock()

	if disableLogToFile.Load() {
		if h.logF != nil {
			_ = h.logF.Close()
			h.logF = nil
		}
		return nil
	}
	// Opening the file is delayed until first use to prevent the bugsnag parent process
	// from opening a file handle and breaking log rotation.
	if err := h.open(); err != nil {
		return err
	}

	b, err := h.formatter.Format(entry)
	if err != nil {
		return errors.Wrap(err, "unable to convert a logrus.Entry to a string")
	}
	_, err = h.logF.Write(b)
	return err
}
