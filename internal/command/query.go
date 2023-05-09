// Copyright 2021 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/mimiro-io/datahub-cli/internal/datasets/printer"
	"github.com/mimiro-io/datahub-cli/internal/docs"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/queries"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/pkg/api"
)

type cmds struct {
	id            string
	entity        []string
	via           string
	inverse       bool
	json          bool
	pretty        bool
	expanded      bool
	datasets      []string
	details       bool
	namespaces    bool
	limit         int
	continuations []string
}

// describeCmd represents the describe command
var QueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query for entities",
	Long: `For example:
mim query --id <entityURI> or
mim query --entity <entityURI> --via <predicateURI> --inverse true | false
`,

	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		c := resolveCmds(cmd, args)
		if len(args) == 0 && c.id == "" && len(c.entity) == 0 {
			_ = cmd.Usage()
			os.Exit(1)
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		sink := outputSink(format)
		if c.expanded {
			sink = api.SinkExpander{Sink: sink}
		}
		if c.id != "" {
			out, err := queryScalar(c, server, token)
			utils.HandleError(err)
			err = outputEntities(out, sink)
			utils.HandleError(err)
		} else {
			result, err := queryEntities(c, server, token)
			utils.HandleError(err)

			outputAsEntities, _ := cmd.Flags().GetBool("output-entities")
			if outputAsEntities && format == "json" {
				entities := getEntities(result)
				err = outputEntities(entities, sink)
				utils.HandleError(err)
			} else {
				pr := newPrinter(format, 50)
				if c.expanded {
					pr = &printer.ExpandingPrinter{Printer: pr}
				}
				pr.Header(result[0])
				pr.Print(result[1:])
				pr.Footer()
			}

		}
		pterm.Println()
	},
}

func outputSink(format string) api.Sink {
	switch format {
	case "term":
		return &api.ConsoleSink{}
	case "pretty":
		return &api.PrettySink{}
	default:
		return &api.RawSink{}
	}
}

func getEntities(result []interface{}) []*api.Entity {
	entities := make([]*api.Entity, 0)

	for i, e := range result {
		if i == 0 || i == len(result)-1 { // this will always be an entity
			entities = append(entities, e.(*api.Entity))
		} else {
			items := e.([]interface{})
			entities = append(entities, items[2].(*api.Entity))
		}
	}
	return entities
}

func newPrinter(format string, batchSize int) printer.Printer {
	switch format {
	case "pretty":
		return &printer.PrettyPrint{Batch: batchSize}
	case "json":
		return &printer.Raw{Batch: 1000}
	default:
		return &term{batchSize: batchSize}
	}
}

type term struct {
	batchSize int
	header    bool
}

func (t *term) Print(entities []interface{}) {
	out := make([][]string, 0)
	if !t.header {
		out = append(out, []string{"Uri", "PredicateUri", "Id", "Recorded", "Deleted", "Props", "Refs"})
	}

	for _, e := range entities {
		raw, isSearchRes := e.([]interface{})
		var obj *api.Entity
		if isSearchRes {
			obj = raw[2].(*api.Entity)
		} else {
			obj = e.(*api.Entity)
		}
		id := obj.ID
		switch id {
		case "@context":
			t.prettyContext(obj)
		case "@continuation":
			pterm.DefaultSection.Println(fmt.Sprintf("Continuation token: %s", obj.Properties["token"]))
		default:
			out = append(out, t.prettyEntity(id, raw))
		}
	}

	if !t.header {
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
		t.header = true
	} else {
		pterm.DefaultTable.WithData(out).Render()
	}
}

func (t *term) Header(entity interface{}) {
	if entity != nil {
		raw := entity.(*api.Entity)
		t.prettyContext(raw)
	}
}

func (t *term) Footer() {
	// do nothing
}

func (t *term) BatchSize() int {
	return t.batchSize + 2 // to accord for cont token and context
}

func (t *term) prettyEntity(id string, e []interface{}) []string {
	obj := e[2].(*api.Entity)

	return []string{
		e[0].(string),
		e[1].(string),
		id,
		fmt.Sprintf("%d", obj.Recorded),
		fmt.Sprintf("%v", obj.IsDeleted),
		fmt.Sprintf("%v", obj.Properties),
		fmt.Sprintf("%v", obj.References),
	}
}

