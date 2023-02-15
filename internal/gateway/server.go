package gateway

import (
	"context"
	"embed"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/queries"
	"io"
	"io/fs"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/web"
)

//go:embed static templates
var embeddedFiles embed.FS

func StartGateway(alias string, port string) {
	e := echo.New()
	e.HideBanner = true

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseFS(embeddedFiles, "templates/*.html")),
	}
	e.Renderer = renderer

	fsys, _ := fs.Sub(embeddedFiles, "static")
	assetHandler := http.FileServer(http.FS(fsys))
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", assetHandler)))

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index", alias)
	})

	e.GET("/views/datasets", func(c echo.Context) error {
		server := web.GetServerFromAlias(alias)
		token, _ := web.ResolveCredentialsFromAlias(alias)

		// get datasets list
		dsm := api.NewDatasetManager(server, token.AccessToken)
		datasets, _ := dsm.List()
		datasetItems := make([]DatasetItemView, 0)
		for _, ds := range datasets {
			datasetItems = append(datasetItems, DatasetItemView{Name: ds.Name, Id: ds.Name})
		}

		// pass it to template
		return c.Render(http.StatusOK, "datasets", datasetItems)
	})

	e.GET("/views/datasets/:dataset/changes", func(c echo.Context) error {
		server := web.GetServerFromAlias(alias)
		token, _ := web.ResolveCredentialsFromAlias(alias)

		dataset := c.Param("dataset")
		since := c.QueryParam("since")

		em := api.NewEntityManager(server, token.AccessToken, context.Background(), api.Changes)
		var s api.Sink
		s = &api.SinkExpander{Sink: &api.CollectorSink{}}

		em.Read(dataset, since, 2, true, s)

		collector := s.(*api.SinkExpander).Sink.(*api.CollectorSink)

		entityListView := &EntityListView{ContinuationToken: collector.ContinuationToken, DatasetId: dataset}
		entityListView.Items = make([]EntityListItemView, 0)

		for _, e := range collector.Entities {
			entityListView.Items = append(entityListView.Items, EntityListItemView{Id: e.ID, Deleted: e.IsDeleted})
		}

		// pass it to template
		return c.Render(http.StatusOK, "changes-list", entityListView)
	})

	e.GET("/views/entity", func(c echo.Context) error {
		server := web.GetServerFromAlias(alias)
		token, _ := web.ResolveCredentialsFromAlias(alias)
		entity := c.QueryParam("id")

		qb := queries.NewQueryBuilder(server, token.AccessToken)

		res, ctx, err := qb.QuerySingle(entity, false, nil)
		if err != nil {
			return err
		}

		viewModel := buildEntityTableView(res, ctx)
		err = c.Render(http.StatusOK, "entity-view", viewModel)
		return err
	})

	e.GET("/views/datasets/:dataset/entities", func(c echo.Context) error {
		server := web.GetServerFromAlias(alias)
		token, _ := web.ResolveCredentialsFromAlias(alias)

		dataset := c.Param("dataset")
		since := c.QueryParam("since")
		form := c.QueryParam("form")

		// get entities list
		em := api.NewEntityManager(server, token.AccessToken, context.Background(), api.Entities)
		var s api.Sink
		s = &api.SinkExpander{Sink: &api.CollectorSink{}}

		em.Read(dataset, since, 10, false, s)

		if form == "" {
			form = "list"
		}

		type EntityDetails struct {
			Entity       *api.Entity
			IsoTimestamp string
		}

		collector := s.(*api.SinkExpander).Sink.(*api.CollectorSink)
		var enhancedEntities []EntityDetails
		for _, entity := range collector.Entities {
			newTimestamp := time.Unix(0, int64(entity.Recorded)).Format(time.RFC3339)
			newEntity := EntityDetails{entity, newTimestamp}
			enhancedEntities = append(enhancedEntities, newEntity)
		}

		if form == "list" {
			entityListView := &EntityListView{ContinuationToken: collector.ContinuationToken, DatasetId: dataset}
			entityListView.Items = make([]EntityListItemView, 0)

			for _, e := range enhancedEntities {
				// jsonData, _ := json.Marshal(e)
				entityListView.Items = append(entityListView.Items, EntityListItemView{Id: e.Entity.ID, Deleted: e.Entity.IsDeleted, Recorded: e.IsoTimestamp})
			}

			// pass it to template
			return c.Render(http.StatusOK, "entities-list", entityListView)

		} else if form == "table" {
			table := buildTableFromEntities(collector.Entities)
			return c.Render(http.StatusOK, "entities-table", TableView{DatasetId: dataset, Table: table, ContinuationToken: collector.ContinuationToken})
		}

		return c.NoContent(http.StatusBadRequest)
	})

	e.GET("/views/jobs", func(c echo.Context) error {
		server := web.GetServerFromAlias(alias)
		token, _ := web.ResolveCredentialsFromAlias(alias)

		jm := api.NewJobManager(server, token.AccessToken)

		jobs := jm.GetJobListWithHistory()

		sort.Slice(jobs, func(i, j int) bool {
			return jobs[i].Job.Title < jobs[j].Job.Title
		})

		return c.Render(http.StatusOK, "jobs", jobs)
	})
	e.Debug = true
	e.Logger.Fatal(e.Start(":" + port))

}

