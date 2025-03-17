package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/static"

	"github.com/docker/model-distribution/pkg/image"
	"github.com/docker/model-distribution/pkg/types"
)

const (
	// CurrentVersion is the current version of the store layout
	CurrentVersion = "1.0.0"
)

// LocalStore implements the Store interface for local storage
type LocalStore struct {
	rootPath string
}

// RootPath returns the root path of the store
func (s *LocalStore) RootPath() string {
	return s.rootPath
}

// New creates a new LocalStore
func New(opts types.StoreOptions) (*LocalStore, error) {
	store := &LocalStore{
		rootPath: opts.RootPath,
	}

	// Initialize store if it doesn't exist
	if err := store.initialize(); err != nil {
		return nil, fmt.Errorf("initializing store: %w", err)
	}

	return store, nil
}

// initialize creates the store directory structure if it doesn't exist
func (s *LocalStore) initialize() error {
	// Create root directory if it doesn't exist
	if err := os.MkdirAll(s.rootPath, 0755); err != nil {
		return fmt.Errorf("creating root directory: %w", err)
	}

	// Create blobs directory
	blobsDir := filepath.Join(s.rootPath, "blobs", "sha256")
	if err := os.MkdirAll(blobsDir, 0755); err != nil {
		return fmt.Errorf("creating blobs directory: %w", err)
	}

	// Create manifests directory
	manifestsDir := filepath.Join(s.rootPath, "manifests", "sha256")
	if err := os.MkdirAll(manifestsDir, 0755); err != nil {
		return fmt.Errorf("creating manifests directory: %w", err)
	}

	// Check if layout.json exists, create if not
	layoutPath := filepath.Join(s.rootPath, "layout.json")
	if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
		layout := types.StoreLayout{
			Version: CurrentVersion,
		}
		layoutData, err := json.MarshalIndent(layout, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling layout: %w", err)
		}
		if err := os.WriteFile(layoutPath, layoutData, 0644); err != nil {
			return fmt.Errorf("writing layout file: %w", err)
		}
	}

	// Check if models.json exists, create if not
	modelsPath := filepath.Join(s.rootPath, "models.json")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		models := types.ModelIndex{
			Models: []types.Model{},
		}
		modelsData, err := json.MarshalIndent(models, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling models: %w", err)
		}
		if err := os.WriteFile(modelsPath, modelsData, 0644); err != nil {
			return fmt.Errorf("writing models file: %w", err)
		}
	}

	return nil
}

// Push pushes a model to the store with the given tags
func (s *LocalStore) Push(modelPath string, tags []string) error {
	// Read model file
	modelContent, err := os.ReadFile(modelPath)
	if err != nil {
		return fmt.Errorf("reading model file: %w", err)
	}

	// Create layer from model content
	ggufLayer := static.NewLayer(modelContent, "application/vnd.docker.ai.model.file.v1+gguf")

	// Create image with layer
	img, err := image.CreateImage(ggufLayer)
	if err != nil {
		return fmt.Errorf("creating image: %w", err)
	}

	// Get manifest from image
	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("getting manifest: %w", err)
	}

	// Gets SHA256 digest
	digest := manifest.Layers[0].Digest
	digestHex := digest.Hex

	// Store the model blob
	blobPath := filepath.Join(s.rootPath, "blobs", "sha256", digestHex)
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		if err := os.WriteFile(blobPath, modelContent, 0644); err != nil {
			return fmt.Errorf("writing blob file: %w", err)
		}
	}

	// Marshal the manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	// Calculate manifest digest
	manifestDigest := sha256.Sum256(manifestData)
	manifestDigestHex := hex.EncodeToString(manifestDigest[:])

	// Store the manifest
	manifestPath := filepath.Join(s.rootPath, "manifests", "sha256", manifestDigestHex)
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("writing manifest file: %w", err)
	}

	// Update the models index with the layer digest
	if err := s.updateModelsIndex(manifestDigestHex, tags, digestHex); err != nil {
		return fmt.Errorf("updating models index: %w", err)
	}

	return nil
}

