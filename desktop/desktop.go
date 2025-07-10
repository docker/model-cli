package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/model-runner/pkg/inference"
	dmrm "github.com/docker/model-runner/pkg/inference/models"
	"github.com/docker/model-runner/pkg/inference/scheduling"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
)

var (
	ErrNotFound           = errors.New("model not found")
	ErrServiceUnavailable = errors.New("service unavailable")
)

type otelErrorSilencer struct{}

func (oes *otelErrorSilencer) Handle(error) {}

func init() {
	otel.SetErrorHandler(&otelErrorSilencer{})
}

type Client struct {
	modelRunner *ModelRunnerContext
}

//go:generate mockgen -source=desktop.go -destination=../mocks/mock_desktop.go -package=mocks DockerHttpClient
type DockerHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func New(modelRunner *ModelRunnerContext) *Client {
	return &Client{modelRunner}
}

type Status struct {
	Running bool   `json:"running"`
	Status  []byte `json:"status"`
	Error   error  `json:"error"`
}

// normalizeHuggingFaceModelName converts Hugging Face model names to lowercase
func normalizeHuggingFaceModelName(model string) string {
	if strings.HasPrefix(model, "hf.co/") {
		return strings.ToLower(model)
	}
	return model
}

func (c *Client) Status() Status {
	// TODO: Query "/".
	resp, err := c.doRequest(http.MethodGet, inference.ModelsPrefix, nil)
	if err != nil {
		err = c.handleQueryError(err, inference.ModelsPrefix)
		if errors.Is(err, ErrServiceUnavailable) {
			return Status{
				Running: false,
			}
		}
		return Status{
			Running: false,
			Error:   err,
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var status []byte
		statusResp, err := c.doRequest(http.MethodGet, inference.InferencePrefix+"/status", nil)
		if err != nil {
			status = []byte(fmt.Sprintf("error querying status: %v", err))
		} else {
			defer statusResp.Body.Close()
			statusBody, err := io.ReadAll(statusResp.Body)
			if err != nil {
				status = []byte(fmt.Sprintf("error reading status body: %v", err))
			} else {
				status = statusBody
			}
		}
		return Status{
			Running: true,
			Status:  status,
		}
	}
	return Status{
		Running: false,
		Error:   fmt.Errorf("unexpected status code: %d", resp.StatusCode),
	}
}

func (c *Client) Pull(model string, progress func(string)) (string, bool, error) {
	model = normalizeHuggingFaceModelName(model)
	jsonData, err := json.Marshal(dmrm.ModelCreateRequest{From: model})
	if err != nil {
		return "", false, fmt.Errorf("error marshaling request: %w", err)
	}

	createPath := inference.ModelsPrefix + "/create"
	resp, err := c.doRequest(
		http.MethodPost,
		createPath,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return "", false, c.handleQueryError(err, createPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("pulling %s failed with status %s: %s", model, resp.Status, string(body))
	}

	progressShown := false

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		// Parse the progress message
		var progressMsg ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			return "", progressShown, fmt.Errorf("error parsing progress message: %w", err)
		}

		// Handle different message types
		switch progressMsg.Type {
		case "progress":
			progress(progressMsg.Message)
			progressShown = true
		case "error":
			return "", progressShown, fmt.Errorf("error pulling model: %s", progressMsg.Message)
		case "success":
			return progressMsg.Message, progressShown, nil
		default:
			return "", progressShown, fmt.Errorf("unknown message type: %s", progressMsg.Type)
		}
	}

	// If we get here, something went wrong
	return "", progressShown, fmt.Errorf("unexpected end of stream while pulling model %s", model)
}

