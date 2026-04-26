// Copyright 2026 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package transform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/gofrs/uuid"

	"github.com/mimiro-io/datahub-cli/pkg/api"
)

type transformer struct {
	query            *queryClient
	assertedPrefixes map[string]string
	logs             *logCollector
}

func newTransformer(hubURL, bearer string, logs *logCollector) *transformer {
	asserted := make(map[string]string)
	if hubURL != "" {
		if ns, err := fetchNamespaces(hubURL, bearer); err == nil {
			for prefix, expansion := range ns {
				asserted[expansion] = prefix
			}
		}
	}
	return &transformer{
		query:            newQueryClient(hubURL, bearer),
		assertedPrefixes: asserted,
		logs:             logs,
	}
}

func hookEngine(engine *goja.Runtime, tf *transformer) {
	engine.Set("Query", tf.Query)
	engine.Set("FindById", tf.ById)
	engine.Set("GetNamespacePrefix", tf.GetNamespacePrefix)
	engine.Set("AssertNamespacePrefix", tf.AssertNamespacePrefix)
	engine.Set("Log", tf.Log)
	engine.Set("NewEntity", tf.NewEntity)
	engine.Set("IsValidEntity", tf.IsValidEntity)
	engine.Set("ToString", tf.ToString)
	engine.Set("AsEntity", tf.AsEntity)
	engine.Set("UUID", tf.UUID)
}

func (tf *transformer) Log(thing interface{}, logLevel string) {
	level := strings.ToLower(logLevel)
	switch level {
	case "info", "warn", "warning", "err", "error":
	default:
		level = "info"
	}
	if level == "warning" {
		level = "warn"
	}
	if level == "err" {
		level = "error"
	}
	tf.logs.add(LogEntry{
		Level:     level,
		Message:   fmt.Sprintf("%v", thing),
		Timestamp: time.Now(),
	})
}

func (tf *transformer) UUID() string {
	uid, _ := uuid.NewV4()
	return uid.String()
}

func (tf *transformer) GetNamespacePrefix(urlExpansion string) string {
	prefix, err := tf.query.GetNamespacePrefix(urlExpansion)
	if err != nil {
		// 404 → fall back to ns0, matching internal/transform behaviour.
		if strings.Contains(err.Error(), "404") {
			return "ns0"
		}
		tf.logs.add(LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("GetNamespacePrefix(%q): %s", urlExpansion, err.Error()),
			Timestamp: time.Now(),
		})
		return ""
	}
	return prefix
}

func (tf *transformer) AssertNamespacePrefix(urlExpansion string) string {
	if val, ok := tf.assertedPrefixes[urlExpansion]; ok {
		return val
	}
	prefix := "ns" + strconv.Itoa(len(tf.assertedPrefixes))
	tf.assertedPrefixes[urlExpansion] = prefix
	return prefix
}

func (tf *transformer) Query(startingEntities []string, predicate string, inverse bool, datasets []string) [][]interface{} {
	result, err := tf.query.Query(startingEntities, predicate, inverse, datasets)
	if err != nil {
		tf.logs.add(LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("Query: %s", err.Error()),
			Timestamp: time.Now(),
		})
		return nil
	}
	out := make([][]interface{}, len(result))
	for i, r := range result {
		out[i] = []interface{}{r.Uri, r.PredicateUri, r.Entity}
	}
	return out
}

func (tf *transformer) ById(entityId string, datasets []string) *api.Entity {
	entity, err := tf.query.QuerySingle(entityId, datasets)
	if err != nil {
		tf.logs.add(LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("FindById(%q): %s", entityId, err.Error()),
			Timestamp: time.Now(),
		})
		return nil
	}
	return entity
}

func (tf *transformer) NewEntity() *api.Entity {
	return &api.Entity{
		References: map[string]interface{}{},
		Properties: map[string]interface{}{},
	}
}

func (tf *transformer) ToString(obj interface{}) string {
	if obj == nil {
		return "undefined"
	}
	switch obj.(type) {
	case *api.Entity:
		return fmt.Sprintf("%v", obj)
	case map[string]interface{}:
		return fmt.Sprintf("%v", obj)
	case int, int32, int64:
		return fmt.Sprintf("%d", obj)
	case float32, float64:
		return fmt.Sprintf("%g", obj)
	case bool:
		return fmt.Sprintf("%v", obj)
	default:
		return fmt.Sprintf("%s", obj)
	}
}

func (tf *transformer) IsValidEntity(entity *api.Entity) bool {
	if entity == nil {
		return false
	}
	return entity.Recorded != 0
}

func (tf *transformer) AsEntity(val interface{}) (res *api.Entity) {
	if e, ok := val.(*api.Entity); ok {
		return e
	}
	if m, ok := val.(map[string]interface{}); ok {
		defer func() {
			if recover() != nil {
				res = nil
			}
		}()
		return api.NewEntityFromMap(m)
	}
	return nil
}
