package fswalk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Options struct {
	IncludeSize     bool
	IncludeSHA256   bool
	IncludeContent  bool
	MaxContentBytes int64
	SymlinkPolicy   SymlinkPolicy
}

type SymlinkPolicy int

const (
	SymlinkError  SymlinkPolicy = iota
	SymlinkIgnore
	SymlinkRecord
	SymlinkFollow
)

func Walk(root string, opts Options) (map[string]any, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory: %s", root)
	}
	return walkDirInner(root, opts, nil, make(map[string]bool))
}

func WalkWithSchema(root string, opts Options, schema map[string]any) (map[string]any, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory: %s", root)
	}
	return walkDirInner(root, opts, schema, make(map[string]bool))
}

func walkDirInner(dir string, opts Options, schema map[string]any, visited map[string]bool) (map[string]any, error) {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve symlink %s: %w", dir, err)
	}
	if visited[realDir] {
		return nil, fmt.Errorf("symlink cycle detected: %s", dir)
	}
	visited[realDir] = true
	defer delete(visited, realDir)

	var schemaProps, schemaPatterns map[string]any
	if schema != nil {
		schemaProps, _ = schema["properties"].(map[string]any)
		schemaPatterns, _ = schema["patternProperties"].(map[string]any)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	out := make(map[string]any, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		full := filepath.Join(dir, name)

		if entry.Type()&fs.ModeSymlink != 0 {
			// Schema-guided handling first
			if schema != nil {
				handled, herr := handleSchemaSymlink(name, full, opts, schemaProps, schemaPatterns, visited, out)
				if herr != nil {
					return nil, herr
				}
				if handled {
					continue
				}
			}

			// SymlinkFollow: follow all symlinks (for export --follow-symlinks)
			if opts.SymlinkPolicy == SymlinkFollow {
				handled, ferr := handleFollowSymlink(name, full, opts, visited, out)
				if ferr != nil {
					return nil, ferr
				}
				if handled {
					continue
				}
			}

			// Existing fallback policies
			switch opts.SymlinkPolicy {
			case SymlinkIgnore:
				continue
			case SymlinkRecord:
				target, rerr := os.Readlink(full)
				if rerr != nil {
					return nil, fmt.Errorf("read symlink %s: %w", full, rerr)
				}
				out[name] = map[string]any{"symlink": target}
				continue
			default:
				return nil, fmt.Errorf("symlink not supported: %s", full)
			}
		}

		if entry.IsDir() {
			var childSchema map[string]any
			if cs, ok := schemaExpectsDir(name, schemaProps, schemaPatterns); ok {
				childSchema = cs
			}
			child, derr := walkDirInner(full, opts, childSchema, visited)
			if derr != nil {
				return nil, derr
			}
			out[name+"/"] = child
			continue
		}

		value, ferr := fileValue(full, opts)
		if ferr != nil {
			return nil, ferr
		}
		out[name] = value
	}

	return out, nil
}

// handleSchemaSymlink decides how to handle a symlink based on schema hints.
// Returns (true, nil) if handled, (false, nil) if not matched, or (false, err) on error.
func handleSchemaSymlink(name, full string, opts Options, schemaProps, schemaPatterns map[string]any, visited map[string]bool, out map[string]any) (bool, error) {
	// Check if schema expects a directory at name+"/"
	if childSchema, ok := schemaExpectsDir(name, schemaProps, schemaPatterns); ok {
		// Resolve the symlink and check it's a directory
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			return false, fmt.Errorf("resolve symlink %s: %w", full, err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return false, fmt.Errorf("stat symlink target %s: %w", full, err)
		}
		if !info.IsDir() {
			// Schema expects dir but target is file — fall through to policy
			return false, nil
		}
		child, err := walkDirInner(full, opts, childSchema, visited)
		if err != nil {
			return false, err
		}
		out[name+"/"] = child
		return true, nil
	}

	// Check if schema expects symlink metadata (has "symlink" property)
	if schemaExpectsSymlink(name, schemaProps, schemaPatterns) {
		target, err := os.Readlink(full)
		if err != nil {
			return false, fmt.Errorf("read symlink %s: %w", full, err)
		}
		out[name] = map[string]any{"symlink": target}
		return true, nil
	}

	// Check if schema expects a file (name without "/", no "symlink" property)
	if schemaExpectsFile(name, schemaProps, schemaPatterns) {
		// Resolve the symlink and treat as a regular file
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			return false, fmt.Errorf("resolve symlink %s: %w", full, err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return false, fmt.Errorf("stat symlink target %s: %w", full, err)
		}
		if info.IsDir() {
			// Symlink points to dir but schema expects file — fall through
			return false, nil
		}
		value, err := fileValue(resolved, opts)
		if err != nil {
			return false, err
		}
		out[name] = value
		return true, nil
	}

	return false, nil
}

