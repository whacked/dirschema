package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed meta.json
var metaSchemaJSON string

var (
	metaSchema     *jsonschema.Schema
	metaSchemaOnce sync.Once
	metaSchemaErr  error
)

// MetaSchema returns the compiled meta-schema for validating dirschema schemas.
// The schema is loaded and compiled lazily on first call.
func MetaSchema() (*jsonschema.Schema, error) {
	metaSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("meta.json", jsonStringReader(metaSchemaJSON)); err != nil {
			metaSchemaErr = fmt.Errorf("add meta schema: %w", err)
			return
		}
		metaSchema, metaSchemaErr = compiler.Compile("meta.json")
	})
	return metaSchema, metaSchemaErr
}

// ValidateSchema validates that the given schema conforms to dirschema conventions.
func ValidateSchema(schema map[string]any) error {
	meta, err := MetaSchema()
	if err != nil {
		return fmt.Errorf("load meta-schema: %w", err)
	}

	if err := meta.Validate(schema); err != nil {
		return fmt.Errorf("schema validation: %w", err)
	}
	return nil
}

type stringReader struct {
	s string
	i int
}

func jsonStringReader(s string) *stringReader {
	return &stringReader{s: s}
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

// MetaSchemaRaw returns the raw meta-schema JSON for inspection.
func MetaSchemaRaw() (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(metaSchemaJSON), &m); err != nil {
		return nil, err
	}
	return m, nil
}
