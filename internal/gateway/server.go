package gateway

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/web"
)

//go:embed static templates
var embededFiles embed.FS

func StartGateway(alias string, port string) {
	e := echo.New()
	e.HideBanner = true

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseFS(embededFiles, "templates/*.html")),
	}
	e.Renderer = renderer

	fsys, _ := fs.Sub(embededFiles, "static")
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

	e.GET("/views/jobs", func(c echo.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.Logger.Fatal(e.Start(":" + port))
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
