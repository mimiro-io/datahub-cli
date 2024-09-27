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
)

var AllCmd = &cobra.Command{
	Use:   "all",
	Short: "show complete lineage",
	Long: `get the complete lineage graph info. For example:
mim lineage all
`,
	Run: func(cmd *cobra.Command, args []string) {
		showAll(cmd)
	},

	TraverseChildren: true,
}

func showAll(cmd *cobra.Command) {
	format := utils.ResolveFormat(cmd)
	if format != "term" { // turn of pterm output
		pterm.DisableOutput()
	}
	server, token, err := login.ResolveCredentials()

	utils.HandleError(err)

	pterm.EnableDebugMessages()

	pterm.DefaultSection.Println("Fetching all lineage for " + server)

	lineage, err := web.GetRequest(server, token, "/lineage")
	utils.HandleError(err)

	output(lineage, format)

	pterm.Println()
}

type LineageRel struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
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
		result := make([]LineageRel, 0)
		err := json.Unmarshal(statsBytes, &result)
		utils.HandleError(err)

		pterm.DefaultSection.Println("Lineage graph")
		pterm.DefaultHeader.Println("=> = copy, -> = transform, ~> = transform-hop")
		for _, rel := range result {
			pterm.Println(rel.From + ref(rel.Type) + rel.To)
		}
	}
}

func ref(rel string) string {
	switch rel {
	case "copy":
		return " => "
	case "transform":
		return " -> "
	case "transform-hop":
		return " ~> "
	default:
		return " - "
	}
}
