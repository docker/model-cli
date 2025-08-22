package types

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Model interface {
	ID() (string, error)
	GGUFPaths() ([]string, error)
	MMPROJPath() (string, error)
	Config() (Config, error)
	Tags() []string
	Descriptor() (Descriptor, error)
}

type ModelArtifact interface {
	ID() (string, error)
	Config() (Config, error)
	Descriptor() (Descriptor, error)
	v1.Image
}

type ModelBundle interface {
	RootDir() string
	GGUFPath() string
	MMPROJPath() string
	RuntimeConfig() Config
}