// handleFollowSymlink follows a symlink regardless of schema (for export --follow-symlinks).
func handleFollowSymlink(name, full string, opts Options, visited map[string]bool, out map[string]any) (bool, error) {
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return false, fmt.Errorf("resolve symlink %s: %w", full, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return false, fmt.Errorf("stat symlink target %s: %w", full, err)
	}
	if info.IsDir() {
		child, derr := walkDirInner(full, opts, nil, visited)
		if derr != nil {
			return false, derr
		}
		out[name+"/"] = child
		return true, nil
	}
	value, ferr := fileValue(resolved, opts)
	if ferr != nil {
		return false, ferr
	}
	out[name] = value
	return true, nil
}

// schemaExpectsDir checks if the schema expects name+"/" as a directory entry.
// Returns the child schema and true if found.
func schemaExpectsDir(name string, schemaProps, schemaPatterns map[string]any) (map[string]any, bool) {
	dirKey := name + "/"

	// Check properties
	if schemaProps != nil {
		if raw, ok := schemaProps[dirKey]; ok {
			if cs, ok := raw.(map[string]any); ok {
				return cs, true
			}
			return nil, true
		}
	}

	// Check patternProperties
	for pattern, raw := range schemaPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(dirKey) {
			if cs, ok := raw.(map[string]any); ok {
				return cs, true
			}
			return nil, true
		}
	}

	return nil, false
}

// schemaExpectsSymlink checks if the schema expects a symlink metadata object for name.
func schemaExpectsSymlink(name string, schemaProps, schemaPatterns map[string]any) bool {
	if raw := schemaLookupFile(name, schemaProps, schemaPatterns); raw != nil {
		if props, ok := raw["properties"].(map[string]any); ok {
			if _, ok := props["symlink"]; ok {
				return true
			}
		}
	}
	return false
}

// schemaExpectsFile checks if the schema expects a regular file at name (not a symlink object, not a directory).
func schemaExpectsFile(name string, schemaProps, schemaPatterns map[string]any) bool {
	if raw := schemaLookupFile(name, schemaProps, schemaPatterns); raw != nil {
		// Has properties with "symlink" → not a plain file expectation
		if props, ok := raw["properties"].(map[string]any); ok {
			if _, ok := props["symlink"]; ok {
				return false
			}
		}
		return true
	}
	return false
}

// schemaLookupFile looks up a non-directory entry in schema properties/patternProperties.
func schemaLookupFile(name string, schemaProps, schemaPatterns map[string]any) map[string]any {
	// Don't match directory keys
	if strings.HasSuffix(name, "/") {
		return nil
	}

	if schemaProps != nil {
		if raw, ok := schemaProps[name]; ok {
			if cs, ok := raw.(map[string]any); ok {
				return cs
			}
		}
	}

	for pattern, raw := range schemaPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(name) {
			if cs, ok := raw.(map[string]any); ok {
				return cs
			}
		}
	}

	return nil
}

func fileValue(path string, opts Options) (any, error) {
	if !opts.IncludeSize && !opts.IncludeSHA256 && !opts.IncludeContent {
		return true, nil
	}

	attrs := map[string]any{}
	if opts.IncludeSize {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		attrs["size"] = info.Size()
	}

	if opts.IncludeSHA256 || opts.IncludeContent {
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if opts.MaxContentBytes > 0 && int64(len(contents)) > opts.MaxContentBytes {
			return nil, fmt.Errorf("content exceeds max bytes: %s", path)
		}
		if opts.IncludeSHA256 {
			sum := sha256.Sum256(contents)
			attrs["sha256"] = hex.EncodeToString(sum[:])
		}
		if opts.IncludeContent {
			attrs["content"] = string(contents)
		}
	}

	if len(attrs) == 0 {
		return true, nil
	}
	return attrs, nil
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
