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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// stubHub serves the minimal /namespaces and /query endpoints the transform
// runtime calls. /query inspects the request body to dispatch between the
// graph-style Query (startingEntities + predicate) and the single-entity
// FindById path.
func stubHub(t *testing.T, hits map[string][]queryResultPart, single map[string]*api.Entity) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/namespaces", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{})
	})
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if id, ok := body["entityId"].(string); ok {
			ent, found := single[id]
			if !found {
				http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"id": "@context"}, ent})
			return
		}
		startersRaw, _ := body["startingEntities"].([]any)
		key := ""
		if len(startersRaw) > 0 {
			key, _ = startersRaw[0].(string)
		}
		parts := hits[key]
		tuples := make([]any, len(parts))
		for i, p := range parts {
			tuples[i] = []any{p.Uri, p.PredicateUri, p.Entity}
		}
		raw, _ := json.Marshal(tuples)
		_, _ = w.Write([]byte(`[{},` + string(raw) + `]`))
	})
	return httptest.NewServer(mux)
}

type recorderStub struct {
	mu      sync.Mutex
	records []QueryRecord
}

func (r *recorderStub) RecordQuery(rec QueryRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, rec)
}

type resolverFunc func(*api.Entity) *api.Entity

func (f resolverFunc) ResolveOverlay(e *api.Entity) *api.Entity { return f(e) }

func TestRun_QueryRecorderObservesCalls(t *testing.T) {
	hits := map[string][]queryResultPart{
		"ns0:src": {{
			Uri:          "ns0:src",
			PredicateUri: "ns0:hasFoo",
			Entity: &api.Entity{
				ID:         "ns0:target",
				Properties: map[string]any{"ns0:flag": false},
				References: map[string]any{},
			},
		}},
	}
	srv := stubHub(t, hits, nil)
	defer srv.Close()

	rec := &recorderStub{}
	src := `
		function transform_entities(entities: any[]) {
			let hits = Query(["ns0:src"], "ns0:hasFoo", false, []);
			return entities;
		}
	`
	_, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:x", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, QueryRecorder: rec})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.records) != 1 {
		t.Fatalf("want 1 recorded call, got %d", len(rec.records))
	}
	got := rec.records[0]
	if got.Kind != "query" || got.Predicate != "ns0:hasFoo" {
		t.Errorf("record kind/predicate = %s/%s, want query/ns0:hasFoo", got.Kind, got.Predicate)
	}
	if len(got.Entities) != 1 || got.Entities[0].ID != "ns0:target" {
		t.Errorf("record entities = %+v, want [ns0:target]", got.Entities)
	}
}

func TestRun_OverlayResolverMutatesQueryEntity(t *testing.T) {
	hits := map[string][]queryResultPart{
		"ns0:src": {{
			Uri:          "ns0:src",
			PredicateUri: "ns0:hasFoo",
			Entity: &api.Entity{
				ID:         "ns0:target",
				Properties: map[string]any{"ns0:flag": false},
				References: map[string]any{},
			},
		}},
	}
	srv := stubHub(t, hits, nil)
	defer srv.Close()

	resolver := resolverFunc(func(e *api.Entity) *api.Entity {
		if e == nil || e.ID != "ns0:target" {
			return e
		}
		clone := *e
		clone.Properties = make(map[string]any, len(e.Properties))
		for k, v := range e.Properties {
			clone.Properties[k] = v
		}
		clone.Properties["ns0:flag"] = true
		return &clone
	})

	src := `
		function transform_entities(entities: any[]) {
			let hits = Query(["ns0:src"], "ns0:hasFoo", false, []);
			let target = hits[0][2];
			let out = NewEntity();
			SetId(out, GetId(entities[0]));
			SetProperty(out, "ns0", "flagged", GetProperty(target, "ns0", "flag", false));
			return [out];
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:x", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, OverlayResolver: resolver})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1 entity, got %d (logs=%+v)", len(res.Entities), res.Logs)
	}
	got := res.Entities[0]
	if v, ok := got.Properties["ns0:flagged"]; !ok || v != true {
		t.Errorf("overlay didn't reach transform: ns0:flagged = %v", v)
	}
}

type syntheticStub struct {
	entities map[string]syntheticEntry
}

type syntheticEntry struct {
	dataset string
	entity  *api.Entity
}

func (s *syntheticStub) Lookup(id string, datasets []string) *api.Entity {
	e, ok := s.entities[id]
	if !ok {
		return nil
	}
	if !syntheticDatasetAllowed(e.dataset, datasets) {
		return nil
	}
	clone := *e.entity
	return &clone
}

func (s *syntheticStub) All(datasets []string) []*api.Entity {
	var out []*api.Entity
	for _, e := range s.entities {
		if !syntheticDatasetAllowed(e.dataset, datasets) {
			continue
		}
		clone := *e.entity
		out = append(out, &clone)
	}
	return out
}

func syntheticDatasetAllowed(have string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, d := range filter {
		if d == have {
			return true
		}
	}
	return false
}

func TestRun_FindByIdReturnsSyntheticWhenHubEmpty(t *testing.T) {
	srv := stubHub(t, nil, nil)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns0:Mock-1": {
			dataset: "ns0.MockDb",
			entity: &api.Entity{
				ID:         "ns0:Mock-1",
				Properties: map[string]any{"ns0:flag": true},
				References: map[string]any{},
			},
		},
	}}
	rec := &recorderStub{}
	src := `
		function transform_entities(entities: any[]) {
			let mock = FindById("ns0:Mock-1", []);
			if (mock === null) { Log("missing", "error"); return entities; }
			let out = NewEntity();
			SetId(out, "ns0:result");
			SetProperty(out, "ns0", "flag", GetProperty(mock, "ns0", "flag", false));
			return [out];
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, SyntheticProvider: prov, QueryRecorder: rec})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1 result, got %d (logs=%+v)", len(res.Entities), res.Logs)
	}
	if v := res.Entities[0].Properties["ns0:flag"]; v != true {
		t.Errorf("flag = %v, want true (synthetic should be visible to transform)", v)
	}
	if len(rec.records) != 1 || rec.records[0].Kind != "findById" {
		t.Errorf("recorder: want one findById entry, got %+v", rec.records)
	}
}