// updateModelsIndex updates the models index with a new model
func (s *LocalStore) updateModelsIndex(manifestDigest string, tags []string, blobDigest string) error {
	// Ensure the manifest digest has the correct format (sha256:...)
	if !strings.Contains(manifestDigest, ":") {
		manifestDigest = fmt.Sprintf("sha256:%s", manifestDigest)
	}

	// Read the models index
	modelsPath := filepath.Join(s.rootPath, "models.json")
	modelsData, err := os.ReadFile(modelsPath)
	if err != nil {
		return fmt.Errorf("reading models file: %w", err)
	}

	// Unmarshal the models index
	var models types.ModelIndex
	if err := json.Unmarshal(modelsData, &models); err != nil {
		return fmt.Errorf("unmarshaling models: %w", err)
	}

	// Check if the model already exists
	var model *types.Model
	for i, m := range models.Models {
		if m.ID == manifestDigest {
			model = &models.Models[i]
			break
		}
	}

	if model == nil {
		// Model doesn't exist, add it
		models.Models = append(models.Models, types.Model{
			ID:      manifestDigest,
			Tags:    tags,
			Files:   []string{fmt.Sprintf("sha256:%s", blobDigest)},
			Created: time.Now().Unix(),
		})
	} else {
		// Model exists, update tags
		existingTags := make(map[string]bool)
		for _, tag := range model.Tags {
			existingTags[tag] = true
		}

		// Add new tags
		for _, tag := range tags {
			if !existingTags[tag] {
				model.Tags = append(model.Tags, tag)
				existingTags[tag] = true
			}
		}
	}

	// Marshal the models index
	modelsData, err = json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling models: %w", err)
	}

	// Write the models index
	if err := os.WriteFile(modelsPath, modelsData, 0644); err != nil {
		return fmt.Errorf("writing models file: %w", err)
	}

	return nil
}

// Pull pulls a model from the store by tag
func (s *LocalStore) Pull(tag string, destPath string) error {
	// Get the model by tag
	model, err := s.GetByTag(tag)
	if err != nil {
		return fmt.Errorf("getting model by tag: %w", err)
	}

	// Read the manifest
	manifestDigestParts := strings.Split(model.ID, ":")
	var algorithm, hash string

	if len(manifestDigestParts) == 2 {
		// Format is already "algorithm:hash"
		algorithm = manifestDigestParts[0]
		hash = manifestDigestParts[1]
	} else if len(manifestDigestParts) == 1 {
		// Format is just the hash, assume sha256
		algorithm = "sha256"
		hash = manifestDigestParts[0]
	} else {
		return fmt.Errorf("invalid manifest digest format: %s", model.ID)
	}

	manifestPath := filepath.Join(s.rootPath, "manifests", algorithm, hash)
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest file: %w", err)
	}

	// Unmarshal the manifest
	var manifest v1.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("unmarshaling manifest: %w", err)
	}

	// Get the layer
	if len(manifest.Layers) == 0 {
		return fmt.Errorf("no layers in manifest")
	}

	// Use the first layer (assuming there's only one for models)
	layer := manifest.Layers[0]
	layerDigest := layer.Digest.String()

	// Parse the layer digest
	layerDigestParts := strings.Split(layerDigest, ":")
	var layerAlgorithm, layerHash string

	if len(layerDigestParts) == 2 {
		// Format is already "algorithm:hash"
		layerAlgorithm = layerDigestParts[0]
		layerHash = layerDigestParts[1]
	} else if len(layerDigestParts) == 1 {
		// Format is just the hash, assume sha256
		layerAlgorithm = "sha256"
		layerHash = layerDigestParts[0]
	} else {
		return fmt.Errorf("invalid digest format: %s", layerDigest)
	}

	// Read the layer blob
	blobPath := filepath.Join(s.rootPath, "blobs", layerAlgorithm, layerHash)
	blobContent, err := os.ReadFile(blobPath)
	if err != nil {
		return fmt.Errorf("reading blob file: %w", err)
	}

	// Write the blob content to the destination file
	if err := os.WriteFile(destPath, blobContent, 0644); err != nil {
		return fmt.Errorf("writing to destination file: %w", err)
	}

	return nil
}

// List lists all models in the store
func (s *LocalStore) List() ([]types.Model, error) {
	// Read the models index
	modelsPath := filepath.Join(s.rootPath, "models.json")
	modelsData, err := os.ReadFile(modelsPath)
	if err != nil {
		return nil, fmt.Errorf("reading models file: %w", err)
	}

	// Unmarshal the models index
	var models types.ModelIndex
	if err := json.Unmarshal(modelsData, &models); err != nil {
		return nil, fmt.Errorf("unmarshaling models: %w", err)
	}

	return models.Models, nil
}

