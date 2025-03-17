package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
)

var (
	doubleClose      = regexp.MustCompile("close.*: use of closed network connection")
	disableLogToFile atomic.Bool
)

// Default is the default logger
var Default = Make()

// Logger is the type of Pinata loggers.
type Logger struct {
	logrus.Logger
}

type ComponentLogger interface {
	Writer() *io.PipeWriter
	WriterLevel(level logrus.Level) *io.PipeWriter
	WithField(key string, value any) *logrus.Entry
	WithFields(fields logrus.Fields) *logrus.Entry
	WithError(err error) *logrus.Entry
	WithContext(ctx context.Context) *logrus.Entry
	WithTime(t time.Time) *logrus.Entry
	Logf(level logrus.Level, format string, args ...any)
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Printf(format string, args ...any)
	Warnf(format string, args ...any)
	Warningf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)
	Log(level logrus.Level, args ...any)
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Print(args ...any)
	Warn(args ...any)
	Warning(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)
	Logln(level logrus.Level, args ...any)
	Traceln(args ...any)
	Debugln(args ...any)
	Infoln(args ...any)
	Println(args ...any)
	Warnln(args ...any)
	Warningln(args ...any)
	Errorln(args ...any)
	Fatalln(args ...any)
	Panicln(args ...any)
}

func (log *Logger) WithComponent(name string) ComponentLogger {
	return log.WithField(ComponentKey, name)
}

// Logrus is the unwrapped logrus logger.
func (log *Logger) Logrus() *logrus.Logger {
	return &log.Logger
}

// SetOutput changes the output.
func (log *Logger) SetOutput(out io.Writer) {
	log.Logger.SetOutput(out)
}

// Close closes and logs errors.  Useful for defer clauses.
func Close(log ComponentLogger, c io.Closer) {
	MaybeWarn(log, c.Close())
}

// MaybeWarn logs an error if `err` is not nil.
func MaybeWarn(log ComponentLogger, err error) {
	if err != nil && !IsClosed(err) {
		log.Warnf("ignored error: %v", err)
	}
}

// Close closes and logs errors.  Useful for defer clauses.
func (log *Logger) Close(c io.Closer) {
	log.MaybeWarn(c.Close())
}

// MaybeWarn logs an error if `err` is not nil.
func (log *Logger) MaybeWarn(err error) {
	if err != nil && !IsClosed(err) {
		log.Warnf("ignored error: %v", err)
	}
}

// IsClosed checks whether err is about closing something already
// closed.
func IsClosed(err error) bool {
	if e, ok := err.(*os.PathError); ok {
		return e.Err == os.ErrClosed
	}
	// There are also net/http errors like
	// close unix ->vms/0/00000003.00000948: use of closed network connection
	return doubleClose.MatchString(err.Error())
}

// Make a logger with default options for the platform.
func Make() *Logger {
	logger := &Logger{Logger: *logrus.New()}
	logger.SetLevel(logrus.InfoLevel)
	component := filepath.Base(os.Args[0])
	platformSpecificCustomisation(logger, component)
	logger.Formatter = NewDesktopFormatter(component, false, DateFormat)
	logger.AddHook(otellogrus.NewHook(otellogrus.WithLevels(logrus.AllLevels...)))
	return logger
}

func MakeFileOnly(dir, componentName string) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.AddHook(NewLogFileHook(dir, componentName))
	return logger
}

// StopFileLogging deactivates logging.
// File handlers need one more log call to be released.
func StopFileLogging() {
	disableLogToFile.Store(true)
}

// StartFileLogging reactivates logging after a stop.
func StartFileLogging() {
	disableLogToFile.Store(false)
}
