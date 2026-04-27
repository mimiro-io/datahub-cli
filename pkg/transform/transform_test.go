// Copyright 2026 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package transform

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mimiro-io/datahub-cli/pkg/api"
)

func TestRun_PureTSPassthrough(t *testing.T) {
	src := `
		function transform_entities(entities: any[]) {
			return entities;
		}
	`
	in := []*api.Entity{
		{ID: "ns0:a", Properties: map[string]any{}, References: map[string]any{}},
		{ID: "ns0:b", Properties: map[string]any{}, References: map[string]any{}},
	}

	res, err := Run(context.Background(), src, in, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 2 {
		t.Fatalf("want 2 entities, got %d", len(res.Entities))
	}
	if res.Entities[0].ID != "ns0:a" || res.Entities[1].ID != "ns0:b" {
		t.Fatalf("wrong order: %+v", res.Entities)
	}
}

func TestRun_HelpersInjected(t *testing.T) {
	src := `
		function transform_entities(entities: any[]) {
			let out = [];
			for (let e of entities) {
				let n = NewEntity();
				SetId(n, GetId(e));
				SetProperty(n, "test", "doubled", (GetProperty(e, "test", "x", 0) as number) * 2);
				out.push(n);
			}
			return out;
		}
	`
	in := []*api.Entity{
		{ID: "x:1", Properties: map[string]any{"test:x": 5}, References: map[string]any{}},
	}

	res, err := Run(context.Background(), src, in, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1, got %d", len(res.Entities))
	}
	got := res.Entities[0]
	if got.ID != "x:1" {
		t.Errorf("ID = %q, want x:1", got.ID)
	}
	if v, ok := got.Properties["test:doubled"]; !ok || v != int64(10) {
		t.Errorf("test:doubled = %v (%T), want 10", v, v)
	}
}

func TestRun_LogCapture(t *testing.T) {
	src := `
		function transform_entities(entities: any[]) {
			Log("hello", "info");
			Log("careful", "warn");
			Log("oops", "error");
			return entities;
		}
	`
	res, err := Run(context.Background(), src, nil, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Logs) != 3 {
		t.Fatalf("want 3 logs, got %d: %+v", len(res.Logs), res.Logs)
	}
	want := []struct{ level, msg string }{
		{"info", "hello"},
		{"warn", "careful"},
		{"error", "oops"},
	}
	for i, w := range want {
		if res.Logs[i].Level != w.level || res.Logs[i].Message != w.msg {
			t.Errorf("Logs[%d] = %+v, want %s/%s", i, res.Logs[i], w.level, w.msg)
		}
	}
}

func TestRun_CompileErrorBecomesLogEntry(t *testing.T) {
	src := `function transform_entities(entities`
	res, err := Run(context.Background(), src, nil, Options{})
	if err != nil {
		t.Fatalf("Run returned error (should surface compile errors as log entries): %v", err)
	}
	if len(res.Entities) != 0 {
		t.Errorf("want empty Entities on compile failure, got %d", len(res.Entities))
	}
	if len(res.Logs) == 0 || res.Logs[0].Level != "error" {
		t.Fatalf("want an error log entry, got %+v", res.Logs)
	}
}

func TestRun_RuntimeErrorBecomesLogEntry(t *testing.T) {
	src := `
		function transform_entities(entities: any[]) {
			throw new Error("boom");
		}
	`
	res, err := Run(context.Background(), src, nil, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 0 {
		t.Errorf("want empty Entities on runtime failure, got %d", len(res.Entities))
	}
	if len(res.Logs) == 0 || res.Logs[0].Level != "error" {
		t.Fatalf("want runtime error in logs, got %+v", res.Logs)
	}
	if !strings.Contains(res.Logs[0].Message, "boom") {
		t.Errorf("log = %q, want it to contain 'boom'", res.Logs[0].Message)
	}
}

func TestRun_ContextCancellationInterrupts(t *testing.T) {
	src := `
		function transform_entities(entities: any[]) {
			while (true) {}
		}
	`
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := Run(ctx, src, nil, Options{})
	if err == nil {
		t.Fatal("want context error, got nil")
	}
	if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "context") {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestRun_TimeoutOptionInterrupts(t *testing.T) {
	src := `
		function transform_entities(entities: any[]) {
			while (true) {}
		}
	`
	_, err := Run(context.Background(), src, nil, Options{Timeout: 100 * time.Millisecond})
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

// Regression: Result.Logs/Entities must marshal to `[]`, not `null`.
func TestRun_LogsAndEntitiesNeverNilOnSuccess(t *testing.T) {
	src := `function transform_entities(entities) { return entities; }`
	res, err := Run(context.Background(), src, []*api.Entity{}, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Logs == nil {
		t.Errorf("Result.Logs is nil; want empty slice (would marshal to JSON null)")
	}
	if res.Entities == nil {
		t.Errorf("Result.Entities is nil; want empty slice")
	}
}

func TestRun_ExternalLoggerDualWrite(t *testing.T) {
	var captured []LogEntry
	logger := loggerFunc(func(e LogEntry) { captured = append(captured, e) })

	src := `function transform_entities(entities: any[]) { Log("x", "info"); return entities; }`
	res, err := Run(context.Background(), src, nil, Options{Logger: logger})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Logs) != 1 || len(captured) != 1 {
		t.Fatalf("want 1 in both, got Result.Logs=%d external=%d", len(res.Logs), len(captured))
	}
}

type loggerFunc func(LogEntry)

func (f loggerFunc) Log(e LogEntry) { f(e) }
