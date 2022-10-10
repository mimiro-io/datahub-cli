package api

import (
	"net/url"
	"strings"
)

type SinkExpander struct {
	Sink Sink
}

func (s SinkExpander) ProcessEntities(entities []*Entity) error {
	var nsMap map[string]interface{}
	fExpand := ValueExpander(nil) // default, no mappings so no expansion
	for _, e := range entities {
		if e.ID == "@context" && e.Properties["namespaces"] != nil {
			nsMap = e.Properties["namespaces"].(map[string]interface{})
			fExpand = ValueExpander(nsMap)
		} else {
			ExpandEntity(e, fExpand)
		}
	}
	return s.Sink.ProcessEntities(entities)
}

func ExpandEntity(e *Entity, fExpand func(string) string) {
	e.ID = fExpand(e.ID)
	newRefs := map[string]interface{}{}
	for k, v := range e.References {
		newVal := expandInterface(v, fExpand)
		newRefs[fExpand(k)] = newVal
	}
	e.References = newRefs
	newProps := map[string]interface{}{}
	for k, v := range e.Properties {
		newVal := expandInterface(v, fExpand)
		newProps[fExpand(k)] = newVal
	}
	e.Properties = newProps
}

func expandInterface(v interface{}, fExpand func(string) string) interface{} {
	newVal := v
	switch val := v.(type) {
	case string:
		newVal = fExpand(val)
	case []string:
		for i, str := range val {
			val[i] = fExpand(str)
		}
		newVal = val
	case []interface{}:
		for i, inter := range val {
			val[i] = expandInterface(inter, fExpand)
		}
		newVal = val
	case Entity:
		ExpandEntity(&val, fExpand)
		newVal = val
	case *Entity:
		ExpandEntity(val, fExpand)
		newVal = val
	}
	return newVal
}

func ValueExpander(nsMap map[string]interface{}) func(string) string {
	return func(value string) string {
		tokens := strings.Split(value, ":")
		if len(tokens) >= 2 {
			expansion := nsMap[tokens[0]]
			if uri, ok := expansion.(string); ok {
				res, err := url.JoinPath(uri, strings.TrimPrefix(value, tokens[0]+":"))
				if err != nil {
					return value
				}
				return res
			}
		}
		return value
	}
}

func (s SinkExpander) Start() {
	s.Sink.Start()
}

func (s SinkExpander) End() {
	s.Sink.End()
}
