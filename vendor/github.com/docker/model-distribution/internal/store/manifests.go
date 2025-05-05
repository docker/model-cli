package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/v1"
)

const (
	manifestsDir = "manifests"
)

// manifestPath returns the path to the manifest file for the given hash.
func (s *LocalStore) manifestPath(hash v1.Hash) string {
	return filepath.Join(s.rootPath, manifestsDir, hash.Algorithm, hash.Hex)
}

// writeManifest writes the model's manifest to the store
func (s *LocalStore) writeManifest(mdl v1.Image) error {
	digest, err := mdl.Digest()
	if err != nil {
		return fmt.Errorf("get digest: %w", err)
	}
	rm, err := mdl.RawManifest()
	if err != nil {
		return fmt.Errorf("get raw manifest: %w", err)
	}
	return writeFile(s.manifestPath(digest), rm)
}

// removeManifest removes the manifest file from the store
func (s *LocalStore) removeManifest(hash v1.Hash) error {
	return os.Remove(s.manifestPath(hash))
}

// writeFile is a wrapper around os.WriteFile that creates any parent directories as needed.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return fmt.Errorf("create parent directory %q: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, data, 0666)
}
