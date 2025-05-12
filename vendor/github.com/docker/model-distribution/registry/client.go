package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/docker/model-distribution/internal/progress"
	"github.com/docker/model-distribution/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	DefaultUserAgent = "model-distribution"
)

var (
	DefaultTransport = remote.DefaultTransport
)

type Client struct {
	transport http.RoundTripper
	userAgent string
	keychain  authn.Keychain
}

type ClientOption func(*Client)

func WithTransport(transport http.RoundTripper) ClientOption {
	return func(c *Client) {
		if transport != nil {
			c.transport = transport
		}
	}
}

func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		if userAgent != "" {
			c.userAgent = userAgent
		}
	}
}

func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		transport: remote.DefaultTransport,
		userAgent: DefaultUserAgent,
		keychain:  authn.DefaultKeychain,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

func (c *Client) Model(ctx context.Context, reference string) (types.ModelArtifact, error) {
	// Parse the reference
	ref, err := name.ParseReference(reference)
	if err != nil {
		return nil, NewReferenceError(reference, err)
	}

	// Return the artifact at the given reference
	remoteImg, err := remote.Image(ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(c.keychain),
		remote.WithTransport(c.transport),
		remote.WithUserAgent(c.userAgent),
	)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "UNAUTHORIZED") {
			return nil, NewRegistryError(reference, "UNAUTHORIZED", "Authentication required for this model", err)
		}
		if strings.Contains(errStr, "MANIFEST_UNKNOWN") {
			return nil, NewRegistryError(reference, "MANIFEST_UNKNOWN", "Model not found", err)
		}
		if strings.Contains(errStr, "NAME_UNKNOWN") {
			return nil, NewRegistryError(reference, "NAME_UNKNOWN", "Repository not found", err)
		}
		return nil, NewRegistryError(reference, "UNKNOWN", err.Error(), err)
	}
	return &artifact{remoteImg}, nil
}

type Target struct {
	reference name.Reference
	transport http.RoundTripper
	userAgent string
	keychain  authn.Keychain
}

func (c *Client) NewTarget(tag string) (*Target, error) {
	ref, err := name.NewTag(tag)
	if err != nil {
		return nil, fmt.Errorf("invalid tag: %q: %w", tag, err)
	}
	return &Target{
		reference: ref,
		transport: c.transport,
		userAgent: c.userAgent,
		keychain:  c.keychain,
	}, nil
}

func (t *Target) Write(ctx context.Context, model types.ModelArtifact, progressWriter io.Writer) error {
	pr := progress.NewProgressReporter(progressWriter, progress.PushMsg)
	defer pr.Wait()

	if err := remote.Write(t.reference, model,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(t.keychain),
		remote.WithTransport(t.transport),
		remote.WithUserAgent(t.userAgent),
		remote.WithProgress(pr.Updates()),
	); err != nil {
		return fmt.Errorf("write to registry %q: %w", t.reference.String(), err)
	}
	return nil
}
