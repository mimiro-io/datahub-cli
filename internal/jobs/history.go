package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"os"
)

// StatusCmd represents the staus command on a job
var HistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "history for a job",
	Long: "history for a job",
	Example: "mim jobs history --id <jobid>",
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		idOrTitle, err := cmd.Flags().GetString("id")
		utils.HandleError(err)
		if idOrTitle == "" && len(args) > 0 {
			idOrTitle = args[0]
		}

		if idOrTitle == "" {
			pterm.Warning.Println("You must provide an id")
			pterm.Println()
			os.Exit(1)
		}

		pterm.DefaultSection.Printf("Get history of job " + idOrTitle + " on " + server)

		id := ResolveId(server,token, idOrTitle)

		hist := getHistory(id, server, token)
		utils.HandleError(err)

		renderHistory(hist, format)

	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetJobsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	TraverseChildren: true,
}

func init() {
	HistoryCmd.Flags().StringP("id", "i", "", "The name of the job you want to get status on")
}

func getHistory(id string, server string, token string) api.JobHistory {
	endpoint := "/jobs/_/history"

	body, err := utils.GetRequest(server, token, endpoint)
	utils.HandleError(err)

	histories := make([]api.JobHistory, 0)
	err = json.Unmarshal(body, &histories)
	utils.HandleError(err)

	for _, hist := range histories {
		if hist.Id == id {
			return hist
		}
	}
	return api.JobHistory{}
}

func renderHistory(history api.JobHistory, format string) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(history)
	utils.HandleError(err)
	jd := bf.String()

	switch format {
	case "json":
		fmt.Println(jd)
	default:
		p := pretty.Pretty([]byte(jd))
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	}

}
