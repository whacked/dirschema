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

func TestExportCommand(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("yo"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"export", "--root", root}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, ExitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var got []any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	want := []any{
		"a.txt",
		map[string]any{
			"sub/": []any{
				"b.txt",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("export mismatch: got %#v want %#v", got, want)
	}
}

func TestExportCommandSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior varies on windows")
	}

	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Symlink("target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run([]string{"export", "--root", root}, &stdout, &stderr)
	if exitCode != ExitSuccess {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", exitCode, ExitSuccess, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	var got []any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var link map[string]any
	for _, entry := range got {
		if m, ok := entry.(map[string]any); ok {
			if val, ok := m["link.txt"]; ok {
				link, _ = val.(map[string]any)
				break
			}
		}
	}
	if link == nil {
		t.Fatalf("expected link.txt entry")
	}
	if link["symlink"] != "target.txt" {
		t.Fatalf("expected symlink target, got %#v", link["symlink"])
	}
}
