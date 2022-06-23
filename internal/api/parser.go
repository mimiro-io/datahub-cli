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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type EntityStreamParser struct {
	localNamespaces       map[string]string
	localPropertyMappings map[string]string
	processingContext     bool
}

func NewEntityStreamParser() *EntityStreamParser {
	esp := &EntityStreamParser{}
	esp.localNamespaces = make(map[string]string)
	esp.localPropertyMappings = make(map[string]string)
	return esp
}

func (esp *EntityStreamParser) ParseStream(reader io.Reader, emitEntity func(*Entity) error) error {

	decoder := json.NewDecoder(reader)

	// expect Start of array
	t, err := decoder.Token()
	if err != nil {
		return errors.New("parsing error: Bad token at Start of stream " + err.Error())
	}

	if delim, ok := t.(json.Delim); !ok || delim != '[' {
		return errors.New("parsing error: Expected [ at Start of document")
	}

	// decode context object
	context := make(map[string]interface{})
	err = decoder.Decode(&context)
	if err != nil {
		return errors.New("parsing error: Unable to decode context " + err.Error())
	}

	if context["id"] == "@context" {
		for k, v := range context["namespaces"].(map[string]interface{}) {
			esp.localNamespaces[k] = v.(string)
		}
	} else {
		return errors.New("first entity in array must be a context")
	}

	contextEntity := NewEntity("@context")
	contextEntity.Properties = context
	_ = emitEntity(contextEntity)

	for {
		t, err = decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return errors.New("parsing error: Unable to read next token " + err.Error())
			}
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '{' {
				e, err := esp.parseEntity(decoder)
				if err != nil {
					return errors.New("parsing error: Unable to parse entity: " + err.Error())
				}
				err = emitEntity(e)
				if err != nil {
					return err
				}
			} else if v == ']' {
				// done
				break
			}
		default:
			return errors.New("parsing error: unexpected value in entity array")
		}
	}

	return nil
}

func (esp *EntityStreamParser) parseEntity(decoder *json.Decoder) (*Entity, error) {
	e := &Entity{}
	e.Properties = make(map[string]interface{})
	e.References = make(map[string]interface{})
	isContinuation := false
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token " + err.Error())
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '}' {
				return e, nil
			}
		case string:
			if v == "id" {
				val, err := decoder.Token()
				if err != nil {
					return nil, errors.New("unable to read token of id value " + err.Error())
				}

				if val.(string) == "@continuation" {
					e.ID = "@continuation"
					isContinuation = true
				} else {
					nsId, err := esp.getNamespacedIdentifier(val.(string), esp.localNamespaces)
					if err != nil {
						return nil, err
					}
					e.ID = nsId
				}
			} else if v == "recorded" {
				val, err := decoder.Token()
				if err != nil {
					return nil, errors.New("unable to read token of recorded value " + err.Error())
				}
				e.Recorded = uint64(val.(float64))

			} else if v == "deleted" {
				val, err := decoder.Token()
				if err != nil {
					return nil, errors.New("unable to read token of deleted value " + err.Error())
				}
				e.IsDeleted = val.(bool)

			} else if v == "props" {
				e.Properties, err = esp.parseProperties(decoder)
				if err != nil {
					return nil, errors.New("unable to parse properties " + err.Error())
				}
			} else if v == "refs" {
				e.References, err = esp.parseReferences(decoder)
				if err != nil {
					return nil, errors.New("unable to parse references " + err.Error())
				}
			} else if v == "token" {
				if !isContinuation {
					return nil, errors.New("token property found but not a continuation entity")
				}
				val, err := decoder.Token()
				if err != nil {
					return nil, errors.New("unable to read continuation token value " + err.Error())
				}
				e.Properties = make(map[string]interface{})
				e.Properties["token"] = val
			} else {
				// log named property
				// read value
				_, err := decoder.Token()
				if err != nil {
					return nil, errors.New("unable to parse value of unknown key: " + v + err.Error())
				}
			}
		default:
			return nil, errors.New("unexpected value in entity")
		}
	}
}

func (esp *EntityStreamParser) parseReferences(decoder *json.Decoder) (map[string]interface{}, error) {
	refs := make(map[string]interface{})

	_, err := decoder.Token()
	if err != nil {
		return nil, errors.New("unable to read token of at Start of references " + err.Error())
	}

	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token in parse references " + err.Error())
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '}' {
				return refs, nil
			}
		case string:
			val, err := esp.parseRefValue(decoder)
			if err != nil {
				return nil, errors.New("unable to parse value of reference key " + v)
			}

			propName := esp.localPropertyMappings[v]
			if propName == "" {
				propName, err = esp.getNamespacedIdentifier(v, esp.localNamespaces)
				if err != nil {
					return nil, err
				}
				esp.localPropertyMappings[v] = propName
			}
			refs[propName] = val
		default:
			return nil, errors.New("unknown type")
		}
	}
}

