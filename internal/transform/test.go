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

package transform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bcicen/jstream"
	"github.com/dop251/goja"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/datasets"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/queries"
	"github.com/mimiro-io/datahub-cli/internal/utils"
)

const wrapperJavascriptFunction = `
function transform_entities_ex(entities) {
	try {
		return transform_entities(entities);
	} catch (e) {
		//Log("js error " + e.name + " : " + e.message);
		throw(e);
	}
}
`

// these are upper cased to prevent the user from accidentally redefining them
// (i mean, not really, but maybe it will help)
const helperJavascriptFunctions = `
function SetProperty(entity, prefix, name, value) {
	if (entity === null || entity === undefined) {
		return;
	}
	if (entity.Properties === null || entity.Properties === undefined) {
		return;
	}
	entity["Properties"][prefix+":"+name] = value;
}
function GetProperty(entity, prefix, name, defaultValue) {
	if (entity === null || entity === undefined) {
		return defaultValue;
	}
	if (entity.Properties === null || entity.Properties === undefined) {
		return defaultValue;
	}
	var value = entity["Properties"][prefix+":"+name]
	if (value === undefined || value === null) {
		return defaultValue;
	}
	return value;
}
function GetReference(entity, prefix, name, defaultValue) {
	if (entity === null || entity === undefined) {
		return defaultValue;
	}
	if (entity.References === null || entity.References === undefined) {
		return defaultValue;
	}
	var value = entity["References"][prefix+":"+name]
	if (value === undefined || value === null) {
		return defaultValue;
	}
	return value;
}
function AddReference(entity, prefix, name, value) {
	if (entity === null || entity === undefined) {
		return;
	}
	if (entity.References === null || entity.References === undefined) {
		return;
	}
	entity["References"][prefix+":"+name] = value;
}
function GetId(entity) {
	if (entity === null || entity === undefined) {
		return;
	}
	return entity["ID"];
}
function SetId(entity, id) {
	if (entity === null || entity === undefined) {
		return;
	}
	entity.ID = id
}

function SetDeleted(entity, deleted) {
	if (entity === null || entity === undefined) {
		return;
	}
	entity.IsDeleted = deleted
}

function GetDeleted(entity) {
	if (entity === null || entity === undefined) {
		return;
	}
	return entity.IsDeleted;
}

function PrefixField(prefix, field) {
    return prefix + ":" + field;
}
function RenameProperty(entity, originalPrefix, originalName, newPrefix, newName) {
	if (entity === null || entity === undefined) {
		return;
	}
	var value = GetProperty(entity, originalPrefix, originalName);
	SetProperty(entity, newPrefix, newName, value);
	RemoveProperty(entity, originalPrefix, originalName);
}

function RemoveProperty(entity, prefix, name){
	if (entity === null || entity === undefined) {
		return;
	}
	delete entity["Properties"][prefix+":"+name];
}

function NewEntityFrom(entity, addType, copyProps, copyRefs){
	if (entity === null || entity === undefined) {
		return NewEntity();
	}

	let newEntity = NewEntity();
	SetId(newEntity, GetId(entity));
	SetDeleted(newEntity, GetDeleted(entity));
	if (addType){
		let rdf = GetNamespacePrefix("http://www.w3.org/1999/02/22-rdf-syntax-ns#");
		let type = GetReference(entity, rdf, "type");
		if (type != null){
			AddReference(newEntity, rdf, "type", type)
		}
	}
	if (copyProps) {
		for (const [key, value] of Object.entries(entity["Properties"])) {
			newEntity["Properties"][key] = value;
		}
	}
	if (copyRefs) {
		for (const [key, value] of Object.entries(entity["References"])) {
			newEntity["References"][key] = value;
		}
	}
	return newEntity;
}
`

