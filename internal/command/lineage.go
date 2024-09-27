package command

import (
	"github.com/mimiro-io/datahub-cli/internal/lineage"
	"github.com/spf13/cobra"
	"os"
)

var LineageCmd = &cobra.Command{
	Use:   `lineage all|select [<datasetName>]`,
	Short: "retrieve dataset lineage from datahub.",
	Long: `Lineage is a graph of dataset transformation. It shows which datasets go into a dataset and which datasets are produced by a dataset.
Examples:
mim lineage select <datasetName>
mim lineage all
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {
	LineageCmd.AddCommand(lineage.SelectCmd)
	LineageCmd.AddCommand(lineage.AllCmd)
}
