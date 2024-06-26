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
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"strconv"
	"strings"

	"github.com/mimiro-io/datahub-cli/pkg/api"

	"github.com/gofrs/uuid"

	"github.com/pterm/pterm"

	"github.com/mimiro-io/datahub-cli/internal/queries"
)

type transformer struct {
	query            *queries.QueryBuilder
	assertedPrefixes map[string]string
}

func newTransformer(server string, token string) *transformer {
	namespaces := getNamespaces(server, token)
	assertedPrefixes := make(map[string]string, len(namespaces))
	for k, v := range namespaces {
		assertedPrefixes[v] = k
	}
	return &transformer{
		query:            queries.NewQueryBuilder(server, token),
		assertedPrefixes: assertedPrefixes,
	}
}

func getNamespaces(server string, token string) map[string]string {
	ns, err := web.GetRequest(server, token, "/namespaces")
	if err != nil {
		utils.HandleError(err)
	}
	var namespaces map[string]string
	err = json.Unmarshal(ns, &namespaces)
	if err != nil {
		utils.HandleError(err)
	}
	return namespaces

}

func (tf *transformer) Log(thing interface{}, logLevel string) {
	switch strings.ToLower(logLevel) {
	case "info":
		pterm.Info.Println(fmt.Sprintf("- %v", thing))
	case "warn", "warning":
		pterm.Warning.Println(fmt.Sprintf("- %v", thing))
	case "err", "error":
		pterm.Error.Println(fmt.Sprintf("- %v", thing))
	default:
		pterm.Info.Println(fmt.Sprintf("- %v", thing))
	}
}

func (tf *transformer) UUID() string {
	uid, _ := uuid.NewV4()
	return fmt.Sprintf("%s", uid)
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

func (tf *transformer) Query(startingEntities []string, predicate string, inverse bool, datasets []string) [][]interface{} {
	result, err := tf.query.Query(startingEntities, predicate, inverse, datasets)
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

func (tf *transformer) ById(entityId string, datasets []string) *api.Entity {
	entity, _, err := tf.query.QuerySingle(entityId, false, datasets)
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

func (tf *transformer) ToString(obj interface{}) string {
	if obj == nil {
		return "undefined"
	}

	switch obj.(type) {
	case *api.Entity:
		return fmt.Sprintf("%v", obj)
	case map[string]interface{}:
		return fmt.Sprintf("%v", obj)
	case int, int32, int64:
		return fmt.Sprintf("%d", obj)
	case float32, float64:
		return fmt.Sprintf("%g", obj)
	case bool:
		return fmt.Sprintf("%v", obj)
	default:
		return fmt.Sprintf("%s", obj)
	}
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

func (javascriptTransform *transformer) AsEntity(val interface{}) (res *api.Entity) {
	if e, ok := val.(*api.Entity); ok {
		res = e
		return
	}
	if m, ok := val.(map[string]interface{}); ok {
		defer func() {
			if recover() != nil {
				res = nil
			}
		}()
		res = api.NewEntityFromMap(m)
		return
	}
	res = nil
	return
}
