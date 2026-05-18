// Copyright 2026 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

// Package transform runs MIMIRO transform scripts (TypeScript or JavaScript)
// against a hub via esbuild + goja.
package transform

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"

	"github.com/mimiro-io/datahub-cli/pkg/api"
)

type Logger interface {
	Log(entry LogEntry)
}

type LogEntry struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"ts"`
}

type Options struct {
	HubURL  string
	Bearer  string
	Logger  Logger
	Timeout time.Duration

	// All three nil in production. The operator UI uses QueryRecorder to
	// discover what a transform pulled in via Query/FindById, OverlayResolver
	// to substitute mocked entity fields before user code observes them, and
	// SyntheticProvider to inject user-authored mock entities into FindById /
	// Query results when the hub itself wouldn't surface them.
	QueryRecorder     QueryRecorder
	OverlayResolver   OverlayResolver
	SyntheticProvider SyntheticProvider
}

// QueryRecorder observes the entities surfaced through Query / FindById, in
// the order they reach user code (i.e. after OverlayResolver has run).
type QueryRecorder interface {
	RecordQuery(record QueryRecord)
}

type QueryRecord struct {
	Kind        string        // "query" | "findById"
	StartingIds []string
	Predicate   string
	Inverse     bool
	Datasets    []string
	Entities    []*api.Entity
}

// OverlayResolver gets a shot at every entity emerging from Query / FindById
// before it reaches user code. Implementations return the entity to hand on —
// either the original or a (cloned and) mutated version. Returning nil signals
// "no overlay; use the original".
type OverlayResolver interface {
	ResolveOverlay(e *api.Entity) *api.Entity
}

// SyntheticProvider injects user-authored mock entities into the runtime.
//
// Lookup is called by tf.ById and the cli's Query-augmentation path to resolve
// a specific id against the operator's authored mocks. If a synthetic exists
// with that id (and a matching dataset, when the call passes a non-empty
// datasets filter), it stands in for any hub result. The synthetic is then
// run through OverlayResolver just like a hub entity would be, so pinned
// overrides still apply.
//
// All returns every synthetic that passes the datasets filter — used by the
// cli's Query augmentation to walk the candidate pool for inverse lookups
// (entities whose refs[predicate] points at one of the startingIds) and as
// the fallback target pool when the hub returns no tuples.
//
// The datasets slice is honoured in both methods: when non-empty, the
// synthetic's dataset must appear in the slice to participate.
type SyntheticProvider interface {
	Lookup(id string, datasets []string) *api.Entity
	All(datasets []string) []*api.Entity
}

// Compile and runtime errors surface as level=error entries on Logs with
// empty Entities; the error return is reserved for cancellation / timeout.
type Result struct {
	Entities   []*api.Entity `json:"entities"`
	Logs       []LogEntry    `json:"logs"`
	DurationMs int64         `json:"durationMs"`
}

func Run(ctx context.Context, source string, entities []*api.Entity, opts Options) (*Result, error) {
	start := time.Now()
	res := &Result{
		Entities: []*api.Entity{},
		Logs:     []LogEntry{},
	}

	collector := &logCollector{logger: opts.Logger, entries: []LogEntry{}}

	code, compileErr := compile(source)
	if compileErr != nil {
		collector.add(LogEntry{
			Level:     "error",
			Message:   compileErr.Error(),
			Timestamp: time.Now(),
		})
		res.Logs = collector.entries
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}

	tf := newTransformer(opts.HubURL, opts.Bearer, collector, opts.QueryRecorder, opts.OverlayResolver, opts.SyntheticProvider, entities)
	engine := goja.New()
	hookEngine(engine, tf)

	// Helpers must load before user code: the namespace shim emitted by
	// compile() captures helper references eagerly via object-literal binding.
	if _, err := engine.RunString(helperJavascriptFunctions); err != nil {
		return nil, fmt.Errorf("install helpers: %w", err)
	}

	if _, err := engine.RunString(code); err != nil {
		collector.add(LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("script load: %s", err.Error()),
			Timestamp: time.Now(),
		})
		res.Logs = collector.entries
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}
	if _, err := engine.RunString(wrapperJavascriptFunction); err != nil {
		return nil, fmt.Errorf("install wrapper: %w", err)
	}

	runCtx := ctx
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	out, runErr := transformEntities(runCtx, engine, entities)
	if runErr != nil {
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(runErr, context.Canceled) {
			return nil, runErr
		}
		collector.add(LogEntry{
			Level:     "error",
			Message:   runErr.Error(),
			Timestamp: time.Now(),
		})
		res.Logs = collector.entries
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}

	res.Entities = out
	res.Logs = collector.entries
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

func transformEntities(ctx context.Context, engine *goja.Runtime, entities []*api.Entity) ([]*api.Entity, error) {
	var transFunc func(entities []*api.Entity) (interface{}, error)
	if err := engine.ExportTo(engine.Get("transform_entities_ex"), &transFunc); err != nil {
		return nil, fmt.Errorf("export transform_entities: %w", err)
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			engine.Interrupt(ctx.Err())
		case <-done:
		}
	}()

	result, err := transFunc(entities)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	switch v := result.(type) {
	case nil:
		return []*api.Entity{}, nil
	case []interface{}:
		out := make([]*api.Entity, 0, len(v))
		for _, e := range v {
			ent, ok := e.(*api.Entity)
			if !ok {
				return nil, fmt.Errorf("transform_entities returned non-entity element: %T", e)
			}
			out = append(out, ent)
		}
		return out, nil
	case []*api.Entity:
		return v, nil
	default:
		return nil, fmt.Errorf("transform_entities returned unsupported type: %T", v)
	}
}

// entries is initialised non-nil so a zero-log run marshals to `[]`, not `null`.
type logCollector struct {
	logger  Logger
	entries []LogEntry
}

func (c *logCollector) add(e LogEntry) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	c.entries = append(c.entries, e)
	if c.logger != nil {
		c.logger.Log(e)
	}
}
