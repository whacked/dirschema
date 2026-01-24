package fswalk

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	SymlinkError SymlinkPolicy = iota
	SymlinkIgnore
	SymlinkRecord
)

func Walk(root string, opts Options) (map[string]any, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory: %s", root)
	}
	return walkDir(root, opts)
}

func walkDir(dir string, opts Options) (map[string]any, error) {
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
			switch opts.SymlinkPolicy {
			case SymlinkIgnore:
				continue
			case SymlinkRecord:
				target, err := os.Readlink(full)
				if err != nil {
					return nil, fmt.Errorf("read symlink %s: %w", full, err)
				}
				out[name] = map[string]any{"symlink": target}
				continue
			default:
				return nil, fmt.Errorf("symlink not supported: %s", full)
			}
		}

		if entry.IsDir() {
			child, err := walkDir(full, opts)
			if err != nil {
				return nil, err
			}
			out[name+"/"] = child
			continue
		}

		value, err := fileValue(full, opts)
		if err != nil {
			return nil, err
		}
		out[name] = value
	}

	return out, nil
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
