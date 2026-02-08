package fswalk

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
}

func skipWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior varies on windows")
	}
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
	skipWindows(t)

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

// Test 1: Follow symlink to directory when schema expects directory contents
func TestWalkWithSchemaFollowSymlinkDir(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	mkdirAll(t, realDir)
	writeFile(t, realDir, "main.go", "package main")
	symlink(t, realDir, filepath.Join(root, "linked"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"linked/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"main.go": map[string]any{
						"oneOf": []any{
							map[string]any{"const": true},
							map[string]any{"type": "object"},
						},
					},
				},
				"required": []any{"main.go"},
			},
		},
		"required": []any{"linked/"},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	want := map[string]any{
		"linked/": map[string]any{
			"main.go": true,
		},
		"real/": map[string]any{
			"main.go": true,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch:\n  got:  %#v\n  want: %#v", got, want)
	}
}

// Test 2: Record symlink when schema expects symlink metadata
func TestWalkWithSchemaRecordSymlink(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	symlink(t, "target.txt", filepath.Join(root, "link.txt"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"link.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"symlink": map[string]any{"const": "target.txt"},
				},
				"required": []any{"symlink"},
			},
		},
		"required": []any{"link.txt"},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	want := map[string]any{
		"link.txt": map[string]any{"symlink": "target.txt"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch:\n  got:  %#v\n  want: %#v", got, want)
	}
}

// Test 3: Resolve file symlink when schema expects file (not symlink)
func TestWalkWithSchemaResolveFileSymlink(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	writeFile(t, root, "real.txt", "hello")
	symlink(t, filepath.Join(root, "real.txt"), filepath.Join(root, "data.txt"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"data.txt": map[string]any{
				"oneOf": []any{
					map[string]any{"const": true},
					map[string]any{"type": "object"},
				},
			},
		},
		"required": []any{"data.txt"},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	// data.txt resolved as file, real.txt is a regular file
	gotData, ok := got["data.txt"]
	if !ok {
		t.Fatalf("expected data.txt in output, got %#v", got)
	}
	if gotData != true {
		t.Fatalf("expected data.txt to be true, got %#v", gotData)
	}
}

// Test 4: Fall back to SymlinkPolicy when no schema match
func TestWalkWithSchemaFallback(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	symlink(t, "somewhere", filepath.Join(root, "unknown"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"other.txt": map[string]any{
				"oneOf": []any{
					map[string]any{"const": true},
					map[string]any{"type": "object"},
				},
			},
		},
		"required": []any{"other.txt"},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	want := map[string]any{
		"unknown": map[string]any{"symlink": "somewhere"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch:\n  got:  %#v\n  want: %#v", got, want)
	}
}

// Test 5: Cycle detection
func TestWalkWithSchemaCycleDetection(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	// Create a -> b and b -> a symlink cycle
	symlink(t, filepath.Join(root, "b"), filepath.Join(root, "a"))
	symlink(t, filepath.Join(root, "a"), filepath.Join(root, "b"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"a/": map[string]any{
				"type":     "object",
				"required": []any{},
			},
			"b/": map[string]any{
				"type":     "object",
				"required": []any{},
			},
		},
		"required": []any{"a/", "b/"},
	}

	_, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err == nil {
		t.Fatalf("expected error for symlink cycle")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "cycle") && !strings.Contains(errStr, "too many links") {
		t.Fatalf("expected cycle-related error, got: %v", err)
	}
}

// Test 6: Nested schema guidance — symlink inside subdirectory
func TestWalkWithSchemaNestedSymlink(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	libsDir := filepath.Join(root, "libs")
	mkdirAll(t, libsDir)
	writeFile(t, libsDir, "util.go", "package libs")

	srcDir := filepath.Join(root, "src")
	mkdirAll(t, srcDir)
	symlink(t, libsDir, filepath.Join(srcDir, "linked"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"src/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"linked/": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"util.go": map[string]any{
								"oneOf": []any{
									map[string]any{"const": true},
									map[string]any{"type": "object"},
								},
							},
						},
						"required": []any{"util.go"},
					},
				},
				"required": []any{"linked/"},
			},
		},
		"required": []any{"src/"},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	srcVal, ok := got["src/"].(map[string]any)
	if !ok {
		t.Fatalf("expected src/ in output, got %#v", got)
	}
	linkedVal, ok := srcVal["linked/"].(map[string]any)
	if !ok {
		t.Fatalf("expected linked/ in src/, got %#v", srcVal)
	}
	if linkedVal["util.go"] != true {
		t.Fatalf("expected util.go in linked/, got %#v", linkedVal)
	}
}

