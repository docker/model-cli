package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"path/filepath"

	"github.com/docker/model-distribution/builder"
	"github.com/docker/model-distribution/registry"
	"github.com/docker/model-distribution/tarball"
	"github.com/docker/model-distribution/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
)

func newPackagedCmd() *cobra.Command {
	var opts packageOptions

	c := &cobra.Command{
		Use:   "package --gguf <path> [--license <path>...] [--context-size <tokens>] [--push] [<tag>]",
		Short: "Package a GGUF file into a Docker model OCI artifact, with optional licenses. The package is sent to the model-runner, unless --push is specified",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model package' requires 1 argument.\n\n"+
						"Usage:  %s\n\n"+
						"See 'docker model package --help' for more information",
					cmd.Use,
				)
			}
			if opts.ggufPath == "" {
				return fmt.Errorf(
					"GGUF path is required.\n\n" +
						"See 'docker model package --help' for more information",
				)
			}
			if !filepath.IsAbs(opts.ggufPath) {
				return fmt.Errorf(
					"GGUF path must be absolute.\n\n" +
						"See 'docker model package --help' for more information",
				)
			}
			opts.ggufPath = filepath.Clean(opts.ggufPath)

			for i, l := range opts.licensePaths {
				if !filepath.IsAbs(l) {
					return fmt.Errorf(
						"license path must be absolute.\n\n" +
							"See 'docker model package --help' for more information",
					)
				}
				opts.licensePaths[i] = filepath.Clean(l)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := packageModel(cmd, opts); err != nil {
				cmd.PrintErrln("Failed to package model")
				return fmt.Errorf("package model: %w", err)
			}
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}

	c.Flags().StringVar(&opts.ggufPath, "gguf", "", "absolute path to gguf file (required)")
	c.Flags().StringArrayVarP(&opts.licensePaths, "license", "l", nil, "absolute path to a license file")
	c.Flags().BoolVar(&opts.push, "push", false, "push to registry (if not set, the model is loaded into the Model Runner content store.")
	c.Flags().Uint64Var(&opts.contextSize, "context-size", 0, "context size in tokens")
	return c
}

type packageOptions struct {
	ggufPath     string
	licensePaths []string
	push         bool
	contextSize  uint64
	tag          string
}

func packageModel(cmd *cobra.Command, opts packageOptions) error {
	var (
		target builder.Target
		err    error
	)
	if opts.push {
		target, err = registry.NewClient(
			registry.WithUserAgent("docker-model-cli/" + desktop.Version),
		).NewTarget(opts.tag)
	} else {
		target, err = newModelRunnerTarget(desktopClient, opts.tag)
	}
	if err != nil {
		return err
	}

	// Create package builder with GGUF file
	cmd.PrintErrf("Adding GGUF file from %q\n", opts.ggufPath)
	pkg, err := builder.FromGGUF(opts.ggufPath)
	if err != nil {
		return fmt.Errorf("add gguf file: %w", err)
	}

	// Set context size
	if opts.contextSize > 0 {
		cmd.PrintErrf("Setting context size %d\n", opts.contextSize)
		pkg = pkg.WithContextSize(opts.contextSize)
	}

	// Add license files
	for _, path := range opts.licensePaths {
		cmd.PrintErrf("Adding license file from %q\n", path)
		pkg, err = pkg.WithLicense(path)
		if err != nil {
			return fmt.Errorf("add license file: %w", err)
		}
	}

	if opts.push {
		cmd.PrintErrln("Pushing model to registry...")
	} else {
		cmd.PrintErrln("Loading model to Model Runner...")
	}
	pr, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		defer pw.Close()
		done <- pkg.Build(cmd.Context(), target, pw)
	}()

	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		// Parse the progress message
		var progressMsg desktop.ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			cmd.PrintErrln("Error displaying progress:", err)
		}

		// Print progress messages
		TUIProgress(progressMsg.Message)
	}
	cmd.PrintErrln("") // newline after progress

	if err := scanner.Err(); err != nil {
		cmd.PrintErrln("Error streaming progress:", err)
	}
	if err := <-done; err != nil {
		if opts.push {
			return fmt.Errorf("failed to save packaged model: %w", err)
		}
		return fmt.Errorf("failed to load packaged model: %w", err)
	}

	if opts.push {
		cmd.PrintErrln("Model pushed successfully")
	} else {
		cmd.PrintErrln("Model loaded successfully")
	}
	return nil
}

// modelRunnerTarget loads model to Docker Model Runner via models/load endpoint
type modelRunnerTarget struct {
	client *desktop.Client
	tag    name.Tag
}

func newModelRunnerTarget(client *desktop.Client, tag string) (*modelRunnerTarget, error) {
	target := &modelRunnerTarget{
		client: client,
	}
	if tag != "" {
		var err error
		target.tag, err = name.NewTag(tag)
		if err != nil {
			return nil, fmt.Errorf("invalid tag: %w", err)
		}
	}
	return target, nil
}

func (t *modelRunnerTarget) Write(ctx context.Context, mdl types.ModelArtifact, progressWriter io.Writer) error {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		target, err := tarball.NewTarget(pw)
		if err != nil {
			errCh <- err
			return
		}
		errCh <- target.Write(ctx, mdl, progressWriter)
	}()

	loadErr := t.client.LoadModel(ctx, pr)
	writeErr := <-errCh

	if loadErr != nil {
		return fmt.Errorf("loading model archive: %w", loadErr)
	}
	if writeErr != nil {
		return fmt.Errorf("writing model archive: %w", writeErr)
	}
	id, err := mdl.ID()
	if err != nil {
		return fmt.Errorf("get model ID: %w", err)
	}
	if t.tag.String() != "" {
		if err := desktopClient.Tag(id, parseRepo(t.tag), t.tag.TagStr()); err != nil {
			return fmt.Errorf("tag model: %w", err)
		}
	}
	return nil
}
