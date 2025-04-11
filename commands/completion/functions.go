package completion

import (
	"encoding/json"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func NoComplete(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// ModelNames offers completion for models present within the local store.
func ModelNames(desktopClient *desktop.Client, limit int) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if limit > 0 && len(args) >= limit {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		modelsString, err := desktopClient.List(true, false, false, "")
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var models []desktop.Model
		if err := json.Unmarshal([]byte(modelsString), &models); err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for _, m := range models {
			names = append(names, m.Tags...)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
