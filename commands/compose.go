package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newComposeCmd() *cobra.Command {

	c := &cobra.Command{
		Use: "compose EVENT",
	}
	c.AddCommand(newUpCommand(desktopClient))
	c.AddCommand(newDownCommand())
	c.Hidden = true
	c.PersistentFlags().String("project-name", "", "compose project name") // unused by model

	return c
}

type Options struct {
	Model string `json:"model,omitempty"`
}

func newUpCommand(desktopClient *desktop.Client) *cobra.Command {
	c := &cobra.Command{
		Use: "up",
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts Options
			err := json.NewDecoder(os.Stdin).Decode(opts)
			if err != nil {
				sendError("failed to parse options")
				return err
			}
			if opts.Model == "" {
				sendError("options.model is required")
				return err
			}

			_, _, err = desktopClient.Pull(opts.Model, func(s string) {
				sendInfo(s)
			})
			if err != nil {
				sendErrorf("Failed to pull model: %v", err)
				return fmt.Errorf("Failed to pull model: %v\n", err)
			}

			// FIXME get actual URL from Docker Desktop
			setenv("URL", "http://model-runner.docker.internal/engines/v1/")
			setenv("MODEL", opts.Model)

			return nil
		},
	}
	return c
}

func newDownCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "down",
		RunE: func(cmd *cobra.Command, args []string) error {
			// No required cleanup on down
			return nil
		},
	}
	return c
}

type jsonMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func setenv(k, v string) error {
	marshal, err := json.Marshal(jsonMessage{
		Type:    "setenv",
		Message: fmt.Sprintf("%v=%v", k, v),
	})
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(marshal))
	return err
}

func sendErrorf(message string, args ...any) error {
	return sendError(fmt.Sprintf(message, args...))
}

func sendError(message string) error {
	marshal, err := json.Marshal(jsonMessage{
		Type:    "error",
		Message: message,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(marshal))
	return err
}

func sendInfo(s string) error {
	marshal, err := json.Marshal(jsonMessage{
		Type:    "info",
		Message: s,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(marshal))
	return err
}
