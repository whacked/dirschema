package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func decodeJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return out
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.json", `{"foo": "bar"}`)

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("json mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", "foo: bar\n")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("json mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadJsonnet(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.jsonnet", `{ foo: "bar" }`)

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("json mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", "foo: [\n")

	_, err := Load(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestInferKind(t *testing.T) {
	dsl := map[string]any{"src/": map[string]any{"main.go": true}}
	kind, err := InferKind(dsl)
	if err != nil {
		t.Fatalf("InferKind DSL: %v", err)
	}
	if kind != KindDSL {
		t.Fatalf("InferKind DSL: got %v want %v", kind, KindDSL)
	}

	schema := map[string]any{"type": "object", "properties": map[string]any{}}
	kind, err = InferKind(schema)
	if err != nil {
		t.Fatalf("InferKind schema: %v", err)
	}
	if kind != KindSchema {
		t.Fatalf("InferKind schema: got %v want %v", kind, KindSchema)
	}

	mixed := map[string]any{"type": "object", "src/": map[string]any{}}
	if _, err := InferKind(mixed); err == nil {
		t.Fatalf("expected mixed spec error")
	}
}

func decodeJSONAny(t *testing.T, raw []byte) any {
	t.Helper()
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return out
}

func TestLoadFromReader_YAMLList(t *testing.T) {
	// Input starting with '-' should be parsed as YAML
	input := "- foo/\n- bar/"
	loaded, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	got := decodeJSONAny(t, loaded.JSON)
	want := []any{"foo/", "bar/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadFromReader_JSONObject(t *testing.T) {
	// Input starting with '{' should be parsed as Jsonnet (which handles JSON)
	input := `{"foo": "bar"}`
	loaded, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadFromReader_JSONArray(t *testing.T) {
	// Input starting with '[' should be parsed as Jsonnet
	input := `["foo/", "bar/"]`
	loaded, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	got := decodeJSONAny(t, loaded.JSON)
	want := []any{"foo/", "bar/"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadFromReader_Jsonnet(t *testing.T) {
	// Jsonnet with object syntax (must start with { for stdin detection)
	input := `{ local x = "bar", foo: x }`
	loaded, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadFromReader_YAMLMap(t *testing.T) {
	// YAML map (doesn't start with -, {, or [)
	input := "foo: bar\nbaz: qux"
	loaded, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar", "baz": "qux"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadFromReader_WhitespacePrefix(t *testing.T) {
	// Whitespace before the actual content
	input := "  \n  {\"foo\": \"bar\"}"
	loaded, err := LoadFromReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("LoadFromReader: %v", err)
	}

	got := decodeJSON(t, loaded.JSON)
	want := map[string]any{"foo": "bar"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mismatch: got %#v want %#v", got, want)
	}
}

func TestLoadFromReader_Empty(t *testing.T) {
	_, err := LoadFromReader(strings.NewReader(""))
	if err == nil {
		t.Fatalf("expected error for empty input")
	}
}

func TestLoadFromReader_WhitespaceOnly(t *testing.T) {
	_, err := LoadFromReader(strings.NewReader("   \n\t  "))
	if err == nil {
		t.Fatalf("expected error for whitespace-only input")
	}
}
