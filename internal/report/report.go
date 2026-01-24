package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"dirschema/internal/hydrate"
	"dirschema/internal/validate"
)

func FormatText(result validate.Result) string {
	if result.Valid {
		return ""
	}
	var b strings.Builder
	for i, err := range result.Errors {
		path := err.InstancePath
		if path == "" {
			path = "/"
		}
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "%s: %s (keyword=%s, schemaPath=%s, instancePath=%s)", path, err.Message, err.Keyword, err.SchemaPath, err.InstancePath)
	}
	return b.String()
}

func FormatJSON(result validate.Result) ([]byte, error) {
	return json.Marshal(result)
}

// HydrateResult combines hydration operations with validation result
type HydrateResult struct {
	Ops    []hydrate.Op     `json:"ops"`
	Valid  bool             `json:"valid"`
	Errors []validate.Item  `json:"errors,omitempty"`
}

func FormatHydrateJSON(plan hydrate.Plan, result validate.Result) ([]byte, error) {
	hr := HydrateResult{
		Ops:    plan.Ops,
		Valid:  result.Valid,
		Errors: result.Errors,
	}
	return json.Marshal(hr)
}
