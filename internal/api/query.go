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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bcicen/jstream"
	"io"
	"net/http"
	"net/url"
)

type EntityQuery struct {
	server string
	token  string
}

func NewEntityQuery(server string, token string) *EntityQuery {
	return &EntityQuery{
		server: server,
		token:  token,
	}
}

func (eq *EntityQuery) Query(entity []string, via string, inverse bool, datasets []string) ([]interface{}, error) {
	q := make(map[string]interface{})
	q["startingEntities"] = entity
	q["predicate"] = via
	q["inverse"] = inverse
	q["datasets"] = datasets

	if via == "" {
		q["predicate"] = "*"
	}

	content, err := json.Marshal(&q)
	if err != nil {
		return nil, err
	}

	endpoint, _ := url.Parse(fmt.Sprintf("%s/query", eq.server))
	req, err := http.NewRequest("POST", endpoint.String(), bytes.NewBuffer(content))
	if err != nil {
		return nil, err
	}
	if eq.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", eq.token))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New("error wrong http status: " + res.Status)
	}

	return eq.readBody(res.Body)
}

func (eq *EntityQuery) readBody(body io.Reader) ([]interface{}, error) {
	entities := make([]interface{}, 0)

	isFirst := true
	err := eq.parseStream(body, func(value *jstream.MetaValue) {
		if isFirst {
			//p.Header(value.Value)
			// emmit a @context
			context := NewContext()
			context.Properties = value.Value.(map[string]interface{})

			entities = append(entities, context)
			isFirst = false
		} else {
			for _, v := range value.Value.([]interface{}) {
				current := 1
				result := make([]interface{}, 3)
				for _, ent := range v.([]interface{}) {
					switch current {
					case 1:
						result[0] = ent.(string)
					case 2:
						result[1] = ent.(string)
					default:
						result[2] = NewEntityFromMap(ent.(map[string]interface{}))
						entities = append(entities, result)
					}
					current++
				}

			}

		}
	})
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (eq *EntityQuery) parseStream(reader io.Reader, emitEntity func(value *jstream.MetaValue)) error {
	decoder := jstream.NewDecoder(reader, 1)

	for mv := range decoder.Stream() {
		emitEntity(mv)
	}

	return nil
}
