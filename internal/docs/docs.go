package docs

import (
	"bytes"
	"embed"
	"fmt"
	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

//go:embed *.md
var docFiles embed.FS

func RenderMarkdown(c *cobra.Command, filename string) string {

	source, err := docFiles.ReadFile(filename)

	if err != nil {
		pterm.Error.Print(err)
		return ""
	}

	writer := bytes.NewBufferString("")
	err = utils.Tmpl(writer, c.HelpTemplate(), c)
	if err != nil {
		pterm.Error.Print(err)
		return ""
	}

	help := fmt.Sprintf(string(source), writer.String())

	result := markdown.Render(help, 120, 2)
	return string(result)
}
