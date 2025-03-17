package logger

import (
	"os"
	"os/user"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func platformSpecificCustomisation(res *Logger, component string) {
	// We only log to %HOME/.docker/desktop/log/host in code which runs on the DD for Linux host
	if _, err := os.Stat("/etc/docker-desktop-vm"); err == nil {
		return
	}

	fileLogger, err := NewLocalFileHook(component)
	if err == nil {
		res.AddHook(fileLogger)
	} else {
		log.Warnf("failed to create file hook: %v", err)
	}
}

func hostLogDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".docker", "desktop", "log", "host"), nil
}
