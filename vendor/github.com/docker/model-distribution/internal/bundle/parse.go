package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/model-distribution/types"
)

// Parse returns the Bundle at the given rootDir
func Parse(rootDir string) (*Bundle, error) {
	if fi, err := os.Stat(rootDir); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("inspect bundle root dir: %w", err)
	}
	ggufPath, err := findGGUFFile(rootDir)
	if err != nil {
		return nil, err
	}
	mmprojPath, err := findMultiModalProjectorFile(rootDir)
	if err != nil {
		return nil, err
	}
	templatePath, err := findChatTemplateFile(rootDir)
	if err != nil {
		return nil, err
	}
	cfg, err := parseRuntimeConfig(rootDir)
	if err != nil {
		return nil, err
	}
	return &Bundle{
		dir:              rootDir,
		mmprojPath:       mmprojPath,
		ggufFile:         ggufPath,
		runtimeConfig:    cfg,
		chatTemplatePath: templatePath,
	}, nil
}

func parseRuntimeConfig(rootDir string) (types.Config, error) {
	f, err := os.Open(filepath.Join(rootDir, "config.json"))
	if err != nil {
		return types.Config{}, fmt.Errorf("open runtime config: %w", err)
	}
	defer f.Close()
	var cfg types.Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return types.Config{}, fmt.Errorf("decode runtime config: %w", err)
	}
	return cfg, nil
}

func findGGUFFile(rootDir string) (string, error) {
	ggufs, err := filepath.Glob(filepath.Join(rootDir, "[^.]*.gguf"))
	if err != nil {
		return "", fmt.Errorf("find gguf files: %w", err)
	}
	if len(ggufs) == 0 {
		return "", fmt.Errorf("no GGUF files found in bundle directory")
	}
	return filepath.Base(ggufs[0]), nil
}

func findMultiModalProjectorFile(rootDir string) (string, error) {
	mmprojPaths, err := filepath.Glob(filepath.Join(rootDir, "[^.]*.mmproj"))
	if err != nil {
		return "", err
	}
	if len(mmprojPaths) == 0 {
		return "", nil
	}
	if len(mmprojPaths) > 1 {
		return "", fmt.Errorf("found multiple .mmproj files, but only 1 is supported")
	}
	return filepath.Base(mmprojPaths[0]), nil
}

func findChatTemplateFile(rootDir string) (string, error) {
	templatePaths, err := filepath.Glob(filepath.Join(rootDir, "[^.]*.jinja"))
	if err != nil {
		return "", err
	}
	if len(templatePaths) == 0 {
		return "", nil
	}
	if len(templatePaths) > 1 {
		return "", fmt.Errorf("found multiple template files, but only 1 is supported")
	}
	return filepath.Base(templatePaths[0]), nil
}