func TestRun_FindByIdRespectsDatasetFilterOnSynthetic(t *testing.T) {
	srv := stubHub(t, nil, nil)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns0:Mock-1": {
			dataset: "ns0.MockDb",
			entity: &api.Entity{
				ID: "ns0:Mock-1", Properties: map[string]any{}, References: map[string]any{},
			},
		},
	}}
	src := `
		function transform_entities(entities: any[]) {
			let m = FindById("ns0:Mock-1", ["ns0.OtherDb"]);
			Log(m === null ? "miss" : "hit", "info");
			return entities;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, SyntheticProvider: prov})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Filter doesn't include the synthetic's dataset → synthetic doesn't fire,
	// hub stub returns 404 → FindById logs an error and returns null.
	found := false
	for _, l := range res.Logs {
		if l.Level == "info" && l.Message == "miss" {
			found = true
		}
	}
	if !found {
		t.Errorf("want a 'miss' info log, got %+v", res.Logs)
	}
}

func TestRun_QueryForwardEmitsSyntheticAsStartingPoint(t *testing.T) {
	// Hub stub returns no tuples for ns0:Mock-1 — the only path that surfaces
	// a target entity is through the synthetic provider.
	hits := map[string][]queryResultPart{}
	single := map[string]*api.Entity{
		"ns0:Herd-A": {
			ID:         "ns0:Herd-A",
			Properties: map[string]any{"ns0:name": "alpha"},
			References: map[string]any{},
		},
	}
	srv := stubHub(t, hits, single)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns0:Mock-1": {
			dataset: "ns0.MockDb",
			entity: &api.Entity{
				ID:         "ns0:Mock-1",
				Properties: map[string]any{},
				References: map[string]any{"ns0:hasHerd": "ns0:Herd-A"},
			},
		},
	}}
	src := `
		function transform_entities(entities: any[]) {
			let hits = Query(["ns0:Mock-1"], "ns0:hasHerd", false, []);
			let out: any[] = [];
			for (let h of hits) {
				let target = h[2];
				if (target !== null && target !== undefined) {
					let e = NewEntity();
					SetId(e, "ns0:row");
					SetProperty(e, "ns0", "subject", h[0]);
					SetProperty(e, "ns0", "name", GetProperty(target, "ns0", "name", ""));
					out.push(e);
				}
			}
			return out;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, SyntheticProvider: prov})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1 result tuple, got %d (logs=%+v)", len(res.Entities), res.Logs)
	}
	got := res.Entities[0]
	if v := got.Properties["ns0:subject"]; v != "ns0:Mock-1" {
		t.Errorf("subject = %v, want ns0:Mock-1", v)
	}
	if v := got.Properties["ns0:name"]; v != "alpha" {
		t.Errorf("name = %v, want alpha (target resolved through hub)", v)
	}
}

