package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-jsonnet"
	"gopkg.in/yaml.v3"
)

type Kind int

const (
	KindUnknown Kind = iota
	KindDSL
	KindSchema
)

type Loaded struct {
	JSON []byte
}

func Load(path string) (Loaded, error) {
	ext := strings.ToLower(filepath.Ext(path))
	contents, err := os.ReadFile(path)
	if err != nil {
		return Loaded{}, err
	}

	switch ext {
	case ".json":
		if err := validateJSON(contents); err != nil {
			return Loaded{}, err
		}
		return Loaded{JSON: contents}, nil
	case ".yaml", ".yml":
		return loadYAML(contents)
	case ".jsonnet":
		return loadJsonnet(path)
	default:
		return Loaded{}, fmt.Errorf("unsupported spec extension: %s", ext)
	}
}

func InferKind(root any) (Kind, error) {
	switch v := root.(type) {
	case []any:
		if len(v) == 0 {
			return KindUnknown, errors.New("empty spec")
		}
		return KindDSL, nil
	case map[string]any:
		if len(v) == 0 {
			return KindUnknown, errors.New("empty spec")
		}

		schemaKeys := 0
		entryKeys := 0
		for key := range v {
			if isSchemaKeyword(key) {
				schemaKeys++
				continue
			}
			entryKeys++
		}

		switch {
		case schemaKeys > 0 && entryKeys > 0:
			return KindUnknown, errors.New("ambiguous spec: mixed schema keywords and entry keys")
		case schemaKeys > 0:
			return KindSchema, nil
		case entryKeys > 0:
			return KindDSL, nil
		default:
			return KindUnknown, errors.New("unable to infer spec kind")
		}
	default:
		return KindUnknown, errors.New("unsupported spec shape")
	}
}

func validateJSON(raw []byte) error {
	var tmp any
	if err := json.Unmarshal(raw, &tmp); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func loadYAML(contents []byte) (Loaded, error) {
	var decoded any
	if err := yaml.Unmarshal(contents, &decoded); err != nil {
		return Loaded{}, fmt.Errorf("invalid yaml: %w", err)
	}

	normalized, err := normalizeYAML(decoded)
	if err != nil {
		return Loaded{}, err
	}

	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return Loaded{}, fmt.Errorf("yaml to json: %w", err)
	}
	return Loaded{JSON: jsonBytes}, nil
}

func loadJsonnet(path string) (Loaded, error) {
	vm := jsonnet.MakeVM()
	jsonStr, err := vm.EvaluateFile(path)
	if err != nil {
		return Loaded{}, fmt.Errorf("jsonnet eval: %w", err)
	}
	if err := validateJSON([]byte(jsonStr)); err != nil {
		return Loaded{}, err
	}
	return Loaded{JSON: []byte(jsonStr)}, nil
}

func normalizeYAML(value any) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			converted, err := normalizeYAML(child)
			if err != nil {
				return nil, err
			}
			out[key] = converted
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			keyStr, ok := key.(string)
			if !ok {
				return nil, fmt.Errorf("yaml map key is not string: %T", key)
			}
			converted, err := normalizeYAML(child)
			if err != nil {
				return nil, err
			}
			out[keyStr] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			converted, err := normalizeYAML(child)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	default:
		return v, nil
	}
}

func isSchemaKeyword(key string) bool {
	_, ok := schemaKeywords[key]
	return ok
}

var schemaKeywords = map[string]struct{}{
	"$schema":              {},
	"$id":                  {},
	"$ref":                 {},
	"$defs":                {},
	"definitions":          {},
	"type":                 {},
	"properties":           {},
	"patternProperties":    {},
	"additionalProperties": {},
	"required":             {},
	"items":                {},
	"allOf":                {},
	"anyOf":                {},
	"oneOf":                {},
	"not":                  {},
	"const":                {},
	"enum":                 {},
	"minimum":              {},
	"maximum":              {},
	"minLength":            {},
	"maxLength":            {},
	"pattern":              {},
	"description":          {},
	"title":                {},
}