func (esp *EntityStreamParser) parseProperties(decoder *json.Decoder) (map[string]interface{}, error) {
	props := make(map[string]interface{})

	_, err := decoder.Token()
	if err != nil {
		return nil, errors.New("unable to read token of at Start of properties " + err.Error())
	}

	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token in parse properties " + err.Error())
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '}' {
				return props, nil
			}
		case string:
			val, err := esp.parseValue(decoder)
			if err != nil {

				return nil, errors.New("unable to parse property value of key " + v + " err: " + err.Error())
			}

			if val != nil { // basically if both error is nil, and value is nil, we drop the field
				propName := esp.localPropertyMappings[v]
				if propName == "" {
					propName, err = esp.getNamespacedIdentifier(v, esp.localNamespaces)
					if err != nil {
						return nil, err
					}
					esp.localPropertyMappings[v] = propName
				}
				props[propName] = val
			}
		default:
			return nil, errors.New("unknown type")
		}
	}
}

func (esp *EntityStreamParser) parseRefValue(decoder *json.Decoder) (interface{}, error) {
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token in parse value " + err.Error())
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '[' {
				return esp.parseRefArray(decoder)
			}
		case string:
			nsRef, err := esp.getNamespacedIdentifier(v, esp.localNamespaces)
			if err != nil {
				return nil, err
			}
			return nsRef, nil
		default:
			return nil, errors.New("unknown token in parse ref value")
		}
	}
}

func (esp *EntityStreamParser) parseRefArray(decoder *json.Decoder) ([]string, error) {
	array := make([]string, 0)
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token in parse ref array " + err.Error())
		}

		switch v := t.(type) {
		case json.Delim:
			if v == ']' {
				return array, nil
			}
		case string:
			nsRef, err := esp.getNamespacedIdentifier(v, esp.localNamespaces)
			if err != nil {
				return nil, err
			}
			array = append(array, nsRef)
		default:
			return nil, errors.New("unknown type")
		}
	}
}

func (esp *EntityStreamParser) parseArray(decoder *json.Decoder) ([]interface{}, error) {
	array := make([]interface{}, 0)
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token in parse array " + err.Error())
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '{' {
				r, err := esp.parseEntity(decoder)
				if err != nil {
					return nil, errors.New("unable to parse array " + err.Error())
				}
				array = append(array, r)
			} else if v == ']' {
				return array, nil
			} else if v == '[' {
				r, err := esp.parseArray(decoder)
				if err != nil {
					return nil, errors.New("unable to parse array " + err.Error())
				}
				array = append(array, r)
			}
		case string:
			array = append(array, v)
		case int:
			array = append(array, v)
		case float64:
			array = append(array, v)
		case bool:
			array = append(array, v)
		case nil:
			array = append(array, v)
		default:
			return nil, errors.New("unknown type")
		}
	}
}

func (esp *EntityStreamParser) parseValue(decoder *json.Decoder) (interface{}, error) {
	for {
		t, err := decoder.Token()
		if err != nil {
			return nil, errors.New("unable to read token in parse value " + err.Error())
		}

		if t == nil {
			// there is a good chance that we got a null value, and we need to handle that
			return nil, nil
		}

		switch v := t.(type) {
		case json.Delim:
			if v == '{' {
				return esp.parseEntity(decoder)
			} else if v == '[' {
				return esp.parseArray(decoder)
			}
		case string:
			return v, nil
		case int:
			return v, nil
		case float64:
			return v, nil
		case bool:
			return v, nil
		default:
			return nil, errors.New("unknown token in parse value")
		}
	}
}

func (esp *EntityStreamParser) getNamespacedIdentifier(val string, localNamespaces map[string]string) (string, error) {

	if val == "" {
		return "", errors.New("empty value not allowed")
	}

	if strings.HasPrefix(val, "http://") {
		expansion, lastPathPart, err := getUrlParts(val)
		if err != nil {
			return "", err
		}

		// check for global expansion
		prefix, err := esp.assertPrefixMappingForExpansion(expansion)
		if err != nil {
			return "", nil
		}
		return prefix + ":" + lastPathPart, nil
	}

	if strings.HasPrefix(val, "https://") {
		expansion, lastPathPart, err := getUrlParts(val)
		if err != nil {
			return "", err
		}

		// check for global expansion
		prefix, err := esp.assertPrefixMappingForExpansion(expansion)
		if err != nil {
			return "", err
		}
		return prefix + ":" + lastPathPart, nil
	}

	indexOfColon := strings.Index(val, ":")
	if indexOfColon == -1 {
		localExpansion := localNamespaces["_"]
		if localExpansion == "" {
			return "", fmt.Errorf("(property '%v' without prefix) no expansion for default prefix _ ", val)
		}

		prefix, err := esp.assertPrefixMappingForExpansion(localExpansion)
		if err != nil {
			return "", err
		}
		return prefix + ":" + val, nil

	} else {
		return val, nil
	}
}

func (esp *EntityStreamParser) assertPrefixMappingForExpansion(uriExpansion string) (string, error) {

	prefix := esp.localNamespaces[uriExpansion]
	if prefix == "" {
		prefix = "ns" + strconv.Itoa(len(esp.localNamespaces))
		esp.localNamespaces[prefix] = uriExpansion
	}
	return prefix, nil
}

func getUrlParts(url string) (string, string, error) {

	index := strings.LastIndex(url, "#")
	if index > -1 {
		return url[:index+1], url[index+1:], nil
	}

	index = strings.LastIndex(url, "/")
	if index > -1 {
		return url[:index+1], url[index+1:], nil
	}

	return "", "", errors.New("unable to split url") // fixme do something better
}
