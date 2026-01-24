````md
# PLAN.md — dirschema (working title)

A single-binary Go CLI that:
1) **materializes a directory tree into JSON** (an `interface{}` / `any` object graph),
2) **validates it against JSON Schema** (with exhaustive error reporting),
3) optionally **hydrates** missing parts of the tree from defaults.

Unix philosophy: do one thing well, be scriptable, exit codes matter, output is machine-friendly.

---

## 0) Product principles

- **Single static-ish binary** (best effort): macOS + Linux first; Windows later if it doesn’t contort the design.
- **Deterministic**: same inputs ⇒ same outputs (including hashing behavior, ordering, normalization).
- **Schema is source-of-truth**: the filesystem is the instance; schema is the contract.
- **Exhaustive validation**: collect *all* violations (no short-circuit), report them consistently.
- **Minimal surface area**: avoid templating engines, watchers, interactive UI, complex “programming in config”.
- **Composable**: JSON output option, stable exit codes, `--format json|text`, no hidden state.

---

## 1) CLI overview (first cut)

Binary name suggestion: `dirschema` (placeholder).

### Commands

#### 1) Expand (DSL → JSON Schema)
```bash
dirschema spec.yaml --expand [--format json|yaml]
```

* Reads a DSL spec, expands to full JSON Schema, and prints it to stdout.
* If the input is already a full JSON Schema, it is printed as-is (after normalization).

#### 2) Validate (default mode)
```bash
dirschema spec.jsonnet [--root DIR] [--format text|json] [--print-instance] [--fail-fast=false]
````

* Exit `0` if valid, `1` if invalid, `2` for tool/config errors.
* `--print-instance` prints the derived instance JSON (to stdout) for debugging; validation output goes to stderr unless `--format json` requested.


#### 3) Hydrate

```bash
dirschema spec.yaml --hydrate [--root DIR] [--dry-run] [--force] [--format text|json]
```

* Validates the current tree as-is.
* If valid: exit `0` with no changes.
* `--dry-run` prints planned operations and exits with `0` if it *would* succeed.
* `--force` allows overwriting files only when schema says overwritable (default: do not overwrite).

### Global flags

* `--root DIR` (default `$PWD`)
* `--format text|json` for diagnostics
* `--quiet` (only errors)
* `--debug` (trace steps, include evaluation info such as which schema keywords triggered which errors)

---

## 2) Core data model

### 2.1 Instance: filesystem → JSON object

We need a **canonical mapping** from a directory tree to a JSON object that is stable and JSON Schema-friendly.

Directory entries in the spec MUST end in a slash. We normalize to `/` internally.

**Spec syntax (DSL) examples:**

* Directory ⇒ JSON object
* File ⇒ JSON object leaf representing file metadata (and optionally content constraints)

Simplest form:

```yaml
some-directory/:
  - file1
  - file2
  - subdirectory/:
      - file3
      - file4
```

Extended form:

```yaml
some-directory/:
  - file1.any:
      size: 1234
      sha256: abcd
  - file2.txt
  - subdirectory/:
      - file3.dat
      - file4.ini:
          content: |
            foo=bar
