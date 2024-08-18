package command

import (
	"github.com/mimiro-io/datahub-cli/internal/stats"
	"github.com/spf13/cobra"
	"os"
)

var StatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "retrieve storage statistics from datahub.",
	Long: `retrieve storage statistics from datahub. statistics are compiled in a long running task in datahub and may not be available before the task is done.
Examples:
mim stats list
mim stats top
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {
	StatsCmd.AddCommand(stats.ListCmd)
	StatsCmd.AddCommand(stats.TopCmd)
}
