# ADR 0001: JSON Schema library and exhaustive errors

## Need / Motivation
We need JSON Schema validation with detailed error paths and the ability to collect multiple errors per evaluation.

## Decision
Use `github.com/santhosh-tekuri/jsonschema/v5` as the JSON Schema engine.

## Reasoning
- Draft support is stable for draft-07, which aligns with our initial scope.
- Validation errors include instance locations and keyword locations, enabling normalization.
- The library exposes nested causes, which we flatten to provide exhaustive-style reporting.

## Exhaustive errors approach
We flatten `ValidationError` causes into a list of leaf errors and sort them by instance path and schema path. For `oneOf/anyOf/allOf`, the library provides nested causes; we surface all leaf failures to align with the “no short-circuit” policy where feasible.

## Consequences
- We normalize errors to our own `validate.Item` type.
- Some constructs are inherently ambiguous (`oneOf`), so we surface library-provided branch failures rather than re-validating branches.

## Status
Accepted