func (c *Client) Push(model string, progress func(string)) (string, bool, error) {
	model = normalizeHuggingFaceModelName(model)
	pushPath := inference.ModelsPrefix + "/" + model + "/push"
	resp, err := c.doRequest(
		http.MethodPost,
		pushPath,
		nil, // Assuming no body is needed for the push request
	)
	if err != nil {
		return "", false, c.handleQueryError(err, pushPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("pushing %s failed with status %s: %s", model, resp.Status, string(body))
	}

	progressShown := false

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		// Parse the progress message
		var progressMsg ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			return "", progressShown, fmt.Errorf("error parsing progress message: %w", err)
		}

		// Handle different message types
		switch progressMsg.Type {
		case "progress":
			progress(progressMsg.Message)
			progressShown = true
		case "error":
			return "", progressShown, fmt.Errorf("error pushing model: %s", progressMsg.Message)
		case "success":
			return progressMsg.Message, progressShown, nil
		default:
			return "", progressShown, fmt.Errorf("unknown message type: %s", progressMsg.Type)
		}
	}

	// If we get here, something went wrong
	return "", progressShown, fmt.Errorf("unexpected end of stream while pushing model %s", model)
}

func (c *Client) List() ([]dmrm.Model, error) {
	modelsRoute := inference.ModelsPrefix
	body, err := c.listRaw(modelsRoute, "")
	if err != nil {
		return []dmrm.Model{}, err
	}

	var modelsJson []dmrm.Model
	if err := json.Unmarshal(body, &modelsJson); err != nil {
		return modelsJson, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return modelsJson, nil
}

func (c *Client) ListOpenAI() (dmrm.OpenAIModelList, error) {
	modelsRoute := inference.InferencePrefix + "/v1/models"
	rawResponse, err := c.listRaw(modelsRoute, "")
	if err != nil {
		return dmrm.OpenAIModelList{}, err
	}
	var modelsJson dmrm.OpenAIModelList
	if err := json.Unmarshal(rawResponse, &modelsJson); err != nil {
		return modelsJson, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return modelsJson, nil
}

func (c *Client) Inspect(model string, remote bool) (dmrm.Model, error) {
	model = normalizeHuggingFaceModelName(model)
	if model != "" {
		if !strings.Contains(strings.Trim(model, "/"), "/") {
			// Do an extra API call to check if the model parameter isn't a model ID.
			modelId, err := c.fullModelID(model)
			if err != nil {
				return dmrm.Model{}, fmt.Errorf("invalid model name: %s", model)
			}
			model = modelId
		}
	}
	rawResponse, err := c.listRawWithQuery(fmt.Sprintf("%s/%s", inference.ModelsPrefix, model), model, remote)
	if err != nil {
		return dmrm.Model{}, err
	}
	var modelInspect dmrm.Model
	if err := json.Unmarshal(rawResponse, &modelInspect); err != nil {
		return modelInspect, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return modelInspect, nil
}

func (c *Client) InspectOpenAI(model string) (dmrm.OpenAIModel, error) {
	model = normalizeHuggingFaceModelName(model)
	modelsRoute := inference.InferencePrefix + "/v1/models"
	if !strings.Contains(strings.Trim(model, "/"), "/") {
		// Do an extra API call to check if the model parameter isn't a model ID.
		var err error
		if model, err = c.fullModelID(model); err != nil {
			return dmrm.OpenAIModel{}, fmt.Errorf("invalid model name: %s", model)
		}
	}
	rawResponse, err := c.listRaw(fmt.Sprintf("%s/%s", modelsRoute, model), model)
	if err != nil {
		return dmrm.OpenAIModel{}, err
	}
	var modelInspect dmrm.OpenAIModel
	if err := json.Unmarshal(rawResponse, &modelInspect); err != nil {
		return modelInspect, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return modelInspect, nil
}

func (c *Client) listRaw(route string, model string) ([]byte, error) {
	return c.listRawWithQuery(route, model, false)
}

func (c *Client) listRawWithQuery(route string, model string, remote bool) ([]byte, error) {
	if remote {
		route += "?remote=true"
	}

	resp, err := c.doRequest(http.MethodGet, route, nil)
	if err != nil {
		return nil, c.handleQueryError(err, route)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if model != "" && resp.StatusCode == http.StatusNotFound {
			return nil, errors.Wrap(ErrNotFound, model)
		}
		return nil, fmt.Errorf("failed to list models: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, nil
}

func (c *Client) fullModelID(id string) (string, error) {
	bodyResponse, err := c.listRaw(inference.ModelsPrefix, "")
	if err != nil {
		return "", err
	}

	var modelsJson []dmrm.Model
	if err := json.Unmarshal(bodyResponse, &modelsJson); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	for _, m := range modelsJson {
		if m.ID[7:19] == id || strings.TrimPrefix(m.ID, "sha256:") == id || m.ID == id {
			return m.ID, nil
		}
	}

	return "", fmt.Errorf("model with ID %s not found", id)
}

func (c *Client) Chat(backend, model, prompt string) error {
	model = normalizeHuggingFaceModelName(model)
	if !strings.Contains(strings.Trim(model, "/"), "/") {
		// Do an extra API call to check if the model parameter isn't a model ID.
		if expanded, err := c.fullModelID(model); err == nil {
			model = expanded
		}
	}

	reqBody := OpenAIChatRequest{
		Model: model,
		Messages: []OpenAIChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	var completionsPath string
	if backend != "" {
		completionsPath = inference.InferencePrefix + "/" + backend + "/v1/chat/completions"
	} else {
		completionsPath = inference.InferencePrefix + "/v1/chat/completions"
	}

	resp, err := c.doRequest(
		http.MethodPost,
		completionsPath,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return c.handleQueryError(err, completionsPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response: status=%d body=%s", resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		var streamResp OpenAIChatResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return fmt.Errorf("error parsing stream response: %w", err)
		}

		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			chunk := streamResp.Choices[0].Delta.Content
			fmt.Print(chunk)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response stream: %w", err)
	}

	return nil
}

func (c *Client) Remove(models []string, force bool) (string, error) {
	modelRemoved := ""
	for _, model := range models {
		model = normalizeHuggingFaceModelName(model)
		// Check if not a model ID passed as parameter.
		if !strings.Contains(model, "/") {
			if expanded, err := c.fullModelID(model); err == nil {
				model = expanded
			}
		}

		// Construct the URL with query parameters
		removePath := fmt.Sprintf("%s/%s?force=%s",
			inference.ModelsPrefix,
			model,
			strconv.FormatBool(force),
		)

		resp, err := c.doRequest(http.MethodDelete, removePath, nil)
		if err != nil {
			return modelRemoved, c.handleQueryError(err, removePath)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				return modelRemoved, fmt.Errorf("no such model: %s", model)
			}
			var bodyStr string
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				bodyStr = fmt.Sprintf("(failed to read response body: %v)", err)
			} else {
				bodyStr = string(body)
			}
			return modelRemoved, fmt.Errorf("removing %s failed with status %s: %s", model, resp.Status, bodyStr)
		}
		modelRemoved += fmt.Sprintf("Model %s removed successfully\n", model)
	}
	return modelRemoved, nil
}

// BackendStatus to be imported from docker/model-runner when https://github.com/docker/model-runner/pull/42 is merged.
type BackendStatus struct {
	// BackendName is the name of the backend
	BackendName string `json:"backend_name"`
	// ModelName is the name of the model loaded in the backend
	ModelName string `json:"model_name"`
	// Mode is the mode the backend is operating in
	Mode string `json:"mode"`
	// LastUsed represents when this backend was last used (if it's idle)
	LastUsed time.Time `json:"last_used,omitempty"`
}

func (c *Client) PS() ([]BackendStatus, error) {
	psPath := inference.InferencePrefix + "/ps"
	resp, err := c.doRequest(http.MethodGet, psPath, nil)
	if err != nil {
		return []BackendStatus{}, c.handleQueryError(err, psPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []BackendStatus{}, fmt.Errorf("failed to list running models: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	var ps []BackendStatus
	if err := json.Unmarshal(body, &ps); err != nil {
		return []BackendStatus{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return ps, nil
}

// DiskUsage to be imported from docker/model-runner when https://github.com/docker/model-runner/pull/45 is merged.
type DiskUsage struct {
	ModelsDiskUsage         int64 `json:"models_disk_usage"`
	DefaultBackendDiskUsage int64 `json:"default_backend_disk_usage"`
}

func (c *Client) DF() (DiskUsage, error) {
	dfPath := inference.InferencePrefix + "/df"
	resp, err := c.doRequest(http.MethodGet, dfPath, nil)
	if err != nil {
		return DiskUsage{}, c.handleQueryError(err, dfPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DiskUsage{}, fmt.Errorf("failed to get disk usage: %s", resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	var df DiskUsage
	if err := json.Unmarshal(body, &df); err != nil {
		return DiskUsage{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return df, nil
}

// UnloadRequest to be imported from docker/model-runner when https://github.com/docker/model-runner/pull/46 is merged.
type UnloadRequest struct {
	All     bool     `json:"all"`
	Backend string   `json:"backend"`
	Models  []string `json:"models"`
}

// UnloadResponse to be imported from docker/model-runner when https://github.com/docker/model-runner/pull/46 is merged.
type UnloadResponse struct {
	UnloadedRunners int `json:"unloaded_runners"`
}

func (c *Client) Unload(req UnloadRequest) (UnloadResponse, error) {
	unloadPath := inference.InferencePrefix + "/unload"
	jsonData, err := json.Marshal(req)
	if err != nil {
		return UnloadResponse{}, fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, unloadPath, bytes.NewReader(jsonData))
	if err != nil {
		return UnloadResponse{}, c.handleQueryError(err, unloadPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return UnloadResponse{}, fmt.Errorf("unloading failed with status %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UnloadResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var unloadResp UnloadResponse
	if err := json.Unmarshal(body, &unloadResp); err != nil {
		return UnloadResponse{}, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return unloadResp, nil
}

func (c *Client) ConfigureBackend(request scheduling.ConfigureRequest) error {
	configureBackendPath := inference.InferencePrefix + "/_configure"
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := c.doRequest(http.MethodPost, configureBackendPath, bytes.NewReader(jsonData))
	if err != nil {
		return c.handleQueryError(err, configureBackendPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("%s", body)
		}
		return fmt.Errorf("%s (%s)", body, resp.Status)
	}

	return nil
}

// doRequest is a helper function that performs HTTP requests and handles 503 responses
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.modelRunner.URL(path), body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("User-Agent", "docker-model-cli/"+Version)
	resp, err := c.modelRunner.Client().Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		resp.Body.Close()
		return nil, ErrServiceUnavailable
	}

	return resp, nil
}

func (c *Client) handleQueryError(err error, path string) error {
	if errors.Is(err, ErrServiceUnavailable) {
		return ErrServiceUnavailable
	}
	return fmt.Errorf("error querying %s: %w", path, err)
}

func (c *Client) Tag(source, targetRepo, targetTag string) (string, error) {
	source = normalizeHuggingFaceModelName(source)
	// Check if the source is a model ID, and expand it if necessary
	if !strings.Contains(strings.Trim(source, "/"), "/") {
		// Do an extra API call to check if the model parameter might be a model ID
		if expanded, err := c.fullModelID(source); err == nil {
			source = expanded
		}
	}

	// Construct the URL with query parameters
	tagPath := fmt.Sprintf("%s/%s/tag?repo=%s&tag=%s",
		inference.ModelsPrefix,
		source,
		targetRepo,
		targetTag,
	)

	resp, err := c.doRequest(http.MethodPost, tagPath, nil)
	if err != nil {
		return "", c.handleQueryError(err, tagPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tagging failed with status %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}