// Test 7: No schema (nil) — uses SymlinkPolicy only
func TestWalkWithSchemaNil(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	symlink(t, "target", filepath.Join(root, "link"))

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, nil)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	want := map[string]any{
		"link": map[string]any{"symlink": "target"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instance mismatch:\n  got:  %#v\n  want: %#v", got, want)
	}
}

// Test 8: Schema expects directory but symlink points to file
func TestWalkWithSchemaDirExpectedFileTarget(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	writeFile(t, root, "regular_file", "hello")
	symlink(t, filepath.Join(root, "regular_file"), filepath.Join(root, "foo"))

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"foo/": map[string]any{
				"type":     "object",
				"required": []any{},
			},
		},
		"required": []any{"foo/"},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	// Falls through to SymlinkRecord since target is not a dir
	fooVal, ok := got["foo"].(map[string]any)
	if !ok {
		t.Fatalf("expected foo as symlink record, got %#v", got)
	}
	if _, ok := fooVal["symlink"]; !ok {
		t.Fatalf("expected symlink key, got %#v", fooVal)
	}
}

// Test 9: patternProperties guidance — glob schema matches symlinked dir
func TestWalkWithSchemaPatternProperties(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	archiveDir := filepath.Join(root, "archive-logs-2024")
	mkdirAll(t, archiveDir)
	writeFile(t, archiveDir, "app.log", "log data")
	symlink(t, archiveDir, filepath.Join(root, "logs-2024"))

	schema := map[string]any{
		"type": "object",
		"patternProperties": map[string]any{
			"^logs-.*/$": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"app.log": map[string]any{
						"oneOf": []any{
							map[string]any{"const": true},
							map[string]any{"type": "object"},
						},
					},
				},
				"required": []any{"app.log"},
			},
		},
		"required": []any{},
	}

	got, err := WalkWithSchema(root, Options{SymlinkPolicy: SymlinkRecord}, schema)
	if err != nil {
		t.Fatalf("WalkWithSchema: %v", err)
	}

	logsVal, ok := got["logs-2024/"].(map[string]any)
	if !ok {
		t.Fatalf("expected logs-2024/ in output, got %#v", got)
	}
	if logsVal["app.log"] != true {
		t.Fatalf("expected app.log in logs-2024/, got %#v", logsVal)
	}
}

// Test 10: Walk with SymlinkFollow follows all symlinks (no schema needed)
func TestWalkSymlinkFollowDir(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	realDir := filepath.Join(root, "real-dir")
	mkdirAll(t, realDir)
	writeFile(t, realDir, "file.txt", "data")
	symlink(t, realDir, filepath.Join(root, "link"))

	got, err := Walk(root, Options{SymlinkPolicy: SymlinkFollow})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	linkVal, ok := got["link/"].(map[string]any)
	if !ok {
		t.Fatalf("expected link/ as followed directory, got %#v", got)
	}
	if linkVal["file.txt"] != true {
		t.Fatalf("expected file.txt in link/, got %#v", linkVal)
	}
}

// Test 11: Walk with SymlinkFollow resolves file symlinks
func TestWalkSymlinkFollowFile(t *testing.T) {
	skipWindows(t)

	root := t.TempDir()
	writeFile(t, root, "real.txt", "hello")
	symlink(t, filepath.Join(root, "real.txt"), filepath.Join(root, "link.txt"))

	got, err := Walk(root, Options{SymlinkPolicy: SymlinkFollow})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	if got["link.txt"] != true {
		t.Fatalf("expected link.txt to be true (resolved file), got %#v", got["link.txt"])
	}
}
