# AGENTS.md

This repository is maintained by a **multi-agent** team (humans + coding agents).
This file defines invariant agent policies and project-specific conventions.

---

## 0) Project summary

**Project name**
- schedaf

**Purpose**
- validates and scaffolds a given directory against a schema-defined structure

**Project type**
- go-based CLI tool

**Primary stack**
- go

**Target platforms**
- mac/linux main, windows option should be kept open

---

## 1) Prime directive

Agents must optimize for:
- **Correctness**
- **Reproducibility**
- **Traceability**
- **Minimal diffs** (avoid drive-by refactors)

Agents should prefer small, reviewable changes over broad rewrites.
If something is uncertain, verify or state the assumption explicitly.

---

## 2) Development environment

### Nix-based workflow (MANDATORY)

- The development environment is defined using **`shell.nix`**.
- All work is expected to be performed inside `nix-shell` (or an equivalent workflow).
- **All dependency changes must be made by editing `shell.nix`.**

### Dependency policy

Adding a dependency is allowed only if:
- it exists in **nixpkgs**, OR
- it can be encapsulated into a **single Nix expression** and added to `buildInputs`.

Tooling conventions:
- Prefer `rg` over `grep`.
- Do not assume `tree` is available.

---

## 3) Repository organization

<TBD>

---

## 4) Validation philosophy

This repository treats **schemas and validators as authoritative**.

Key principles:
- All committed structured artifacts are **assumed to conform** to their schemas.
- Validator commands **must exist** and **must succeed** when run.
- What is optional is **when** a human runs validation, not whether validation is correct.

Implications:
- Validators may be run manually, via Makefile targets, via pre-commit hooks, or in CI.
- Any transformer that changes data shape must update schemas and ensure validation passes.

---

## 5) Before changing code

For any non-trivial change, agents must record a short plan in the commit body or PR description:

- **Goal**
- **Non-goals**
- **Approach**
- **Verification plan** (what commands were run)

If assumptions are made, they must be stated.

---

## 6) Commit policy (MANDATORY)

### 6.1 Subject line format

Every commit **must** use the following prefix:

[ai] <short summary>

Examples:
- `[ai] Fix incorrect schema normalization`
- `[ai] Add benchmark harness for new tool`
- `[ai] Refactor adapter invocation logic`

---

### 6.2 Co-authorship trailer (MANDATORY)

Claude Code automatically appends a `Co-authored-by` trailer.

- **If you are Claude Code:** do nothing extra.
- **If you are NOT Claude Code:** you **must** add a trailer manually at the end of the commit body.

Use one of:

Preferred:

Co-authored-by: <Agent Name> agent@example.invalid

Fallback:

Co-authored-by: AI Agent ai-agent@local.invalid


Notes:
- Use `.invalid` to avoid leaking personal email addresses.
- One trailer per agent is acceptable.

---

## 7) Commit body templates

Use the template appropriate to the change.

### Bugfix
**Problem**  
**Symptoms**  
**Reproduction**  
**Root cause**  
**Fix strategy**  
**Solution**  
**Resolution / Verification**  
**Impact**

---

### Feature
**Need / Motivation**  
**Design / Reasoning**  
**Implementation**  
**Verification**  
**Impact**

---

### Structural / methodology change
**Question**  
**Method**  
**Constraints / invariants**  
**Verification**  
**Impact**

---

## 8) Multi-agent coordination

Unless otherwise specified:
- Agents may work independently.
- Keep changes self-contained.
- Avoid overlapping scope.

If handoff is needed, include:
- current state
- commands run
- next steps
- known caveats

---

## 9) When uncertain

- Prefer a small validating experiment.
- If verification is not possible, state uncertainty explicitly.
- Propose a follow-up step rather than guessing.

---

## 10) Invariants checklist

- `[ai]` commit prefix
- Co-authored-by requirement
- `shell.nix` as the sole dependency authority
- Schemas define truth; validators must pass
- Deterministic, reproducible artifacts

