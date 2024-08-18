package stats

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"slices"
	"strconv"
	"strings"
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all dataset statistics",
	Long: `List all dataset statistics. For example:
mim stats list
`,
	Run: func(cmd *cobra.Command, args []string) {
		fetch(cmd)
	},

	TraverseChildren: true,
}

func fetch(cmd *cobra.Command) {
	format := utils.ResolveFormat(cmd)
	if format != "term" { // turn of pterm output
		pterm.DisableOutput()
	}
	server, token, err := login.ResolveCredentials()

	utils.HandleError(err)

	pterm.EnableDebugMessages()

	pterm.DefaultSection.Println("Fetching storage statistics for " + server)

	stats, err := web.GetRequest(server, token, "/statistics")
	utils.HandleError(err)

	output(stats, format)

	pterm.Println()
}
func output(statsBytes []byte, format string) {
	switch format {
	case "pretty":
		f := pretty.Pretty(statsBytes)
		result := pretty.Color(f, nil)
		fmt.Println(string(result))
	case "json":
		fmt.Println(string(statsBytes))
	default:
		result := make(map[string]any)
		err := json.Unmarshal(statsBytes, &result)
		utils.HandleError(err)
		rows := toRows(result)

		out := make([][]string, 0)
		out = append(out, []string{"Dataset", "Changes", "Entities", "Refs", "Keys size",
			"Values size", "Refs size", "Changes \nper Entity"})
		for k, v := range rows {
			versionsPerEntity := float64(0)
			if v[1] > 0 {
				versionsPerEntity = float64(v[0]) / float64(v[1])

			}

			out = append(out, []string{
				k,                         //dataset
				fmt.Sprintf("%10d", v[0]), // changes
				fmt.Sprintf("%10d", v[1]), // entities
				fmt.Sprintf("%10d", v[2]), // refs
				ByteCountIEC(v[3]),        // size entity keys
				ByteCountIEC(v[4]),        // size entity values
				ByteCountIEC(v[5]),        // size ref keys
				strings.ReplaceAll(fmt.Sprintf("%.1f", versionsPerEntity), ".0", ""), // avg versions per entity
			})
		}

		sortTable(out, 0, false) // sort by dataset name

		pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).WithMargin(10).Print("Entities")
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
		pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).
			WithMargin(10).Print("Other (URIs, unknown, sys)")

		keyCount, keySize, valueSize := toSystemStats(result)

		box := pterm.DefaultBox.WithLeftPadding(10).WithRightPadding(10).WithTopPadding(1).WithBottomPadding(1)
		p, _ := pterm.DefaultPanel.WithPanels(pterm.Panels{[]pterm.Panel{
			{Data: box.Sprintf(fmt.Sprintf("Keys: %10d", keyCount))},
			{Data: box.Sprintf("Key Size: " + ByteCountIEC(keySize))},
			{Data: box.Sprintf("Value Size: " + ByteCountIEC(valueSize))}}}).
			WithPadding(5).
			Srender()

		pterm.DefaultCenter.Println(p)
	}

}

func sortTable(out [][]string, colNum int, numeric bool) {
	slices.SortStableFunc(out, func(i []string, j []string) int {
		val1 := i[colNum]
		val2 := j[colNum]

		if numeric {
			n1, err := strconv.ParseFloat(strings.TrimSpace(val1), 64)
			utils.HandleError(err)
			n2, err := strconv.ParseFloat(strings.TrimSpace(val2), 64)
			utils.HandleError(err)

			// reverse sort if numeric, since its used in top lists
			if n1 < n2 {
				return 1
			} else if n1 > n2 {
				return -1
			} else {
				return 0
			}
		}
		if val1 < val2 {
			return -1
		} else if val1 > val2 {
			return 1
		} else {
			return 0
		}
	})
}

func toSystemStats(result map[string]any) (int64, int64, int64) {
	sys, _ := result["sys"].(map[string]any)
	unknown, _ := result["unknown"].(map[string]any)
	uris, _ := result["urimap"].(map[string]any)

	var keyCount int64
	var keySize int64
	var valueSize int64
	for _, sub := range sys {
		datasets := sub.(map[string]any)
		for _, v := range datasets {
			ds := v.(map[string]any)
			keys := int64(ds["keys"].(float64))
			sizeKeys := int64(ds["size-keys"].(float64))
			sizeValues := int64(ds["size-values"].(float64))

			keyCount += keys
			keySize += sizeKeys
			valueSize += sizeValues
		}
	}
	for _, sub := range unknown {
		datasets := sub.(map[string]any)
		for _, v := range datasets {
			ds := v.(map[string]any)
			keys := int64(ds["keys"].(float64))
			sizeKeys := int64(ds["size-keys"].(float64))
			sizeValues := int64(ds["size-values"].(float64))

			keyCount += keys
			keySize += sizeKeys
			valueSize += sizeValues
		}
	}
	for _, sub := range uris {
		datasets := sub.(map[string]any)
		for _, v := range datasets {
			ds := v.(map[string]any)
			keys := int64(ds["keys"].(float64))
			sizeKeys := int64(ds["size-keys"].(float64))
			sizeValues := int64(ds["size-values"].(float64))

			keyCount += keys
			keySize += sizeKeys
			valueSize += sizeValues
		}
	}
	return keyCount, keySize, valueSize
}

func toRows(result map[string]any) map[string][7]int64 {
	ents, _ := result["entity"].(map[string]any)
	rows := map[string][7]int64{}
	for ix, sub := range ents {
		datasets := sub.(map[string]any)
		for k, v := range datasets {
			ds := v.(map[string]any)

			keys := int64(ds["keys"].(float64))
			sizeKeys := int64(ds["size-keys"].(float64))
			sizeValues := int64(ds["size-values"].(float64))
			row, _ := rows[k]
			if ix == "DATASET_ENTITY_CHANGE_LOG" {
				row[0] = keys + row[0]
				row[3] = sizeKeys + row[3]
			} else if ix == "DATASET_LATEST_ENTITIES" {
				row[1] = keys + row[1]
				row[3] = sizeKeys + row[3]
			} else {
				row[3] = sizeKeys + row[3]
				row[4] = sizeValues + row[4]
			}
			rows[k] = row
		}
	}
	refs, _ := result["refs"].(map[string]any)
	for _, sub := range refs {
		datasets := sub.(map[string]any)
		for k, v := range datasets {
			ds := v.(map[string]any)

			keys := int64(ds["keys"].(float64))
			sizeKeys := int64(ds["size-keys"].(float64))
			//sizeValues := int64(ds["size-values"].(float64))
			k = strings.ReplaceAll(k, " (deleted)", "")
			row, _ := rows[k]
			row[2] = keys + row[2]
			row[5] = sizeKeys + row[5]
			rows[k] = row
		}
	}
	return rows
}
func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
