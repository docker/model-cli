package commands

import (
	"fmt"

	"github.com/docker/model-runner/pkg/inference/scheduling"
	"github.com/spf13/cobra"
)

func newConfigureCmd() *cobra.Command {
	var opts scheduling.ConfigureRequest

	c := &cobra.Command{
		Use:   "configure [--context-size=<n>] MODEL [-- <runtime-flags...>]",
		Short: "Configure runtime options for a model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf(
					"Model specification is required.\n\n" +
						"See 'docker model configure --help' for more information",
				)
			}
			opts.Model = args[0]
			opts.RuntimeFlags = args[1:]
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return desktopClient.ConfigureBackend(opts)
		},
	}

	c.Flags().Int64Var(&opts.ContextSize, "context-size", -1, "context size (in tokens)")
	return c
}
