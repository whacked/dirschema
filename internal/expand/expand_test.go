package expand

import (
	"reflect"
	"testing"
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
				"required": []string{"main.go"},
			},
			"README.md": existenceSchema,
		},
		"required": []string{"README.md", "src/"},
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
				"required": []string{"symlink"},
			},
		},
		"required": []string{"link.txt"},
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
				"required": []string{"content"},
			},
		},
		"required": []string{"config.txt"},
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
				"required": []string{"sha256"},
			},
		},
		"required": []string{"data.bin"},
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
	required := fileSchema["required"].([]string)

	if len(fileProps) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(fileProps))
	}
	if len(required) != 3 {
		t.Fatalf("expected 3 required, got %d", len(required))
	}
}
