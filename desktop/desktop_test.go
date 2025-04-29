package desktop

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/docker/pinata/common/pkg/inference/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestPullHuggingFaceModel(t *testing.T) {
	// Test case for pulling a Hugging Face model with mixed case
	modelName := "hf.co/Bartowski/Llama-3.2-1B-Instruct-GGUF"
	expectedLowercase := "hf.co/bartowski/llama-3.2-1b-instruct-gguf"

	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is converted to lowercase in the request
				var reqBody models.ModelCreateRequest
				err := json.NewDecoder(req.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, expectedLowercase, reqBody.From)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"type":"success","message":"Model pulled successfully"}`)),
				}, nil
			},
		},
	}

	_, _, err := client.Pull(modelName, func(s string) {})
	assert.NoError(t, err)
}

func TestChatHuggingFaceModel(t *testing.T) {
	// Test case for chatting with a Hugging Face model with mixed case
	modelName := "hf.co/Bartowski/Llama-3.2-1B-Instruct-GGUF"
	expectedLowercase := "hf.co/bartowski/llama-3.2-1b-instruct-gguf"
	prompt := "Hello"

	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is converted to lowercase in the request
				var reqBody OpenAIChatRequest
				err := json.NewDecoder(req.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, expectedLowercase, reqBody.Model)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("data: {\"choices\":[{\"delta\":{\"content\":\"Hello there!\"}}]}\n")),
				}, nil
			},
		},
	}

	err := client.Chat(modelName, prompt)
	assert.NoError(t, err)
}

func TestInspectHuggingFaceModel(t *testing.T) {
	// Test case for inspecting a Hugging Face model with mixed case
	modelName := "hf.co/Bartowski/Llama-3.2-1B-Instruct-GGUF"
	expectedLowercase := "hf.co/bartowski/llama-3.2-1b-instruct-gguf"

	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is converted to lowercase in the request URL
				assert.Contains(t, req.URL.Path, expectedLowercase)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewBufferString(`{
						"id": "sha256:123456789012",
						"tags": ["` + expectedLowercase + `"],
						"created": 1234567890,
						"config": {
							"format": "gguf",
							"quantization": "Q4_K_M",
							"parameters": "1B",
							"architecture": "llama",
							"size": "1.2GB"
						}
					}`)),
				}, nil
			},
		},
	}

	model, err := client.Inspect(modelName)
	assert.NoError(t, err)
	assert.Equal(t, expectedLowercase, model.Tags[0])
}

func TestNonHuggingFaceModel(t *testing.T) {
	// Test case for a non-Hugging Face model (should not be converted to lowercase)
	modelName := "docker.io/library/llama2"
	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is not converted to lowercase
				var reqBody models.ModelCreateRequest
				err := json.NewDecoder(req.Body).Decode(&reqBody)
				require.NoError(t, err)
				assert.Equal(t, modelName, reqBody.From)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"type":"success","message":"Model pulled successfully"}`)),
				}, nil
			},
		},
	}

	_, _, err := client.Pull(modelName, func(s string) {})
	assert.NoError(t, err)
}

func TestPushHuggingFaceModel(t *testing.T) {
	// Test case for pushing a Hugging Face model with mixed case
	modelName := "hf.co/Bartowski/Llama-3.2-1B-Instruct-GGUF"
	expectedLowercase := "hf.co/bartowski/llama-3.2-1b-instruct-gguf"

	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is converted to lowercase in the request URL
				assert.Contains(t, req.URL.Path, expectedLowercase)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"type":"success","message":"Model pushed successfully"}`)),
				}, nil
			},
		},
	}

	_, _, err := client.Push(modelName, func(s string) {})
	assert.NoError(t, err)
}

func TestRemoveHuggingFaceModel(t *testing.T) {
	// Test case for removing a Hugging Face model with mixed case
	modelName := "hf.co/Bartowski/Llama-3.2-1B-Instruct-GGUF"
	expectedLowercase := "hf.co/bartowski/llama-3.2-1b-instruct-gguf"

	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is converted to lowercase in the request URL
				assert.Contains(t, req.URL.Path, expectedLowercase)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("Model removed successfully")),
				}, nil
			},
		},
	}

	_, err := client.Remove([]string{modelName}, false)
	assert.NoError(t, err)
}

func TestTagHuggingFaceModel(t *testing.T) {
	// Test case for tagging a Hugging Face model with mixed case
	sourceModel := "hf.co/Bartowski/Llama-3.2-1B-Instruct-GGUF"
	expectedLowercase := "hf.co/bartowski/llama-3.2-1b-instruct-gguf"
	targetRepo := "myrepo"
	targetTag := "latest"

	client := &Client{
		dockerClient: &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify the model name is converted to lowercase in the request URL
				assert.Contains(t, req.URL.Path, expectedLowercase)

				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewBufferString("Tag created successfully")),
				}, nil
			},
		},
	}

	_, err := client.Tag(sourceModel, targetRepo, targetTag)
	assert.NoError(t, err)
}
