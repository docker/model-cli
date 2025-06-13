package commands

import (
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func newTagCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "tag SOURCE TARGET",
		Short: "Tag a model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf(
					"'docker model tag' requires 2 arguments.\n\n" +
						"Usage:  docker model tag SOURCE TARGET\n\n" +
						"See 'docker model tag --help' for more information",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}
			return tagModel(cmd, desktopClient, args[0], args[1])
		},
	}
	return c
}

func tagModel(cmd *cobra.Command, desktopClient *desktop.Client, source, target string) error {
	// Parse the target to extract repo and tag
	tag, err := name.NewTag(target)
	if err != nil {
		return fmt.Errorf("invalid tag: %w", err)
	}
	targetRepo := tag.Repository.String()
	targetTag := tag.TagStr()

	// Make the POST request
	resp, err := desktopClient.Tag(source, targetRepo, targetTag)
	if err != nil {
		return fmt.Errorf("failed to tag model: %w", err)
	}

	cmd.Println(resp)
	return nil
}