func TestRun_QueryInverseEmitsSyntheticAsTarget(t *testing.T) {
	srv := stubHub(t, nil, nil)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns0:Mock-1": {
			dataset: "ns0.MockDb",
			entity: &api.Entity{
				ID:         "ns0:Mock-1",
				Properties: map[string]any{"ns0:name": "beta"},
				References: map[string]any{"ns0:knows": "ns0:Animal-X"},
			},
		},
	}}
	src := `
		function transform_entities(entities: any[]) {
			let hits = Query(["ns0:Animal-X"], "ns0:knows", true, []);
			let out: any[] = [];
			for (let h of hits) {
				let target = h[2];
				if (target !== null && target !== undefined) {
					let e = NewEntity();
					SetId(e, "ns0:row");
					SetProperty(e, "ns0", "found", GetProperty(target, "ns0", "name", ""));
					out.push(e);
				}
			}
			return out;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, SyntheticProvider: prov})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1 result, got %d (logs=%+v)", len(res.Entities), res.Logs)
	}
	if v := res.Entities[0].Properties["ns0:found"]; v != "beta" {
		t.Errorf("found = %v, want beta", v)
	}
}

func TestRun_QueryDatasetFilterDropsMismatchedSynthetics(t *testing.T) {
	srv := stubHub(t, nil, nil)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns0:Mock-1": {
			dataset: "ns0.MockDb",
			entity: &api.Entity{
				ID:         "ns0:Mock-1",
				Properties: map[string]any{},
				References: map[string]any{"ns0:rel": "ns0:Anything"},
			},
		},
	}}
	src := `
		function transform_entities(entities: any[]) {
			let hits = Query(["ns0:Mock-1"], "ns0:rel", false, ["ns0.OtherDb"]);
			Log("count " + hits.length, "info");
			return entities;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, SyntheticProvider: prov})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "count 0"
	found := false
	for _, l := range res.Logs {
		if l.Level == "info" && l.Message == want {
			found = true
		}
	}
	if !found {
		t.Errorf("want a %q info log (filter should drop synthetic), got %+v", want, res.Logs)
	}
}

// stubHubQuery500 mirrors stubHub but the /query endpoint returns 500 for
// non-entityId calls — this reproduces the "could not load predicate id" path
// the hub takes when an operator's transform names a predicate the hub
// doesn't have registered yet.
func stubHubQuery500(t *testing.T, single map[string]*api.Entity) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/namespaces", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{})
	})
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if id, ok := body["entityId"].(string); ok {
			ent, found := single[id]
			if !found {
				http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode([]any{map[string]any{"id": "@context"}, ent})
			return
		}
		http.Error(w, `{"message":"could not load predicate id"}`, http.StatusInternalServerError)
	})
	return httptest.NewServer(mux)
}

