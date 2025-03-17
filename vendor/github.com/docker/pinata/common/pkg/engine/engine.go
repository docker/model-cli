package engine

import (
	"os"
	"path/filepath"
)

type Engine int // Engine selects between Linux and Windows
const (
	Linux   Engine = iota // Linux containers
	Windows               // Windows containers
)

func (e Engine) String() string {
	switch e {
	case Linux:
		return "Linux"
	case Windows:
		return "Windows"
	default:
		return "unknown engine"
	}
}

// Other engine.
func (e Engine) Other() Engine {
	if e == Linux {
		return Windows
	}
	return Linux
}

// CliContextName returns the name of the CLI context associated with the given engine.
func (e Engine) CliContextName() string {
	switch e {
	case Linux:
		if devhome, ok := os.LookupEnv("DEVHOME"); ok {
			devhome = filepath.Base(devhome)
			return "desktop-linux-" + devhome
		}
		return "desktop-linux"
	case Windows:
		return "desktop-windows"
	}
	return ""
}
