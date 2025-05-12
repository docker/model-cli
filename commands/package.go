package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"path/filepath"

	"github.com/docker/model-cli/desktop"
	"github.com/docker/model-distribution/builder"
	"github.com/docker/model-distribution/registry"
	"github.com/spf13/cobra"
)

func newPackagedCmd() *cobra.Command {
	var opts packageOptions

	c := &cobra.Command{
		Use:   "package --gguf <path> [--license <path>...] --push TARGET",
		Short: "package a model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model package' requires 1 argument.\n\n"+
						"Usage:  %s\n\n"+
						"See 'docker model package --help' for more information",
					cmd.Use,
				)
			}
			if opts.push != true {
				return fmt.Errorf(
					"This version of 'docker model package' requires --push and will write the resulting package directly to the registry.\n\n" +
						"See 'docker model package --help' for more information",
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
			if err := packageModel(cmd, args[0], opts); err != nil {
				cmd.PrintErrln("Failed to package model")
				return fmt.Errorf("package model: %w", err)
			}
			return nil
		},
	}

	c.Flags().StringVar(&opts.ggufPath, "gguf", "", "absolute path to gguf file (required)")
	c.Flags().StringArrayVarP(&opts.licensePaths, "license", "l", nil, "absolute path to a license file")
	c.Flags().BoolVar(&opts.push, "push", false, "push to registry (required)")
	return c
}

type packageOptions struct {
	ggufPath     string
	licensePaths []string
	push         bool
}

func packageModel(cmd *cobra.Command, tag string, opts packageOptions) error {
	// Parse the reference
	cmd.PrintErrln("Packaging model %q\n", tag)
	target, err := registry.NewClient(
		registry.WithUserAgent("model-cli"),
	).NewTarget(tag)
	if err != nil {
		return err
	}

	// Create package builder with GGUF file
	cmd.PrintErrf("Adding GGUF file from %q\n", opts.ggufPath)
	pkg, err := builder.FromGGUF(opts.ggufPath)
	if err != nil {
		return fmt.Errorf("add gguf file: %w", err)
	}

	// Add license files
	for _, path := range opts.licensePaths {
		cmd.PrintErrf("Adding license file from %q\n", path)
		pkg, err = pkg.WithLicense(path)
		if err != nil {
			return fmt.Errorf("add license file: %w", err)
		}
	}

	// Write the artifact to the registry
	cmd.PrintErrln("Pushing to registry...")
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
		return fmt.Errorf("push: %w", err)
	}
	cmd.PrintErrln("Model pushed successfully")
	return nil
}
