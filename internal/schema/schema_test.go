package schema

import (
	"testing"
)

func TestMetaSchemaLoads(t *testing.T) {
	schema, err := MetaSchema()
	if err != nil {
		t.Fatalf("MetaSchema: %v", err)
	}
	if schema == nil {
		t.Fatal("MetaSchema returned nil")
	}
}

func TestValidateSchemaValid(t *testing.T) {
	// A valid expanded schema with existence-only file
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"README.md": map[string]any{
				"oneOf": []any{
					map[string]any{"const": true},
					map[string]any{"type": "object"},
				},
			},
		},
		"required": []any{"README.md"},
	}

	if err := ValidateSchema(schema); err != nil {
		t.Fatalf("ValidateSchema: %v", err)
	}
}

func TestValidateSchemaWithContent(t *testing.T) {
	// A valid expanded schema with content constraint
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"config.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{"const": "hello"},
				},
				"required": []any{"content"},
			},
		},
		"required": []any{"config.txt"},
	}

	if err := ValidateSchema(schema); err != nil {
		t.Fatalf("ValidateSchema: %v", err)
	}
}

func TestValidateSchemaWithDirectory(t *testing.T) {
	// A valid expanded schema with nested directory
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"src/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"main.go": map[string]any{
						"oneOf": []any{
							map[string]any{"const": true},
							map[string]any{"type": "object"},
						},
					},
				},
				"required": []any{"main.go"},
			},
		},
		"required": []any{"src/"},
	}

	if err := ValidateSchema(schema); err != nil {
		t.Fatalf("ValidateSchema: %v", err)
	}
}