// GetByTag gets a model by tag
func (s *LocalStore) GetByTag(tag string) (*types.Model, error) {
	// Read the models index
	modelsPath := filepath.Join(s.rootPath, "models.json")
	modelsData, err := os.ReadFile(modelsPath)
	if err != nil {
		return nil, fmt.Errorf("reading models file: %w", err)
	}

	// Unmarshal the models index
	var models types.ModelIndex
	if err := json.Unmarshal(modelsData, &models); err != nil {
		return nil, fmt.Errorf("unmarshaling models: %w", err)
	}

	// Find the model by tag
	for _, model := range models.Models {
		for _, modelTag := range model.Tags {
			if modelTag == tag {
				return &model, nil
			}
		}
	}

	return nil, fmt.Errorf("model with tag %s not found", tag)
}

// GetBlobPath returns the direct path to the blob file for a model
func (s *LocalStore) GetBlobPath(tag string) (string, error) {
	// Get the model by tag
	model, err := s.GetByTag(tag)
	if err != nil {
		return "", fmt.Errorf("getting model by tag: %w", err)
	}

	// Read the manifest
	manifestDigestParts := strings.Split(model.ID, ":")
	var algorithm, hash string

	if len(manifestDigestParts) == 2 {
		// Format is already "algorithm:hash"
		algorithm = manifestDigestParts[0]
		hash = manifestDigestParts[1]
	} else if len(manifestDigestParts) == 1 {
		// Format is just the hash, assume sha256
		algorithm = "sha256"
		hash = manifestDigestParts[0]
	} else {
		return "", fmt.Errorf("invalid manifest digest format: %s", model.ID)
	}

	manifestPath := filepath.Join(s.rootPath, "manifests", algorithm, hash)
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("reading manifest file: %w", err)
	}

	// Unmarshal the manifest
	var manifest v1.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return "", fmt.Errorf("unmarshaling manifest: %w", err)
	}

	// Get the layer
	if len(manifest.Layers) == 0 {
		return "", fmt.Errorf("no layers in manifest")
	}

	// Use the first layer (assuming there's only one for models)
	layer := manifest.Layers[0]
	layerDigest := layer.Digest.String()

	// Parse the layer digest
	layerDigestParts := strings.Split(layerDigest, ":")
	var layerAlgorithm, layerHash string

	if len(layerDigestParts) == 2 {
		// Format is already "algorithm:hash"
		layerAlgorithm = layerDigestParts[0]
		layerHash = layerDigestParts[1]
	} else if len(layerDigestParts) == 1 {
		// Format is just the hash, assume sha256
		layerAlgorithm = "sha256"
		layerHash = layerDigestParts[0]
	} else {
		return "", fmt.Errorf("invalid digest format: %s", layerDigest)
	}

	// Return the path to the blob file
	return filepath.Join(s.rootPath, "blobs", layerAlgorithm, layerHash), nil
}

// Delete deletes a model by tag
func (s *LocalStore) Delete(tag string) error {
	// Read the models index
	modelsPath := filepath.Join(s.rootPath, "models.json")
	modelsData, err := os.ReadFile(modelsPath)
	if err != nil {
		return fmt.Errorf("reading models file: %w", err)
	}

	// Unmarshal the models index
	var models types.ModelIndex
	if err := json.Unmarshal(modelsData, &models); err != nil {
		return fmt.Errorf("unmarshaling models: %w", err)
	}

	// Find the model by tag
	var modelIndex = -1
	var tagIndex = -1
	for i, model := range models.Models {
		for j, modelTag := range model.Tags {
			if modelTag == tag {
				modelIndex = i
				tagIndex = j
				break
			}
		}
		if modelIndex != -1 {
			break
		}
	}

	if modelIndex == -1 {
		return fmt.Errorf("model with tag %s not found", tag)
	}

	// Remove the tag
	model := &models.Models[modelIndex]
	model.Tags = append(model.Tags[:tagIndex], model.Tags[tagIndex+1:]...)

	// If no more tags, remove the model
	if len(model.Tags) == 0 {
		models.Models = append(models.Models[:modelIndex], models.Models[modelIndex+1:]...)
	}

	// Marshal the models index
	modelsData, err = json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling models: %w", err)
	}

	// Write the models index
	if err := os.WriteFile(modelsPath, modelsData, 0644); err != nil {
		return fmt.Errorf("writing models file: %w", err)
	}

	return nil
}

