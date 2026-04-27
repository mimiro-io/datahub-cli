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

	tf := newTransformer(opts.HubURL, opts.Bearer, collector)
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
