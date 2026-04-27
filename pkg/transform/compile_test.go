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

	"github.com/mimiro-io/datahub-cli/pkg/api"
)

func TestStripImportsExports(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single import line dropped",
			in:   "import { Entity } from \"datahub-tslib\";\nfunction f() { return 1; }",
			want: "function f() { return 1; }",
		},
		{
			name: "import \"side-effect\" dropped",
			in:   "import \"./bootstrap\";\nfunction f() {}",
			want: "function f() {}",
		},
		{
			name: "export keyword stripped from function",
			in:   "export function transform_entities(e) { return e; }",
			want: "function transform_entities(e) { return e; }",
		},
		{
			name: "export keyword stripped from const",
			in:   "export const X = 1;",
			want: "const X = 1;",
		},
		{
			name: "export default expr dropped entirely",
			in:   "export default 42;",
			want: "",
		},
		{
			name: "export default function unwrapped",
			in:   "export default function f() { return 1; }",
			want: "function f() { return 1; }",
		},
		{
			name: "export { x, y } dropped",
			in:   "function f() {}\nexport { f };",
			want: "function f() {}",
		},
		{
			name: "export * from \"x\" dropped",
			in:   "export * from \"./other\";\nfunction f() {}",
			want: "function f() {}",
		},
		{
			name: "indented `export` keyword inside object literal preserved",
			in: `const o = {
  export: 1,
};`,
			want: `const o = {
  export: 1,
};`,
		},
		{
			name: "no top-level module syntax — passthrough",
			in:   "function f() { return 1; }",
			want: "function f() { return 1; }",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripImportsExports(c.in)
			if got != c.want {
				t.Errorf("stripImportsExports() mismatch\nGOT:\n%s\nWANT:\n%s", got, c.want)
			}
		})
	}
}

func TestRun_StripsImportSyntaxBeforeRunning(t *testing.T) {
	src := `
		import { Entity, Query, NewEntity } from "datahub-tslib";

		export function transform_entities(entities: Entity[]) {
			return entities.map((e) => {
				const n = NewEntity();
				SetId(n, GetId(e));
				return n;
			});
		}
	`
	in := []*api.Entity{
		{ID: "x:1", Properties: map[string]any{}, References: map[string]any{}},
	}

	res, err := Run(context.Background(), src, in, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, l := range res.Logs {
		if l.Level == "error" {
			t.Fatalf("unexpected error log entry: %+v", l)
		}
	}
	if len(res.Entities) != 1 {
		t.Fatalf("want 1 entity, got %d", len(res.Entities))
	}
	if res.Entities[0].ID != "x:1" {
		t.Errorf("ID = %q, want x:1", res.Entities[0].ID)
	}
}

func TestRun_BareExportFunctionRuns(t *testing.T) {
	src := `
		export function transform_entities(entities: any[]) {
			return entities;
		}
	`
	res, err := Run(context.Background(), src, []*api.Entity{
		{ID: "a", Properties: map[string]any{}, References: map[string]any{}},
	}, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, l := range res.Logs {
		if l.Level == "error" {
			t.Fatalf("unexpected error: %s", l.Message)
		}
	}
	if len(res.Entities) != 1 || res.Entities[0].ID != "a" {
		t.Errorf("got %+v", res.Entities)
	}
}

func TestRun_NamespaceImportAliasResolvesAllMembers(t *testing.T) {
	src := `
		import * as dh from "datahub-tslib/datahub";

		export function transform_entities(entities: dh.Entity[]): dh.Entity[] {
			const ns = dh.AssertNamespacePrefix("http://example.org/");
			let out: dh.Entity[] = [];
			entities.forEach((e: dh.Entity) => {
				let n = dh.NewEntityFrom(e, false, false, false);
				dh.SetId(n, dh.GetId(e));
				dh.AddReference(n, ns, "ref", "ns0:thing");
				dh.SetProperty(n, ns, "prop", 42);
				out.push(n);
			});
			return out;
		}
	`
	in := []*api.Entity{
		{ID: "x:1", Properties: map[string]any{}, References: map[string]any{}},
		{ID: "x:2", Properties: map[string]any{}, References: map[string]any{}},
	}

	res, err := Run(context.Background(), src, in, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, l := range res.Logs {
		if l.Level == "error" {
			t.Fatalf("unexpected error log: %s", l.Message)
		}
	}
	if len(res.Entities) != 2 {
		t.Fatalf("want 2 entities, got %d", len(res.Entities))
	}
	if res.Entities[0].ID != "x:1" || res.Entities[1].ID != "x:2" {
		t.Errorf("ids = %s, %s; want x:1, x:2", res.Entities[0].ID, res.Entities[1].ID)
	}
	hasRef := false
	for _, v := range res.Entities[0].References {
		if v == "ns0:thing" {
			hasRef = true
		}
	}
	if !hasRef {
		t.Errorf("entity[0].References = %+v; want one entry with value ns0:thing", res.Entities[0].References)
	}
}

func TestDetectTslibNamespaceAliases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "datahub-tslib root",
			in:   `import * as dh from "datahub-tslib";`,
			want: []string{"dh"},
		},
		{
			name: "datahub-tslib/datahub",
			in:   `import * as dh from "datahub-tslib/datahub";`,
			want: []string{"dh"},
		},
		{
			name: "ignores non-tslib namespace imports",
			in:   `import * as fs from "node:fs";`,
			want: nil,
		},
		{
			name: "multiple aliases captured",
			in:   "import * as dh from \"datahub-tslib\";\nimport * as ns2 from \"datahub-tslib/datahub\";",
			want: []string{"dh", "ns2"},
		},
		{
			name: "named imports do not produce aliases",
			in:   `import { Query } from "datahub-tslib";`,
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectTslibNamespaceAliases(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("aliases[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestCompile_RealSyntaxErrorStillSurfaces(t *testing.T) {
	src := "function transform_entities(entities"
	res, err := Run(context.Background(), src, nil, Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Logs) == 0 || res.Logs[0].Level != "error" {
		t.Fatalf("want error log entry, got %+v", res.Logs)
	}
	if !strings.Contains(strings.ToLower(res.Logs[0].Message), "expected") {
		t.Errorf("error message %q doesn't look like an esbuild parse error", res.Logs[0].Message)
	}
}
