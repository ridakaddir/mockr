package generate

import (
	"context"
	"fmt"
)

// Options is the top-level configuration for the generator.
type Options struct {
	Spec   string // file path or URL to the OpenAPI spec
	OutDir string // output directory
	Format string // "toml" | "yaml" | "json"
	Split  bool   // split into one file per tag
}

// Result summarises what was generated.
type Result struct {
	SpecTitle   string
	SpecVersion string
	Operations  int
	Routes      int
	ConfigFiles []string
	StubFiles   []string
}

// Run loads the spec, parses operations, and writes the output.
func Run(opts Options) (*Result, error) {
	// Defaults.
	if opts.Format == "" {
		opts.Format = "toml"
	}
	// Accept "yml" as an alias for "yaml".
	if opts.Format == "yml" {
		opts.Format = "yaml"
	}
	// Validate format.
	switch opts.Format {
	case "toml", "yaml", "json":
		// valid
	default:
		return nil, fmt.Errorf("unsupported format %q — use toml, yaml, or json", opts.Format)
	}
	if opts.OutDir == "" {
		opts.OutDir = "mocks"
	}

	// Load spec.
	doc, err := LoadSpec(opts.Spec)
	if err != nil {
		return nil, fmt.Errorf("loading spec: %w", err)
	}

	// Validate leniently — many real-world specs have minor issues so we warn rather than fail.
	if err := doc.Validate(context.Background()); err != nil {
		fmt.Printf("warning: spec validation: %v\n", err)
	}

	title := ""
	version := ""
	if doc.Info != nil {
		title = doc.Info.Title
		version = doc.Info.Version
	}

	// Parse all operations.
	ops, err := ParseOperations(doc)
	if err != nil {
		return nil, fmt.Errorf("parsing operations: %w", err)
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("no operations found in spec")
	}

	// Group by tag.
	groups := groupByTag(ops)

	// Write files.
	writeOpts := WriteOptions{
		OutDir:  opts.OutDir,
		Format:  opts.Format,
		Split:   opts.Split,
		SpecSrc: opts.Spec,
	}

	written, err := Write(groups, writeOpts)
	if err != nil {
		return nil, fmt.Errorf("writing output: %w", err)
	}

	return &Result{
		SpecTitle:   title,
		SpecVersion: version,
		Operations:  len(ops),
		Routes:      len(ops),
		ConfigFiles: written.ConfigFiles,
		StubFiles:   written.StubFiles,
	}, nil
}

// groupByTag organises operations by their first tag.
func groupByTag(ops []Operation) map[string][]Operation {
	groups := make(map[string][]Operation)
	for _, op := range ops {
		groups[op.Tag] = append(groups[op.Tag], op)
	}
	return groups
}