func (t *term) prettyContext(context *api.Entity) {
	pterm.DefaultSection.Println("Namespaces:")
	namespaces := context.Properties["namespaces"].(map[string]interface{})
	out := make([][]string, 0)
	out = append(out, []string{"#", "Namespace"})
	for k, v := range namespaces {
		out = append(out, []string{
			k,
			fmt.Sprintf("%s", v),
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(out).Render()
}

func outputEntities(output []*api.Entity, sink api.Sink) error {
	source := &api.EntityListDatasource{Entities: output}
	pipeline := api.NewPipeline(source, sink)
	return pipeline.Sync(context.Background(), "", 0)
}

func resolveCmds(cmd *cobra.Command, args []string) cmds {
	c := cmds{}
	c.id, _ = cmd.Flags().GetString("id")
	if c.id == "" && len(args) > 1 {
		c.id = args[0]
	}
	c.entity, _ = cmd.Flags().GetStringArray("entity")
	c.via, _ = cmd.Flags().GetString("via")
	c.inverse, _ = cmd.Flags().GetBool("inverse")
	c.json, _ = cmd.Flags().GetBool("json")
	c.pretty, _ = cmd.Flags().GetBool("pretty")
	c.datasets, _ = cmd.Flags().GetStringArray("datasets")
	c.expanded, _ = cmd.Flags().GetBool("expanded")
	c.details, _ = cmd.Flags().GetBool("details")
	c.namespaces, _ = cmd.Flags().GetBool("namespaces")
	c.limit, _ = cmd.Flags().GetInt("limit")
	c.continuations, _ = cmd.Flags().GetStringArray("continuations")
	return c
}

func queryScalar(c cmds, server string, token string) ([]*api.Entity, error) {
	pterm.DefaultSection.Printf("Query for entity " + c.id + " on " + server)

	qb := queries.NewQueryBuilder(server, token)

	res, namespaces, err := qb.QuerySingle(c.id, c.details, c.datasets)
	if err != nil {
		return nil, err
	}
	out := make([]*api.Entity, 0)
	if c.namespaces {
		out = append(out, api.NewContextWithNamespaces(namespaces))
	} else {
		out = append(out, api.NewContext())
	}

	out = append(out, res)

	return out, nil
}

func queryEntities(c cmds, server string, token string) ([]interface{}, error) {
	eq := api.NewEntityQuery(server, token)

	return eq.Query(c.entity, c.via, c.inverse, c.datasets, c.limit, c.continuations)
}

func init() {
	QueryCmd.Flags().StringP("id", "i", "", "The id of the entity you want to fetch")
	QueryCmd.Flags().StringArray("entity", make([]string, 0),
		"The URI of the entity to use as start of traversal. May be repeated for batch lookups")
	QueryCmd.Flags().String("via", "", "The URI of the traversal reference type")
	QueryCmd.Flags().Bool("inverse", false, "Indicates if the traversal is out from the entities or incoming")
	QueryCmd.Flags().Bool("output-entities", true,
		"If this is an entity query, and the output is json, then this outputs only the list of entities")
	QueryCmd.Flags().StringArray("datasets", make([]string, 0),
		"add a list of datasets to filter in with '<dataset-name>, <dataset-name>'")
	QueryCmd.Flags().BoolP("expanded", "e", false, "Expand namespace prefixes in entities to full namespace URIs")
	QueryCmd.Flags().Bool("details", false, "Works only with --id/-i query. Inject entity details into entity result.")
	QueryCmd.Flags().Bool("namespaces", false, "Works only with --id/-i query. Add context with namespaces.")
	QueryCmd.Flags().Int("limit", 0, "Limit number of search results. If exceeded, response may contain continuations")
	QueryCmd.Flags().StringArray("continuations", nil,
		"list of continuation tokens. provide to continue previously started query")

	QueryCmd.RegisterFlagCompletionFunc(
		"datasets",
		func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			parts := strings.Split(toComplete, ",")
			if len(parts) > 1 {
				pattern := parts[len(parts)-1]
				result := api.GetDatasetsCompletion(pattern)
				var values []string
				for _, res := range result {
					values = append(values, strings.TrimSuffix(toComplete, pattern)+res)
				}
				return values, cobra.ShellCompDirectiveNoFileComp
			}
			return api.GetDatasetsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
		},
	)

	QueryCmd.SetHelpFunc(func(command *cobra.Command, _ []string) {
		pterm.Println()
		result := docs.RenderMarkdown(command, "doc-query.md")
		pterm.Println(result)
	})
}
