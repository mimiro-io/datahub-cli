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
	"github.com/mimiro-io/datahub-cli/internal/api"
)

type Printer interface {
	Print(entities []interface{})
	BatchSize() int
	Header(entity interface{})
	Footer()
}

type ExpandingPrinter struct {
	Printer Printer
	fExpand func(string) string
}

func (p *ExpandingPrinter) BatchSize() int {
	return p.Printer.BatchSize()
}

func (p *ExpandingPrinter) Header(entity interface{}) {
	nsMap := entity.(*api.Entity).Properties["namespaces"].(map[string]interface{})
	p.fExpand = api.ValueExpander(nsMap)
	p.Printer.Header(entity)
}

func (p *ExpandingPrinter) Footer() {
	p.Printer.Footer()
}

func (p *ExpandingPrinter) Print(input []interface{}) {
	for _, e := range input {
		row := e.([]interface{})
		row[0] = p.fExpand(row[0].(string)) // expand start id
		row[1] = p.fExpand(row[1].(string)) // expand predicatetUri
		obj := row[2].(*api.Entity)
		api.ExpandEntity(obj, p.fExpand)
	}
	p.Printer.Print(input)
}