// TestRun_QueryHubErrorWithInputSourceFindsSyntheticTarget reproduces the
// reported bug: a transform runs with a real-looking input entity whose
// refs[predicate] points at a synthetic entity, but the hub doesn't recognise
// the predicate and returns 500. The cli should downgrade the failure to a
// warning, walk the input's refs, and emit the synthetic as the tuple target.
func TestRun_QueryHubErrorWithInputSourceFindsSyntheticTarget(t *testing.T) {
	srv := stubHubQuery500(t, nil)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns823:1000": {
			dataset: "raavaretorget.Anmerkningene",
			entity: &api.Entity{
				ID: "ns823:1000",
				Properties: map[string]any{
					"ns823:id":   900,
					"ns823:navn": "Hoyt baktfoo",
				},
				References: map[string]any{
					"ns2:type":    "ns361:Anmerkning",
					"ns823:pred":  "ns825:2680067",
					"ns823:sameAs": "ns368:900",
				},
			},
		},
	}}

	input := &api.Entity{
		ID:       "ns825:2680067",
		Recorded: 1778688855839914800,
		Properties: map[string]any{
			"ns825:id":     2680067,
			"ns825:mengde": 3519,
		},
		References: map[string]any{
			"ns2:type":            "ns361:Leveranse",
			"ns825:produsent_id":  "ns824:26003",
			"ns825:foobar":        "ns823:1000",
		},
	}
	src := `
		function transform_entities(entities: any[]) {
			let out: any[] = [];
			for (let e of entities) {
				let hits = Query([GetId(e)], "ns825:foobar", false, ["raavaretorget.Leveranse", "raavaretorget.Anmerkningene"]);
				for (let h of hits) {
					let target = h[2];
					if (target !== null && target !== undefined) {
						let row = NewEntity();
						SetId(row, "ns:row");
						SetProperty(row, "ns823", "navn", GetProperty(target, "ns823", "navn", ""));
						out.push(row);
					}
				}
			}
			return out;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{input}, Options{
		HubURL:            srv.URL,
		SyntheticProvider: prov,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1 result tuple, got %d (logs=%+v)", len(res.Entities), res.Logs)
	}
	if v := res.Entities[0].Properties["ns823:navn"]; v != "Hoyt baktfoo" {
		t.Errorf("navn = %v; want 'Hoyt baktfoo' (the synthetic should be visible)", v)
	}
	// The hub failure must NOT surface as warn/error when augmentation
	// rescued the run — the operator authored the synthetic on purpose.
	for _, l := range res.Logs {
		if l.Level == "warn" || l.Level == "error" {
			t.Errorf("expected silent rescue, got %s log: %q", l.Level, l.Message)
		}
	}
}

// Sibling case: hub fails AND no synthetic/input matches. The warn surfaces
// so the operator sees what happened instead of getting silent zero results.
func TestRun_QueryHubErrorWithoutAugmentationSurfacesWarn(t *testing.T) {
	srv := stubHubQuery500(t, nil)
	defer srv.Close()

	src := `
		function transform_entities(entities: any[]) {
			Query(["ns:nothing"], "ns:missing", false, []);
			return entities;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	sawWarn := false
	for _, l := range res.Logs {
		if l.Level == "warn" && strings.Contains(l.Message, "could not load predicate") {
			sawWarn = true
		}
	}
	if !sawWarn {
		t.Errorf("hub failure with no augmentation should surface a warn; got %+v", res.Logs)
	}
}

func TestRun_OverlayAppliesOnTopOfSynthetic(t *testing.T) {
	srv := stubHub(t, nil, nil)
	defer srv.Close()

	prov := &syntheticStub{entities: map[string]syntheticEntry{
		"ns0:Mock-1": {
			dataset: "ns0.MockDb",
			entity: &api.Entity{
				ID:         "ns0:Mock-1",
				Properties: map[string]any{"ns0:flag": false},
				References: map[string]any{},
			},
		},
	}}
	resolver := resolverFunc(func(e *api.Entity) *api.Entity {
		if e == nil || e.ID != "ns0:Mock-1" {
			return e
		}
		clone := *e
		clone.Properties = map[string]any{"ns0:flag": true}
		return &clone
	})
	src := `
		function transform_entities(entities: any[]) {
			let m = FindById("ns0:Mock-1", []);
			let out = NewEntity();
			SetId(out, "ns0:result");
			SetProperty(out, "ns0", "flag", GetProperty(m, "ns0", "flag", false));
			return [out];
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:src", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, SyntheticProvider: prov, OverlayResolver: resolver})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1, got %d (logs=%+v)", len(res.Entities), res.Logs)
	}
	if v := res.Entities[0].Properties["ns0:flag"]; v != true {
		t.Errorf("flag = %v; want true (overlay should layer onto synthetic)", v)
	}
}

func TestRun_FindByIdHooksFireToo(t *testing.T) {
	single := map[string]*api.Entity{
		"ns0:lookup": {
			ID:         "ns0:lookup",
			Properties: map[string]any{"ns0:n": 1},
			References: map[string]any{},
		},
	}
	srv := stubHub(t, nil, single)
	defer srv.Close()

	rec := &recorderStub{}
	resolver := resolverFunc(func(e *api.Entity) *api.Entity {
		if e == nil {
			return nil
		}
		clone := *e
		clone.Properties = map[string]any{"ns0:n": int64(42)}
		return &clone
	})
	src := `
		function transform_entities(entities: any[]) {
			let found = FindById("ns0:lookup", []);
			return entities;
		}
	`
	_, err := Run(context.Background(), src, []*api.Entity{
		{ID: "ns0:x", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{HubURL: srv.URL, QueryRecorder: rec, OverlayResolver: resolver})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.records) != 1 || rec.records[0].Kind != "findById" {
		t.Fatalf("want 1 findById record, got %+v", rec.records)
	}
	got := rec.records[0]
	if len(got.Entities) != 1 || got.Entities[0].Properties["ns0:n"] != int64(42) {
		t.Errorf("overlay didn't reach recorder; entities=%+v", got.Entities)
	}
}
