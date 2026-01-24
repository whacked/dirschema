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

	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"src/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"main.go": map[string]any{"const": true},
				},
				"required": []string{"main.go"},
			},
			"README.md": map[string]any{"const": true},
		},
		"required": []string{"README.md", "src/"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema mismatch: got %#v want %#v", got, want)
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
