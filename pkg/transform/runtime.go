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
	recorder         QueryRecorder
	resolver         OverlayResolver
	synthetics       SyntheticProvider
	// inputs indexes the entities slice passed to Run so Query/FindById can
	// consult the user's working set before falling through to synthetics or
	// the hub. This lets a transform run end-to-end even when the source
	// entity isn't on the hub yet (a common case during operator scratchpad
	// work).
	inputs map[string]*api.Entity
}

func newTransformer(hubURL, bearer string, logs *logCollector, recorder QueryRecorder, resolver OverlayResolver, synthetics SyntheticProvider, inputs []*api.Entity) *transformer {
	asserted := make(map[string]string)
	if hubURL != "" {
		if ns, err := fetchNamespaces(hubURL, bearer); err == nil {
			for prefix, expansion := range ns {
				asserted[expansion] = prefix
			}
		}
	}
	idx := make(map[string]*api.Entity, len(inputs))
	for _, e := range inputs {
		if e != nil && e.ID != "" {
			idx[e.ID] = e
		}
	}
	return &transformer{
		query:            newQueryClient(hubURL, bearer),
		assertedPrefixes: asserted,
		logs:             logs,
		recorder:         recorder,
		resolver:         resolver,
		synthetics:       synthetics,
		inputs:           idx,
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
	result, hubErr := tf.query.Query(startingEntities, predicate, inverse, datasets)
	out := make([][]interface{}, 0)
	recorded := make([]*api.Entity, 0)
	seen := make(map[string]struct{})

	if hubErr == nil {
		for _, r := range result {
			entity := tf.applyOverlay(r.Entity)
			out = append(out, []interface{}{r.Uri, r.PredicateUri, entity})
			if entity != nil {
				recorded = append(recorded, entity)
				seen[queryTupleKey(r.Uri, entity.ID)] = struct{}{}
			}
		}
	}

	pool := tf.candidatePool(datasets)
	starting := make(map[string]struct{}, len(startingEntities))
	for _, s := range startingEntities {
		starting[s] = struct{}{}
	}

	if !inverse {
		for _, startId := range startingEntities {
			src, ok := pool[startId]
			if !ok || src == nil || src.References == nil {
				continue
			}
			refVal, refOK := src.References[predicate]
			if !refOK {
				continue
			}
			for _, targetId := range refValueToStrings(refVal) {
				key := queryTupleKey(startId, targetId)
				if _, dup := seen[key]; dup {
					continue
				}
				tgt := tf.resolveTarget(targetId, datasets)
				if tgt == nil {
					continue
				}
				tgt = tf.applyOverlay(tgt)
				if tgt == nil {
					continue
				}
				out = append(out, []interface{}{startId, predicate, tgt})
				recorded = append(recorded, tgt)
				seen[key] = struct{}{}
			}
		}
	} else {
		for _, cand := range pool {
			if cand == nil || cand.References == nil {
				continue
			}
			refVal, refOK := cand.References[predicate]
			if !refOK {
				continue
			}
			for _, targetId := range refValueToStrings(refVal) {
				if _, isStart := starting[targetId]; !isStart {
					continue
				}
				key := queryTupleKey(targetId, cand.ID)
				if _, dup := seen[key]; dup {
					continue
				}
				resolved := tf.applyOverlay(cand)
				if resolved == nil {
					continue
				}
				out = append(out, []interface{}{targetId, predicate, resolved})
				recorded = append(recorded, resolved)
				seen[key] = struct{}{}
				break
			}
		}
	}

	// Surface the hub failure only when augmentation didn't rescue the call.
	// If inputs/synthetics produced tuples, the operator authored a mock on
	// purpose — the run is succeeding by design and a warn would be friction.
	if hubErr != nil && len(out) == 0 {
		tf.logs.add(LogEntry{
			Level:     "warn",
			Message:   fmt.Sprintf("Query: hub returned %q; no inputs/synthetics matched", hubErr.Error()),
			Timestamp: time.Now(),
		})
	}

	tf.recordQuery(QueryRecord{
		Kind:        "query",
		StartingIds: startingEntities,
		Predicate:   predicate,
		Inverse:     inverse,
		Datasets:    datasets,
		Entities:    recorded,
	})
	return out
}

// candidatePool merges the input working set (always — inputs have no dataset
// tag) with synthetics filtered by the datasets argument. The returned map is
// keyed by entity id; on collision the synthetic wins, since the operator has
// explicitly authored it as the source of truth.
func (tf *transformer) candidatePool(datasets []string) map[string]*api.Entity {
	pool := make(map[string]*api.Entity, len(tf.inputs))
	for id, e := range tf.inputs {
		pool[id] = e
	}
	if tf.synthetics != nil {
		for _, syn := range tf.synthetics.All(datasets) {
			if syn != nil && syn.ID != "" {
				pool[syn.ID] = syn
			}
		}
	}
	return pool
}

// resolveTarget walks inputs → synthetics → hub. Used by Query forward
// augmentation when chasing a ref target out of the user's working set.
func (tf *transformer) resolveTarget(id string, datasets []string) *api.Entity {
	if e, ok := tf.inputs[id]; ok {
		return e
	}
	if tf.synthetics != nil {
		if syn := tf.synthetics.Lookup(id, datasets); syn != nil {
			return syn
		}
	}
	if e, err := tf.query.QuerySingle(id, datasets); err == nil && e != nil {
		return e
	}
	return nil
}

func queryTupleKey(subject, target string) string {
	return subject + "\x00" + target
}

func refValueToStrings(v interface{}) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func (tf *transformer) ById(entityId string, datasets []string) *api.Entity {
	// Inputs win — the user's working set is in scope even when no hub-side
	// representation exists yet. Dataset filter doesn't apply to inputs
	// (they aren't tagged with a dataset).
	if e, ok := tf.inputs[entityId]; ok {
		entity := tf.applyOverlay(e)
		if entity != nil {
			tf.recordQuery(QueryRecord{
				Kind:        "findById",
				StartingIds: []string{entityId},
				Datasets:    datasets,
				Entities:    []*api.Entity{entity},
			})
		}
		return entity
	}
	// Synthetic mocks take precedence over the hub — operators author them to
	// short-circuit a real lookup (typically because the dataset doesn't exist
	// on the hub yet).
	if tf.synthetics != nil {
		if syn := tf.synthetics.Lookup(entityId, datasets); syn != nil {
			syn = tf.applyOverlay(syn)
			if syn != nil {
				tf.recordQuery(QueryRecord{
					Kind:        "findById",
					StartingIds: []string{entityId},
					Datasets:    datasets,
					Entities:    []*api.Entity{syn},
				})
			}
			return syn
		}
	}
	entity, err := tf.query.QuerySingle(entityId, datasets)
	if err != nil {
		tf.logs.add(LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("FindById(%q): %s", entityId, err.Error()),
			Timestamp: time.Now(),
		})
		return nil
	}
	entity = tf.applyOverlay(entity)
	if entity != nil {
		tf.recordQuery(QueryRecord{
			Kind:        "findById",
			StartingIds: []string{entityId},
			Datasets:    datasets,
			Entities:    []*api.Entity{entity},
		})
	}
	return entity
}

func (tf *transformer) applyOverlay(e *api.Entity) *api.Entity {
	if tf.resolver == nil || e == nil {
		return e
	}
	if merged := tf.resolver.ResolveOverlay(e); merged != nil {
		return merged
	}
	return e
}

func (tf *transformer) recordQuery(record QueryRecord) {
	if tf.recorder == nil {
		return
	}
	tf.recorder.RecordQuery(record)
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
