package commands

import (
	"fmt"

	"github.com/docker/pinata/common/cmd/docker-model/desktop"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "status",
		Short: "Check if the Docker Model Runner is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := desktop.New()
			if err != nil {
				return fmt.Errorf("Failed to create Docker client: %v\n", err)
			}
			status, err := client.Status()
			if err != nil {
				return fmt.Errorf("Failed to get Docker Model Runner status: %v\n", err)
			}
			cmd.Println(status)
			return nil
		},
	}
	return c
}
