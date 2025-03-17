package distribution

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/docker/model-distribution/pkg/image"
	"github.com/docker/model-distribution/pkg/store"
	"github.com/docker/model-distribution/pkg/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
)

// Client provides model distribution functionality
type Client struct {
	store *store.LocalStore
	log   *logrus.Entry
}

// GetStorePath returns the root path where models are stored
func (c *Client) GetStorePath() string {
	return c.store.RootPath()
}

// ClientOptions represents options for creating a new Client
type ClientOptions struct {
	storeRootPath string
	logger        *logrus.Entry
}

// WithStoreRootPath sets the store root path
func WithStoreRootPath(path string) func(*ClientOptions) {
	return func(o *ClientOptions) {
		o.storeRootPath = path
	}
}

// WithLogger sets the logger
func WithLogger(logger *logrus.Entry) func(*ClientOptions) {
	return func(o *ClientOptions) {
		o.logger = logger
	}
}

// NewClient creates a new distribution client
func NewClient(opts ...func(*ClientOptions)) (*Client, error) {
	options := &ClientOptions{
		logger: logrus.NewEntry(logrus.StandardLogger()),
	}
	for _, opt := range opts {
		opt(options)
	}

	if options.storeRootPath == "" {
		return nil, fmt.Errorf("store root path is required")
	}

	s, err := store.New(types.StoreOptions{RootPath: options.storeRootPath})
	if err != nil {
		return nil, fmt.Errorf("initializing store: %w", err)
	}

	options.logger.Infoln("Successfully initialized store")
	return &Client{
		store: s,
		log:   options.logger,
	}, nil
}

// PullModel pulls a model from a registry and returns the local file path
func (c *Client) PullModel(ctx context.Context, reference string, progressWriter io.Writer) (string, error) {
	c.log.Infoln("Starting model pull:", reference)

	// Check if model exists in local store
	_, err := c.store.GetByTag(reference)
	if err == nil {
		c.log.Infoln("Model found in local store:", reference)
		// Model exists in local store, get its path
		blobPath, err := c.GetModelPath(reference)
		if err != nil {
			return "", err
		}

		// Get file size for progress reporting
		fileInfo, err := os.Stat(blobPath)
		if err != nil {
			return "", fmt.Errorf("getting file info: %w", err)
		}

		// Report progress for local model
		if progressWriter != nil {
			size := fileInfo.Size()
			fmt.Fprintf(progressWriter, "Using cached model: %.2f MB\n", float64(size)/1024/1024)
		}

		return blobPath, nil
	}

	c.log.Infoln("Model not found in local store, pulling from remote:", reference)
	// Model doesn't exist in local store, pull from remote
	ref, err := name.ParseReference(reference)
	if err != nil {
		c.log.Errorln("Failed to parse reference:", err, "reference:", reference)
		return "", fmt.Errorf("parsing reference: %w", err)
	}

	// Create a buffered channel for progress updates
	progress := make(chan v1.Update, 100)
	defer close(progress)

	// Start a goroutine to handle progress updates
	go func() {
		var lastComplete int64
		var lastUpdate time.Time
		const updateInterval = 500 * time.Millisecond // Update every 500ms
		const minBytesForUpdate = 1024 * 1024         // At least 1MB difference

		for p := range progress {
			if progressWriter != nil {
				now := time.Now()
				bytesDownloaded := p.Complete - lastComplete

				// Only update if enough time has passed or enough bytes downloaded
				if now.Sub(lastUpdate) >= updateInterval || bytesDownloaded >= minBytesForUpdate {
					fmt.Fprintf(progressWriter, "Downloaded: %.2f MB\n", float64(p.Complete)/1024/1024)
					lastUpdate = now
					lastComplete = p.Complete
				}
			}
		}
	}()

	// Configure remote options with progress tracking
	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithContext(ctx),
		remote.WithProgress(progress),
	}

	// Pull the image with progress tracking
	img, err := remote.Image(ref, remoteOpts...)
	if err != nil {
		c.log.Errorln("Failed to pull image:", err, "reference:", reference)
		return "", fmt.Errorf("pulling image: %w", err)
	}

	// Create a temporary file to store the model content
	tempFile, err := os.CreateTemp("", "model-*.gguf")
	if err != nil {
		c.log.Errorln("Failed to create temporary file:", err)
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Get the model content from the image
	layers, err := img.Layers()
	if err != nil {
		c.log.Errorln("Failed to get image layers:", err)
		return "", fmt.Errorf("getting layers: %w", err)
	}

	if len(layers) == 0 {
		c.log.Errorln("No layers found in image")
		return "", fmt.Errorf("no layers in image")
	}

	// Use the first layer (assuming there's only one for models)
	layer := layers[0]

	// Get the layer content
	rc, err := layer.Uncompressed()
	if err != nil {
		c.log.Errorln("Failed to get layer content:", err)
		return "", fmt.Errorf("getting layer content: %w", err)
	}
	defer rc.Close()

	// Create a progress reader to track layer download progress
	progressReader := &ProgressReader{
		Reader:       rc,
		ProgressChan: progress,
	}

	// Write the layer content to the temporary file
	if _, err := io.Copy(tempFile, progressReader); err != nil {
		c.log.Errorln("Failed to write layer content:", err)
		return "", fmt.Errorf("writing layer content: %w", err)
	}

	// Push the model to the local store
	if err := c.store.Push(tempFile.Name(), []string{reference}); err != nil {
		c.log.Errorln("Failed to store model in local store:", err, "reference:", reference)
		return "", fmt.Errorf("storing model in local store: %w", err)
	}

	c.log.Infoln("Successfully pulled and stored model:", reference)
	// Get the model path
	return c.GetModelPath(reference)
}

