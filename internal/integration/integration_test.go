package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"dirschema/internal/cli"
)

func TestValidateIntegrationValid(t *testing.T) {
	spec := fixturePath(t, "simple_valid", "spec.json")
	root := fixturePath(t, "simple_valid", "root")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cli.Run([]string{"validate", "--root", root, spec}, &stdout, &stderr)
	if exitCode != cli.ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, cli.ExitSuccess, stderr.String())
	}
}

func TestValidateIntegrationInvalid(t *testing.T) {
	spec := fixturePath(t, "simple_invalid_missing", "spec.json")
	root := fixturePath(t, "simple_invalid_missing", "root")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cli.Run([]string{"validate", "--root", root, spec}, &stdout, &stderr)
	if exitCode != cli.ExitValidation {
		t.Fatalf("exit code: got %d want %d", exitCode, cli.ExitValidation)
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected stderr output")
	}
}

func TestValidateIntegrationGlobNoMatch(t *testing.T) {
	spec := fixturePath(t, "invalid_glob_no_match", "spec.yaml")
	root := fixturePath(t, "invalid_glob_no_match", "root")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cli.Run([]string{"validate", "--root", root, spec}, &stdout, &stderr)
	if exitCode != cli.ExitValidation {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, cli.ExitValidation, stderr.String())
	}
	if stderr.Len() == 0 {
		t.Fatalf("expected stderr output for unmatched glob")
	}
}

func TestHydrateIntegration(t *testing.T) {
	spec := fixturePath(t, "hydrate_empty", "spec.json")
	root := t.TempDir()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := cli.Run([]string{"hydrate", "--root", root, spec}, &stdout, &stderr)
	if exitCode != cli.ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, cli.ExitSuccess, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(root, "README.md")); err != nil {
		t.Fatalf("expected README.md: %v", err)
	}

	appConf := filepath.Join(root, "config", "app.conf")
	if _, err := os.Stat(appConf); err != nil {
		t.Fatalf("expected app.conf: %v", err)
	}

	// Verify content was written from schema
	content, err := os.ReadFile(appConf)
	if err != nil {
		t.Fatalf("read app.conf: %v", err)
	}
	if string(content) != "key=value\n" {
		t.Fatalf("app.conf content: got %q want %q", string(content), "key=value\n")
	}
}

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "testdata"))
	return filepath.Join(append([]string{root}, parts...)...)
}