func buildEntityTableView(entity *api.Entity, namespaces map[string]interface{}) *EntityTableView {
	etv := &EntityTableView{}
	etv.EntityId = expandEntityRef(entity.ID, namespaces)
	etv.EntityTable = &EntityTable{}
	etv.EntityTable.Rows = make([]*Row, 0)

	// id row
	row := &Row{}
	row.Values = make([][]*Value, 0)
	idCol := makeValue("http://data.mimiro.io/core/uda/id", true, identityFunc)
	row.Values = append(row.Values, idCol)
	idVal := makeValue(etv.EntityId, true, identityFunc)
	row.Values = append(row.Values, idVal)
	etv.EntityTable.Rows = append(etv.EntityTable.Rows, row)

	// props
	for k, v := range entity.Properties {
		row := &Row{}
		row.Values = make([][]*Value, 0)

		prop := makeValue(k, true, func(val string) string { return expandEntityRef(val, namespaces) })
		row.Values = append(row.Values, prop)

		val := makeValue(v, false, identityFunc)
		row.Values = append(row.Values, val)

		etv.EntityTable.Rows = append(etv.EntityTable.Rows, row)
	}

	// refs
	for k, v := range entity.References {
		row := &Row{}
		row.Values = make([][]*Value, 0)

		prop := makeValue(k, true, func(val string) string { return expandEntityRef(val, namespaces) })
		row.Values = append(row.Values, prop)

		val := makeValue(v, true, func(val string) string { return expandEntityRef(val, namespaces) })
		row.Values = append(row.Values, val)

		etv.EntityTable.Rows = append(etv.EntityTable.Rows, row)
	}

	return etv
}

type EntityTableView struct {
	EntityId    string
	EntityTable *EntityTable
}

type EntityTable struct {
	Rows []*Row
}

type TableView struct {
	DatasetId         string
	Table             *Table
	ContinuationToken string
}

// do the best to create a table view that can be
// rendered by the table template
func buildTableFromEntities(entities []*api.Entity) *Table {
	// build headers - ignore nested entities but support lists of values and refs
	table := &Table{}
	table.Headers = make([]*EntityRef, 0)
	table.Rows = make([]*Row, 0)

	props := make([]string, 0)
	refs := make([]string, 0)
	for _, e := range entities {

		for k, _ := range e.Properties {
			if !contains(props, k) {
				props = append(props, k)
			}
		}

		for k, _ := range e.References {
			if !contains(refs, k) {
				refs = append(refs, k)
			}
		}
	}

	// add a header column
	table.Headers = append(table.Headers, &EntityRef{Label: "Id", URI: "http://data.mimiro.io/uda/id"})

	// add props and refs to header
	for _, h := range props {
		table.Headers = append(table.Headers, makeEntityRef(h))
	}
	for _, h := range refs {
		table.Headers = append(table.Headers, makeEntityRef(h))
	}

	// add values
	for _, e := range entities {
		row := &Row{}
		table.Rows = append(table.Rows, row)
		row.Values = make([][]*Value, 0)

		// add id
		val := makeValue(e.ID, true, identityFunc)
		row.Values = append(row.Values, val)

		// add props
		for _, k := range props {
			val := makeValue(e.Properties[k], false, identityFunc)
			row.Values = append(row.Values, val)
		}

		// add refs
		for _, k := range refs {
			val := makeValue(e.References[k], true, identityFunc)
			row.Values = append(row.Values, val)
		}

	}

	return table
}

