package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/docker/model-cli/desktop"
	"github.com/spf13/cobra"
)

func newComposeCmd() *cobra.Command {

	c := &cobra.Command{
		Use: "compose EVENT",
	}
	c.AddCommand(newUpCommand())
	c.AddCommand(newDownCommand())
	c.Hidden = true
	c.PersistentFlags().String("project-name", "", "compose project name") // unused by model

	return c
}

func newUpCommand() *cobra.Command {
	var models []string
	c := &cobra.Command{
		Use: "up",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(models) == 0 {
				err := errors.New("options.model is required")
				_ = sendError(err.Error())
				return err
			}

			kind := modelRunner.EngineKind()
			standalone, err := ensureStandaloneRunnerAvailable(cmd.Context(), nil)
			if err != nil {
				_ = sendErrorf("Failed to initialize standalone model runner: %v", err)
				return fmt.Errorf("Failed to initialize standalone model runner: %w", err)
			} else if (kind == desktop.ModelRunnerEngineKindMoby || kind == desktop.ModelRunnerEngineKindCloud) &&
				standalone == nil || standalone.gatewayIP == "" || standalone.gatewayPort == 0 {
				return errors.New("unable to determine standalone runner endpoint")
			}

			if err := downloadModelsOnlyIfNotFound(desktopClient, models); err != nil {
				return err
			}

			switch kind {
			case desktop.ModelRunnerEngineKindDesktop:
				_ = setenv("URL", "http://model-runner.docker.internal/engines/v1/")
			case desktop.ModelRunnerEngineKindMobyManual:
				_ = setenv("URL", modelRunner.URL("/engines/v1/"))
			case desktop.ModelRunnerEngineKindCloud:
				fallthrough
			case desktop.ModelRunnerEngineKindMoby:
				_ = setenv("URL", fmt.Sprintf("http://%s:%d/engines/v1", standalone.gatewayIP, standalone.gatewayPort))
			default:
				return fmt.Errorf("unhandled engine kind: %v", kind)
			}
			return nil
		},
	}
	c.Flags().StringArrayVar(&models, "model", nil, "model to use")
	return c
}

func newDownCommand() *cobra.Command {
	var model []string
	c := &cobra.Command{
		Use: "down",
		RunE: func(cmd *cobra.Command, args []string) error {
			// No required cleanup on down
			return nil
		},
	}
	c.Flags().StringArrayVar(&model, "model", nil, "model to use")
	return c
}

func downloadModelsOnlyIfNotFound(desktopClient *desktop.Client, models []string) error {
	modelsDownloaded, err := desktopClient.List()
	if err != nil {
		_ = sendErrorf("Failed to get models list: %v", err)
		return err
	}
	for _, model := range models {
		// Download the model if not already present in the local model store
		if !slices.ContainsFunc(modelsDownloaded, func(m desktop.Model) bool {
			if model == m.ID {
				return true
			}
			for _, tag := range m.Tags {
				if tag == model {
					return true
				}
			}
			return false
		}) {
			_, _, err = desktopClient.Pull(model, func(s string) {
				_ = sendInfo(s)
			})
			if err != nil {
				_ = sendErrorf("Failed to pull model: %v", err)
				return fmt.Errorf("Failed to pull model: %v\n", err)
			}
		}

	}
	_ = setenv("MODEL", strings.Join(models, ","))
	return nil
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
