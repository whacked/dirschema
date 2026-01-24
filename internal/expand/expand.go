package expand

import (
	"errors"
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
		return map[string]any{"const": true}, nil
	}
	boolVal, ok := value.(bool)
	if ok {
		if boolVal {
			return map[string]any{"const": true}, nil
		}
		return nil, fmt.Errorf("file %q must be true or object, got false", key)
	}
	if obj, ok := value.(map[string]any); ok {
		return expandFileDescriptor(key, obj)
	}
	return nil, fmt.Errorf("file %q must be true or object", key)
}

func expandFileDescriptor(key string, obj map[string]any) (map[string]any, error) {
	if len(obj) != 1 {
		return nil, errors.New("file descriptor objects must contain a single key")
	}
	if raw, ok := obj["symlink"]; ok {
		target, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("file %q symlink target must be string", key)
		}
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symlink": map[string]any{"const": target},
			},
			"required":             []string{"symlink"},
		}, nil
	}
	return nil, errors.New("file descriptor objects are not supported yet")
}
