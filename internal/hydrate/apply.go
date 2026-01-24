package hydrate

import (
	"fmt"
	"os"
	"path/filepath"
)

type ApplyOptions struct {
	DryRun bool
}

func Apply(plan Plan, opts ApplyOptions) error {
	for _, op := range plan.Ops {
		switch op.Kind {
		case OpMkdir:
			if opts.DryRun {
				continue
			}
			if err := os.MkdirAll(op.Path, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", op.RelPath, err)
			}
		case OpWriteFile:
			if err := applyWrite(op, opts); err != nil {
				return err
			}
		case OpSymlink:
			if err := applySymlink(op, opts); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown op: %s", op.Kind)
		}
	}
	return nil
}

func applyWrite(op Op, opts ApplyOptions) error {
	if opts.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(op.Path), 0o755); err != nil {
		return fmt.Errorf("mkdir for file %s: %w", op.RelPath, err)
	}

	content := []byte{}
	if op.Content != nil {
		content = []byte(*op.Content)
	}
	if err := os.WriteFile(op.Path, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", op.RelPath, err)
	}
	return nil
}

func applySymlink(op Op, opts ApplyOptions) error {
	if opts.DryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(op.Path), 0o755); err != nil {
		return fmt.Errorf("mkdir for symlink %s: %w", op.RelPath, err)
	}
	if err := os.Symlink(op.Target, op.Path); err != nil {
		return fmt.Errorf("symlink %s: %w", op.RelPath, err)
	}
	return nil
}
