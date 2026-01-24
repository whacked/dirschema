# dirschema

A Go-based CLI that validates and scaffolds a directory tree against a schema-defined structure.

## Premise
`dirschema` treats the filesystem as the instance and a JSON Schema (or compact DSL that expands to JSON Schema) as the contract. It can:

- Expand a compact DSL to JSON Schema.
- Validate a directory tree against a schema with exhaustive-style error reporting.
- Hydrate missing required paths (dirs/files) using defaults.

## Usage

### Validate (default)

```bash
dirschema spec.json
```

### Expand (DSL -> JSON Schema)

```bash
dirschema expand spec.yaml
```

### Export (filesystem -> simplified DSL)

```bash
dirschema export --root /path/to/tree
```

Outputs the simplest DSL representation in list form (files as strings, directories as `dir/: [ ... ]`). A list-form DSL is also supported when authoring specs (see below).
Symlinks are emitted as `{ "symlink": "target" }`.

### Validate (explicit)

```bash
dirschema validate spec.json --root /path/to/tree
```

- Exit codes: 0 valid, 1 invalid, 2 config/IO error.
- `--format json` for machine output.
- `--print-instance` to emit derived instance JSON.

### Hydrate

```bash
dirschema hydrate spec.json --root /path/to/tree
```

- Creates missing required files/dirs; existing paths are never modified.
- `--dry-run` prints planned operations without changes.

### Version

```bash
dirschema version
```

## Schema and DSL notes

- Files are modeled as `true` (existence-only) or a descriptor object (future: size/sha256/content constraints).
- Directories are JSON objects keyed by entry names, with directory entries ending in `/`.
- The DSL is a deterministic expansion to JSON Schema; advanced JSON Schema features (e.g. `oneOf`, `patternProperties`) require full schema form.
- Symlinks are represented as file descriptors: `"link.txt": { "symlink": "target.txt" }`.
- DSL list form is supported:\n+\n+```yaml\n+src/:\n+  - main.go\n+  - link:\n+      symlink: main.go\n+```\n+\n+List entries must be either strings (file names) or single-key maps; duplicate names are rejected case-insensitively.

## Development

This repo is intended to be worked on inside `nix-shell`.

```bash
nix-shell

go test ./...
```

### Build

```bash
make build
```

### Project layout

```
cmd/dirschema/            CLI entrypoint
internal/cli/             command wiring
internal/spec/            spec loading + DSL/schema inference
internal/expand/          DSL -> JSON Schema expansion
internal/fswalk/          filesystem -> instance
internal/instance/        instance helpers (schema-guided attributes)
internal/validate/        JSON Schema validation + error normalization
internal/report/          text/json reporting
internal/hydrate/         hydrate plan/apply
internal/integration/     fixture-based integration tests
schemas/                  (reserved for meta-schema)
```

## Prior art

- https://github.com/mozilla-releng/dirschema
- https://materials-data-science-and-informatics.github.io/dirschema/main/quickstart/#getting-started

These tools validate and scaffold directory layouts using schema-like specifications. `dirschema` differs in a few ways:

- It treats full JSON Schema as the authoritative format, with a compact DSL as a convenience layer.
- It targets exhaustive error reporting and stable, machine-friendly output formats.
- It provides explicit CLI subcommands for expansion, validation, and hydration rather than a single multi-mode command.
- It emphasizes deterministic instance shaping and schema-guided attribute collection.
