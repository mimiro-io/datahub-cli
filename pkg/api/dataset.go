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

package api

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
)

type Dataset struct {
	Name  string   `json:"name"`
	Type  []string `json:"type"`
	Items int      `json:"items"`
}

type DatasetManager struct {
	server string
	token  string
}

func NewDatasetManager(server string, token string) *DatasetManager {
	return &DatasetManager{
		server: server,
		token:  token,
	}
}

func (dm *DatasetManager) List() ([]Dataset, error) {
	datasets := make([]Dataset, 0)
	data, err := web.GetRequest(dm.server, dm.token, "/datasets")
	if err != nil {
		return nil, err
	} else {
		err = json.Unmarshal(data, &datasets)
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(datasets, func(i, j int) bool {
		return datasets[i].Name < datasets[j].Name
	})

	return datasets, nil
}

func (dm *DatasetManager) Get(name string) (*Entity, error) {
	res, err := web.GetRequest(dm.server, dm.token, "/datasets/"+name)
	if err != nil {
		return nil, err
	}

	e := &Entity{}

	err = json.Unmarshal(res, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (dm *DatasetManager) Rename(name string, newName string) error {
	payload, err := json.Marshal(map[string]string{"ID": newName})
	if err != nil {
		return err
	}

	_, err = web.PatchRequest(dm.server, dm.token, "/datasets/"+name, payload)
	if err != nil {
		return err
	}
	return nil
}

func GetDatasetsCompletion(pattern string) []string {
	server, token, err := login.ResolveCredentials()
	utils.HandleError(err)

	datasets, err := web.GetRequest(server, token, "/datasets")
	utils.HandleError(err)

	datasetlist := make([]Dataset, 0)
	err = json.Unmarshal(datasets, &datasetlist)
	utils.HandleError(err)

	var datasetIds []string

	for _, dataset := range datasetlist {
		if strings.HasPrefix(strings.ToLower(dataset.Name), strings.ToLower(pattern)) {
			datasetIds = append(datasetIds, dataset.Name)
		}
	}
	return datasetIds
}
