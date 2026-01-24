package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"dirschema/internal/expand"
	"dirschema/internal/fswalk"
	"dirschema/internal/hydrate"
	"dirschema/internal/instance"
	"dirschema/internal/report"
	"dirschema/internal/spec"
	"dirschema/internal/validate"
)

const (
	ExitSuccess     = 0
	ExitValidation  = 1
	ExitConfigError = 2
)

const Version = "dev"

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return ExitConfigError
	}

	switch args[0] {
	case "expand":
		return runExpand(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "hydrate":
		return runHydrate(args[1:], stdout, stderr)
	case "version", "--version":
		fmt.Fprintln(stdout, Version)
		return ExitSuccess
	case "-h", "--help", "help":
		printUsage(stdout)
		return ExitSuccess
	default:
		return runValidate(args, stdout, stderr)
	}
}

func runExpand(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("expand", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return ExitConfigError
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "expand requires a single spec path")
		return ExitConfigError
	}

	specPath := fs.Arg(0)
	loaded, err := spec.Load(specPath)
	if err != nil {
		fmt.Fprintf(stderr, "failed to load spec: %v\n", err)
		return ExitConfigError
	}

	root, err := decodeRoot(loaded.JSON)
	if err != nil {
		fmt.Fprintf(stderr, "failed to parse spec json: %v\n", err)
		return ExitConfigError
	}

	kind, err := spec.InferKind(root)
	if err != nil {
		fmt.Fprintf(stderr, "failed to infer spec kind: %v\n", err)
		return ExitConfigError
	}

	var output map[string]any
	switch kind {
	case spec.KindSchema:
		asMap, ok := root.(map[string]any)
		if !ok {
			fmt.Fprintln(stderr, "schema must be a JSON object")
			return ExitConfigError
		}
		output = asMap
	case spec.KindDSL:
		output, err = expand.ExpandDSL(root)
		if err != nil {
			fmt.Fprintf(stderr, "failed to expand DSL: %v\n", err)
			return ExitConfigError
		}
	default:
		fmt.Fprintln(stderr, "unable to infer spec kind")
		return ExitConfigError
	}

	encoded, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(stderr, "failed to encode schema: %v\n", err)
		return ExitConfigError
	}
	if _, err := stdout.Write(encoded); err != nil {
		fmt.Fprintf(stderr, "failed to write output: %v\n", err)
		return ExitConfigError
	}
	if _, err := stdout.Write([]byte("\n")); err != nil {
		fmt.Fprintf(stderr, "failed to write output: %v\n", err)
		return ExitConfigError
	}

	return ExitSuccess
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rootFlag := fs.String("root", "", "root directory")
	formatFlag := fs.String("format", "text", "output format (text|json)")
	printInstance := fs.Bool("print-instance", false, "print derived instance JSON")
	if err := fs.Parse(args); err != nil {
		return ExitConfigError
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "validate requires a single spec path")
		return ExitConfigError
	}
	if *formatFlag != "text" && *formatFlag != "json" {
		fmt.Fprintln(stderr, "invalid --format (must be text or json)")
		return ExitConfigError
	}
	if *printInstance && *formatFlag == "json" {
		fmt.Fprintln(stderr, "--print-instance cannot be used with --format json")
		return ExitConfigError
	}

	specPath := fs.Arg(0)
	schema, err := loadSchema(specPath)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitConfigError
	}

	root := *rootFlag
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "failed to get working directory: %v\n", err)
			return ExitConfigError
		}
	}
	root, err = filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve root: %v\n", err)
		return ExitConfigError
	}

	walkOpts := instance.ScanAttributes(schema)
	inst, err := fswalk.Walk(root, walkOpts)
	if err != nil {
		fmt.Fprintf(stderr, "failed to walk filesystem: %v\n", err)
		return ExitConfigError
	}

	if *printInstance {
		if err := writeJSON(stdout, inst); err != nil {
			fmt.Fprintf(stderr, "failed to write instance: %v\n", err)
			return ExitConfigError
		}
	}

	result, err := validate.Validate(schema, inst)
	if err != nil {
		fmt.Fprintf(stderr, "validation failed: %v\n", err)
		return ExitConfigError
	}

	if result.Valid {
		return ExitSuccess
	}

	if *formatFlag == "json" {
		payload, err := report.FormatJSON(result)
		if err != nil {
			fmt.Fprintf(stderr, "failed to encode report: %v\n", err)
			return ExitConfigError
		}
		if _, err := stdout.Write(payload); err != nil {
			fmt.Fprintf(stderr, "failed to write report: %v\n", err)
			return ExitConfigError
		}
		if _, err := stdout.Write([]byte("\n")); err != nil {
			fmt.Fprintf(stderr, "failed to write report: %v\n", err)
			return ExitConfigError
		}
	} else {
		text := report.FormatText(result)
		if text != "" {
			if _, err := stderr.Write([]byte(text + "\n")); err != nil {
				fmt.Fprintf(stderr, "failed to write report: %v\n", err)
				return ExitConfigError
			}
		}
	}

	return ExitValidation
}

