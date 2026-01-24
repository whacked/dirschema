package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeJSONFile(t *testing.T, dir, name string, value string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func TestValidateValid(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	spec := `{"type":"object","properties":{"a.txt":{"const":true}},"required":["a.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "--root", root, specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, ExitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestValidateInvalid(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	spec := `{"type":"object","properties":{"a.txt":{"const":true}},"required":["a.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "--root", root, specPath}, &stdout, &stderr)
	if exitCode != ExitValidation {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitValidation)
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected stderr output")
	}
}

func TestValidateJSONFormat(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	spec := `{"type":"object","properties":{"a.txt":{"const":true}},"required":["a.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "--root", root, "--format", "json", specPath}, &stdout, &stderr)
	if exitCode != ExitValidation {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitValidation)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode stdout: %v", err)
	}
	if valid, ok := payload["valid"].(bool); !ok || valid {
		t.Fatalf("expected valid=false, got %v", payload["valid"])
	}
}

func TestValidatePrintInstance(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	spec := `{"type":"object","properties":{"a.txt":{"const":true}},"required":["a.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"validate", "--root", root, "--print-instance", specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var instance map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &instance); err != nil {
		t.Fatalf("decode instance: %v", err)
	}
	if _, ok := instance["a.txt"]; !ok {
		t.Fatalf("expected instance to include a.txt")
	}
}

func TestValidateDefaultCommand(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	spec := `{"type":"object","properties":{"a.txt":{"const":true}},"required":["a.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"--root", root, specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, ExitSuccess, stderr.String())
	}
}
