package validate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Result struct {
	Valid  bool   `json:"valid"`
	Errors []Item `json:"errors,omitempty"`
}

type Item struct {
	InstancePath string      `json:"instancePath"`
	SchemaPath   string      `json:"schemaPath"`
	Keyword      string      `json:"keyword"`
	Message      string      `json:"message"`
	Details      interface{} `json:"details,omitempty"`
}

func Validate(schema map[string]any, instance map[string]any) (Result, error) {
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return Result{}, fmt.Errorf("encode schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return Result{}, fmt.Errorf("add schema: %w", err)
	}

	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		return Result{}, fmt.Errorf("compile schema: %w", err)
	}

	if err := compiled.Validate(instance); err != nil {
		ve, ok := err.(*jsonschema.ValidationError)
		if !ok {
			return Result{}, fmt.Errorf("validate instance: %w", err)
		}
		items := flattenErrors(ve)
		rewriteGlobPresenceErrors(items, schema)
		return Result{Valid: false, Errors: items}, nil
	}

	return Result{Valid: true}, nil
}

func flattenErrors(err *jsonschema.ValidationError) []Item {
	var items []Item
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			items = append(items, Item{
				InstancePath: normalizePath(e.InstanceLocation),
				SchemaPath:   schemaPath(e),
				Keyword:      keywordFromLocation(e.KeywordLocation),
				Message:      e.Message,
			})
			return
		}
		for _, cause := range e.Causes {
			walk(cause)
		}
	}
	walk(err)

	sort.Slice(items, func(i, j int) bool {
		if items[i].InstancePath == items[j].InstancePath {
			return items[i].SchemaPath < items[j].SchemaPath
		}
		return items[i].InstancePath < items[j].InstancePath
	})

	return items
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	if path == "/" {
		return "/"
	}
	return path
}

func keywordFromLocation(location string) string {
	if location == "" {
		return ""
	}
	trimmed := strings.TrimSuffix(location, "/")
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	return parts[len(parts)-1]
}

func schemaPath(err *jsonschema.ValidationError) string {
	if err.AbsoluteKeywordLocation != "" {
		return err.AbsoluteKeywordLocation
	}
	return err.KeywordLocation
}

// rewriteGlobPresenceErrors detects errors caused by the not/propertyNames/not
// double-negation pattern (used to require at least one glob match) and rewrites
// their messages to be human-readable.
//
// It resolves each error's SchemaPath against the original schema. If the
// sub-schema at that path has the shape:
//
//	{"propertyNames": {"not": {"pattern": R}}}
//
// then the error is rewritten to: "no entries matching pattern <R>"
func rewriteGlobPresenceErrors(items []Item, schema map[string]any) {
	for i := range items {
		if items[i].Keyword != "not" {
			continue
		}

		// Extract the JSON pointer fragment from the schema path.
		// SchemaPath is like "file:///...schema.json#/allOf/0/not"
		fragment := extractFragment(items[i].SchemaPath)
		if fragment == "" {
			continue
		}

		// Resolve the pointer against the schema to get the value
		// under the "not" key — the inner sub-schema.
		subSchema := resolveJSONPointer(schema, fragment)
		if subSchema == nil {
			continue
		}

		// Check if it matches: {"propertyNames": {"not": {"pattern": R}}}
		pattern := extractGlobPresencePattern(subSchema)
		if pattern != "" {
			items[i].Message = fmt.Sprintf("no entries matching pattern %s", pattern)
			items[i].Keyword = "glob-presence"
		}
	}
}

// extractFragment returns the fragment portion of a URI (after #), or the
// whole string if there's no #.
func extractFragment(uri string) string {
	if idx := strings.Index(uri, "#"); idx >= 0 {
		return uri[idx+1:]
	}
	return uri
}

// resolveJSONPointer walks a schema map following a JSON Pointer (RFC 6901).
// Returns the value at the pointer, or nil if the path is invalid.
func resolveJSONPointer(schema map[string]any, pointer string) any {
	if pointer == "" || pointer == "/" {
		return schema
	}

	// Strip leading /
	pointer = strings.TrimPrefix(pointer, "/")
	parts := strings.Split(pointer, "/")

	var current any = schema
	for _, part := range parts {
		// Decode JSON Pointer escapes: ~1 → /, ~0 → ~
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")

		switch v := current.(type) {
		case map[string]any:
			next, ok := v[part]
			if !ok {
				return nil
			}
			current = next
		case []any:
			// Array index
			idx := 0
			for _, c := range part {
				if c < '0' || c > '9' {
					return nil
				}
				idx = idx*10 + int(c-'0')
			}
			if idx >= len(v) {
				return nil
			}
			current = v[idx]
		default:
			return nil
		}
	}
	return current
}

// extractGlobPresencePattern checks if a sub-schema has the shape:
//
//	{"propertyNames": {"not": {"pattern": R}}}
//
// and returns R if so, or "" otherwise.
func extractGlobPresencePattern(subSchema any) string {
	obj, ok := subSchema.(map[string]any)
	if !ok {
		return ""
	}
	propNames, ok := obj["propertyNames"]
	if !ok {
		return ""
	}
	propNamesObj, ok := propNames.(map[string]any)
	if !ok {
		return ""
	}
	notVal, ok := propNamesObj["not"]
	if !ok {
		return ""
	}
	notObj, ok := notVal.(map[string]any)
	if !ok {
		return ""
	}
	pattern, ok := notObj["pattern"].(string)
	if !ok {
		return ""
	}
	return pattern
}
