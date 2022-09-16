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
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/evanw/esbuild/pkg/cli"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"os"
	"os/exec"
	"strings"
)

type Importer struct {
	file string
}

func NewImporter(file string) *Importer {
	return &Importer{
		file: file,
	}
}

func (imp *Importer) ImportJs() ([]byte, error) {
	result, err := imp.buildCode()
	if err != nil {
		return nil, err
	}

	transform := result.OutputFiles[0]

	code := imp.fix(string(transform.Contents))
	pterm.Println(code)

	return []byte(code), nil
}

func (imp *Importer) ImportTs()([]byte, error)  {
	VerifyNodeInstallation(imp)

	typescriptCmd := []string{"npx", "tt", imp.file}
	code, err := imp.Cmd(typescriptCmd)

	pterm.Println(string(code))
	return code, err
}

func VerifyNodeInstallation(imp *Importer)  {
	//check if node is installed
	checkForNodeCmd := []string{"node", "-v"}
	_, err := imp.Cmd(checkForNodeCmd)
	if err != nil {
		utils.HandleError(err)
	}
	//list out npm packages
	checkForLibCmd := []string{"npm", "list"}
	library, _ := imp.Cmd(checkForLibCmd)

	//check if the package needed is installed.
	pkgList := strings.Split(string(library), "\n")
	pkgName := "datahub-tslib"
	isPackageInstalled := utils.ListContainsSubstr(pkgList,pkgName)

	if isPackageInstalled == false{
		pterm.Error.Println(fmt.Sprintf("Missing datahub-tslib package."))
		pterm.Error.Println(fmt.Sprintf("Please install it. https://open.mimiro.io/software/typescript/"))
		os.Exit(1)
	}
}

func (imp *Importer) Cmd(cmd []string) ([]byte, error) {
	cmdExec := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(cmd, " ")))
	return cmdExec.CombinedOutput()
}

func (imp *Importer) Encode(code []byte) string {
	return base64.StdEncoding.EncodeToString(code)
}

func (imp *Importer) buildCode() (*api.BuildResult, error) {
	options, err := cli.ParseBuildOptions([]string{
		imp.file,
		"--bundle",
		"--format=esm",
		"--target=es2016",
		"--outfile=out.js",
	})
	if err != nil {
		return nil, err
	}

	result := api.Build(options)

	for _, w := range result.Warnings {
		pterm.Warning.Printf(w.Text)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			pterm.Error.Println(fmt.Sprintf("%s:%v", e.Text, e.Location))
		}
		return nil, errors.New("something wrong happened with the compile")
	}

	return &result, nil
}

func (imp *Importer) fix(content string) string {
	if strings.Contains(content, "export") {
		i := strings.Index(content, "export")
		c := content[:i]
		return c
	}
	return content
}
