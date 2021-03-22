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
	"fmt"
	"os"
)

type Raw struct {
	Batch int
}

func (r *Raw) Print(entities []interface{}) {
	for _, e := range entities {
		_, _ = os.Stdout.Write([]byte(","))
		r.writeEntity(e)
	}
}

func (r *Raw) Header(entity interface{}) {
	_, _ = os.Stdout.Write([]byte("["))
	if entity != nil {
		r.writeEntity(entity)
	} else {
		_, _ = os.Stdout.Write([]byte("{\"id\": \"@context\"}"))
	}
}

func (r *Raw) Footer() {
	_, _ = os.Stdout.Write([]byte("]\n"))
}

func (r *Raw) BatchSize() int {
	return r.Batch
}

func (r *Raw) writeEntity(e interface{}) {
	out, err := json.Marshal(e)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(out)
}
