package expand

import (
	"fmt"
	"regexp"
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

	properties := make(map[string]any)
	patternProperties := make(map[string]any)
	required := make([]any, 0)

	for _, key := range keys {
		value := node[key]
		var schema map[string]any
		var err error

		if strings.HasSuffix(key, "/") {
			// Directory - check if it's a pattern
			dirName := key
			if isGlobPattern(dirName) {
				// Pattern directory: strip trailing / for pattern, convert to regex
				patternBase := strings.TrimSuffix(dirName, "/")
				regexPattern, err := globToRegex(patternBase + "/")
				if err != nil {
					return nil, err
				}
				schema, err = expandDirectoryValue(key, value)
				if err != nil {
					return nil, err
				}
				patternProperties[regexPattern] = schema
			} else {
				schema, err = expandDirectoryValue(key, value)
				if err != nil {
					return nil, err
				}
				properties[key] = schema
				required = append(required, key)
			}
		} else {
			// File - check if it's a pattern
			if isGlobPattern(key) {
				regexPattern, err := globToRegex(key)
				if err != nil {
					return nil, err
				}
				schema, err = expandFileValue(key, value)
				if err != nil {
					return nil, err
				}
				patternProperties[regexPattern] = schema
			} else {
				schema, err = expandFileValue(key, value)
				if err != nil {
					return nil, err
				}
				properties[key] = schema
				required = append(required, key)
			}
		}
	}

	result := map[string]any{
		"type": "object",
	}

	if len(properties) > 0 {
		result["properties"] = properties
	}
	if len(patternProperties) > 0 {
		result["patternProperties"] = patternProperties
	}
	if len(required) > 0 {
		result["required"] = required
	} else {
		result["required"] = []any{}
	}

	return result, nil
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
			"required": []any{"symlink"},
		}, nil
	}

	// Regular file with content/size/sha256 (can be combined)
	if hasContent || hasSize || hasSha256 {
		props := make(map[string]any)
		required := make([]any, 0)

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

		sortAnyStrings(required)
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

func sortAnyStrings(s []any) {
	sort.Slice(s, func(i, j int) bool {
		return s[i].(string) < s[j].(string)
	})
}

// isGlobPattern returns true if the key contains glob characters (* ? [)
func isGlobPattern(key string) bool {
	return strings.ContainsAny(key, "*?[")
}

// globToRegex converts a simple glob pattern to a regex pattern.
// Supports: * (any chars), ? (single char), [...] (character class)
// The result is anchored with ^ and $.
func globToRegex(glob string) (string, error) {
	var buf strings.Builder
	buf.WriteString("^")

	i := 0
	for i < len(glob) {
		c := glob[i]
		switch c {
		case '*':
			buf.WriteString(".*")
		case '?':
			buf.WriteString(".")
		case '[':
			// Find matching ]
			j := i + 1
			if j < len(glob) && glob[j] == '!' {
				j++
			}
			if j < len(glob) && glob[j] == ']' {
				j++
			}
			for j < len(glob) && glob[j] != ']' {
				j++
			}
			if j >= len(glob) {
				return "", fmt.Errorf("unclosed character class in glob: %s", glob)
			}
			// Copy the character class as-is (it's valid regex)
			charClass := glob[i : j+1]
			// Convert [!...] to [^...]
			if len(charClass) > 2 && charClass[1] == '!' {
				buf.WriteString("[^")
				buf.WriteString(charClass[2:])
			} else {
				buf.WriteString(charClass)
			}
			i = j
		case '.', '+', '^', '$', '(', ')', '{', '}', '|', '\\':
			// Escape regex metacharacters
			buf.WriteByte('\\')
			buf.WriteByte(c)
		default:
			buf.WriteByte(c)
		}
		i++
	}

	buf.WriteString("$")
	pattern := buf.String()

	// Validate the regex
	if _, err := regexp.Compile(pattern); err != nil {
		return "", fmt.Errorf("invalid glob pattern %q: %w", glob, err)
	}

	return pattern, nil
}
