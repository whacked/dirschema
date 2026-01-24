package fswalk

import (
	"crypto/sha256"
	"encoding/hex"
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

func TestWalkSimple(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "a.txt", "hello")
	subdir := filepath.Join(root, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, subdir, "b.txt", "world")

	got, err := Walk(root, Options{})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := map[string]any{
		"a.txt": true,
		"sub/": map[string]any{
			"b.txt": true,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch: got %#v want %#v", got, want)
	}
}

func TestWalkIncludesSize(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "file.txt", "abc")

	got, err := Walk(root, Options{IncludeSize: true})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := map[string]any{
		"file.txt": map[string]any{
			"size": int64(3),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch: got %#v want %#v", got, want)
	}
}

func TestWalkIncludesSHA256(t *testing.T) {
	root := t.TempDir()
	contents := []byte("abc")
	writeFile(t, root, "file.txt", string(contents))

	sum := sha256.Sum256(contents)
	expected := hex.EncodeToString(sum[:])

	got, err := Walk(root, Options{IncludeSHA256: true})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := map[string]any{
		"file.txt": map[string]any{
			"sha256": expected,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch: got %#v want %#v", got, want)
	}
}

func TestWalkIncludesContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "file.txt", "hello")

	got, err := Walk(root, Options{IncludeContent: true, MaxContentBytes: 16})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	want := map[string]any{
		"file.txt": map[string]any{
			"content": "hello",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch: got %#v want %#v", got, want)
	}
}

func TestWalkRecordsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior varies on windows")
	}

	root := t.TempDir()
	writeFile(t, root, "file.txt", "data")
	linkPath := filepath.Join(root, "link.txt")
	if err := os.Symlink(filepath.Join(root, "file.txt"), linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	got, err := Walk(root, Options{SymlinkPolicy: SymlinkRecord})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	link, ok := got["link.txt"].(map[string]any)
	if !ok {
		t.Fatalf("expected link.txt to be object")
	}
	if _, ok := link["symlink"]; !ok {
		t.Fatalf("expected symlink target")
	}
}
