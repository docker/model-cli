package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var debug bool

	cmdArgs := "MODEL [PROMPT]"
	c := &cobra.Command{
		Use:   "run " + cmdArgs,
		Short: "Run a model and interact with it using a submitted prompt or chat mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := args[0]
			prompt := ""
			if len(args) == 1 {
				if debug {
					cmd.Printf("Running model %s\n", model)
				}
			} else {
				prompt = args[1]
				if debug {
					cmd.Printf("Running model %s with prompt %s\n", model, prompt)
				}
			}

			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}

			modelDetail, err := desktopClient.Inspect(model, false)
			if err != nil {
				if !errors.Is(err, desktop.ErrNotFound) {
					return handleNotRunningError(handleClientError(err, "Failed to list models"))
				}
				cmd.Println("Unable to find model '" + model + "' locally. Pulling from the server.")
				if err := pullModel(cmd, desktopClient, model); err != nil {
					return err
				}
			} else if model != modelDetail.Tags[0] {
				model = modelDetail.Tags[0]
			}

			if prompt != "" {
				if err := desktopClient.Chat(model, prompt); err != nil {
					return handleClientError(err, "Failed to generate a response")
				}
				cmd.Println()
				return nil
			}

			scanner := bufio.NewScanner(os.Stdin)
			cmd.Println("Interactive chat mode started. Type '/bye' to exit.")
			cmd.Print("> ")

			for scanner.Scan() {
				userInput := scanner.Text()

				if strings.ToLower(userInput) == "/bye" {
					cmd.Println("Chat session ended.")
					break
				}

				if strings.TrimSpace(userInput) == "" {
					cmd.Print("> ")
					continue
				}

				if err := desktopClient.Chat(model, userInput); err != nil {
					cmd.PrintErr(handleClientError(err, "Failed to generate a response"))
					cmd.Print("> ")
					continue
				}

				cmd.Print("\n> ")
			}

			if err := scanner.Err(); err != nil {
				return fmt.Errorf("Error reading input: %v\n", err)
			}
			return nil
		},
		ValidArgsFunction: completion.ModelNames(getDesktopClient, 1),
	}
	c.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf(
				"'docker model run' requires at least 1 argument.\n\n" +
					"Usage:  docker model run " + cmdArgs + "\n\n" +
					"See 'docker model run --help' for more information",
			)
		}
		if len(args) > 2 {
			return fmt.Errorf("too many arguments, expected " + cmdArgs)
		}
		return nil
	}

	c.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")

	return c
}
