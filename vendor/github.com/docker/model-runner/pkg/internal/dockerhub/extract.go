package dockerhub

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/model-runner/pkg/internal/archive"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ManifestNotFoundError struct {
	os       string
	platform string
}

func (r *ManifestNotFoundError) Error() string {
	return fmt.Sprintf("unable to find manifest for %s/%s", r.os, r.platform)
}

func newManifestNotFoundError(os, platform string) *ManifestNotFoundError {
	return &ManifestNotFoundError{
		os:       os,
		platform: platform,
	}
}

// Extract all files from a `docker save` tarFile matching a given architecture
// and OS to destination. Note this doesn't handle files which have been deleted
// in layers.
func Extract(tarFile, architecture, OS, destination string) error {
	tmpDir, err := os.MkdirTemp("", "docker-tar-extract")
	if err != nil {
		return fmt.Errorf("creating temp directory for untar: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := unTarFile(tarFile, tmpDir); err != nil {
		return fmt.Errorf("untaring %s: %w", tarFile, err)
	}
	return extract(tmpDir, architecture, OS, destination)
}

func unTarFile(tarFile, destinationFolder string) error {
	from, err := os.Open(tarFile)
	if err != nil {
		return fmt.Errorf("opening tar file %s: %w", tarFile, err)
	}
	defer from.Close()
	return unTar(from, destinationFolder)
}

func unTar(from io.Reader, destinationFolder string) error {
	tarReader := tar.NewReader(from)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		path, err := archive.CheckRelative(destinationFolder, header.Name)
		if err != nil {
			return err
		}
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			if err := archive.CheckSymlink(destinationFolder, header.Name, header.Linkname); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, path); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, tarReader); err != nil {
			_ = file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}
	return nil
}

func extract(dir, architecture, OS, destination string) error {
	indexJSON, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return fmt.Errorf("reading %s/index.json: %w", dir, err)
	}
	var index v1.Index
	if err := json.Unmarshal(indexJSON, &index); err != nil {
		return fmt.Errorf("unmarshalling index: %w", err)
	}
	// Assume only one manifest for now
	if len(index.Manifests) != 1 {
		return errors.New("expected exactly one image manifest")
	}
	digest := index.Manifests[0].Digest
	bs, err := readBlob(dir, digest)
	if err != nil {
		return fmt.Errorf("reading blob %s: %w", digest.String(), err)
	}
	mtype := index.Manifests[0].MediaType
	// is it a manifest or a manifest list?
	if mtype == mediaTypeManifest {
		// not a multi-arch image
		return extractFromDigest(dir, digest, destination)
	}
	if mtype == mediaTypeOCI {
		return extractFromOCI(dir, digest, destination, OS, architecture)
	}
	if mtype != mediaTypeManifestList {
		return fmt.Errorf("unknown mediaType in manifest: %s", mtype)
	}
	// multi-arch image so look up the Architecture and OS
	var manifestList v1.Index
	if err := json.Unmarshal(bs, &manifestList); err != nil {
		return fmt.Errorf("unmarshalling manifest list: %w", err)
	}
	for _, m := range manifestList.Manifests {
		if m.Platform.Architecture != architecture {
			continue
		}
		if m.Platform.OS != OS {
			continue
		}
		return extractFromDigest(dir, m.Digest, destination)
	}

	return newManifestNotFoundError(OS, architecture)
}

const (
	mediaTypeManifest     = "application/vnd.docker.distribution.manifest.v2+json"
	mediaTypeManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
	mediaTypeOCI          = "application/vnd.oci.image.index.v1+json"
	mediaTypeOCIManifest  = "application/vnd.oci.image.manifest.v1+json"
	mediaTypeLayer        = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	mediaTypeOCILayer     = "application/vnd.oci.image.layer.v1.tar+gzip"
)

func blobPath(dir string, digest digest.Digest) string {
	return filepath.Join(dir, "blobs", digest.Algorithm().String(), digest.Hex())
}

func readBlob(dir string, digest digest.Digest) ([]byte, error) {
	return os.ReadFile(blobPath(dir, digest))
}

func extractFromOCI(dir string, digest digest.Digest, destination, OS, architecture string) error {
	bs, err := readBlob(dir, digest)
	if err != nil {
		return fmt.Errorf("extracting digest %s: %w", digest.String(), err)
	}
	var index v1.Index
	if err := json.Unmarshal(bs, &index); err != nil {
		return fmt.Errorf("unmarshalling index: %w", err)
	}
	for _, manifest := range index.Manifests {
		if manifest.MediaType == mediaTypeOCIManifest && manifest.Platform.OS == OS && manifest.Platform.Architecture == architecture {
			return extractFromDigest(dir, manifest.Digest, destination)
		}
	}
	return newManifestNotFoundError(OS, architecture)
}

func extractFromDigest(dir string, digest digest.Digest, destination string) error {
	bs, err := readBlob(dir, digest)
	if err != nil {
		return fmt.Errorf("extracting digest %s: %w", digest.String(), err)
	}
	var manifest v1.Manifest
	if err := json.Unmarshal(bs, &manifest); err != nil {
		return fmt.Errorf("unmarshalling manifest: %w", err)
	}
	for _, layer := range manifest.Layers {
		if err := extractLayer(dir, layer, destination); err != nil {
			return fmt.Errorf("extracting layer %s: %w", layer.Digest.String(), err)
		}
	}
	return nil
}

func extractLayer(dir string, layer v1.Descriptor, destination string) error {
	fmt.Printf("descriptor %s has media type %s\n", layer.Digest.String(), layer.MediaType)
	if layer.MediaType != mediaTypeLayer && layer.MediaType != mediaTypeOCILayer {
		return fmt.Errorf("expected layer %s to have media type %s or %s, received %s", layer.Digest.String(), mediaTypeLayer, mediaTypeOCILayer, layer.MediaType)
	}
	f, err := os.Open(blobPath(dir, layer.Digest))
	if err != nil {
		return fmt.Errorf("reading blob %s: %w", layer.Digest.String(), err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("decompressing %s: %w", layer.Digest.String(), err)
	}
	defer gz.Close()
	return unTar(gz, destination)
}
