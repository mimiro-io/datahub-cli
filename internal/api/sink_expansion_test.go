package api

import (
	"github.com/franela/goblin"
	"testing"
)

type RecordingSink struct {
	Entities []*Entity
}

func (r *RecordingSink) Start() {}
func (r *RecordingSink) End()   {}

func (r *RecordingSink) ProcessEntities(entities []*Entity) error {
	r.Entities = entities
	return nil
}

func TestUriExpansion(t *testing.T) {
	g := goblin.Goblin(t)
	baseInput := []*Entity{{
		ID: "@context",
		Properties: map[string]interface{}{
			"namespaces": map[string]interface{}{"ns1": "http://hello/", "ns2": "http://goodbye"},
		},
	}}
	g.Describe("prefix expansion", func() {
		g.It("should expand ids", func() {
			recorder := &RecordingSink{}
			test := &SinkExpander{Sink: recorder}
			err := test.ProcessEntities(append(baseInput, &Entity{
				ID: "ns1:bob",
			}))
			g.Assert(err).IsNil()
			g.Assert(len(recorder.Entities)).Equal(2, "context and bob")
			g.Assert(recorder.Entities[1].ID).Equal("http://hello/bob")
		})
		g.It("should keep unresolvable prefixes", func() {
			recorder := &RecordingSink{}
			test := &SinkExpander{Sink: recorder}
			err := test.ProcessEntities(append(baseInput, &Entity{
				ID: "bogus:bob",
			}))
			g.Assert(err).IsNil()
			g.Assert(len(recorder.Entities)).Equal(2, "context and bob")
			g.Assert(recorder.Entities[1].ID).Equal("bogus:bob")
		})
		g.It("should expand props", func() {
			recorder := &RecordingSink{}
			test := &SinkExpander{Sink: recorder}
			err := test.ProcessEntities(append(baseInput, &Entity{
				ID: "ns1:bob",
				Properties: map[string]interface{}{
					"ns1:friend": "ns2:frank",
					"family":     []string{"ns1:alice", "ns2:john", "ns3:jim"},
					"ns1:workplace": Entity{
						ID: "ns2:home",
						Properties: map[string]interface{}{
							"ns3:boss": "ns1:Ken",
						},
						References: map[string]interface{}{
							"ns1:friend": "ns2:alice",
						},
					},
				},
			}))
			g.Assert(err).IsNil()
			g.Assert(len(recorder.Entities)).Equal(2, "context and bob")
			g.Assert(recorder.Entities[1].Properties).Equal(map[string]interface{}{
				"family":              []string{"http://hello/alice", "http://goodbye/john", "ns3:jim"},
				"http://hello/friend": "http://goodbye/frank",
				"http://hello/workplace": Entity{
					ID:         "http://goodbye/home",
					References: map[string]interface{}{"http://hello/friend": "http://goodbye/alice"},
					Properties: map[string]interface{}{"ns3:boss": "http://hello/Ken"},
				},
			})
		})
		g.It("should expand refs", func() {
			recorder := &RecordingSink{}
			test := &SinkExpander{Sink: recorder}
			err := test.ProcessEntities(append(baseInput, &Entity{
				ID: "ns1:bob",
				References: map[string]interface{}{
					"ns1:friend": "ns2:frank",
					"family":     []string{"ns1:alice", "ns2:john", "ns3:jim"},
				},
			}))
			g.Assert(err).IsNil()
			g.Assert(len(recorder.Entities)).Equal(2, "context and bob")
			g.Assert(recorder.Entities[1].References).Equal(map[string]interface{}{
				"family":              []string{"http://hello/alice", "http://goodbye/john", "ns3:jim"},
				"http://hello/friend": "http://goodbye/frank",
			})
		})
	})
}
