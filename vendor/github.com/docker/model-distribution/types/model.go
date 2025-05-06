package types

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type Model interface {
	ID() (string, error)
	GGUFPath() (string, error)
	Config() (Config, error)
	Tags() []string
	Descriptor() (Descriptor, error)
}

type ModelArtifact interface {
	Model
	v1.Image
}
