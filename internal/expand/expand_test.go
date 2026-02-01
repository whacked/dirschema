package expand

import (
	"reflect"
	"testing"

	"dirschema/internal/schema"
)

func TestExpandSimpleDSL(t *testing.T) {
	dsl := map[string]any{
		"src/": map[string]any{
			"main.go": true,
		},
		"README.md": true,
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	// Existence-only files expand to oneOf: [const true, type object]
	existenceSchema := map[string]any{
		"oneOf": []any{
			map[string]any{"const": true},
			map[string]any{"type": "object"},
		},
	}

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"src/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"main.go": existenceSchema,
				},
				"required": []any{"main.go"},
			},
			"README.md": existenceSchema,
		},
		"required": []any{"README.md", "src/"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestExpandSymlinkDSL(t *testing.T) {
	dsl := map[string]any{
		"link.txt": map[string]any{"symlink": "target.txt"},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"link.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symlink": map[string]any{"const": "target.txt"},
				},
				"required": []any{"symlink"},
			},
		},
		"required": []any{"link.txt"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema mismatch: got %#v want %#v", got, want)
	}
}

func TestExpandListDSL(t *testing.T) {
	dsl := map[string]any{
		"root/": []any{
			"file1",
			map[string]any{"link.txt": map[string]any{"symlink": "file1"}},
		},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	root := got["properties"].(map[string]any)["root/"].(map[string]any)
	props := root["properties"].(map[string]any)

	if _, ok := props["file1"]; !ok {
		t.Fatalf("expected file1 in properties")
	}
	link := props["link.txt"].(map[string]any)
	linkProps := link["properties"].(map[string]any)
	if linkProps["symlink"].(map[string]any)["const"] != "file1" {
		t.Fatalf("expected symlink const target")
	}
}

func TestExpandListDSLRejectsDuplicates(t *testing.T) {
	dsl := map[string]any{
		"root/": []any{
			"File.txt",
			"file.txt",
		},
	}

	_, err := ExpandDSL(dsl)
	if err == nil {
		t.Fatalf("expected duplicate error")
	}
}

func TestExpandContentDSL(t *testing.T) {
	dsl := map[string]any{
		"config.txt": map[string]any{"content": "key=value"},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"config.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{"const": "key=value"},
				},
				"required": []any{"content"},
			},
		},
		"required": []any{"config.txt"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestExpandContentListDSL(t *testing.T) {
	dsl := map[string]any{
		"dir/": []any{
			map[string]any{"readme.md": map[string]any{"content": "# Hello"}},
		},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	dir := got["properties"].(map[string]any)["dir/"].(map[string]any)
	props := dir["properties"].(map[string]any)
	readme := props["readme.md"].(map[string]any)
	readmeProps := readme["properties"].(map[string]any)

	if readmeProps["content"].(map[string]any)["const"] != "# Hello" {
		t.Fatalf("expected content const")
	}
}

func TestExpandRejectsSymlinkAndContent(t *testing.T) {
	dsl := map[string]any{
		"file.txt": map[string]any{
			"symlink": "target.txt",
			"content": "hello",
		},
	}

	_, err := ExpandDSL(dsl)
	if err == nil {
		t.Fatalf("expected error for symlink+content")
	}
}

func TestExpandSha256DSL(t *testing.T) {
	dsl := map[string]any{
		"data.bin": map[string]any{"sha256": "abc123def456"},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"data.bin": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sha256": map[string]any{"const": "abc123def456"},
				},
				"required": []any{"sha256"},
			},
		},
		"required": []any{"data.bin"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema mismatch:\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestExpandSizeExactDSL(t *testing.T) {
	dsl := map[string]any{
		"file.dat": map[string]any{"size": float64(1024)},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	props := got["properties"].(map[string]any)
	fileSchema := props["file.dat"].(map[string]any)
	fileProps := fileSchema["properties"].(map[string]any)
	sizeSchema := fileProps["size"].(map[string]any)

	if sizeSchema["const"] != int64(1024) {
		t.Fatalf("expected size const 1024, got %v", sizeSchema)
	}
}

func TestExpandSizeRangeDSL(t *testing.T) {
	dsl := map[string]any{
		"file.dat": map[string]any{
			"size": map[string]any{"min": float64(100), "max": float64(1000)},
		},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	props := got["properties"].(map[string]any)
	fileSchema := props["file.dat"].(map[string]any)
	fileProps := fileSchema["properties"].(map[string]any)
	sizeSchema := fileProps["size"].(map[string]any)

	if sizeSchema["type"] != "integer" {
		t.Fatalf("expected size type integer, got %v", sizeSchema["type"])
	}
	if sizeSchema["minimum"] != int64(100) {
		t.Fatalf("expected minimum 100, got %v", sizeSchema["minimum"])
	}
	if sizeSchema["maximum"] != int64(1000) {
		t.Fatalf("expected maximum 1000, got %v", sizeSchema["maximum"])
	}
}

func TestExpandCombinedConstraints(t *testing.T) {
	dsl := map[string]any{
		"file.txt": map[string]any{
			"content": "hello",
			"size":    float64(5),
			"sha256":  "abc123",
		},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	props := got["properties"].(map[string]any)
	fileSchema := props["file.txt"].(map[string]any)
	fileProps := fileSchema["properties"].(map[string]any)
	required := fileSchema["required"].([]any)

	if len(fileProps) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(fileProps))
	}
	if len(required) != 3 {
		t.Fatalf("expected 3 required, got %d", len(required))
	}
}

func TestExpandGlobPattern(t *testing.T) {
	dsl := map[string]any{
		"src/": map[string]any{
			"*.go":   true,
			"main.c": true,
		},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	srcSchema := got["properties"].(map[string]any)["src/"].(map[string]any)

	// main.c should be in properties
	props := srcSchema["properties"].(map[string]any)
	if _, ok := props["main.c"]; !ok {
		t.Fatalf("expected main.c in properties")
	}

	// *.go should be in patternProperties as regex
	patternProps := srcSchema["patternProperties"].(map[string]any)
	if _, ok := patternProps["^.*\\.go$"]; !ok {
		t.Fatalf("expected ^.*\\.go$ in patternProperties, got %v", patternProps)
	}

	// main.c should be required, but *.go should not
	required := srcSchema["required"].([]any)
	if len(required) != 1 || required[0] != "main.c" {
		t.Fatalf("expected only main.c in required, got %v", required)
	}
}

func TestExpandDirectoryGlobPattern(t *testing.T) {
	dsl := map[string]any{
		"logs-*/": map[string]any{
			"*.log": true,
		},
	}

	got, err := ExpandDSL(dsl)
	if err != nil {
		t.Fatalf("ExpandDSL: %v", err)
	}

	// logs-*/ should be in patternProperties
	patternProps := got["patternProperties"].(map[string]any)
	if _, ok := patternProps["^logs-.*/$"]; !ok {
		t.Fatalf("expected ^logs-.*/ in patternProperties, got %v", patternProps)
	}

	// Root should have no required entries (only pattern)
	required := got["required"].([]any)
	if len(required) != 0 {
		t.Fatalf("expected no required entries for pattern-only schema, got %v", required)
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		glob  string
		regex string
	}{
		{"*.go", "^.*\\.go$"},
		{"test_*.py", "^test_.*\\.py$"},
		{"file?.txt", "^file.\\.txt$"},
		{"[abc].txt", "^[abc]\\.txt$"},
		{"[!abc].txt", "^[^abc]\\.txt$"},
		{"data.json", "^data\\.json$"},
		{"logs-*/", "^logs-.*/$"},
	}

	for _, tc := range tests {
		t.Run(tc.glob, func(t *testing.T) {
			got, err := globToRegex(tc.glob)
			if err != nil {
				t.Fatalf("globToRegex(%q): %v", tc.glob, err)
			}
			if got != tc.regex {
				t.Fatalf("globToRegex(%q) = %q, want %q", tc.glob, got, tc.regex)
			}
		})
	}
}

func TestIsGlobPattern(t *testing.T) {
	tests := []struct {
		key    string
		isGlob bool
	}{
		{"main.go", false},
		{"*.go", true},
		{"test_?.py", true},
		{"[abc].txt", true},
		{"normal-file.txt", false},
		{"logs-2024/", false},
		{"logs-*/", true},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := isGlobPattern(tc.key)
			if got != tc.isGlob {
				t.Fatalf("isGlobPattern(%q) = %v, want %v", tc.key, got, tc.isGlob)
			}
		})
	}
}

// TestExpandOutputPassesMetaSchema validates that all expanded schemas
// conform to the meta-schema. This ensures the expansion produces
// correctly shaped output for all entry types.
func TestExpandOutputPassesMetaSchema(t *testing.T) {
	tests := []struct {
		name string
		dsl  map[string]any
	}{
		{
			name: "existence-only file",
			dsl: map[string]any{
				"README.md": true,
			},
		},
		{
			name: "nil file (implicit existence)",
			dsl: map[string]any{
				"file.txt": nil,
			},
		},
		{
			name: "symlink",
			dsl: map[string]any{
				"link.txt": map[string]any{"symlink": "target.txt"},
			},
		},
		{
			name: "content",
			dsl: map[string]any{
				"config.ini": map[string]any{"content": "[app]\nkey=value"},
			},
		},
		{
			name: "sha256",
			dsl: map[string]any{
				"data.bin": map[string]any{"sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
			},
		},
		{
			name: "size exact",
			dsl: map[string]any{
				"file.dat": map[string]any{"size": float64(1024)},
			},
		},
		{
			name: "size range",
			dsl: map[string]any{
				"file.dat": map[string]any{"size": map[string]any{"min": float64(0), "max": float64(1000)}},
			},
		},
		{
			name: "combined content+size+sha256",
			dsl: map[string]any{
				"file.txt": map[string]any{
					"content": "hello",
					"size":    float64(5),
					"sha256":  "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
				},
			},
		},
		{
			name: "directory with files",
			dsl: map[string]any{
				"src/": map[string]any{
					"main.go": true,
					"lib.go":  true,
				},
			},
		},
		{
			name: "nested directories",
			dsl: map[string]any{
				"project/": map[string]any{
					"src/": map[string]any{
						"main.go": true,
					},
					"README.md": map[string]any{"content": "# Project"},
				},
			},
		},
		{
			name: "list form DSL",
			dsl: map[string]any{
				"dir/": []any{
					"file1.txt",
					"file2.txt",
					map[string]any{"config.ini": map[string]any{"content": "key=value"}},
				},
			},
		},
		{
			name: "mixed entry types",
			dsl: map[string]any{
				"README.md":  true,
				"config.ini": map[string]any{"content": "[app]"},
				"link":       map[string]any{"symlink": "README.md"},
				"src/": map[string]any{
					"main.go": true,
				},
			},
		},
		{
			name: "glob pattern",
			dsl: map[string]any{
				"src/": map[string]any{
					"*.go": true,
				},
			},
		},
		{
			name: "directory glob pattern",
			dsl: map[string]any{
				"logs-*/": map[string]any{
					"*.log": true,
				},
			},
		},
		{
			name: "mixed patterns and literals",
			dsl: map[string]any{
				"src/": map[string]any{
					"main.go": true,
					"*.test.go": true,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expanded, err := ExpandDSL(tc.dsl)
			if err != nil {
				t.Fatalf("ExpandDSL: %v", err)
			}

			if err := schema.ValidateSchema(expanded); err != nil {
				t.Fatalf("meta-schema validation failed: %v", err)
			}
		})
	}
}
