package hydrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type OpKind string

const (
	OpMkdir     OpKind = "mkdir"
	OpWriteFile OpKind = "writefile"
	OpSymlink   OpKind = "symlink"
)

type Op struct {
	Kind    OpKind
	Path    string
	RelPath string
	Content *string
	Target  string
}

type Plan struct {
	Ops []Op
}

func BuildPlan(schema map[string]any, root string) (Plan, error) {
	ops, err := collectOps(schema, root, "")
	if err != nil {
		return Plan{}, err
	}
	stableSortOps(ops)
	return Plan{Ops: ops}, nil
}

func collectOps(schema map[string]any, root, rel string) ([]Op, error) {
	// Check for patternProperties - cannot hydrate patterns
	if _, hasPatterns := schema["patternProperties"]; hasPatterns {
		if rel == "" {
			return nil, fmt.Errorf("cannot hydrate schema with patternProperties at root level")
		}
		return nil, fmt.Errorf("cannot hydrate schema with patternProperties at %q", rel)
	}

	props, _ := schema["properties"].(map[string]any)
	required := requiredKeys(schema)

	var ops []Op
	for _, name := range required {
		childSchemaRaw, ok := props[name]
		if !ok {
			return nil, fmt.Errorf("required entry %q missing schema", name)
		}
		childSchema, ok := childSchemaRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schema for %q must be object", name)
		}

		childRel := filepath.Join(rel, name)
		if isDirectorySchema(childSchema, name) {
			dirRel := strings.TrimSuffix(childRel, string(filepath.Separator)+"")
			dirRel = strings.TrimSuffix(dirRel, "/")
			childOps, err := collectOps(childSchema, root, dirRel)
			if err != nil {
				return nil, err
			}
			dirPath := filepath.Join(root, dirRel)
			if !pathExists(dirPath) {
				op := Op{
					Kind:    OpMkdir,
					Path:    dirPath,
					RelPath: dirRel,
				}
				ops = append(ops, op)
			}
			ops = append(ops, childOps...)
			continue
		}

		if target, ok, err := symlinkTargetFromSchema(childSchema); err != nil {
			return nil, err
		} else if ok {
			op := Op{
				Kind:    OpSymlink,
				Path:    filepath.Join(root, childRel),
				RelPath: childRel,
				Target:  target,
			}
			if !pathExists(op.Path) {
				ops = append(ops, op)
			}
			continue
		}

		content := contentFromSchema(childSchema)
		op := Op{
			Kind:    OpWriteFile,
			Path:    filepath.Join(root, childRel),
			RelPath: childRel,
			Content: content,
		}
		if !pathExists(op.Path) {
			ops = append(ops, op)
		}
	}

	return ops, nil
}

func requiredKeys(schema map[string]any) []string {
	var out []string
	if raw, ok := schema["required"]; ok {
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
		case []string:
			out = append(out, v...)
		}
	}
	sort.Strings(out)
	return out
}

func isDirectorySchema(schema map[string]any, name string) bool {
	if strings.HasSuffix(name, "/") {
		return true
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return false
	}
	for key := range props {
		if strings.HasSuffix(key, "/") {
			return true
		}
	}

	if isFileDescriptorProperties(props) {
		return false
	}
	return true
}

func isFileDescriptorProperties(props map[string]any) bool {
	for key := range props {
		switch key {
		case "size", "sha256", "content", "mode", "symlink":
			continue
		default:
			return false
		}
	}
	return len(props) > 0
}

func contentFromSchema(schema map[string]any) *string {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	contentSchema, ok := props["content"].(map[string]any)
	if !ok {
		return nil
	}
	if val, ok := contentSchema["const"]; ok {
		if str, ok := val.(string); ok {
			return &str
		}
	}
	return nil
}

func stableSortOps(ops []Op) {
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].RelPath == ops[j].RelPath {
			return ops[i].Kind < ops[j].Kind
		}
		return ops[i].RelPath < ops[j].RelPath
	})
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func symlinkTargetFromSchema(schema map[string]any) (string, bool, error) {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	raw, ok := props["symlink"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	if _, ok := raw["const"]; !ok {
		return "", false, fmt.Errorf("symlink requires const target for hydration")
	}
	if val, ok := raw["const"]; ok {
		target, ok := val.(string)
		if !ok {
			return "", false, fmt.Errorf("symlink const must be string")
		}
		return target, true, nil
	}
	return "", false, nil
}
