package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/docker/model-distribution/pkg/distribution"
	"github.com/docker/model-distribution/pkg/types"
	"github.com/docker/pinata/common/pkg/inference"
	"github.com/docker/pinata/common/pkg/logger"
	"github.com/docker/pinata/common/pkg/paths"
)

const (
	// maximumConcurrentModelPulls is the maximum number of concurrent model
	// pulls that a model manager will allow.
	maximumConcurrentModelPulls = 2
)

// Manager manages inference model pulls and storage.
type Manager struct {
	// log is the associated logger.
	log logger.ComponentLogger
	// httpClient is the HTTP client to use for model pulls.
	httpClient *http.Client
	// pullTokens is a semaphore used to restrict the maximum number of
	// concurrent pull requests.
	pullTokens chan struct{}
	// router is the HTTP request router.
	router *http.ServeMux
	// distributionClient is the client for model distribution.
	distributionClient *distribution.Client
}

// NewManager creates a new model's manager.
func NewManager(log logger.ComponentLogger, httpClient *http.Client) *Manager {
	// Create the distribution client
	distributionClient, err := distribution.NewClient(
		distribution.WithStoreRootPath(paths.DockerHome("models")),
		distribution.WithLogger(log.WithField("component", "model-distribution")))
	if err != nil {
		log.Errorf("Failed to create distribution client: %v", err)
		// Continue without distribution client
	}
	// Create the manager.
	m := &Manager{
		log:                log,
		httpClient:         httpClient,
		pullTokens:         make(chan struct{}, maximumConcurrentModelPulls),
		router:             http.NewServeMux(),
		distributionClient: distributionClient,
	}

	// Register routes.
	m.router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	m.router.HandleFunc("POST "+inference.ModelsPrefix+"/create", m.handleCreateModel)
	m.router.HandleFunc("GET "+inference.ModelsPrefix, m.handleGetModels)
	m.router.HandleFunc("GET "+inference.ModelsPrefix+"/{namespace}/{name}", m.handleGetModel)
	m.router.HandleFunc("DELETE "+inference.ModelsPrefix+"/{namespace}/{name}", m.handleDeleteModel)
	m.router.HandleFunc("GET "+inference.InferencePrefix+"/{backend}/v1/models", m.handleOpenAIGetModels)
	m.router.HandleFunc("GET "+inference.InferencePrefix+"/{backend}/v1/models/{namespace}/{name}", m.handleOpenAIGetModel)
	m.router.HandleFunc("GET "+inference.InferencePrefix+"/v1/models", m.handleOpenAIGetModels)
	m.router.HandleFunc("GET "+inference.InferencePrefix+"/v1/models/{namespace}/{name}", m.handleOpenAIGetModel)

	// Populate the pull concurrency semaphore.
	for i := 0; i < maximumConcurrentModelPulls; i++ {
		m.pullTokens <- struct{}{}
	}

	// Manager successfully initialized.
	return m
}

// handleCreateModel handles POST <inference-prefix>/models/create requests.
func (m *Manager) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Decode the request.
	var request ModelCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Pull the model. In the future, we may support additional operations here
	// besides pulling (such as model building).
	if err := m.PullModel(r.Context(), request.From, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleGetModels handles GET <inference-prefix>/models requests.
func (m *Manager) handleGetModels(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Query models.
	models, err := m.getModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models); err != nil {
		m.log.Warnln("Error while encoding model listing response:", err)
	}
}

