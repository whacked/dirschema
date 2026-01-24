package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHydrateCreatesMissing(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	spec := `{"type":"object","properties":{"dir/":{"type":"object","properties":{"file.txt":{"const":true}},"required":["file.txt"]},"root.txt":{"const":true}},"required":["dir/","root.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"hydrate", "--root", root, specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, ExitSuccess, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(root, "root.txt")); err != nil {
		t.Fatalf("expected root.txt created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "dir", "file.txt")); err != nil {
		t.Fatalf("expected file.txt created: %v", err)
	}
}

func TestHydrateDryRun(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	spec := `{"type":"object","properties":{"root.txt":{"const":true}},"required":["root.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"hydrate", "--root", root, "--dry-run", specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitSuccess)
	}
	if _, err := os.Stat(filepath.Join(root, "root.txt")); err == nil {
		t.Fatalf("expected file not created in dry-run")
	}
}

func TestHydrateReportsInvalid(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	spec := `{"type":"object","properties":{"root.txt":{"const":true,"defaultContent":"x"}},"required":["root.txt"]}`
	specPath := writeJSONFile(t, dir, "spec.json", spec)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"hydrate", "--root", root, "--format", "json", specPath}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d", exitCode, ExitSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode stdout: %v", err)
	}
	if valid, ok := payload["valid"].(bool); !ok || !valid {
		t.Fatalf("expected valid=true after hydrate, got %v", payload["valid"])
	}
}
