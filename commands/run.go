package commands

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var debug bool

	cmdArgs := "MODEL [PROMPT]"
	c := &cobra.Command{
		Use:   "run " + cmdArgs,
		Short: "Run a model with the Docker Model Runner",
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

			client, err := desktop.New()
			if err != nil {
				return fmt.Errorf("Failed to create Docker client: %v\n", err)
			}

			if _, err := client.List(false, false, model); err != nil {
				if !errors.Is(err, desktop.ErrNotFound) {
					return fmt.Errorf("Failed to list model: %v\n", err)
				}
				cmd.Println("Unable to find model '" + model + "' locally. Pulling from the server.")
				response, err := client.Pull(model)
				if err != nil {
					return fmt.Errorf("Failed to pull model: %v\n", err)
				}
				cmd.Println(response)
			}

			if prompt != "" {
				if err := client.Chat(model, prompt); err != nil {
					return fmt.Errorf("Failed to generate a response: %v\n", err)
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

				if err := client.Chat(model, userInput); err != nil {
					cmd.PrintErrf("Failed to generate a response: %v\n", err)
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
