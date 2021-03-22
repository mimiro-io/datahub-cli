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
	"strings"
	"testing"

	. "github.com/franela/goblin"
)

func TestEntityStreamParser_ParseStream(t *testing.T) {
	json := `
[
	{
		"id" : "@context",
		"namespaces" : {
			"mimiro-people" : "http://data.mimiro.io/people/",
			"_" : "http://data.mimiro.io/core/"
		}
	},
	{
		"id" : "mimiro-people:homer",
		"props" : {
			"Name" : "Homer Simpson"
		},
		"refs" : {
			"friends" : [ "mimiro-people:marge" , "mimiro-people:bert"]
		}
	},
	{
		"id" : "@continuation",
	    "token" : "next-20"
	}
]`

	g := Goblin(t)
	g.Describe("Test parsing", func() {
		g.It("when given valid json it should parse", func() {
			reader := strings.NewReader(json)

			esp := NewEntityStreamParser()
			entities := make([]*Entity, 0)
			err := esp.ParseStream(reader, func(e *Entity) error {
				entities = append(entities, e)
				return nil
			})
			g.Assert(err).IsNil()
			g.Assert(len(entities)).Equal(3)
			g.Assert(entities[0].ID).Equal("@context")
			g.Assert(entities[2].ID).Equal("@continuation")
		})
	})
}
