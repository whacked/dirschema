package expand

import (
	"errors"
	"fmt"
	"strings"
)

type ParseOptions struct {
	CaseSensitive bool
}

func ParseDSL(root any, opts ParseOptions) (map[string]any, error) {
	switch v := root.(type) {
	case map[string]any:
		return parseNode(v, opts)
	case []any:
		return parseList("root", v, opts)
	default:
		return nil, fmt.Errorf("unsupported DSL root")
	}
}

func parseNode(node map[string]any, opts ParseOptions) (map[string]any, error) {
	out := make(map[string]any, len(node))
	seen := map[string]struct{}{}
	for key, value := range node {
		norm := normalizeKey(key, opts)
		if _, ok := seen[norm]; ok {
			return nil, fmt.Errorf("duplicate entry %q", key)
		}
		seen[norm] = struct{}{}

		parsed, err := parseValue(key, value, opts)
		if err != nil {
			return nil, err
		}
		out[key] = parsed
	}
	return out, nil
}

func parseValue(key string, value any, opts ParseOptions) (any, error) {
	if key == "symlink" {
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return nil, fmt.Errorf("symlink must be string")
		}
	}
	switch v := value.(type) {
	case nil:
		return nil, nil
	case bool:
		return v, nil
	case map[string]any:
		return parseNode(v, opts)
	case []any:
		return parseList(key, v, opts)
	default:
		return nil, fmt.Errorf("unsupported value for %q", key)
	}
}

func parseList(parent string, list []any, opts ParseOptions) (map[string]any, error) {
	out := make(map[string]any, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		switch v := item.(type) {
		case string:
			if err := addEntry(out, seen, v, true, opts); err != nil {
				return nil, err
			}
		case map[string]any:
			if len(v) != 1 {
				return nil, fmt.Errorf("list entry under %q must have a single key", parent)
			}
			for k, raw := range v {
				parsed, err := parseValue(k, raw, opts)
				if err != nil {
					return nil, err
				}
				if err := addEntry(out, seen, k, parsed, opts); err != nil {
					return nil, err
				}
			}
		default:
			return nil, fmt.Errorf("list entry under %q must be string or map", parent)
		}
	}
	return out, nil
}

func addEntry(out map[string]any, seen map[string]struct{}, key string, value any, opts ParseOptions) error {
	norm := normalizeKey(key, opts)
	if _, ok := seen[norm]; ok {
		return fmt.Errorf("duplicate entry %q", key)
	}
	seen[norm] = struct{}{}
	out[key] = value
	return nil
}

func normalizeKey(key string, opts ParseOptions) string {
	if opts.CaseSensitive {
		return key
	}
	return strings.ToLower(key)
}

func ParseDSLOrPanic(root map[string]any) map[string]any {
	parsed, err := ParseDSL(root, ParseOptions{})
	if err != nil {
		panic(err)
	}
	return parsed
}

var ErrUnsupportedDSL = errors.New("unsupported DSL form")
