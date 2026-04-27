// Copyright 2026 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package transform

import (
	"errors"
	"fmt"
	"strings"

	esbuild "github.com/evanw/esbuild/pkg/api"
)

// goja can't parse import/export, so we strip module syntax after esbuild
// emits ES2016. Host functions are injected as globals; datahub-tslib is
// ambient-`declare` only, so no module wiring is needed at runtime.
func compile(source string) (string, error) {
	result := esbuild.Transform(source, esbuild.TransformOptions{
		Loader: esbuild.LoaderTS,
		Format: esbuild.FormatDefault,
		Target: esbuild.ES2016,
	})

	if len(result.Errors) > 0 {
		var msgs []string
		for _, e := range result.Errors {
			msgs = append(msgs, formatDiagnostic(e))
		}
		return "", errors.New(strings.Join(msgs, "\n"))
	}

	raw := string(result.Code)
	aliases := detectTslibNamespaceAliases(raw)
	stripped := stripImportsExports(raw)
	if len(aliases) > 0 {
		return buildNamespaceShim(aliases) + "\n" + stripped, nil
	}
	return stripped, nil
}

// Named imports already resolve to globals once stripped; only namespace
// aliases (`import * as dh from "datahub-tslib"`) need a shim.
func detectTslibNamespaceAliases(code string) []string {
	var out []string
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "import") {
			continue
		}
		if !strings.Contains(trimmed, "datahub-tslib") {
			continue
		}
		idx := strings.Index(trimmed, "* as ")
		if idx < 0 {
			continue
		}
		rest := trimmed[idx+len("* as "):]
		end := 0
		for end < len(rest) && isIdentChar(rest[end]) {
			end++
		}
		if end == 0 {
			continue
		}
		out = append(out, rest[:end])
	}
	return out
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' || b == '$'
}

// Keep in lockstep with runtime.go::hookEngine and js_helpers.go::helperJavascriptFunctions.
var nsShimMembers = []string{
	"Query", "FindById", "GetNamespacePrefix", "AssertNamespacePrefix",
	"Log", "NewEntity", "IsValidEntity", "ToString", "AsEntity", "UUID",
	"SetProperty", "GetProperty", "GetReference", "AddReference",
	"GetId", "SetId", "GetDeleted", "SetDeleted", "PrefixField",
	"RenameProperty", "RemoveProperty", "NewEntityFrom",
}

func buildNamespaceShim(aliases []string) string {
	var b strings.Builder
	for _, alias := range aliases {
		b.WriteString("var ")
		b.WriteString(alias)
		b.WriteString(" = {")
		for i, m := range nsShimMembers {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(m)
			b.WriteByte(':')
			b.WriteString(m)
		}
		b.WriteString("};\n")
	}
	return b.String()
}

// Line-based strip of top-level ES module syntax. Relies on esbuild's
// Transform canonicalising multi-line imports to a single line.
func stripImportsExports(code string) string {
	var b strings.Builder
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "import{") ||
			strings.HasPrefix(trimmed, "import\"") ||
			strings.HasPrefix(trimmed, "import'") {
			continue
		}
		if strings.HasPrefix(trimmed, "export default ") {
			rest := strings.TrimPrefix(trimmed, "export default ")
			if strings.HasPrefix(rest, "function ") ||
				strings.HasPrefix(rest, "function(") ||
				strings.HasPrefix(rest, "class ") ||
				strings.HasPrefix(rest, "class{") ||
				strings.HasPrefix(rest, "async ") {
				b.WriteString(rest)
				b.WriteByte('\n')
			}
			continue
		}
		if strings.HasPrefix(trimmed, "export ") {
			rest := strings.TrimPrefix(trimmed, "export ")
			if strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "*") {
				continue
			}
			b.WriteString(rest)
			b.WriteByte('\n')
			continue
		}
		if trimmed == "export{}" || trimmed == "export {}" || trimmed == "export {};" || trimmed == "export{};" {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatDiagnostic(m esbuild.Message) string {
	if m.Location != nil {
		return fmt.Sprintf("[%d:%d] %s", m.Location.Line, m.Location.Column, m.Text)
	}
	return m.Text
}
