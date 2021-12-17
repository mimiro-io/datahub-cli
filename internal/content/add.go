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

package content

import (
	"encoding/json"
	"errors"
	"github.com/mimiro-io/datahub-cli/internal/web"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type content struct {
	Id   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds a new content or updates an existing.",
	Long: `Adds a new content or updates an existing. For example:
mim content add file=myfile.json

or

cat myfile.json | mim content add

`,
	Run: func(cmd *cobra.Command, args []string) {
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		id, err := cmd.Flags().GetString("id")
		utils.HandleError(err)
		file, err := cmd.Flags().GetString("file")

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Adding content to " + server + "/content/" + id)

		conf, err := utils.ReadInput(file)
		utils.HandleError(err)

		content, err := convert(conf)
		utils.HandleError(err)

		conf, err = updateId(id, content)
		utils.HandleError(err)

		pterm.Success.Println("Read content file")

		_, err = web.PostRequest(server, token, "/content", conf)
		utils.HandleError(err)

		pterm.Success.Println("Added content to server")
		pterm.Println()
	},
	TraverseChildren: true,
}

func convert(contentBytes []byte) (*content, error) {
	config := make(map[string]interface{})
	err := json.Unmarshal(contentBytes, &config)
	if err != nil {
		return nil, err
	}

	contentObj := &content{
		Id:   "",
		Data: config,
	}
	if id, ok := config["id"]; ok {
		contentObj.Id = id.(string)
	}

	return contentObj, nil
}

func updateId(id string, contentObj *content) ([]byte, error) {
	if id != "" {
		contentObj.Id = id
	}
	if contentObj.Id == "" {
		return nil, errors.New("you have to provide an id")
	}

	k, err := json.Marshal(contentObj)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func init() {
	AddCmd.Flags().StringP("id", "i", "", "The id of the content to add. This overrides the file id.")
	AddCmd.Flags().StringP("file", "f", "", "The input file. Must be json.")
}
