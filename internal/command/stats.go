package command

import (
	"github.com/mimiro-io/datahub-cli/internal/stats"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "retrieve storage statistics from datahub.",
	Long: `retrieve storage statistics from datahub. statistics are compiled in a long running task in datahub and may not be available before the task is done.
Examples:
mim stats
`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.Success.Println("Fetching storage statistics")
		stats.Fetch(cmd)
	},
}

func init() {
}
