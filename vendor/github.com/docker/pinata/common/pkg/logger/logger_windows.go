package logger

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func platformSpecificCustomisation(res *Logger, component string) {
	fileLogger, err := NewLocalFileHook(component)
	if err == nil {
		res.AddHook(fileLogger)
	} else {
		log.Warnf("failed to create file hook: %v", err)
	}
}

func appLocalData() (string, error) {
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		return "", fmt.Errorf("unable to get 'LOCALAPPDATA'")
	}
	if devhome, ok := os.LookupEnv("DEVHOME"); ok {
		appData = filepath.Join(devhome, "Local")
	}
	dir := filepath.Join(appData, "Docker")
	return dir, nil
}

func hostLogDir() (string, error) {
	dataDir, err := appLocalData()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "log", "host"), nil
}
