package completion

import (
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func NoComplete(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// ModelNames offers completion for models present within the local store.
func ModelNames(desktopClient func() *desktop.Client, limit int) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// HACK: Invoke rootCmd's PersistentPreRunE, which is needed for context
		// detection and client initialization. This function isn't invoked
		// automatically on autocompletion paths.
		cmd.Parent().PersistentPreRunE(cmd, args)

		if limit > 0 && len(args) >= limit {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		models, err := desktopClient().List(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, m := range models {
			names = append(names, m.Tags...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
