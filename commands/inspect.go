package commands

import (
	"fmt"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/commands/formatter"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var openai bool
	c := &cobra.Command{
		Use:   "inspect MODEL",
		Short: "Display detailed information on one model",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf(
					"'docker model inspect' requires 1 argument.\n\n" +
						"Usage:  docker model inspect MODEL\n\n" +
						"See 'docker model inspect --help' for more information",
				)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			inspectedModel, err := inspectModel(args, openai, desktopClient)
			if err != nil {
				return err
			}
			cmd.Print(inspectedModel)
			return nil
		},
		ValidArgsFunction: completion.ModelNames(desktopClient, 1),
	}
	c.Flags().BoolVar(&openai, "openai", false, "List model in an OpenAI format")
	return c
}

func inspectModel(args []string, openai bool, desktopClient *desktop.Client) (string, error) {
	modelName := args[0]
	if openai {
		model, err := desktopClient.InspectOpenAI(modelName)
		if err != nil {
			err = handleClientError(err, "Failed to get model "+modelName)
			return "", handleNotRunningError(err)
		}
		return formatter.ToStandardJSON(model)
	}
	model, err := desktopClient.Inspect(modelName)
	if err != nil {
		err = handleClientError(err, "Failed to get model "+modelName)
		return "", handleNotRunningError(err)
	}
	return formatter.ToStandardJSON(model)
}
