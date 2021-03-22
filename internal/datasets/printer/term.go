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

package printer

import (
	"fmt"
	"github.com/pterm/pterm"
)

type term struct {
	batchSize int
}

func (t *term) Print(entities []interface{}) {
	out := make([][]string, 0)
	out = append(out, []string{"Id", "InternalId", "Recorded", "Deleted", "Props", "Refs"})

	for _, e := range entities {
		raw := e.(map[string]interface{})
		id := raw["id"].(string)

		if id == "@context" {
			t.prettyContext(raw)
		} else if id == "@continuation" {
			pterm.DefaultSection.Println(fmt.Sprintf("Continuation token: %s", raw["token"]))
		} else {
			out = append(out, t.prettyEntity(id, raw))
		}
	}
	pterm.DefaultTable.WithHasHeader().WithData(out).Render()
}

func (t *term) Header(entity interface{}) {
	if entity != nil {
		raw := entity.(map[string]interface{})
		t.prettyContext(raw)
	}
}

func (t *term) Footer() {
	// do nothing
}

func (t *term) BatchSize() int {
	if t.batchSize > 100 { // prevent the console from crapping out
		return 102
	} else if t.batchSize < 1 {
		return 12
	}
	return t.batchSize + 2 // to accord for cont token and context
}

func (t *term) prettyEntity(id string, e map[string]interface{}) []string {
	internalId := "0"
	recorded := "0"

	if e["internalId"] != nil {
		internalId = fmt.Sprintf("%.f", e["internalId"].(float64))
	}
	if e["recorded"] != nil {
		recorded = fmt.Sprintf("%.f", e["recorded"].(float64))
	}

	return []string{
		id,
		internalId,
		recorded,
		fmt.Sprintf("%v", e["deleted"]),
		fmt.Sprintf("%v", e["props"]),
		fmt.Sprintf("%v", e["refs"]),
	}
}

func (t *term) prettyContext(context map[string]interface{}) {
	pterm.DefaultSection.Println("Namespaces:")
	namespaces := context["namespaces"].(map[string]interface{})
	out := make([][]string, 0)
	out = append(out, []string{"#", "Namespace"})
	for k, v := range namespaces {
		out = append(out, []string{
			k,
			fmt.Sprintf("%s", v),
		})
	}
	pterm.DefaultTable.WithHasHeader().WithData(out).Render()
}
