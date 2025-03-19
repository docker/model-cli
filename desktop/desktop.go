package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/docker/docker/client"
	"github.com/docker/pinata/common/pkg/engine"
	"github.com/docker/pinata/common/pkg/inference"
	"github.com/docker/pinata/common/pkg/inference/models"
	"github.com/docker/pinata/common/pkg/paths"
)

func init() {
	paths.Init(paths.OnHost)
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

func (c *Client) List(openai bool, model string) (string, error) {
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
		return "", fmt.Errorf("failed to list models: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(body), nil
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