// handleGetModel handles GET <inference-prefix>/models/{namespace}/{name} requests.
func (m *Manager) handleGetModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Query the model.
	model, err := m.GetModel(r.PathValue("namespace") + "/" + r.PathValue("name"))
	if err != nil {
		if errors.Is(err, distribution.ErrModelNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(model); err != nil {
		m.log.Warnln("Error while encoding model response:", err)
	}
}

// handleDeleteModel handles DELETE <inference-prefix>/models/{namespace}/{name} requests.
func (m *Manager) handleDeleteModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// TODO: We probably want the manager to have a lock / unlock mechanism for
	// models so that active runners can retain / release a model, analogous to
	// a container blocking the release of an image. However, unlike containers,
	// runners are only evicted when idle or when memory is needed, so users
	// won't be able to release the images manually. Perhaps we can unlink the
	// corresponding GGUF files from disk and allow the OS to clean them up once
	// the runner process exits (though this won't work for Windows, where we
	// might need some separate cleanup process).

	err := m.distributionClient.DeleteModel(r.PathValue("namespace") + "/" + r.PathValue("name"))
	if err != nil {
		m.log.Warnln("Error while deleting model:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleOpenAIGetModels handles GET <inference-prefix>/<backend>/v1/models and
// GET /<inference-prefix>/v1/models requests.
func (m *Manager) handleOpenAIGetModels(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Query models.
	models, err := m.getModels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(models.toOpenAI()); err != nil {
		m.log.Warnln("Error while encoding OpenAI model listing response:", err)
	}
}

// handleOpenAIGetModel handles GET <inference-prefix>/<backend>/v1/models/{namespace}/{name}
// and GET <inference-prefix>/v1/models/{namespace}/{name} requests.
func (m *Manager) handleOpenAIGetModel(w http.ResponseWriter, r *http.Request) {
	if m.distributionClient == nil {
		http.Error(w, "model distribution service unavailable", http.StatusServiceUnavailable)
		return
	}

	// Query the model.
	model, err := m.GetModel(r.PathValue("namespace") + "/" + r.PathValue("name"))
	if err != nil {
		if errors.Is(err, distribution.ErrModelNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Write the response.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToOpenAI(model)); err != nil {
		m.log.Warnln("Error while encoding OpenAI model response:", err)
	}
}

// ServeHTTP implement net/http.Handler.ServeHTTP.
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.router.ServeHTTP(w, r)
}

// getModels returns a list of all models.
func (m *Manager) getModels() (ModelList, error) {
	// Initialize the model list. We always want to return a non-nil list (even
	// if it's empty) so that it can be encoded directly to JSON.
	models := make(ModelList, 0)

	// Get all models from the distribution client
	available, err := m.distributionClient.ListModels()
	if err != nil {
		return nil, fmt.Errorf("error while listing models: %w", err)
	}

	// Convert distribution models to our model format
	for _, current := range available {
		models = append(models, current)
	}

	return models, nil
}

// GetModel returns a single model.
func (m *Manager) GetModel(ref string) (*types.Model, error) {
	return m.distributionClient.GetModel(ref)
}

// GetModelPath returns the path to a model's files.
func (m *Manager) GetModelPath(ref string) (string, error) {
	model, err := m.GetModel(ref)
	if err != nil {
		return "", err
	}
	// TODO: Handle multiple files
	// Convert <algorithm>:<digest> to <algorithm>/<digest>
	blobName := model.Files[0]
	parts := strings.Split(blobName, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid blob format: %s", blobName)
	}
	return fmt.Sprintf("%s/blobs/%s/%s", m.distributionClient.GetStorePath(), parts[0], parts[1]), nil
}

// PullModel pulls a model to local storage. Any error it returns is suitable
// for writing back to the client.
func (m *Manager) PullModel(ctx context.Context, model string, w http.ResponseWriter) error {
	// Restrict model pull concurrency.
	select {
	case <-m.pullTokens:
	case <-ctx.Done():
		return context.Canceled
	}
	defer func() {
		m.pullTokens <- struct{}{}
	}()

	// Set up response headers for streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Create a flusher to ensure chunks are sent immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// Create a progress writer that writes to the response
	progressWriter := &progressResponseWriter{
		writer:  w,
		flusher: flusher,
	}

	// Pull the model using the Docker model distribution client
	m.log.Infoln("Pulling model:", model)
	if _, err := m.distributionClient.PullModel(ctx, model, progressWriter); err != nil {
		return fmt.Errorf("error while pulling model: %w", err)
	}

	return nil
}

// progressResponseWriter implements io.Writer to write progress updates to the HTTP response
type progressResponseWriter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
}

func (w *progressResponseWriter) Write(p []byte) (n int, err error) {
	escapedData := html.EscapeString(string(p))
	n, err = w.writer.Write([]byte(escapedData))
	if err != nil {
		return 0, err
	}
	// Flush the response to ensure the chunk is sent immediately
	w.flusher.Flush()
	return n, nil
}
