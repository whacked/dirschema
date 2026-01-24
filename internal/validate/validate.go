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
