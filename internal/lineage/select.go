package lineage

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"os"
)

var SelectCmd = &cobra.Command{
	Use:   "select",
	Short: "select a dataset to show lineage",
	Long: `Zoom into a single dataset in the lineage graph and only show it with its adjacent nodes. For example:
mim lineage select <datasetName>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
		dataset := args[0]
		showDs(cmd, dataset)
	},

	TraverseChildren: true,
}

func showDs(cmd *cobra.Command, dataset string) {
	format := utils.ResolveFormat(cmd)
	if format != "term" { // turn of pterm output
		pterm.DisableOutput()
	}
	server, token, err := login.ResolveCredentials()

	utils.HandleError(err)

	pterm.EnableDebugMessages()

	pterm.DefaultSection.Println("Fetching lineage of " + dataset + " for " + server)

	stats, err := web.GetRequest(server, token, "/lineage/"+dataset)
	utils.HandleError(err)

	outputDataset(stats, format, dataset)

	pterm.Println()
}
func outputDataset(statsBytes []byte, format string, datasetName string) {
	switch format {
	case "pretty":
		f := pretty.Pretty(statsBytes)
		result := pretty.Color(f, nil)
		fmt.Println(string(result))
	case "json":
		fmt.Println(string(statsBytes))
	default:
		result := make([]LineageRel, 0)
		err := json.Unmarshal(statsBytes, &result)
		utils.HandleError(err)

		pterm.DefaultSection.Println("Lineage graph for " + datasetName)
		pterm.DefaultHeader.Println("=> copy, -> transform, ~> transform-hop")
		inputItems := make([]pterm.BulletListItem, 0)
		outputItems := make([]pterm.BulletListItem, 0)
		for _, rel := range result {
			if rel.To == datasetName {
				inputItems = append(inputItems, pterm.BulletListItem{Level: lvl(rel.Type), Text: rel.From, Bullet: ref(rel.Type)})
			} else {
				outputItems = append(outputItems, pterm.BulletListItem{Level: lvl(rel.Type), Text: rel.To, Bullet: ref(rel.Type)})
			}

		}
		inputs, _ := pterm.DefaultBulletList.WithItems(inputItems).Srender()
		outputs, _ := pterm.DefaultBulletList.WithItems(outputItems).Srender()
		pterm.DefaultPanel.WithSameColumnWidth(true).WithPanels([][]pterm.Panel{
			{{Data: pterm.DefaultBox.WithTitle("Inputs").Sprint(inputs)},
				{Data: pterm.DefaultBox.WithTitle("Outputs").Sprint(outputs)}},
		}).WithPadding(5).Render()
	}
}

func lvl(t string) int {
	switch t {
	case "copy":
		return 1
	case "transform":
		return 1
	case "transform-hop":
		return 2
	default:
		return 0
	}
}
