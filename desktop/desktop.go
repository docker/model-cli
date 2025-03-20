package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/docker/pinata/common/pkg/engine"
	"github.com/docker/pinata/common/pkg/inference"
	"github.com/docker/pinata/common/pkg/inference/models"
	"github.com/docker/pinata/common/pkg/paths"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
)

var ErrNotFound = errors.New("model not found")

type otelErrorSilencer struct{}

func (oes *otelErrorSilencer) Handle(error) {}

func init() {
	paths.Init(paths.OnHost)
	otel.SetErrorHandler(&otelErrorSilencer{})
}

type Client struct {
	dockerClient *client.Client
}

func New() (*Client, error) {
	dockerClient, err := client.NewClientWithOpts(
		// TODO: Make sure it works while running in Windows containers mode.
		client.WithHost(paths.HostServiceSockets().DockerHost(engine.Linux)),
	)
	if err != nil {
		return nil, err
	}
	return &Client{dockerClient}, nil
}

func (c *Client) Status() (string, error) {
	// TODO: Query "/".
	resp, err := c.dockerClient.HTTPClient().Get(url(inference.ModelsPrefix))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return "Docker Model Runner is running", nil
	}
	return "Docker Model Runner is not running", nil
}

func (c *Client) Pull(model string) (string, error) {
	jsonData, err := json.Marshal(models.ModelCreateRequest{From: model})
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %w", err)
	}

	resp, err := c.dockerClient.HTTPClient().Post(
		url(inference.ModelsPrefix+"/create"),
		"application/json",
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("error querying %s: %w", inference.ModelsPrefix+"/create", err)
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
			fmt.Print("\r\033[K", progressLine)
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
		if len(strings.Split(strings.Trim(model, "/"), "/")) != 2 {
			return "", fmt.Errorf("invalid model name: %s", model)
		}
		modelsRoute += "/" + model
	}
	resp, err := c.dockerClient.HTTPClient().Get(url(modelsRoute))
	if err != nil {
		return "", err
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

	resp, err := c.dockerClient.HTTPClient().Post(
		url(inference.InferencePrefix+"/v1/chat/completions"),
		"application/json",
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return fmt.Errorf("error querying %s: %w", inference.InferencePrefix+"/v1/chat/completions", err)
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
	req, err := http.NewRequest(http.MethodDelete, url(inference.ModelsPrefix+"/"+model), nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	resp, err := c.dockerClient.HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("error querying %s: %w", inference.ModelsPrefix+"/"+model, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK { // from common/pkg/inference/models/manager.go
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

func url(path string) string {
	return fmt.Sprintf("http://localhost" + inference.ExperimentalEndpointsPrefix + path)
}

func prettyPrintModels(models []Model) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"MODEL", "PARAMETERS", "QUANTIZATION", "ARCHITECTURE", "FORMAT", "MODEL ID", "CREATED", "SIZE"})

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
		tablewriter.ALIGN_LEFT, // FORMAT
		tablewriter.ALIGN_LEFT, // IMAGE ID
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
			string(m.Config.Format),
			m.ID[7:19],
			timeAgo(time.Unix(m.Created, 0)),
			m.Config.Size,
		})
	}

	table.Render()
	return buf.String()
}

func timeAgo(t time.Time) string {
	duration := time.Since(t)
	hours := int(duration.Hours())
	days := hours / 24
	months := days / 30
	years := months / 12

	switch {
	case hours < 1:
		return strconv.Itoa(int(duration.Minutes())) + " minutes ago"
	case hours < 24:
		return strconv.Itoa(hours) + " hours ago"
	case days < 30:
		return strconv.Itoa(days) + " days ago"
	case months < 12:
		return strconv.Itoa(months) + " months ago"
	default:
		return strconv.Itoa(years) + " years ago"
	}
}
