package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/model-distribution/types"
)

// Unpack creates and return a Bundle by unpacking files and config from model into dir.
func Unpack(dir string, model types.Model) (*Bundle, error) {
	bundle := &Bundle{
		dir: dir,
	}
	if err := unpackGGUFs(bundle, model); err != nil {
		return nil, fmt.Errorf("add GGUF file(s) to runtime bundle: %w", err)
	}
	if err := unpackMultiModalProjector(bundle, model); err != nil {
		return nil, fmt.Errorf("add multi-model projector file to runtime bundle: %w", err)
	}
	if err := unpackTemplate(bundle, model); err != nil {
		return nil, fmt.Errorf("add chat template file to runtime bundle: %w", err)
	}
	if err := unpackRuntimeConfig(bundle, model); err != nil {
		return nil, fmt.Errorf("add config.json to runtime bundle: %w", err)
	}
	return bundle, nil
}

func unpackRuntimeConfig(bundle *Bundle, mdl types.Model) error {
	cfg, err := mdl.Config()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(bundle.dir, "config.json"))
	if err != nil {
		return fmt.Errorf("create runtime config file: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encode runtime config: %w", err)
	}
	bundle.runtimeConfig = cfg
	return nil
}

func unpackGGUFs(bundle *Bundle, mdl types.Model) error {
	ggufPaths, err := mdl.GGUFPaths()
	if err != nil {
		return fmt.Errorf("get GGUF files for model: %w", err)
	}

	if len(ggufPaths) == 1 {
		if err := unpackFile(filepath.Join(bundle.dir, "model.gguf"), ggufPaths[0]); err != nil {
			return err
		}
		bundle.ggufFile = "model.gguf"
		return err
	}

	for i := range ggufPaths {
		name := fmt.Sprintf("model-%05d-of-%05d.gguf", i+1, len(ggufPaths))
		if err := unpackFile(filepath.Join(bundle.dir, name), ggufPaths[i]); err != nil {
			return err
		}
		bundle.ggufFile = name
	}

	return nil
}

func unpackMultiModalProjector(bundle *Bundle, mdl types.Model) error {
	path, err := mdl.MMPROJPath()
	if err != nil {
		return nil // no such file
	}
	if err = unpackFile(filepath.Join(bundle.dir, "model.mmproj"), path); err != nil {
		return err
	}
	bundle.mmprojPath = "model.mmproj"
	return nil
}

func unpackTemplate(bundle *Bundle, mdl types.Model) error {
	path, err := mdl.ChatTemplatePath()
	if err != nil {
		return nil // no such file
	}
	if err = unpackFile(filepath.Join(bundle.dir, "template.jinja"), path); err != nil {
		return err
	}
	bundle.chatTemplatePath = "template.jinja"
	return nil
}

func unpackFile(bundlePath string, srcPath string) error {
	return os.Link(srcPath, bundlePath)
}
