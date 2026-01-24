package hydrate

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestBuildPlanBasic(t *testing.T) {
	root := t.TempDir()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"dir/": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"file.txt": map[string]any{"const": true},
				},
				"required": []any{"file.txt"},
			},
			"root.txt": map[string]any{"const": true},
		},
		"required": []any{"dir/", "root.txt"},
	}

	plan, err := BuildPlan(schema, root)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	gotKinds := []OpKind{}
	gotRel := []string{}
	for _, op := range plan.Ops {
		gotKinds = append(gotKinds, op.Kind)
		gotRel = append(gotRel, op.RelPath)
	}

	wantKinds := []OpKind{OpMkdir, OpWriteFile, OpWriteFile}
	wantRel := []string{"dir", filepath.Join("dir", "file.txt"), "root.txt"}

	if !reflect.DeepEqual(gotKinds, wantKinds) || !reflect.DeepEqual(gotRel, wantRel) {
		t.Fatalf("plan mismatch: got %v %v want %v %v", gotKinds, gotRel, wantKinds, wantRel)
	}
}

func TestApplyCreatesFiles(t *testing.T) {
	root := t.TempDir()
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file.txt": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{"const": "hello"},
				},
			},
		},
		"required": []any{"file.txt"},
	}

	plan, err := BuildPlan(schema, root)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if err := Apply(plan, ApplyOptions{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, "file.txt"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestApplyDryRun(t *testing.T) {
	root := t.TempDir()
	plan := Plan{Ops: []Op{{
		Kind:    OpWriteFile,
		Path:    filepath.Join(root, "file.txt"),
		RelPath: "file.txt",
	}}}

	if err := Apply(plan, ApplyOptions{DryRun: true}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "file.txt")); err == nil {
		t.Fatalf("expected file not to be created in dry-run")
	}
}

func TestApplySymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior varies on windows")
	}

	root := t.TempDir()
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

	plan, err := BuildPlan(schema, root)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if err := Apply(plan, ApplyOptions{}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	target, err := os.Readlink(filepath.Join(root, "link.txt"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != "target.txt" {
		t.Fatalf("unexpected target: %q", target)
	}
}

func TestRequiredKeysTypes(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]any
		want   []string
	}{
		{
			name:   "nil required",
			schema: map[string]any{},
			want:   nil,
		},
		{
			name: "[]any type",
			schema: map[string]any{
				"required": []any{"b", "a", "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "[]string type",
			schema: map[string]any{
				"required": []string{"z", "y", "x"},
			},
			want: []string{"x", "y", "z"},
		},
		{
			name: "[]any with non-strings ignored",
			schema: map[string]any{
				"required": []any{"a", 123, "b"},
			},
			want: []string{"a", "b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := requiredKeys(tc.schema)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("requiredKeys() = %v, want %v", got, tc.want)
			}
		})
	}
}