```

Verbose form (still DSL):

Represent the directory tree as nested objects **where directory entries are keys**, and file values are either:

* `true` (vanilla existence-only), or
* a **file descriptor object** (for content constraints), or
* a **directory descriptor object** (rarely needed; mostly implicit)

Example instance (derived from a walk, not user-authored):

```jsonnet
{
  "src/": {
    "main.go": { "size": 913, "sha256": "…" },
    "pkg/": {
      "types.go": { "size": 1200 }
    }
  },
  "README.md": { "size": 420 },
  "empty-directory/": {},
  "EMPTYFILE": true,
}
```

### 2.2 Two representations: vanilla vs complex

You explicitly want to support both:

* vanilla: directory as `{ "file.txt": true }`
* complex: leaf has attributes `{ "file.txt": { "size": {...} } }`

**Rule to disambiguate**

* If a leaf value is boolean `true` ⇒ existence-only file
* If a leaf value is object ⇒ file descriptor
* Directories are objects whose keys are entry names; *but then file descriptors are also objects.*

So we need a sentinel to distinguish directory objects vs file descriptor objects if we ever allow directory-level attributes.

**Decision (first cut): no directory descriptors.**

* A JSON object is always a directory node unless it appears in a position where the schema demands a file descriptor.
* In practice: schema determines it; instance doesn’t need a sentinel.

If later you want directory descriptors, add reserved key like `"$dir": {...attrs...}` while keeping children as siblings.

### 2.3 Walker output (instance shape)

The filesystem walker outputs the **instance object** that is validated by the compiled JSON Schema:

* Directories ⇒ JSON objects whose keys are entry names.
* Files ⇒ either `true` (existence-only) or a file descriptor object (size/sha256/content when requested by schema).

The walker should be **schema-guided**:

* Pre-walk the compiled schema to build a lookup table of which paths require `sha256`, `content`, or `size`.
* During walk, compute only the attributes requested by the schema.

The walker never outputs the DSL. It only outputs the instance shape shown above.

---

## 3) Spec formats and compilation

### Supported inputs

We accept **either**:

* a **compact DSL** (YAML/JSON/Jsonnet) that describes a directory tree and file constraints, **or**
* a **full JSON Schema** (verbatim).

The DSL is **deterministically expanded** into a JSON Schema before validation. This keeps the CLI simple while standing on JSON Schema semantics.
This expansion is core functionality and is exposed via the `--expand` CLI mode.

Inputs:

* `spec.json` (DSL or full schema)
* `spec.yaml` (DSL → JSON)
* `spec.jsonnet` (DSL → JSON)

### Pipeline

1. Read spec file
2. If yaml/jsonnet: render to JSON bytes
3. Parse JSON into `map[string]any`
4. Determine spec kind by **shape inference** at the top level (DSL vs full schema). If ambiguous, **hard error**.
5. If DSL: expand to full JSON Schema (deterministically)
6. Validate JSON Schema against a **baked-in meta-schema** (embedded in binary)
7. Use compiled JSON Schema to validate derived instance

### Embedded meta-schema

We embed a JSON Schema that validates “your directory-spec schema” shape.
This meta-schema ensures:

* it is a valid JSON Schema draft (whatever library supports),
* plus any house rules we impose (reserved keys, allowed keywords, etc.)

First cut: meta-schema validates:

* top-level is JSON Schema object
* forbids unknown keywords only if feasible (may be hard across drafts)
* enforces that root is `type: object` (directory root)

### DSL expansion rules (first cut)

The compact syntax is a porcelain layer. It maps deterministically to a full JSON Schema:

* A directory entry (`"name/"`) expands to an object schema for its children.
* A file entry (`"name.ext"`) expands to a file-leaf schema.
* `true` expands to “existence-only file”.
* A file descriptor object (e.g., `{size, sha256, content}`) expands to a schema that constrains those attributes.
* The DSL does **not** support advanced JSON Schema constructs (e.g., `patternProperties`, `oneOf`, `$ref`); those require the full schema form and are rejected in DSL mode.

We infer DSL vs full schema by the **top-level shape**:

* DSL: keys look like directory/file entries (e.g., `"foo/"`, `"bar.txt"`).
* Full schema: keys are JSON Schema keywords (e.g., `type`, `properties`, `required`).
* Ambiguity: if top-level mixes both styles, **error**.

**Ambiguity examples (hard error):**

* `{ "type": "object", "src/": { "main.go": true } }`
* `{ "properties": { "src/": {} }, "README.md": true }`

---

## 4) JSON Schema engine requirements

### 4.1 Exhaustive evaluation

We must report all violations, not just the first.

This has implications:

* Choose a validator library that can return a list of errors and ideally preserve error paths and keyword context.
* If the library short-circuits internally (e.g., `oneOf`), we may have to:

  * configure it to keep all errors, or
  * re-run evaluation per-branch to collect union, or
  * accept that some constructs are inherently “pick-one” (like `anyOf/oneOf`) and report per-branch failures as part of the error payload.

**Definition of exhaustive (first cut)**

* For ordinary constraints (required, type, properties, patternProperties, min/max, etc.): collect all.
* For `oneOf`: report:

  * if no branches match: include errors from all branches
  * if multiple branches match: report that condition
* For `anyOf`: if none match: include errors from all branches
* For `allOf`: include errors from all subschemas
* Do not stop after first `required` violation; list all missing required keys.

### 4.2 Draft support

Pick one JSON Schema draft and be consistent (draft-07 is often safest for Go libs; but this depends on the library).

First cut recommendation:

* support a single draft end-to-end (whichever best supported + stable),
* meta-schema is aligned with that draft.

---

## 5) Content-based validation scope (first cut)

We need a crisp line to avoid feature creep.

### 5.1 Supported file attributes in schema (v1)

* `exists` (implicit by presence / required)
* `size`:

  * exact: `{"const": 123}`
  * range: `{"minimum": 0, "maximum": 1000}` (as JSON Schema numeric constraints)
* `sha256` (string exact match or enum)
* `content`:

  * exact match string (small files only; enforce max size)
  * OR regex via `pattern` (text files only; UTF-8 decode required)
* `mode` (unix permissions): keep this as a possible future feature. Do not implement in v1; move to a "FUTURE" section.

  * numeric (e.g., 0644 expressed as integer)
  * optional; only on unix; on mac/linux this is meaningful.

### 5.2 Explicit non-goals (v1)

* Mtime/ctime/atime constraints
* DOS/Unix line endings checks (unless you later define it as a “text normalization mode”)
* Arbitrary encodings (UTF-16, Shift-JIS) — start with UTF-8 only
* “ReadOnly: true” as a semantic property beyond `mode` (can be derived later)
* Templating of generated content (cookiecutter-like)

### 5.3 Handling binary vs text

* `content` checks require decoding as UTF-8.
* If decode fails and schema asked for `content`/`pattern`, it’s a validation error.
* Hashing always works on bytes.

### 5.4 Performance guardrails

* Hashing large files can be expensive.
* Implement per-file policy:

  * only compute sha256 if schema requests it (detected by pre-walking the compiled schema and building a lookup table)
  * only read content if schema requests it
  * impose max bytes for `content` check unless overridden

---

## 6) Skeleton hydration

### 6.1 Behavior

* Read spec
* Derive current instance from filesystem under root
* Validate current instance **as-is**
  * If valid: exit (no hydration needed)
  * If invalid: continue to hydration step
* Compute missing paths from the spec vs current tree
* Hydrate **only what does not exist**, using the spec’s defaults
* Validate again and print any errors to stdout

**Important clarification:**

* We do **not** attempt “partial validation.” JSON Schema does not define such a mode.
* Hydrate is best-effort creation of missing paths, not a guarantee of post-hydration validity.
* Post-hydration validation may still fail due to schema requirements that cannot be satisfied by defaults alone (e.g., content hash mismatch, pattern-only constraints).

### 6.2 Defaults

We need a place in schema to express defaults for hydration.
JSON Schema has `default`, but it’s annotation, not validation.

**Convention (first cut):**

* For file leaf schemas:

  * `defaultContent`: string (write file with this content)
  * `defaultBytesBase64`: base64 for binary (optional; may be too much for v1)
* If neither specified: create empty file.

**Edge case to call out:**

* Defaults are annotations in JSON Schema, and multiple schemas can disagree.
  * Example: `oneOf` two file schemas with different defaults — hydrate can’t know which to pick unless validation decides a winner.
* Another concrete edge: if we implicitly create empty files, but the schema requires `content`/`pattern` or `size > 0`, hydration will succeed then immediate post-validate will fail. We should either require an explicit default for those cases or treat them as “cannot hydrate without explicit default.”

**If we hydrate from `content`:**

* Only safe when `content` is an exact string (not a regex/pattern).
* Doesn’t cover binary files, large files, or explicit `sha256`/`size` constraints unless we accept the exact content as the source of truth.
* Needs a clear max-size and encoding rule (UTF-8 only), and a policy for line endings so we don’t create content that immediately fails validation on another platform.

**Pattern properties rule (v1):**

* If a directory schema uses `patternProperties`, hydrate should **skip** those entries unless a default is explicitly provided.

For directories: created as needed.

### 6.3 Overwrite policy

* Never overwrite existing files by default.
* If `--force` and schema includes `overwritable: true` on that leaf, allow overwrite.

---

## 7) Schema patterns for filesystem validation

### 8.1 Existence checks

* Use `required` for fixed names.
* Use `patternProperties` for naming rules.

### 8.2 Distinguishing file vs directory

Since instance values are:

* directory: object with children
* file: object with file attributes OR boolean true

Schema can enforce:

* a file leaf is either `true` or an object with allowed keys `size/sha256/content/mode`.
* a directory is an object whose properties are child entries and those child schemas.

In practice, your directory schemas will be recursive and verbose; we should provide **schema helpers** (via Jsonnet) but the tool does not own templating.

---

## 8) Output and error reporting

### 9.1 Text format (human)

* One violation per line:

  * `PATH: message (keyword=..., schemaPath=..., instancePath=...)`
* Stable ordering: sort by instancePath then schemaPath.

### 9.2 JSON format (machine)

```json
{
  "valid": false,
  "errors": [
    {
      "instancePath": "/src/main.go/sha256",
      "schemaPath": "/properties/src/properties/main.go/properties/sha256/const",
      "keyword": "const",
      "message": "expected sha256 to equal …",
      "details": { "expected": "…", "actual": "…" }
    }
  ]
}
```

### 9.3 Exit codes

* 0: success (valid / hydrated / refactor done)
* 1: validation failure (including post-hydration or post-refactor validation)
* 2: operational/config error (invalid schema file, jsonnet render error, IO failure, permission denied, etc.)

---

## 9) Project structure (Go)

Suggested repo layout:

```
/cmd/dirschema/main.go
/internal/cli/...
/internal/spec/        (load yaml/json/jsonnet, meta-validate)
/internal/fswalk/      (traverse root, build instance tree)
/internal/instance/    (types + helpers, canonicalization)
/internal/validate/    (json schema validation glue, exhaustive error shaping)
/internal/hydrate/     (plan + apply ops)
/internal/refactor/    (plan + apply ops)
/internal/report/      (formatters text/json)
/schemas/              (embedded meta-schema JSON)
/testdata/             (fixtures)
```

---

## 10) Dependency selection (chosen)

### JSON Schema

* Use `github.com/santhosh-tekuri/jsonschema/v5`.
  * Supports draft-07 (plus newer drafts if we decide later).
  * Produces detailed errors with instance and keyword locations, which we can normalize into our error model.
  * Validates schemas against the meta-schema.

### Jsonnet

* Use `github.com/google/go-jsonnet`.

### YAML

* Use `gopkg.in/yaml.v3`.

**ADR:** still write a short ADR capturing the library choice and how we collect exhaustive errors in our output model.

---

## 11) Test strategy (must-have for v1)

### Fixture-based integration tests

Under `/testdata`:

* `simple_valid/` dir + `spec.json`
* `simple_invalid_missing/`
* `pattern_properties/`
* `content_hash/` with known sha256
* `hydrate_empty/` spec that produces expected tree
* `refactor_basic/` with source tree + expected output tree

Test harness runs binary in temp dirs and checks:

* exit codes
* json output matches expected
* files created correctly (hydrate)
* refactor results validated

### Unit tests

* fswalk determinism (ordering, symlink handling decision)
* hash/content reading behavior only when requested
* error normalization and sorting

---

## 12) Open questions (agents must lock for first cut)

1. **Symlinks**

   * Options: ignore, treat as file, follow (dangerous).
   * v1 recommendation: *do not follow*, represent symlink as validation error unless schema explicitly allows `symlink: true` (but that’s extra). Simplest: ignore symlinks and optionally warn.

2. **Hidden files**

   * Include by default (since this is validation), with schema controlling allowance.
   * Provide `--ignore-hidden` if needed later; avoid v1 unless required.

3. **Path normalization**

   * Always use forward-slash in instancePath regardless of OS (JSON Pointer style).

4. **File permissions portability**

   * On mac/linux, OK.
   * On Windows later, either omit `mode` or map best-effort.

5. **Schema ergonomics**

   * Expect Jsonnet users to build helpers; tool stays minimal.

---

## 13) Milestones and agent task breakdown

### Milestone A — MVP validate (must land first)

**Deliverable:** `dirschema validate` supports json/yaml/jsonnet specs, derives instance, validates, reports exhaustive errors.

Tasks:

* A1: CLI scaffolding + flag parsing + exit code conventions
* A2: Spec loader pipeline (yaml/jsonnet → json) + embedded meta-schema validation
* A3: Filesystem walker → instance tree
* A4: JSON Schema validator integration + exhaustive error collection + normalized error model
* A5: Report formatters (text/json)
* A6: Integration tests for validate

Acceptance:

* `dirschema validate` works on fixtures
* Errors are stable + exhaustive for typical keywords

### Milestone B — Hydrate (safe create)

**Deliverable:** `dirschema hydrate` creates missing structure using defaults/empty files.

Tasks:

* B1: Define schema convention for defaults (`defaultContent`, `overwritable`)
* B2: Build hydration plan (list of mkdir/write operations)
* B3: Apply plan (dry-run + force rules)
* B4: Post-hydration full validation
* B5: Tests

Acceptance:

* Dry-run shows plan
* No overwrites by default
* Post-hydration validate passes on fixtures

---

## 14) Non-goals (repeat, to prevent drift)

* Cookiecutter-like templating or variable substitution
* Dynamic rule engines in bridge mappings
* Watching filesystem / incremental re-validation
* Network access or remote backends
* “Fix my tree automatically” beyond hydration defaults
* Refactor/bridge workflows (explicitly deferred for v1)

---

## 15) Naming and repository hygiene (recommended)

* Keep the binary name and module name aligned (`dirschema`).
* Provide `--version` and `--help` with concise examples.
* Provide manpage-style docs later; v1 just `README.md` + this PLAN.

---

## 16) Definition of done (v1)

* `validate` and `hydrate` commands implemented as above.
* Deterministic output, stable errors, clear exit codes.
* Works on macOS + Linux CI.
* Testdata fixtures cover core behaviors.
* ADR for JSON Schema library + “exhaustive errors” approach included in repo.

```
