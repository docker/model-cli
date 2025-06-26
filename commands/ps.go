package commands

import (
	"bytes"
	"time"

	"github.com/docker/go-units"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newPSCmd() *cobra.Command {
	c := &cobra.Command{
		GroupID: groupIDModel,
		Use:     "ps",
		Short:   "List running models",
		RunE: func(cmd *cobra.Command, args []string) error {
			ps, err := desktopClient.PS()
			if err != nil {
				err = handleClientError(err, "Failed to list running models")
				return handleNotRunningError(err)
			}
			cmd.Print(psTable(ps))
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}

func psTable(ps []desktop.BackendStatus) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"MODEL NAME", "BACKEND", "MODE", "LAST USED"})

	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT, // MODEL
		tablewriter.ALIGN_LEFT, // BACKEND
		tablewriter.ALIGN_LEFT, // MODE
		tablewriter.ALIGN_LEFT, // LAST USED
	})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	for _, status := range ps {
		table.Append([]string{
			status.ModelName,
			status.BackendName,
			status.Mode,
			units.HumanDuration(time.Since(status.LastUsed)) + " ago",
		})
	}

	table.Render()
	return buf.String()
}
