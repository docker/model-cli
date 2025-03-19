package desktop

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface
type MockHTTPClient struct {
	GetFunc  func(url string) (*http.Response, error)
	PostFunc func(url, contentType string, body io.Reader) (*http.Response, error)
	DoFunc   func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	return m.GetFunc(url)
}

func (m *MockHTTPClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return m.PostFunc(url, contentType, body)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// MockDockerClient is a mock implementation of the DockerClient interface
type MockDockerClient struct {
	httpClient HTTPClient
}

func (m *MockDockerClient) HTTPClient() HTTPClient {
	return m.httpClient
}

// NewTestClient creates a new Client with a mock Docker client for testing
func NewTestClient(mockHTTPClient *MockHTTPClient) *Client {
	mockDockerClient := &MockDockerClient{
		httpClient: mockHTTPClient,
	}
	return &Client{
		dockerClient:    mockDockerClient,
		inferencePrefix: "/inference",
		modelsPrefix:    "/models",
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus string
		expectError    bool
	}{
		{
			name:           "running",
			statusCode:     http.StatusOK,
			expectedStatus: "Docker Model Runner is running",
			expectError:    false,
		},
		{
			name:           "not running",
			statusCode:     http.StatusServiceUnavailable,
			expectedStatus: "Docker Model Runner is not running",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTPClient := &MockHTTPClient{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				},
			}

			client := NewTestClient(mockHTTPClient)
			status, err := client.Status()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if status != tt.expectedStatus {
				t.Errorf("Expected status %q but got %q", tt.expectedStatus, status)
			}
		})
	}
}

func TestList(t *testing.T) {
	mockResponse := `[
		{
			"id": "sha256:1234567890abcdef",
			"tags": ["model1:latest"],
			"created": 1647270123,
			"config": {
				"format": "gguf",
				"quantization": "Q4_K_M",
				"parameters": "7B",
				"architecture": "llama",
				"size": "4.2GB"
			}
		},
		{
			"id": "sha256:0987654321fedcba",
			"tags": ["model2:latest"],
			"created": 1647270123,
			"config": {
				"format": "gguf",
				"quantization": "IQ4_XS",
				"parameters": "1.24 B",
				"architecture": "llama",
				"size": "584.87 MiB"
			}
		}
	]`

	tests := []struct {
		name             string
		jsonFormat       bool
		openai           bool
		model            string
		mockResponse     string
		expectedOutput   []string
		unexpectedOutput []string
	}{
		{
			name:         "json format",
			jsonFormat:   true,
			openai:       false,
			model:        "",
			mockResponse: mockResponse,
			expectedOutput: []string{
				"model1:latest",
				"model2:latest",
				"sha256:1234567890abcdef",
				"sha256:0987654321fedcba",
				"Q4_K_M",
				"IQ4_XS",
				"7B",
				"1.24 B",
			},
		},
		{
			name:         "table format",
			jsonFormat:   false,
			openai:       false,
			model:        "",
			mockResponse: mockResponse,
			expectedOutput: []string{
				"MODEL",
				"PARAMETERS",
				"QUANTIZATION",
				"ARCHITECTURE",
				"FORMAT",
				"MODEL ID",
				"CREATED",
				"SIZE",
				"model1:latest",
				"model2:latest",
				"7B",
				"1.24 B",
				"Q4_K_M",
				"IQ4_XS",
				"llama",
				"gguf",
			},
			// Table format should not contain the full SHA256 IDs
			unexpectedOutput: []string{
				"sha256:1234567890abcdef",
				"sha256:0987654321fedcba",
			},
		},
		// TODO what should happen when openai is true? Can openai be true and json format false?
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTPClient := &MockHTTPClient{
				GetFunc: func(url string) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(tt.mockResponse)),
					}, nil
				},
			}

			client := NewTestClient(mockHTTPClient)
			result, err := client.List(tt.jsonFormat, tt.openai, tt.model)

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q but got: %s", expected, result)
				}
			}

			if tt.unexpectedOutput != nil {
				for _, unexpected := range tt.unexpectedOutput {
					if strings.Contains(result, unexpected) {
						t.Errorf("Expected result NOT to contain %q but it did: %s", unexpected, result)
					}
				}
			}

			// For table format, verify that it has the expected structure
			if !tt.jsonFormat && !tt.openai && tt.model == "" {
				// Check that the output has multiple lines
				lines := strings.Split(strings.TrimSpace(result), "\n")
				if len(lines) < 3 {
					t.Errorf("Expected table output to have at least 3 lines, got %d", len(lines))
				}

				// Check that the first line contains all the column headers
				headers := []string{"MODEL", "PARAMETERS", "QUANTIZATION", "ARCHITECTURE", "FORMAT", "MODEL ID", "CREATED", "SIZE"}
				for _, header := range headers {
					if !strings.Contains(lines[0], header) {
						t.Errorf("Expected first line to contain header %q but got: %s", header, lines[0])
					}
				}

				// Check that the model names are in the output
				modelFound := false
				for _, line := range lines {
					if strings.Contains(line, "model1:latest") || strings.Contains(line, "model2:latest") {
						modelFound = true
						break
					}
				}
				if !modelFound {
					t.Errorf("Expected to find model names in the table output")
				}
			}
		})
	}
}

