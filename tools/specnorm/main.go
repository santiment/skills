// Command specnorm normalizes an OpenAPI document so that oapi-codegen
// (built on kin-openapi, which targets OpenAPI 3.0) can consume it cleanly.
//
// It performs three transformations:
//
//  1. Downgrades the declared version from 3.1.x to 3.0.3.
//  2. Rewrites JSON-Schema-style "type" arrays into 3.0 form:
//     - ["X","null"]            -> {"type":"X","nullable":true}
//     - ["X","Y", ...]          -> drop "type" (free-form), keep nullable if "null" present
//     - ["null"]                -> drop "type", nullable:true
//  3. Injects a deterministic operationId for every operation that lacks one,
//     derived from the HTTP method and path, so generated Go method names are stable.
//
// Usage:
//
//	specnorm <input.json> <output.json>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: specnorm <input.json> <output.json>")
		os.Exit(2)
	}
	in, out := os.Args[1], os.Args[2]

	raw, err := os.ReadFile(in)
	if err != nil {
		fatal("read %s: %v", in, err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		fatal("parse %s: %v", in, err)
	}

	if v, ok := doc["openapi"].(string); ok && strings.HasPrefix(v, "3.1") {
		doc["openapi"] = "3.0.3"
	}

	injectOperationIDs(doc)
	normalizeParameters(doc)
	normalizeTypes(doc)

	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fatal("encode: %v", err)
	}
	if err := os.WriteFile(out, encoded, 0o644); err != nil {
		fatal("write %s: %v", out, err)
	}
	fmt.Fprintf(os.Stderr, "specnorm: %s -> %s\n", in, out)
}

// normalizeTypes walks the whole document and rewrites schema nodes into an
// OpenAPI 3.0 compatible shape: "type" arrays, scalar "null" types, and the
// 3.1 nullable-union idiom (oneOf/anyOf containing a {"type":"null"} member).
func normalizeTypes(node any) {
	switch n := node.(type) {
	case map[string]any:
		for _, key := range []string{"oneOf", "anyOf"} {
			if members, ok := n[key].([]any); ok {
				collapseNullUnion(n, key, members)
			}
		}
		if t, ok := n["type"].([]any); ok {
			rewriteTypeArray(n, t)
		}
		if s, ok := n["type"].(string); ok && s == "null" {
			// A bare null type cannot be expressed in 3.0; make it free-form+nullable.
			delete(n, "type")
			n["nullable"] = true
		}
		for _, v := range n {
			normalizeTypes(v)
		}
	case []any:
		for _, v := range n {
			normalizeTypes(v)
		}
	}
}

// collapseNullUnion rewrites the 3.1 nullable-union idiom, e.g.
//
//	{"oneOf": [{"$ref": "#/.../Signal"}, {"type": "null"}]}
//
// into a 3.0-friendly schema by dropping the null member and marking the node
// nullable. If a single member remains it is spread into the node (so a nullable
// $ref becomes {"$ref": ..., "nullable": true}); otherwise the remaining members
// are kept under the original key.
func collapseNullUnion(node map[string]any, key string, members []any) {
	kept := make([]any, 0, len(members))
	hadNull := false
	for _, m := range members {
		if mm, ok := m.(map[string]any); ok && isNullOnly(mm) {
			hadNull = true
			continue
		}
		kept = append(kept, m)
	}
	if !hadNull {
		return
	}
	node["nullable"] = true
	if len(kept) == 1 {
		delete(node, key)
		if mm, ok := kept[0].(map[string]any); ok {
			for k, v := range mm {
				node[k] = v
			}
		}
		return
	}
	node[key] = kept
}

// isNullOnly reports whether a schema is exactly {"type": "null"} (scalar or
// single-element array form) with no other constraints.
func isNullOnly(m map[string]any) bool {
	if len(m) != 1 {
		return false
	}
	switch t := m["type"].(type) {
	case string:
		return t == "null"
	case []any:
		if len(t) == 1 {
			s, _ := t[0].(string)
			return s == "null"
		}
	}
	return false
}

func rewriteTypeArray(node map[string]any, types []any) {
	var nonNull []string
	hasNull := false
	for _, t := range types {
		s, _ := t.(string)
		if s == "null" {
			hasNull = true
			continue
		}
		if s != "" {
			nonNull = append(nonNull, s)
		}
	}
	if hasNull {
		node["nullable"] = true
	}
	switch len(nonNull) {
	case 1:
		node["type"] = nonNull[0]
	default:
		// Zero or multiple concrete types cannot be expressed by a single 3.0
		// "type"; drop it so the schema becomes free-form (generated as any).
		delete(node, "type")
	}
}

// normalizeParameters ensures every parameter has a typed schema. Some specs
// declare query parameters with a schema that carries only an example and no
// "type"; oapi-codegen then generates an untyped interface{} field that it
// serializes unconditionally, panicking when the value is nil. Giving the
// schema a concrete type makes the generated field an optional pointer that is
// safely skipped when unset.
func normalizeParameters(node any) {
	switch n := node.(type) {
	case map[string]any:
		if isParameter(n) {
			fixParameterSchema(n)
		}
		for _, v := range n {
			normalizeParameters(v)
		}
	case []any:
		for _, v := range n {
			normalizeParameters(v)
		}
	}
}

func isParameter(n map[string]any) bool {
	_, hasIn := n["in"].(string)
	_, hasName := n["name"].(string)
	return hasIn && hasName
}

func fixParameterSchema(param map[string]any) {
	schema, ok := param["schema"].(map[string]any)
	if !ok {
		param["schema"] = map[string]any{"type": "string"}
		return
	}
	if _, hasType := schema["type"]; hasType {
		return
	}
	for _, composite := range []string{"$ref", "allOf", "oneOf", "anyOf"} {
		if _, ok := schema[composite]; ok {
			return // a composed/ref schema already conveys a type
		}
	}
	schema["type"] = inferType(schema["example"])
}

func inferType(example any) string {
	switch v := example.(type) {
	case bool:
		return "boolean"
	case float64:
		if v == float64(int64(v)) {
			return "integer"
		}
		return "number"
	default:
		return "string"
	}
}

// injectOperationIDs assigns a deterministic operationId to every operation
// under paths that does not already declare one.
func injectOperationIDs(doc map[string]any) {
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return
	}
	methods := map[string]bool{
		"get": true, "put": true, "post": true, "delete": true,
		"patch": true, "options": true, "head": true, "trace": true,
	}
	// Sort paths for deterministic output ordering of generated code.
	keys := make([]string, 0, len(paths))
	for p := range paths {
		keys = append(keys, p)
	}
	sort.Strings(keys)

	for _, p := range keys {
		item, ok := paths[p].(map[string]any)
		if !ok {
			continue
		}
		for method, raw := range item {
			if !methods[strings.ToLower(method)] {
				continue
			}
			op, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if id, _ := op["operationId"].(string); id == "" {
				op["operationId"] = operationID(method, p)
			}
		}
	}
}

// operationID builds a CamelCase identifier from an HTTP method and path,
// e.g. ("get", "/v2/predictions/{id}") -> "getV2PredictionsId".
func operationID(method, path string) string {
	var b strings.Builder
	b.WriteString(strings.ToLower(method))
	for _, seg := range strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '{' || r == '}' || r == '-' || r == '_' || r == '.'
	}) {
		b.WriteString(strings.ToUpper(seg[:1]))
		if len(seg) > 1 {
			b.WriteString(seg[1:])
		}
	}
	return b.String()
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "specnorm: "+format+"\n", args...)
	os.Exit(1)
}
