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
	"encoding/json"
	"github.com/pterm/pterm"
	"github.com/tidwall/pretty"
	"os"
)

type PrettyPrint struct {
	Batch int
}

func (p *PrettyPrint) Print(entities []interface{}) {
	for _, e := range entities {
		p.println(e, ",")
	}
}

func (p *PrettyPrint) Header(entity interface{}) {
	if entity != nil {
		p.println(entity, "")
	}
}

func (p *PrettyPrint) Footer() {
	// do nothing here
}

func (p *PrettyPrint) BatchSize() int {
	return p.Batch
}

func (p *PrettyPrint) println(e interface{}, separator string) {
	layer, err := json.Marshal(e)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	f := pretty.Pretty(layer)
	result := pretty.Color(f, nil)

	pterm.Printf("%s%s", separator, string(result))
}
