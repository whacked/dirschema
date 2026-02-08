package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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

// Test 12: End-to-end validate with symlinked subdir
func TestValidateWithSymlinkedSubdir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior varies on windows")
	}

	// Create directory structure:
	//   root/src/main.go
	//   root/real-lib/util.go
	//   root/lib -> root/real-lib (symlink)
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, srcDir, "main.go", "package main")

	realLib := filepath.Join(root, "real-lib")
	if err := os.MkdirAll(realLib, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, realLib, "util.go", "package lib")

	if err := os.Symlink(realLib, filepath.Join(root, "lib")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	// DSL spec: expects lib/ as a directory with util.go
	specDir := t.TempDir()
	specPath := writeFile(t, specDir, "spec.yaml", `
src/:
  main.go: true
lib/:
  util.go: true
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "--root", root, specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, ExitSuccess, stderr.String())
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
