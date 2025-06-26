package commands

import (
	"bytes"

	"github.com/docker/go-units"
	"github.com/docker/model-cli/commands/completion"
	"github.com/docker/model-cli/desktop"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newDFCmd() *cobra.Command {
	c := &cobra.Command{
		GroupID: groupIDModelRunner,
		Use:     "df",
		Short:   "Show Docker Model Runner disk usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			df, err := desktopClient.DF()
			if err != nil {
				err = handleClientError(err, "Failed to list running models")
				return handleNotRunningError(err)
			}
			cmd.Print(diskUsageTable(df))
			return nil
		},
		ValidArgsFunction: completion.NoComplete,
	}
	return c
}

func diskUsageTable(df desktop.DiskUsage) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)

	table.SetHeader([]string{"TYPE", "SIZE"})

	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)

	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT, // TYPE
		tablewriter.ALIGN_LEFT, // SIZE
	})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	table.Append([]string{"Models", units.HumanSize(float64(df.ModelsDiskUsage))})
	if df.DefaultBackendDiskUsage != 0 {
		table.Append([]string{"Inference engine", units.HumanSize(float64(df.DefaultBackendDiskUsage))})
	}

	table.Render()
	return buf.String()
}
