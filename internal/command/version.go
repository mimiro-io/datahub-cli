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
	"embed"
	_ "embed"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

//go:embed *
var versionDir embed.FS

var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"version"},
	Short:   "See cli version",
	Long:    `See semantic version for this cli release.`,
	Run: func(cmd *cobra.Command, args []string) {
		versionFile, err := versionDir.ReadFile("VERSION")
		var version string
		if err == nil {
			version = string(versionFile)
		} else {
			version = "Only available on released binaries"
		}
		pterm.Println(version)
	},
}

func init() {

}
