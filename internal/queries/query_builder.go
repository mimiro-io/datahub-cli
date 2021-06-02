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

package queries

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/utils"
)

type QueryResult struct {
	Data []ResultPart
}

type ResultPart struct {
	Uri          string     `json:"uri"`
	PredicateUri string     `json:"predicateUri"`
	Entity       api.Entity `json:"entity"`
}

type QueryBuilder struct {
	token  string
	server string
}

type namespace struct {
	Prefix    string `json:"prefix"`
	Expansion string `json:"expansion"`
}

func (part *ResultPart) UnmarshalJSON(data []byte) error {
	var v []interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	part.Uri, _ = v[0].(string)
	part.PredicateUri, _ = v[1].(string)

	if v[2] != nil {
		raw, err := json.Marshal(v[2])
		if err != nil {
			return err
		}
		entity := api.Entity{}
		err = json.Unmarshal(raw, &entity)
		if err != nil {
			return err
		}
		part.Entity = entity
	}

	return nil
}

func NewQueryBuilder(server string, token string) *QueryBuilder {
	return &QueryBuilder{
		server: server,
		token:  token,
	}
}

func (qb *QueryBuilder) QuerySingle(entityId string) (*api.Entity, error) {
	q := make(map[string]interface{})
	q["entityId"] = entityId

	content, err := json.Marshal(&q)
	if err != nil {
		return nil, err
	}

	res, err := utils.PostRequest(qb.server, qb.token, "/query", content)
	if err != nil {
		return nil, err
	}

	entity := make([]api.Entity, 0)
	err = json.Unmarshal(res, &entity)

	if err != nil {
		return nil, err
	}

	if len(entity) < 2 {
		return nil, errors.New("unexpected response")
	}

	return &entity[1], nil
}

func (qb *QueryBuilder) Query(startingEntities []string, predicate string, inverse bool) (*QueryResult, error) {
	q := make(map[string]interface{})
	q["startingEntities"] = startingEntities
	q["predicate"] = predicate
	q["inverse"] = inverse

	if predicate == "" {
		q["predicate"] = "*"
	}

	content, err := json.Marshal(&q)
	if err != nil {
		return nil, err
	}

	res, err := utils.PostRequest(qb.server, qb.token, "/query", content)
	if err != nil {
		return nil, err
	}

	// get rid of the context, a bit dirty, but it works
	ents := make([]interface{}, 0)
	err = json.Unmarshal(res, &ents)
	if err != nil {
		return nil, err
	}

	obj, err := json.Marshal(ents[1])
	if err != nil {
		return nil, err
	}

	entities := make([]ResultPart, 0)
	err = json.Unmarshal(obj, &entities)
	if err != nil {
		return nil, err
	}

	//pterm.Println(entities)

	return &QueryResult{Data: entities}, nil

}

func (qb *QueryBuilder) GetNamespacePrefix(urlExpansion string) (string, error) {
	res, err := utils.GetRequest(qb.server, qb.token, fmt.Sprintf("/query/namespace?expansion=%s", urlExpansion))
	if err != nil {
		return "", err
	}

	n := &namespace{}
	err = json.Unmarshal(res, n)
	if err != nil {
		return "", err
	}

	return n.Prefix, nil
}
