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
	"fmt"
	"strconv"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/api"

	"github.com/mimiro-io/datahub-cli/internal/queries"
	"github.com/pterm/pterm"
)

type transformer struct {
	query            *queries.QueryBuilder
	assertedPrefixes map[string]string
}

func (tf *transformer) Log(thing interface{}) {
	pterm.Info.Println(fmt.Sprintf("- %v", thing))
}

func (tf *transformer) MakeEntityArray(entities []interface{}) []*api.Entity {
	newArray := make([]*api.Entity, 0)
	for _, e := range entities {
		newArray = append(newArray, e.(*api.Entity))
	}
	return newArray
}

func (tf *transformer) GetNamespacePrefix(urlExpansion string) string {
	result, err := tf.query.GetNamespacePrefix(urlExpansion)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return "ns0"
		}

		pterm.Error.Print(err)
		return ""
	}

	return result
}

func (tf *transformer) AssertNamespacePrefix(urlExpansion string) string {
	if val, ok := tf.assertedPrefixes[urlExpansion]; ok {
		return val
	} else {
		l := len(tf.assertedPrefixes)
		prefix := "ns" + strconv.Itoa(l)
		tf.assertedPrefixes[urlExpansion] = prefix
		return prefix
	}
}

func (tf *transformer) Query(startingEntities []string, predicate string, inverse bool) [][]interface{} {
	result, err := tf.query.Query(startingEntities, predicate, inverse)
	if err != nil {
		pterm.Error.Print(err)
		return nil
	}
	data := make([][]interface{}, len(result.Data))
	for i, e := range result.Data {
		r := make([]interface{}, 3)
		r[0] = e.Uri
		r[1] = e.PredicateUri
		r[2] = e.Entity

		data[i] = r
	}

	return data
}

func (tf *transformer) ById(entityId string) *api.Entity {
	entity, err := tf.query.QuerySingle(entityId)
	if err != nil {
		return nil
	}
	return entity
}

func (tf *transformer) NewEntity() *api.Entity {
	entity := &api.Entity{}
	entity.References = map[string]interface{}{}
	entity.Properties = map[string]interface{}{}
	return entity
}

func (tf *transformer) IsValidEntity(entity *api.Entity) bool {
	if entity == nil {
		return false
	}
	if entity.Recorded == 0 {
		return false
	}
	return true
}
