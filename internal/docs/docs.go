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
