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

package display

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"reflect"
)

func ToMap(input interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	elem := reflect.ValueOf(input).Elem()
	relType := elem.Type()
	for i := 0; i < relType.NumField(); i++ {
		m[relType.Field(i).Name] = elem.Field(i).Interface()
	}
	return m
}

type Driver interface {
	Render(items [][]string, header bool)
	RenderHeader(line string)
	RenderSuccess(line string)
	RenderWarning(line string, fatal bool)
	RenderError(err error, fatal bool)
	Msg(lines ...string)
	Must(v interface{}, err error) interface{}
}

func ResolveDriver(cmd *cobra.Command) Driver {
	return &Console{}
}

type Console struct {
}

func (c *Console) Must(v interface{}, err error) interface{} {
	c.RenderError(err, true)
	return v
}

func (c *Console) Msg(lines ...string) {
	for _, line := range lines {
		pterm.Println(line)
	}
}

func (c *Console) Render(items [][]string, header bool) {
	if header {
		pterm.DefaultTable.WithHasHeader().WithData(items).Render()
	} else {
		pterm.DefaultTable.WithData(items).Render()
	}
}

func (c *Console) RenderHeader(line string) {
	pterm.DefaultSection.Println(line)
}

func (c *Console) RenderSuccess(line string) {
	pterm.Success.Println(line)
	pterm.Println()
}

func (c *Console) RenderWarning(line string, fatal bool) {
	pterm.Warning.Println(line)
	pterm.Println()
	if fatal {
		os.Exit(1)
	}
}

func (c *Console) RenderError(err error, fatal bool) {
	if err != nil {
		pterm.Error.Println(err.Error())
		pterm.Println()
		if fatal {
			os.Exit(1)
		}
	}
}