// AddTags adds tags to an existing model
func (s *LocalStore) AddTags(tag string, newTags []string) error {
	// Get the model by tag
	model, err := s.GetByTag(tag)
	if err != nil {
		return fmt.Errorf("getting model by tag: %w", err)
	}

	// Read the models index
	modelsPath := filepath.Join(s.rootPath, "models.json")
	modelsData, err := os.ReadFile(modelsPath)
	if err != nil {
		return fmt.Errorf("reading models file: %w", err)
	}

	// Unmarshal the models index
	var models types.ModelIndex
	if err := json.Unmarshal(modelsData, &models); err != nil {
		return fmt.Errorf("unmarshaling models: %w", err)
	}

	// Find the model in the index
	for i, m := range models.Models {
		if m.ID == model.ID {
			// Add new tags
			existingTags := make(map[string]bool)
			for _, t := range m.Tags {
				existingTags[t] = true
			}

			for _, newTag := range newTags {
				if !existingTags[newTag] {
					models.Models[i].Tags = append(models.Models[i].Tags, newTag)
					existingTags[newTag] = true
				}
			}
			break
		}
	}

	// Marshal the models index
	modelsData, err = json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling models: %w", err)
	}

	// Write the models index
	if err := os.WriteFile(modelsPath, modelsData, 0644); err != nil {
		return fmt.Errorf("writing models file: %w", err)
	}

	return nil
}

// RemoveTags removes tags from models
func (s *LocalStore) RemoveTags(tags []string) error {
	// Read the models index
	modelsPath := filepath.Join(s.rootPath, "models.json")
	modelsData, err := os.ReadFile(modelsPath)
	if err != nil {
		return fmt.Errorf("reading models file: %w", err)
	}

	// Unmarshal the models index
	var models types.ModelIndex
	if err := json.Unmarshal(modelsData, &models); err != nil {
		return fmt.Errorf("unmarshaling models: %w", err)
	}

	// Create a map of tags to remove
	tagsToRemove := make(map[string]bool)
	for _, tag := range tags {
		tagsToRemove[tag] = true
	}

	// Remove tags from models
	var modelsToRemove []int
	for i, model := range models.Models {
		var newTags []string
		for _, tag := range model.Tags {
			if !tagsToRemove[tag] {
				newTags = append(newTags, tag)
			}
		}
		models.Models[i].Tags = newTags

		// If no more tags, mark model for removal
		if len(models.Models[i].Tags) == 0 {
			modelsToRemove = append(modelsToRemove, i)
		}
	}

	// Remove models with no tags (in reverse order to avoid index issues)
	for i := len(modelsToRemove) - 1; i >= 0; i-- {
		index := modelsToRemove[i]
		models.Models = append(models.Models[:index], models.Models[index+1:]...)
	}

	// Marshal the models index
	modelsData, err = json.MarshalIndent(models, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling models: %w", err)
	}

	// Write the models index
	if err := os.WriteFile(modelsPath, modelsData, 0644); err != nil {
		return fmt.Errorf("writing models file: %w", err)
	}

	return nil
}

// Version returns the store version
func (s *LocalStore) Version() string {
	// Read the layout file
	layoutPath := filepath.Join(s.rootPath, "layout.json")
	layoutData, err := os.ReadFile(layoutPath)
	if err != nil {
		return "unknown"
	}

	// Unmarshal the layout
	var layout types.StoreLayout
	if err := json.Unmarshal(layoutData, &layout); err != nil {
		return "unknown"
	}

	return layout.Version
}

// Upgrade upgrades the store to the latest version
func (s *LocalStore) Upgrade() error {
	// Read the layout file
	layoutPath := filepath.Join(s.rootPath, "layout.json")
	layoutData, err := os.ReadFile(layoutPath)
	if err != nil {
		return fmt.Errorf("reading layout file: %w", err)
	}

	// Unmarshal the layout
	var layout types.StoreLayout
	if err := json.Unmarshal(layoutData, &layout); err != nil {
		return fmt.Errorf("unmarshaling layout: %w", err)
	}

	// Check if upgrade is needed
	if layout.Version == CurrentVersion {
		return nil
	}

	// Implement upgrade logic here
	// For now, just update the version
	layout.Version = CurrentVersion

	// Marshal the layout
	layoutData, err = json.MarshalIndent(layout, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling layout: %w", err)
	}

	// Write the layout file
	if err := os.WriteFile(layoutPath, layoutData, 0644); err != nil {
		return fmt.Errorf("writing layout file: %w", err)
	}

	return nil
}
