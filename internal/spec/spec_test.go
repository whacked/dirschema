package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
