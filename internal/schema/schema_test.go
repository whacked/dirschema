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

func TestValidateSchemaWithSizeRange(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"data.bin": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"size": map[string]any{
						"type":    "integer",
						"minimum": int64(0),
						"maximum": int64(1024),
					},
				},
				"required": []any{"size"},
			},
		},
		"required": []any{"data.bin"},
	}

	if err := ValidateSchema(schema); err != nil {
		t.Fatalf("ValidateSchema: %v", err)
	}
}

func TestValidateSchemaRejectsMissingType(t *testing.T) {
	// Missing "type": "object" at root
	schema := map[string]any{
		"properties": map[string]any{
			"file.txt": map[string]any{
				"oneOf": []any{
					map[string]any{"const": true},
					map[string]any{"type": "object"},
				},
			},
		},
		"required": []any{"file.txt"},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for missing type")
	}
}

func TestValidateSchemaRejectsMissingRequired(t *testing.T) {
	// Missing "required" array
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file.txt": map[string]any{
				"oneOf": []any{
					map[string]any{"const": true},
					map[string]any{"type": "object"},
				},
			},
		},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for missing required")
	}
}

func TestValidateSchemaRejectsInvalidFileDescriptor(t *testing.T) {
	// File descriptor with unknown property
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"unknown_prop": map[string]any{"const": "value"},
				},
				"required": []any{"unknown_prop"},
			},
		},
		"required": []any{"file.txt"},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for invalid file descriptor property")
	}
}

func TestValidateSchemaRejectsWrongConstType(t *testing.T) {
	// content.const should be string, not number
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{"const": 12345},
				},
				"required": []any{"content"},
			},
		},
		"required": []any{"file.txt"},
	}

	if err := ValidateSchema(schema); err == nil {
		t.Fatal("expected error for non-string content const")
	}
}
