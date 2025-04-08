package commands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newComposeCmd(desktopClient *desktop.Client) *cobra.Command {

	c := &cobra.Command{
		Use: "compose EVENT",
	}
	c.AddCommand(newUpCommand(desktopClient))
	c.AddCommand(newDownCommand())
	c.Hidden = true
	c.Flags().String("project-name", "", "compose project name") // unused by model

	return c
}

func newUpCommand(desktopClient *desktop.Client) *cobra.Command {
	var model string
	c := &cobra.Command{
		Use: "up",
		RunE: func(cmd *cobra.Command, args []string) error {
			if model == "" {
				err := errors.New("options.model is required")
				sendError(err.Error())
				return err
			}

			_, err := desktopClient.Pull(model, func(s string) {
				sendInfo(s)
			})
			if err != nil {
				sendErrorf("Failed to pull model", err)
				return fmt.Errorf("Failed to pull model: %v\n", err)
			}

			// FIXME get actual URL from Docker Desktop
			setenv("URL", "http://model-runner.docker.internal/engines/v1/")
			setenv("MODEL", model)

			return nil
		},
	}
	c.Flags().StringVar(&model, "model", "", "model to use")
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