func runExport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rootFlag := fs.String("root", "", "root directory")
	if err := fs.Parse(args); err != nil {
		return ExitConfigError
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "export does not accept positional arguments")
		return ExitConfigError
	}

	root := *rootFlag
	var err error
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "failed to get working directory: %v\n", err)
			return ExitConfigError
		}
	}
	root, err = filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve root: %v\n", err)
		return ExitConfigError
	}

	inst, err := fswalk.Walk(root, fswalk.Options{SymlinkPolicy: fswalk.SymlinkRecord})
	if err != nil {
		fmt.Fprintf(stderr, "failed to walk filesystem: %v\n", err)
		return ExitConfigError
	}

	list := expand.FormatListDSL(inst)
	if err := writeJSON(stdout, list); err != nil {
		fmt.Fprintf(stderr, "failed to write export: %v\n", err)
		return ExitConfigError
	}
	return ExitSuccess
}

func runHydrate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hydrate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	rootFlag := fs.String("root", "", "root directory")
	formatFlag := fs.String("format", "text", "output format (text|json)")
	dryRun := fs.Bool("dry-run", false, "print planned operations without applying")
	if err := fs.Parse(args); err != nil {
		return ExitConfigError
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "hydrate requires a single spec path")
		return ExitConfigError
	}
	if *formatFlag != "text" && *formatFlag != "json" {
		fmt.Fprintln(stderr, "invalid --format (must be text or json)")
		return ExitConfigError
	}

	specPath := fs.Arg(0)
	schema, err := loadSchema(specPath)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return ExitConfigError
	}

	root := *rootFlag
	if root == "" {
		root, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "failed to get working directory: %v\n", err)
			return ExitConfigError
		}
	}
	root, err = filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve root: %v\n", err)
		return ExitConfigError
	}

	plan, err := hydrate.BuildPlan(schema, root)
	if err != nil {
		fmt.Fprintf(stderr, "failed to build hydrate plan: %v\n", err)
		return ExitConfigError
	}

	// Text mode: always print ops to stdout
	if *formatFlag == "text" {
		text := hydrate.FormatOpsText(plan)
		if text != "" {
			if _, err := stdout.Write([]byte(text + "\n")); err != nil {
				fmt.Fprintf(stderr, "failed to write plan: %v\n", err)
				return ExitConfigError
			}
		}
	}

	// Dry-run: just print plan and exit
	if *dryRun {
		if *formatFlag == "json" {
			payload, err := hydrate.FormatOpsJSON(plan)
			if err != nil {
				fmt.Fprintf(stderr, "failed to encode plan: %v\n", err)
				return ExitConfigError
			}
			if _, err := stdout.Write(payload); err != nil {
				fmt.Fprintf(stderr, "failed to write plan: %v\n", err)
				return ExitConfigError
			}
			if _, err := stdout.Write([]byte("\n")); err != nil {
				fmt.Fprintf(stderr, "failed to write plan: %v\n", err)
				return ExitConfigError
			}
		}
		return ExitSuccess
	}

	if err := hydrate.Apply(plan, hydrate.ApplyOptions{}); err != nil {
		fmt.Fprintf(stderr, "failed to apply hydrate plan: %v\n", err)
		return ExitConfigError
	}

	walkOpts := instance.ScanAttributes(schema)
	inst, err := fswalk.Walk(root, walkOpts)
	if err != nil {
		fmt.Fprintf(stderr, "failed to walk filesystem: %v\n", err)
		return ExitConfigError
	}
	result, err := validate.Validate(schema, inst)
	if err != nil {
		fmt.Fprintf(stderr, "validation failed: %v\n", err)
		return ExitConfigError
	}

	if *formatFlag == "json" {
		payload, err := report.FormatHydrateJSON(plan, result)
		if err != nil {
			fmt.Fprintf(stderr, "failed to encode report: %v\n", err)
			return ExitConfigError
		}
		if _, err := stdout.Write(payload); err != nil {
			fmt.Fprintf(stderr, "failed to write report: %v\n", err)
			return ExitConfigError
		}
		if _, err := stdout.Write([]byte("\n")); err != nil {
			fmt.Fprintf(stderr, "failed to write report: %v\n", err)
			return ExitConfigError
		}
	} else if !result.Valid {
		text := report.FormatText(result)
		if text != "" {
			if _, err := stderr.Write([]byte(text + "\n")); err != nil {
				fmt.Fprintf(stderr, "failed to write report: %v\n", err)
				return ExitConfigError
			}
		}
	}

	if result.Valid {
		return ExitSuccess
	}
	return ExitValidation
}

func decodeRoot(raw []byte) (any, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	return root, nil
}

func loadSchema(path string) (map[string]any, error) {
	loaded, err := spec.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load spec: %w", err)
	}
	root, err := decodeRoot(loaded.JSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec json: %w", err)
	}
	kind, err := spec.InferKind(root)
	if err != nil {
		return nil, fmt.Errorf("failed to infer spec kind: %w", err)
	}
	switch kind {
	case spec.KindSchema:
		asMap, ok := root.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schema must be an object")
		}
		return asMap, nil
	case spec.KindDSL:
		schema, err := expand.ExpandDSL(root)
		if err != nil {
			return nil, fmt.Errorf("failed to expand DSL: %w", err)
		}
		return schema, nil
	default:
		return nil, fmt.Errorf("unable to infer spec kind")
	}
}

func writeJSON(w io.Writer, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := w.Write(encoded); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, "usage: dirschema <spec> [--root DIR] [--format text|json] [--print-instance]\n\ncommands:\n  expand <spec>\n  export [--root DIR]\n  validate <spec> [--root DIR] [--format text|json] [--print-instance]\n  hydrate <spec> [--root DIR] [--format text|json] [--dry-run]\n  version\n")
}

func Main() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}
