package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	Running bool  `json:"running"`
	Error   error `json:"error"`
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
		return Status{
			Running: true,
		}
	}
	return Status{
		Running: false,
		Error:   fmt.Errorf("unexpected status code: %d", resp.StatusCode),
	}
}

func (c *Client) Pull(model string, progress func(string)) (string, error) {
	jsonData, err := json.Marshal(models.ModelCreateRequest{From: model})
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	createPath := inference.ModelsPrefix + "/create"
	resp, err := c.doRequest(
		http.MethodPost,
		createPath,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return "", c.handleQueryError(err, createPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("pulling %s failed with status %s: %s", model, resp.Status, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine != "" {
			progress(progressLine)
		}
	}

	fmt.Println()

	return fmt.Sprintf("Model %s pulled successfully", model), nil
}

func (c *Client) List(jsonFormat, openai bool, model string) (string, error) {
	modelsRoute := inference.ModelsPrefix
	if openai {
		modelsRoute = inference.InferencePrefix + "/v1/models"
	}
	if model != "" {
		if !strings.Contains(strings.Trim(model, "/"), "/") {
			// We assume a model name is invalid if it does not contain a "/".
			return "", fmt.Errorf("invalid model name: %s", model)
		}
		modelsRoute += "/" + model
	}

	resp, err := c.doRequest(http.MethodGet, modelsRoute, nil)
	if err != nil {
		return "", c.handleQueryError(err, modelsRoute)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if model != "" && resp.StatusCode == http.StatusNotFound {
			return "", errors.Wrap(ErrNotFound, model)
		}
		return "", fmt.Errorf("failed to list models: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
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

	return prettyPrintModels(modelsJson), nil
}

func (c *Client) Chat(model, prompt string) error {
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

func (c *Client) Remove(model string) (string, error) {
	removePath := inference.ModelsPrefix + "/" + model
	resp, err := c.doRequest(http.MethodDelete, removePath, nil)
	if err != nil {
		return "", c.handleQueryError(err, removePath)
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
		return "", fmt.Errorf("removing %s failed with status %s: %s", model, resp.Status, bodyStr)
	}

	return fmt.Sprintf("Model %s removed successfully", model), nil
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

	table.SetHeader([]string{"MODEL", "PARAMETERS", "QUANTIZATION", "ARCHITECTURE", "MODEL ID", "CREATED", "SIZE"})

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
			fmt.Fprintf(os.Stderr, "no tags found for model: %v\n", m)
			continue
		}
		if len(m.ID) < 19 {
			fmt.Fprintf(os.Stderr, "invalid image ID for model: %v\n", m)
			continue
		}
		table.Append([]string{
			m.Tags[0],
			m.Config.Parameters,
			m.Config.Quantization,
			m.Config.Architecture,
			m.ID[7:19],
			units.HumanDuration(time.Since(time.Unix(m.Created, 0))) + " ago",
			m.Config.Size,
		})
	}

	table.Render()
	return buf.String()
}