func TestPull(t *testing.T) {
	mockHTTPClient := &MockHTTPClient{
		PostFunc: func(url, contentType string, body io.Reader) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("Progress: 100%")),
			}, nil
		},
	}

	client := NewTestClient(mockHTTPClient)
	result, err := client.Pull("test-model")

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	expectedResult := "Model test-model pulled successfully"
	if result != expectedResult {
		t.Errorf("Expected %q but got %q", expectedResult, result)
	}
}

func TestRemove(t *testing.T) {
	mockHTTPClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodDelete {
				t.Errorf("Expected DELETE request but got %s", req.Method)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		},
	}

	client := NewTestClient(mockHTTPClient)
	result, err := client.Remove("test-model")

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	expectedResult := "Model test-model removed successfully"
	if result != expectedResult {
		t.Errorf("Expected %q but got %q", expectedResult, result)
	}
}

func TestURL(t *testing.T) {
	client := &Client{
		dockerClient:    &MockDockerClient{httpClient: &MockHTTPClient{}},
		inferencePrefix: "/custom/inference",
		modelsPrefix:    "/custom/models",
	}

	// Test URL construction
	url := client.url("/custom/models/test")

	// Just check that the URL contains the expected path and host
	if !strings.Contains(url, "http://localhost") {
		t.Errorf("Expected URL to contain 'http://localhost', got %q", url)
	}

	if !strings.Contains(url, "/custom/models/test") {
		t.Errorf("Expected URL to contain '/custom/models/test', got %q", url)
	}
}

func TestChat(t *testing.T) {
	// Mock streaming response
	mockStreamResponse := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677858242,"model":"gpt-3.5-turbo","choices":[{"delta":{"content":"Hello"},"index":0,"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677858242,"model":"gpt-3.5-turbo","choices":[{"delta":{"content":" world"},"index":0,"finish_reason":null}]}
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677858242,"model":"gpt-3.5-turbo","choices":[{"delta":{"content":"!"},"index":0,"finish_reason":null}]}
data: [DONE]
`

	mockHTTPClient := &MockHTTPClient{
		PostFunc: func(url, contentType string, body io.Reader) (*http.Response, error) {
			// Verify the request URL contains the expected path
			if !strings.Contains(url, "/v1/chat/completions") {
				t.Errorf("Expected URL to contain '/v1/chat/completions', got %s", url)
			}

			// Verify content type is application/json
			if contentType != "application/json" {
				t.Errorf("Expected content type 'application/json', got %s", contentType)
			}

			// Read and verify request body
			bodyBytes, err := io.ReadAll(body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
			}

			bodyStr := string(bodyBytes)
			if !strings.Contains(bodyStr, "test-model") {
				t.Errorf("Expected request body to contain model name, got %s", bodyStr)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(mockStreamResponse)),
			}, nil
		},
	}

	client := NewTestClient(mockHTTPClient)
	err := client.Chat("test-model", "Hello, how are you?")

	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}
