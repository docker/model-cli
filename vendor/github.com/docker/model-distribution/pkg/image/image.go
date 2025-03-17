package image

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// CreateImage creates a new OCI image with the given layer
func CreateImage(ggufLayer v1.Layer) (v1.Image, error) {
	// Create empty image with artifact configuration
	img := empty.Image

	configFile := &v1.ConfigFile{
		Architecture: "unknown",
		OS:           "unknown",
		Config:       v1.Config{},
	}

	var err error
	img, err = mutate.ConfigFile(img, configFile)
	if err != nil {
		return nil, err
	}

	// Set up artifact manifest according to OCI spec
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, "application/vnd.docker.ai.model.config.v1+json")

	// Append layer to image
	img, err = mutate.AppendLayers(img, ggufLayer)
	if err != nil {
		return nil, err
	}

	return img, nil
}
