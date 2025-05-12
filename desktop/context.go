package desktop

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	clientpkg "github.com/docker/docker/client"
	"github.com/docker/model-runner/pkg/inference"
)

// isDesktopContext returns true if the CLI instance points to a Docker Desktop
// context and false otherwise.
func isDesktopContext(cli *command.DockerCli) bool {
	// We don't currently support Docker Model Runner in Docker Desktop for
	// Linux, so we won't treat that as a Docker Desktop case (though it will
	// still work as a standard Moby or Cloud case, depending on configuration).
	if runtime.GOOS == "linux" {
		return false
	}

	// Enforce that we're on macOS or Windows, just in case someone is running
	// a Docker client on (say) BSD.
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		return false
	}

	// Otherwise used name-based heuristics to identify the environment.
	name := cli.CurrentContext()
	if name == "desktop-linux" {
		return true
	} else if name == "default" && os.Getenv("DOCKER_HOST") == "" {
		return true
	} else if runtime.GOOS == "windows" && name == "desktop-windows" {
		// On Windows, we'll still target the Linux engine, even if the Windows
		// engine is currently active.
		return true
	}
	return false
}

// isCloudContext returns true if the CLI instance points to a Docker Cloud
// context and false otherwise.
func isCloudContext(cli *command.DockerCli) bool {
	rawMetadata, err := cli.ContextStore().GetMetadata(cli.CurrentContext())
	if err != nil {
		return false
	}
	metadata, err := command.GetDockerContext(rawMetadata)
	if err != nil {
		return false
	}
	_, ok := metadata.AdditionalFields["cloud.docker.com"]
	return ok
}

// dockerClientForContext creates a Docker client for the specified context.
func dockerClientForContext(cli *command.DockerCli, name string) (*clientpkg.Client, error) {
	c, err := cli.ContextStore().GetMetadata(name)
	if err != nil {
		return nil, fmt.Errorf("unable to load context metadata: %w", err)
	}
	endpoint, err := docker.EndpointFromContext(c)
	if err != nil {
		return nil, fmt.Errorf("unable to determine context endpoint: %w", err)
	}
	return clientpkg.NewClientWithOpts(clientpkg.FromEnv, clientpkg.WithHost(endpoint.Host))
}

// ModelRunnerEngineKind encodes the kind of Docker engine associated with the
// model runner context.
type ModelRunnerEngineKind uint8

const (
	// ModelRunnerEngineKindMoby represents a non-Desktop/Cloud engine on which
	// the Model CLI command is responsible for managing a Model Runner.
	ModelRunnerEngineKindMoby ModelRunnerEngineKind = iota
	// ModelRunnerEngineKindMobyManual represents a non-Desktop/Cloud engine
	// that's explicitly targeted by a DMR_HOST environment variable on which
	// the user is responsible for managing a Model Runner.
	ModelRunnerEngineKindMobyManual
	// ModelRunnerEngineKindDesktop represents a Docker Desktop engine. It only
	// refers to a Docker Desktop Linux engine, i.e. not a Windows container
	// engine in the case of Docker Desktop for Windows.
	ModelRunnerEngineKindDesktop
	// ModelRunnerEngineKindCloud represents a Docker Cloud engine.
	ModelRunnerEngineKindCloud
)

// String returns a human-readable engine kind description.
func (k ModelRunnerEngineKind) String() string {
	switch k {
	case ModelRunnerEngineKindMoby:
		return "Moby"
	case ModelRunnerEngineKindMobyManual:
		return "Moby (Manual Install)"
	case ModelRunnerEngineKindDesktop:
		return "Docker Desktop"
	case ModelRunnerEngineKindCloud:
		return "Docker Cloud"
	default:
		return "Unknown"
	}
}

// ModelRunnerContext encodes the operational context of a Model CLI command and
// provides facilities for inspecting and interacting with the Model Runner.
type ModelRunnerContext struct {
	// kind stores the associated engine kind.
	kind ModelRunnerEngineKind
	// urlPrefix is the prefix URL for all requests.
	urlPrefix *url.URL
	// client is the model runner client.
	client DockerHttpClient
}

// NewContextForMock is a ModelRunnerContext constructor exposed only for the
// purposes of mock testing.
func NewContextForMock(client DockerHttpClient) *ModelRunnerContext {
	urlPrefix, err := url.Parse("http://localhost" + inference.ExperimentalEndpointsPrefix)
	if err != nil {
		panic("error occurred while parsing known-good URL")
	}
	return &ModelRunnerContext{
		kind:      ModelRunnerEngineKindDesktop,
		urlPrefix: urlPrefix,
		client:    client,
	}
}

// DetectContext determines the current Docker Model Runner context.
func DetectContext(cli *command.DockerCli) (*ModelRunnerContext, error) {
	// Check for an explicit endpoint setting.
	dmrHost := os.Getenv("DMR_HOST")

	// Detect the associated engine type.
	kind := ModelRunnerEngineKindMoby
	if dmrHost != "" {
		kind = ModelRunnerEngineKindMobyManual
	} else if isDesktopContext(cli) {
		kind = ModelRunnerEngineKindDesktop
	} else if isCloudContext(cli) {
		kind = ModelRunnerEngineKindCloud
	}

	// Compute the URL prefix based on the associated engine kind.
	var rawURLPrefix string
	if kind == ModelRunnerEngineKindMoby {
		rawURLPrefix = "http://localhost:12434"
	} else if kind == ModelRunnerEngineKindMobyManual {
		rawURLPrefix = dmrHost
	} else if kind == ModelRunnerEngineKindDesktop {
		rawURLPrefix = "http://localhost" + inference.ExperimentalEndpointsPrefix
	} else { // ModelRunnerEngineKindCloud
		rawURLPrefix = "http://localhost/"
	}
	urlPrefix, err := url.Parse(rawURLPrefix)
	if err != nil {
		return nil, fmt.Errorf("invalid model runner URL (%s): %w", rawURLPrefix, err)
	}

	// Construct the HTTP client.
	var client DockerHttpClient
	if kind == ModelRunnerEngineKindMoby || kind == ModelRunnerEngineKindMobyManual {
		client = http.DefaultClient
	} else if kind == ModelRunnerEngineKindDesktop {
		dockerClient, err := dockerClientForContext(cli, "desktop-linux")
		if err != nil {
			return nil, fmt.Errorf("unable to create model runner client: %w", err)
		}
		client = dockerClient.HTTPClient()
	} else { // ModelRunnerEngineKindCloud
		dockerClient, err := dockerClientForContext(cli, cli.CurrentContext())
		if err != nil {
			return nil, fmt.Errorf("unable to create model runner client: %w", err)
		}
		client = dockerClient.HTTPClient()
	}

	// Success.
	return &ModelRunnerContext{
		kind:      kind,
		urlPrefix: urlPrefix,
		client:    client,
	}, nil
}

// EngineKind returns the Docker engine kind associated with the model runner.
func (c *ModelRunnerContext) EngineKind() ModelRunnerEngineKind {
	return c.kind
}

// URL constructs a URL string appropriate for the model runner.
func (c *ModelRunnerContext) URL(path string) string {
	return c.urlPrefix.JoinPath(path).String()
}

// Client returns an HTTP client appropriate for accessing the model runner.
func (c *ModelRunnerContext) Client() DockerHttpClient {
	return c.client
}
