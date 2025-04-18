package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/docker/pinata/common/pkg/inference"
	"github.com/docker/pinata/common/pkg/inference/models"
	"github.com/docker/pinata/common/pkg/paths"
	"github.com/olekukonko/tablewriter"
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
	paths.Init(paths.OnHost)
	otel.SetErrorHandler(&otelErrorSilencer{})
}

type Client struct {
	dockerClient DockerHttpClient
}

//go:generate mockgen -source=desktop.go -destination=../mocks/mock_desktop.go -package=mocks DockerHttpClient
type DockerHttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func New(dockerClient DockerHttpClient) *Client {
	return &Client{dockerClient}
}

type Status struct {
	Running bool   `json:"running"`
	Status  []byte `json:"status"`
	Error   error  `json:"error"`
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
	jsonData, err := json.Marshal(models.ModelCreateRequest{From: model})
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

func (c *Client) List(jsonFormat, openai bool, quiet bool, model string) (string, error) {
	modelsRoute := inference.ModelsPrefix
	if openai {
		modelsRoute = inference.InferencePrefix + "/v1/models"
	}
	if model != "" {
		if !strings.Contains(strings.Trim(model, "/"), "/") {
			// Do an extra API call to check if the model parameter isn't a model ID.
			if expanded, err := c.fullModelID(model); err == nil {
				model = expanded
			}
		}
		modelsRoute += "/" + model
	}

	body, err := c.listRaw(modelsRoute, model)
	if err != nil {
		return "", err
	}

	if openai {
		return string(body), nil
	}

	if model != "" {
		// Handle single model for `docker model inspect`.
		// TODO: Handle this in model-distribution.
		var modelJson Model
		if err := json.Unmarshal(body, &modelJson); err != nil {
			return "", fmt.Errorf("failed to unmarshal response body: %w", err)
		}

		modelJsonPretty, err := json.MarshalIndent(modelJson, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal model: %w", err)
		}

		return string(modelJsonPretty), nil
	}

	var modelsJson []Model
	if err := json.Unmarshal(body, &modelsJson); err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if jsonFormat {
		modelsJsonPretty, err := json.MarshalIndent(modelsJson, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal models list: %w", err)
		}

		return string(modelsJsonPretty), nil
	}

	if quiet {
		var modelIDs string
		for _, m := range modelsJson {
			if len(m.Tags) == 0 {
				fmt.Fprintf(os.Stderr, "no tags found for model: %v\n", m)
				continue
			}
			if len(m.ID) < 19 {
				fmt.Fprintf(os.Stderr, "invalid image ID for model: %v\n", m)
				continue
			}
			modelIDs += fmt.Sprintf("%s\n", m.ID[7:19])
		}
		return modelIDs, nil
	}

	return prettyPrintModels(modelsJson), nil
}

func (c *Client) listRaw(route string, model string) ([]byte, error) {
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

	var modelsJson []Model
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

func (c *Client) Chat(model, prompt string) error {
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

	chatCompletionsPath := inference.InferencePrefix + "/v1/chat/completions"
	resp, err := c.doRequest(
		http.MethodPost,
		chatCompletionsPath,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return c.handleQueryError(err, chatCompletionsPath)
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

func URL(path string) string {
	return fmt.Sprintf("http://localhost" + inference.ExperimentalEndpointsPrefix + path)
}

// doRequest is a helper function that performs HTTP requests and handles 503 responses
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, URL(path), body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.dockerClient.Do(req)
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

func prettyPrintModels(models []Model) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"MODEL NAME", "PARAMETERS", "QUANTIZATION", "ARCHITECTURE", "MODEL ID", "CREATED", "SIZE"})

	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT, // MODEL
		tablewriter.ALIGN_LEFT, // PARAMETERS
		tablewriter.ALIGN_LEFT, // QUANTIZATION
		tablewriter.ALIGN_LEFT, // ARCHITECTURE
		tablewriter.ALIGN_LEFT, // MODEL ID
		tablewriter.ALIGN_LEFT, // CREATED
		tablewriter.ALIGN_LEFT, // SIZE
	})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	for _, m := range models {
		if len(m.Tags) == 0 {
			appendRow(table, "<none>", m)
			continue
		}
		for _, tag := range m.Tags {
			appendRow(table, tag, m)
		}
	}

	table.Render()
	return buf.String()
}

func appendRow(table *tablewriter.Table, tag string, model Model) {
	if len(model.ID) < 19 {
		fmt.Fprintf(os.Stderr, "invalid model ID for model: %v\n", model)
		return
	}
	table.Append([]string{
		tag,
		model.Config.Parameters,
		model.Config.Quantization,
		model.Config.Architecture,
		model.ID[7:19],
		units.HumanDuration(time.Since(time.Unix(model.Created, 0))) + " ago",
		model.Config.Size,
	})
}

func (c *Client) Tag(source, targetRepo, targetTag string) (string, error) {
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
