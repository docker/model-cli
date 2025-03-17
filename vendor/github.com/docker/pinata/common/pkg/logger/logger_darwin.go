package logger

import (
	"os"
	"os/user"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func platformSpecificCustomisation(res *Logger, component string) {
	// If we're running as root (i.e. we were started from launchd),
	// rely on ASL for logging.
	if os.Geteuid() < 1 {
		// root would be 0
		considerAddingASLHook(res)
		rootMustUseASLOnly()
		return
	}

	// If we're running as a regular user, write the logs directly
	// to files.
	if fileLogger, err := NewLocalFileHook(component); err == nil {
		res.AddHook(fileLogger)
	} else {
		log.Warnf("failed to create file hook: %v", err)
	}
}

// LogrusASLHook defines a hook for Logrus that redirects logs
// to ASL API (to be displayed in Console application)
type LogrusASLHook struct{}

// NewLogrusASLHook returns a new LogrusASLHook
func NewLogrusASLHook() *LogrusASLHook {
	return new(LogrusASLHook)
}

// Levels returns the available ASL log levels
func (t *LogrusASLHook) Levels() []log.Level {
	return []log.Level{
		log.PanicLevel,
		log.FatalLevel,
		log.ErrorLevel,
		log.WarnLevel,
		log.InfoLevel,
		log.DebugLevel,
	}
}

// Since the paths package itself uses logging, we need to make the log package
// independent (otherwise where would the logs go?)

func hostLogDir() (string, error) {
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		return filepath.Join(devhome, "Data", "log", "host"), nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, "Library", "Containers", "com.docker.docker", "Data", "log", "host"), nil
}
