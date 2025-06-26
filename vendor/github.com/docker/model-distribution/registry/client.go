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
	auth      authn.Authenticator
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

func WithAuthConfig(username, password string) ClientOption {
	return func(c *Client) {
		if username != "" && password != "" {
			c.auth = &authn.Basic{
				Username: username,
				Password: password,
			}
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

	// Set up authentication options
	authOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(c.transport),
		remote.WithUserAgent(c.userAgent),
	}

	// Use direct auth if provided, otherwise fall back to keychain
	if c.auth != nil {
		authOpts = append(authOpts, remote.WithAuth(c.auth))
	} else {
		authOpts = append(authOpts, remote.WithAuthFromKeychain(c.keychain))
	}

	// Return the artifact at the given reference
	remoteImg, err := remote.Image(ref, authOpts...)
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
	auth      authn.Authenticator
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
		auth:      c.auth,
	}, nil
}

func (t *Target) Write(ctx context.Context, model types.ModelArtifact, progressWriter io.Writer) error {
	pr := progress.NewProgressReporter(progressWriter, progress.PushMsg, nil)
	defer pr.Wait()

	// Set up authentication options
	authOpts := []remote.Option{
		remote.WithContext(ctx),
		remote.WithTransport(t.transport),
		remote.WithUserAgent(t.userAgent),
		remote.WithProgress(pr.Updates()),
	}

	// Use direct auth if provided, otherwise fall back to keychain
	if t.auth != nil {
		authOpts = append(authOpts, remote.WithAuth(t.auth))
	} else {
		authOpts = append(authOpts, remote.WithAuthFromKeychain(t.keychain))
	}

	if err := remote.Write(t.reference, model, authOpts...); err != nil {
		return fmt.Errorf("write to registry %q: %w", t.reference.String(), err)
	}
	return nil
}
