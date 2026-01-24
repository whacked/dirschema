package report

import (
	"testing"

	"dirschema/internal/validate"
)

func TestFormatText(t *testing.T) {
	res := validate.Result{
		Valid: false,
		Errors: []validate.Item{
			{
				InstancePath: "/a",
				SchemaPath:   "/properties/a/type",
				Keyword:      "type",
				Message:      "expected string",
			},
			{
				InstancePath: "",
				SchemaPath:   "/required",
				Keyword:      "required",
				Message:      "missing required property 'b'",
			},
		},
	}

	got := FormatText(res)
	want := "/a: expected string (keyword=type, schemaPath=/properties/a/type, instancePath=/a)\n" +
		"/: missing required property 'b' (keyword=required, schemaPath=/required, instancePath=)"

	if got != want {
		t.Fatalf("unexpected output:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatTextValid(t *testing.T) {
	res := validate.Result{Valid: true}
	if got := FormatText(res); got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
}
