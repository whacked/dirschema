package expand

import (
	"fmt"
	"sort"
	"strings"
)

func ExpandDSL(root any) (map[string]any, error) {
	parsed, err := ParseDSL(root, ParseOptions{})
	if err != nil {
		return nil, err
	}
	return expandDir(parsed)
}

func expandDir(node map[string]any) (map[string]any, error) {
	keys := make([]string, 0, len(node))
	for key := range node {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	properties := make(map[string]any, len(node))
	required := make([]string, 0, len(node))

	for _, key := range keys {
		value := node[key]
		var schema map[string]any
		var err error

		if strings.HasSuffix(key, "/") {
			schema, err = expandDirectoryValue(key, value)
		} else {
			schema, err = expandFileValue(key, value)
		}
		if err != nil {
			return nil, err
		}
		properties[key] = schema
		required = append(required, key)
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
	}, nil
}

func expandDirectoryValue(key string, value any) (map[string]any, error) {
	switch v := value.(type) {
	case nil:
		return expandDir(map[string]any{})
	case map[string]any:
		return expandDir(v)
	default:
		return nil, fmt.Errorf("directory %q must map to an object", key)
	}
}

func expandFileValue(key string, value any) (map[string]any, error) {
	if value == nil {
		return existenceOnlyFileSchema(), nil
	}
	boolVal, ok := value.(bool)
	if ok {
		if boolVal {
			return existenceOnlyFileSchema(), nil
		}
		return nil, fmt.Errorf("file %q must be true or object, got false", key)
	}
	if obj, ok := value.(map[string]any); ok {
		return expandFileDescriptor(key, obj)
	}
	return nil, fmt.Errorf("file %q must be true or object", key)
}

// existenceOnlyFileSchema returns a schema that matches both:
// - true (when no attributes requested)
// - object (when global attributes like content are included)
func existenceOnlyFileSchema() map[string]any {
	return map[string]any{
		"oneOf": []any{
			map[string]any{"const": true},
			map[string]any{"type": "object"},
		},
	}
}

func expandFileDescriptor(key string, obj map[string]any) (map[string]any, error) {
	// Check for mutually exclusive properties
	_, hasSymlink := obj["symlink"]
	_, hasContent := obj["content"]
	_, hasSize := obj["size"]
	_, hasSha256 := obj["sha256"]

	// Symlink is exclusive with everything else
	if hasSymlink && (hasContent || hasSize || hasSha256) {
		return nil, fmt.Errorf("file %q: symlink cannot be combined with content/size/sha256", key)
	}

	// Symlink-only case
	if hasSymlink {
		target, ok := obj["symlink"].(string)
		if !ok {
			return nil, fmt.Errorf("file %q symlink target must be string", key)
		}
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symlink": map[string]any{"const": target},
			},
			"required": []string{"symlink"},
		}, nil
	}

	// Regular file with content/size/sha256 (can be combined)
	if hasContent || hasSize || hasSha256 {
		props := make(map[string]any)
		required := make([]string, 0)

		if hasContent {
			content, ok := obj["content"].(string)
			if !ok {
				return nil, fmt.Errorf("file %q content must be string", key)
			}
			props["content"] = map[string]any{"const": content}
			required = append(required, "content")
		}

		if hasSize {
			sizeSchema, err := expandSizeConstraint(key, obj["size"])
			if err != nil {
				return nil, err
			}
			props["size"] = sizeSchema
			required = append(required, "size")
		}

		if hasSha256 {
			hash, ok := obj["sha256"].(string)
			if !ok {
				return nil, fmt.Errorf("file %q sha256 must be string", key)
			}
			props["sha256"] = map[string]any{"const": hash}
			required = append(required, "sha256")
		}

		sort.Strings(required)
		return map[string]any{
			"type":       "object",
			"properties": props,
			"required":   required,
		}, nil
	}

	// List unsupported keys for better error message
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return nil, fmt.Errorf("file %q has unsupported descriptor keys: %v", key, keys)
}

func expandSizeConstraint(key string, size any) (map[string]any, error) {
	switch v := size.(type) {
	case float64:
		return map[string]any{"const": int64(v)}, nil
	case int:
		return map[string]any{"const": int64(v)}, nil
	case map[string]any:
		schema := map[string]any{"type": "integer"}
		if min, ok := v["min"]; ok {
			switch m := min.(type) {
			case float64:
				schema["minimum"] = int64(m)
			case int:
				schema["minimum"] = int64(m)
			default:
				return nil, fmt.Errorf("file %q size.min must be number", key)
			}
		}
		if max, ok := v["max"]; ok {
			switch m := max.(type) {
			case float64:
				schema["maximum"] = int64(m)
			case int:
				schema["maximum"] = int64(m)
			default:
				return nil, fmt.Errorf("file %q size.max must be number", key)
			}
		}
		return schema, nil
	default:
		return nil, fmt.Errorf("file %q size must be number or {min, max}", key)
	}
}
