package validate

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestValidateCollectsMultipleErrors(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"type": "string"},
			"b": map[string]any{"type": "number"},
		},
		"required": []string{"a", "b"},
	}
	instance := map[string]any{
		"a": 123,
	}

	res, err := Validate(schema, instance)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if res.Valid {
		t.Fatalf("expected invalid result")
	}
	if len(res.Errors) < 2 {
		t.Fatalf("expected multiple errors, got %d", len(res.Errors))
	}

	foundMissingB := false
	foundTypeA := false
	for _, e := range res.Errors {
		switch e.Keyword {
		case "required":
			if e.InstancePath == "" || e.InstancePath == "/" {
				foundMissingB = true
			}
		case "type":
			if e.InstancePath == "/a" {
				foundTypeA = true
			}
		}
	}

	if !foundMissingB {
		t.Fatalf("missing required error for b")
	}
	if !foundTypeA {
		t.Fatalf("missing type error for a")
	}
}

func TestValidateSchemaError(t *testing.T) {
	schema := map[string]any{
		"type": 123,
	}
	_, err := Validate(schema, map[string]any{})
	if err == nil {
		t.Fatalf("expected schema error")
	}
}

func TestValidateGlobPresenceConstraint(t *testing.T) {
	// Schema that requires at least one *.go file via the not-not trick:
	// "at least one property name must match the pattern"
	existenceSchema := map[string]any{
		"oneOf": []any{
			map[string]any{"const": true},
			map[string]any{"type": "object"},
		},
	}

	schema := map[string]any{
		"type": "object",
		"patternProperties": map[string]any{
			"^.*\\.go$": existenceSchema,
		},
		"required": []any{},
		"allOf": []any{
			map[string]any{
				"not": map[string]any{
					"propertyNames": map[string]any{
						"not": map[string]any{
							"pattern": "^.*\\.go$",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		instance map[string]any
		valid    bool
	}{
		{
			name:     "matching file exists",
			instance: map[string]any{"foo.go": true},
			valid:    true,
		},
		{
			name:     "no matching file",
			instance: map[string]any{"main.c": true},
			valid:    false,
		},
		{
			name:     "empty directory",
			instance: map[string]any{},
			valid:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Validate(schema, tc.instance)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if res.Valid != tc.valid {
				t.Fatalf("expected valid=%v, got valid=%v; errors=%v", tc.valid, res.Valid, res.Errors)
			}
		})
	}
}

func TestValidateMixedLiteralAndGlobPresence(t *testing.T) {
	existenceSchema := map[string]any{
		"oneOf": []any{
			map[string]any{"const": true},
			map[string]any{"type": "object"},
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"main.c": existenceSchema,
		},
		"patternProperties": map[string]any{
			"^.*\\.go$": existenceSchema,
		},
		"required": []any{"main.c"},
		"allOf": []any{
			map[string]any{
				"not": map[string]any{
					"propertyNames": map[string]any{
						"not": map[string]any{
							"pattern": "^.*\\.go$",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		instance map[string]any
		valid    bool
	}{
		{
			name:     "both literal and glob satisfied",
			instance: map[string]any{"main.c": true, "foo.go": true},
			valid:    true,
		},
		{
			name:     "literal present but glob unsatisfied",
			instance: map[string]any{"main.c": true},
			valid:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Validate(schema, tc.instance)
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if res.Valid != tc.valid {
				t.Fatalf("expected valid=%v, got valid=%v; errors=%v", tc.valid, res.Valid, res.Errors)
			}
		})
	}
}

func TestValidateJSONRoundTrip(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a": map[string]any{"const": true},
		},
		"required": []string{"a"},
	}
	instance := map[string]any{"a": true}

	res, err := Validate(schema, instance)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Result
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !reflect.DeepEqual(res, decoded) {
		t.Fatalf("round trip mismatch")
	}
}