// ProgressReader wraps an io.Reader to track reading progress
type ProgressReader struct {
	Reader       io.Reader
	ProgressChan chan<- v1.Update
	Total        int64
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.Total += int64(n)
		pr.ProgressChan <- v1.Update{Complete: pr.Total}
	}
	return n, err
}

// GetModelPath returns the local file path for a model
func (c *Client) GetModelPath(reference string) (string, error) {
	c.log.Infoln("Getting model path:", reference)
	// Get the direct path to the blob file
	blobPath, err := c.store.GetBlobPath(reference)
	if err != nil {
		c.log.Errorln("Failed to get blob path:", err, "reference:", reference)
		return "", fmt.Errorf("getting blob path: %w", err)
	}

	return blobPath, nil
}

// ListModels returns all available models
func (c *Client) ListModels() ([]*types.Model, error) {
	c.log.Infoln("Listing available models")
	models, err := c.store.List()
	if err != nil {
		c.log.Errorln("Failed to list models:", err)
		return nil, fmt.Errorf("listing models: %w", err)
	}

	result := make([]*types.Model, len(models))
	for i, model := range models {
		modelCopy := model // Create a copy to avoid issues with the loop variable
		result[i] = &modelCopy
	}

	c.log.Infoln("Successfully listed models, count:", len(result))
	return result, nil
}

// GetModel returns a model by reference
func (c *Client) GetModel(reference string) (*types.Model, error) {
	c.log.Infoln("Getting model by reference:", reference)
	model, err := c.store.GetByTag(reference)
	if err != nil {
		c.log.Errorln("Model not found:", err, "reference:", reference)
		return nil, ErrModelNotFound
	}

	return model, nil
}

// PushModel pushes a model to a registry
func (c *Client) PushModel(ctx context.Context, source, reference string) error {
	c.log.Infoln("Starting model push, source:", source, "reference:", reference)

	// Parse the reference
	ref, err := name.ParseReference(reference)
	if err != nil {
		c.log.Errorln("Failed to parse reference:", err, "reference:", reference)
		return fmt.Errorf("parsing reference: %w", err)
	}

	// Read the model file
	fileContent, err := os.ReadFile(source)
	if err != nil {
		c.log.Errorln("Failed to read model file:", err, "source:", source)
		return fmt.Errorf("reading model file: %w", err)
	}

	// Create layer from model content
	layer := static.NewLayer(fileContent, "application/vnd.docker.ai.model.file.v1+gguf")

	// Create image with layer
	img, err := image.CreateImage(layer)
	if err != nil {
		c.log.Errorln("Failed to create image:", err)
		return fmt.Errorf("creating image: %w", err)
	}

	// Push the image
	if err := remote.Write(ref, img,
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
		remote.WithContext(ctx),
	); err != nil {
		c.log.Errorln("Failed to push image:", err, "reference:", reference)
		return fmt.Errorf("pushing image: %w", err)
	}

	// Store the model in the local store
	if err := c.store.Push(source, []string{reference}); err != nil {
		c.log.Errorln("Failed to store model in local store:", err, "reference:", reference)
		return fmt.Errorf("storing model in local store: %w", err)
	}

	c.log.Infoln("Successfully pushed model:", reference)
	return nil
}

// getImageFromLocalStore creates an image from a model in the local store
func (c *Client) getImageFromLocalStore(model *types.Model) (v1.Image, error) {
	c.log.Infoln("Getting image from local store:", model.Tags[0])

	// Get the direct path to the blob file
	blobPath, err := c.store.GetBlobPath(model.Tags[0])
	if err != nil {
		c.log.Errorln("Failed to get blob path:", err, "model:", model.Tags[0])
		return nil, fmt.Errorf("getting blob path: %w", err)
	}

	// Read the model content directly from the blob file
	modelContent, err := os.ReadFile(blobPath)
	if err != nil {
		c.log.Errorln("Failed to read model content:", err, "path:", blobPath)
		return nil, fmt.Errorf("reading model content: %w", err)
	}

	// Create layer from model content
	layer := static.NewLayer(modelContent, "application/vnd.docker.ai.model.file.v1+gguf")

	// Create image with layer
	img, err := image.CreateImage(layer)
	if err != nil {
		c.log.Errorln("Failed to create image from layer:", err)
		return nil, err
	}

	return img, nil
}

// DeleteModel deletes a model by tag
func (c *Client) DeleteModel(tag string) error {
	c.log.Infoln("Deleting model:", tag)
	if err := c.store.Delete(tag); err != nil {
		c.log.Errorln("Failed to delete model:", err, "tag:", tag)
		return fmt.Errorf("deleting model: %w", err)
	}
	c.log.Infoln("Successfully deleted model:", tag)
	return nil
}
