package cli

import (
	"bytes"
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

func TestExpandCommand(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", "src/:\n  main.go: true\nREADME.md: true\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"expand", path}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", exitCode, stderr.String())
	}

	got := decodeJSON(t, stdout.Bytes())
	want := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"src/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"main.go": map[string]any{"const": true},
				},
				"required": []any{"main.go"},
			},
			"README.md": map[string]any{"const": true},
		},
		"required": []any{"README.md", "src/"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema mismatch: got %#v want %#v", got, want)
	}
}

func TestExpandCommandMixedSpec(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.json", `{"type":"object","src/":{}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"expand", path}, &stdout, &stderr)
	if exitCode != ExitConfigError {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitConfigError)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected stderr output")
	}
}

func TestVersionFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--version"}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitSuccess)
	}
	if stdout.String() == "" {
		t.Fatalf("expected version output")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}
