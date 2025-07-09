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

// readMultilineInput reads input from stdin, supporting both single-line and multiline input.
// For multiline input, it detects triple-quoted strings and shows continuation prompts.
func readMultilineInput(cmd *cobra.Command, scanner *bufio.Scanner) (string, error) {
	cmd.Print("> ")

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading input: %v", err)
		}
		return "", fmt.Errorf("EOF")
	}

	line := scanner.Text()

	// Check if this is the start of a multiline input (triple quotes)
	tripleQuoteStart := ""
	if strings.HasPrefix(line, `"""`) {
		tripleQuoteStart = `"""`
	} else if strings.HasPrefix(line, "'''") {
		tripleQuoteStart = "'''"
	}

	// If no triple quotes, return a single line
	if tripleQuoteStart == "" {
		return line, nil
	}

	// Check if the triple quotes are closed on the same line
	restOfLine := line[3:]
	if strings.HasSuffix(restOfLine, tripleQuoteStart) && len(restOfLine) >= 3 {
		// Complete multiline string on single line
		return line, nil
	}

	// Start collecting multiline input
	var multilineInput strings.Builder
	multilineInput.WriteString(line)
	multilineInput.WriteString("\n")

	// Continue reading lines until we find the closing triple quotes
	for {
		cmd.Print("... ")

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("error reading input: %v", err)
			}
			return "", fmt.Errorf("unclosed multiline input (EOF)")
		}

		line = scanner.Text()
		multilineInput.WriteString(line)

		// Check if this line contains the closing triple quotes
		if strings.Contains(line, tripleQuoteStart) {
			// Found closing quotes, we're done
			break
		}

		multilineInput.WriteString("\n")
	}

	return multilineInput.String(), nil
}

func newRunCmd() *cobra.Command {
	var debug bool

	const cmdArgs = "MODEL [PROMPT]"
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

			_, err := desktopClient.Inspect(model, false)
			if err != nil {
				if !errors.Is(err, desktop.ErrNotFound) {
					return handleNotRunningError(handleClientError(err, "Failed to inspect model"))
				}
				cmd.Println("Unable to find model '" + model + "' locally. Pulling from the server.")
				if err := pullModel(cmd, desktopClient, model); err != nil {
					return err
				}
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

			for {
				userInput, err := readMultilineInput(cmd, scanner)
				if err != nil {
					if err.Error() == "EOF" {
						cmd.Println("\nChat session ended.")
						break
					}
					return fmt.Errorf("Error reading input: %v", err)
				}

				if strings.ToLower(strings.TrimSpace(userInput)) == "/bye" {
					cmd.Println("Chat session ended.")
					break
				}

				if strings.TrimSpace(userInput) == "" {
					continue
				}

				if err := desktopClient.Chat(model, userInput); err != nil {
					cmd.PrintErr(handleClientError(err, "Failed to generate a response"))
					continue
				}

				cmd.Println()
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