// TestCmd allows to import a *.js (or a *.ts) file into an existing job
var TestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test a transformation",
	Long: `Test a transformation from a job or a file. For example:
mim transform test --file <transform.js> --name sdb.Animal --limit 10
or
cat <transform.js> | mim transform test -n sdb.Animal

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		file, err := cmd.Flags().GetString("file")
		utils.HandleError(err)

		dataset, err := cmd.Flags().GetString("name")
		utils.HandleError(err)
		if dataset == "" && len(args) > 0 {
			dataset = args[0]
		}

		limit, err := cmd.Flags().GetInt("limit")
		utils.HandleError(err)

		pterm.DefaultSection.Println("Testing script function")

		if file == "" {
			utils.HandleError(errors.New("missing or empty file parameter"))
		}
		var res []byte
		importer := NewImporter(file)
		if filepath.Ext(file) == ".ts" {
			res, err = importer.ImportTs()
		} else {
			res, err = importer.ImportJs()
		}

		utils.HandleError(err)
		code := string(res)

		// does it compile?
		_, err = goja.Compile("transform_entities", code, false)
		utils.HandleError(err)

		engine := hookEngine(server, token)
		_, err = engine.RunString(code)
		utils.HandleError(err)
		_, err = engine.RunString(wrapperJavascriptFunction)
		utils.HandleError(err)

		// add helper functions
		_, err = engine.RunString(helperJavascriptFunctions)
		utils.HandleError(err)

		entities, err := getEntities(server, token, dataset, limit)
		utils.HandleError(err)

		transformed, err := transformEntities(entities, engine)
		utils.HandleError(err)
		outputResult(transformed, format)

		pterm.Println()
	},
	TraverseChildren: true,
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

func outputResult(entities []*api.Entity, format string) {
	pterm.Println()
	sink := outputSink(format)
	sink.Start()
	_ = sink.ProcessEntities(entities)
	sink.End()
}

func getEntities(server string, token string, dataset string, limit int) ([]*api.Entity, error) {
	if dataset == "" { // we have a bytestream from stdin
		sink := &api.CollectorSink{}
		source := &api.StdinDatasetSource{}
		pipeline := api.NewPipeline(source, sink)
		err := pipeline.Sync(context.Background(), "", limit)
		if err != nil {
			return nil, err
		}

		return sink.Entities, nil
	} else {
		em := api.NewEntityManager(server, token, context.Background(), api.Changes)
		collector := &api.CollectorSink{}

		err := em.Read(dataset, "", datasets.SaneLimit("json", limit), false, collector)
		if err != nil {
			return nil, err
		}

		if len(collector.Entities) == 0 {
			return make([]*api.Entity, 0), nil
		}

		return collector.Entities, nil
	}
}

func parseStdin(input []byte) ([]*api.Entity, error) {
	// expect a list of Entity, and we will need everything, so just parse it, memory be damned
	decoder := jstream.NewDecoder(bytes.NewReader(input), 1)

	max := 17
	if len(input) < max {
		return nil, errors.New("input too short to be usable")
	}

	entities := make([]*api.Entity, 0)
	if string(input[:max]) == "[{\"id\":\"@context\"" {
		decoder = jstream.NewDecoder(bytes.NewReader(input), 2)
		return decodeQuery(input, decoder)
	} else {
		entity := &api.Entity{}
		err := json.Unmarshal(input, entity)
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

func decodeQuery(input []byte, decoder *jstream.Decoder) ([]*api.Entity, error) {
	count := 0
	entities := make([]*api.Entity, 0)
	for mv := range decoder.Stream() {
		if count > 1 { // we are not interested in the 2 first entities as that is related to the @context
			switch mv.ValueType {
			case 5:
				ent := mv.Value.([]interface{})

				raw, err := json.Marshal(ent[2]) // convert to raw
				if err != nil {
					return nil, err
				}

				entity := &api.Entity{}
				err = json.Unmarshal(raw, entity)
				if err != nil {
					return nil, err
				}
				entities = append(entities, entity)
			default:
				fmt.Println(mv.Value)
			}
		}

		count++
	}

	return entities, nil
}

func transformEntities(entities []*api.Entity, engine *goja.Runtime) ([]*api.Entity, error) {
	var transFunc func(entities []*api.Entity) (interface{}, error)
	err := engine.ExportTo(engine.Get("transform_entities"), &transFunc)
	if err != nil {
		return nil, err
	}

	result, err := transFunc(entities)
	if err != nil {
		return nil, err
	}

	var resultEntities []*api.Entity
	switch v := result.(type) {
	case []interface{}:
		resultEntities = make([]*api.Entity, 0)
		for _, e := range v {
			resultEntities = append(resultEntities, e.(*api.Entity))
		}
	case []*api.Entity:
		resultEntities = v
	default:
		return nil, errors.New("bad result from transform")
	}

	return resultEntities, nil
}

func hookEngine(server string, token string) *goja.Runtime {
	tf := &transformer{
		query:            queries.NewQueryBuilder(server, token),
		assertedPrefixes: make(map[string]string),
	}
	engine := goja.New()
	engine.Set("Query", tf.Query)
	engine.Set("FindById", tf.ById)
	engine.Set("GetNamespacePrefix", tf.GetNamespacePrefix)
	engine.Set("AssertNamespacePrefix", tf.AssertNamespacePrefix)
	engine.Set("Log", tf.Log)
	engine.Set("NewEntity", tf.NewEntity)
	engine.Set("IsValidEntity", tf.IsValidEntity)
	engine.Set("ToString", tf.ToString)
	engine.Set("AsEntity", tf.AsEntity)
	engine.Set("UUID", tf.UUID)
	return engine
}

func init() {
	TestCmd.Flags().String("file", "", "The file to run the transform from.")
	TestCmd.Flags().StringP("job-id", "j", "", "The id of the job to run the transform from.")
	TestCmd.Flags().StringP("name", "n", "", "The dataset to transform entities from")
	TestCmd.Flags().Int("limit", 10, "Limits the number of entities to transform")
	TestCmd.Flags().StringP("format", "f", "term", "The output format. Valid options are: term|pretty|raw")
}
