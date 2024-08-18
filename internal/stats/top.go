package stats

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
)

var TopCmd = &cobra.Command{
	Use:   "top",
	Short: "List top 10 dataset statistics",
	Long: `List top 10 dataset statistics. For example:
mim stats top
`,
	Run: func(cmd *cobra.Command, args []string) {
		top(cmd)
	},

	TraverseChildren: true,
}

func top(cmd *cobra.Command) {
	format := utils.ResolveFormat(cmd)
	server, token, err := login.ResolveCredentials()
	utils.HandleError(err)
	pterm.EnableDebugMessages()
	pterm.DefaultSection.Println("Fetching top 10 statistics from " + server)
	stats, err := web.GetRequest(server, token, "/statistics")
	utils.HandleError(err)
	topOutput(stats, format)
	pterm.Println()
}

func topOutput(statsBytes []byte, format string) {
	switch format {
	case "pretty":
		pterm.Error.Println("JSON format not supported for this command. use `mim stats list --pretty` instead")
	case "json":
		pterm.Error.Println("JSON format not supported for this command. use `mim stats list --json` instead")
	default:
		result := make(map[string]any)
		err := json.Unmarshal(statsBytes, &result)
		utils.HandleError(err)
		rows := toRows(result)

		var totals [7]int64
		out := make([][]string, 0)
		//out = append(out, []string{"Dataset", "Changes", "Entities", "Refs", "Keys size",
		//	"Values size", "Refs size", "Changes \nper Entity"})
		for k, v := range rows {
			versionsPerEntity := float64(0)
			if v[1] > 0 {
				versionsPerEntity = float64(v[0]) / float64(v[1])

			}

			if k == "all" {
				totals = v
				totals[6] = int64(versionsPerEntity * 100)
			} else {
				out = append(out, []string{
					k,                                      //dataset
					fmt.Sprintf("%d", v[0]),                // changes
					fmt.Sprintf("%d", v[1]),                // entities
					fmt.Sprintf("%d", v[2]),                // refs
					fmt.Sprintf("%d", v[3]),                // keys
					fmt.Sprintf("%d", v[4]),                // values
					fmt.Sprintf("%d", v[5]),                // refs size
					fmt.Sprintf("%.1f", versionsPerEntity), // avg versions per entity
				})
			}
		}

		printTopList(out, totals, 1, "Top 10 Changes")
		printTopList(out, totals, 2, "Top 10 Entities")
		printTopList(out, totals, 3, "Top 10 Refs")
		printTopList(out, totals, 4, "Top 10 Keys Size in bytes")
		printTopList(out, totals, 5, "Top 10 Values Size in bytes")
		printTopList(out, totals, 6, "Top 10 Refs Size in bytes")
		printTopList(out, totals, 7, "Top 10 Changes per Entity")
	}
}

func printTopList(out [][]string, totals [7]int64, colNum int, title string) {
	sortTable(out, colNum, true) // sort by changes
	totalTxt := fmt.Sprintf("Total: %10d", totals[colNum-1])
	if colNum == 4 || colNum == 5 || colNum == 6 {
		totalTxt = fmt.Sprintf("Total: %10s", ByteCountIEC(totals[colNum-1]))
	}
	if colNum == 7 {
		totalTxt = fmt.Sprintf("Total: %10.1f", float64(totals[colNum-1])/100)
	}

	bars := pterm.DefaultBarChart.WithHorizontal().WithBars(mkBars(out, colNum)).
		WithShowValue().WithWidth(50).WithHorizontalBarCharacter("┉")
	barsTxt, _ := bars.Srender()
	pterm.DefaultPanel.WithPanels(pterm.Panels{[]pterm.Panel{
		{Data: barsTxt}},
		{{Data: pterm.DefaultBasicText.Sprintln(title)},
			{Data: pterm.DefaultBasicText.WithStyle(pterm.GrayBoxStyle).Sprintln(totalTxt)}},
	}).WithPadding(5).Render()
}

func mkBars(out [][]string, colNum int) pterm.Bars {
	var bars pterm.Bars
	for i, _ := range out {
		v, _ := strconv.ParseFloat(strings.TrimSpace(out[i][colNum]), 64)
		bars = append(bars, pterm.Bar{
			Label: out[i][0], // datasetname
			Value: int(v),    // value
			Style: pterm.NewStyle(pterm.FgLightGreen),
		})
		if i == 9 {
			break
		}
	}
	return bars
}