func contains(array []string, val string) bool {
	for _, v := range array {
		if v == val {
			return true
		}
	}
	return false
}

func identityFunc(value string) string {
	return value
}

func expandEntityRef(value string, ctx map[string]interface{}) string {
	tokens := strings.Split(value, ":")
	if len(tokens) == 2 {
		if expansion, ok := ctx[tokens[0]]; ok {
			return expansion.(string) + tokens[1]
		} else {
			return value
		}
	} else {
		return value
	}
}

func makeValue(raw interface{}, isRef bool, valueFunc func(string) string) []*Value {

	multiValue := make([]*Value, 0)

	switch v := raw.(type) {
	case int:
		val := &Value{Value: strconv.Itoa(v), IsRef: isRef}
		multiValue = append(multiValue, val)
	case int64:
		val := &Value{Value: strconv.FormatInt(v, 10), IsRef: isRef}
		multiValue = append(multiValue, val)
	case float64:
		val := &Value{Value: fmt.Sprintf("%f", v), IsRef: isRef}
		multiValue = append(multiValue, val)
	case string:
		if isRef {
			v = valueFunc(v)
			entityRef := makeEntityRef(v)
			val := &Value{IsRef: true, Label: entityRef.Label, URI: entityRef.URI}
			multiValue = append(multiValue, val)
		} else {
			val := &Value{Value: v}
			multiValue = append(multiValue, val)
		}
	case []string:
		for _, v1 := range v {
			if isRef {
				v1 = valueFunc(v1)
				entityRef := makeEntityRef(v1)
				val := &Value{IsRef: true, Label: entityRef.Label, URI: entityRef.URI}
				multiValue = append(multiValue, val)
			} else {
				val := &Value{Value: v1}
				multiValue = append(multiValue, val)
			}
		}
	case []interface{}:
		values := make([]*Value, 0)
		for _, v1 := range v {
			newValues := makeValue(v1, isRef, valueFunc)
			for _, val1 := range newValues {
				exist := false
				for _, val2 := range values {
					if val1.IsRef {
						if val1.URI == val2.URI {
							exist = true
						}
					} else {
						if val1.Value == val2.Value {
							exist = true
						}
					}
				}
				if !exist {
					values = append(values, val1)
				}
			}

		}
		multiValue = append(multiValue, values...)
	case nil:
		val := &Value{Value: "nil", IsRef: false}
		multiValue = append(multiValue, val)
	case bool:
		val := &Value{Value: fmt.Sprintf("%t", v), IsRef: false}
		multiValue = append(multiValue, val)
	default:
		val := &Value{Value: "unknown type", IsRef: false}
		multiValue = append(multiValue, val)
	}

	return multiValue
}

type Table struct {
	Headers []*EntityRef
	Rows    []*Row
}

func makeEntityRef(uri string) *EntityRef {
	lastSlash := strings.LastIndex(uri, "#")
	if lastSlash == -1 {
		lastSlash = strings.LastIndex(uri, "/")
	}

	entityRef := &EntityRef{Label: uri[lastSlash+1:], URI: uri}
	return entityRef
}

type EntityRef struct {
	URI   string
	Label string
}

type Row struct {
	Values [][]*Value
}

type Value struct {
	IsRef bool
	Label string
	Value string
	URI   string
}

type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type DatasetItemView struct {
	Id   string
	Name string
}

type EntityListView struct {
	Items             []EntityListItemView
	ContinuationToken string
	DatasetId         string
}

type EntityListItemView struct {
	Id       string
	Deleted  bool
	Recorded string
}

type Config struct {
	DataHubs []*DataHub
	Services []*Service
}

type DataHub struct {
	Name       string
	Endpoint   string
	ClientId   string
	PrivateKey string
	PublicKey  string
}

type Service struct {
	Name       string
	Endpoint   string
	ClientId   string
	PrivateKey string
	PublicKey  string
}

type Catalog struct {
	DatahubHome   string   // relates to one of the datahubs listed above
	ModelDatasets []string // list of the datasets in the hub that contain data models
}

type Application struct {
	Name        string
	DataHubHome string // datahub storing application definition
	DataSetName string // dataset storing application definition
}
