package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/agentflare-ai/go-xsd"
	"github.com/spf13/cobra"
)

type docOptions struct {
	baseDir     string
	allowRemote bool
	outPath     string
	title       string
	includeTOC  bool
	sections    []string
	format      string
	maxDepth    int
}

func newDocCommand(global *globalOptions) *cobra.Command {
	opts := &docOptions{
		includeTOC: true,
		format:     string(xsd.DocFormatMarkdown),
		maxDepth:   3,
		outPath:    "-",
	}

	cmd := &cobra.Command{
		Use:   "doc <schema.xsd>",
		Short: "Generate schema documentation in Markdown (and future formats)",
		Long:  "Render human-friendly documentation for an XSD schema, producing Markdown by default with hooks for future formats like HTML or AsciiDoc.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("expected exactly one schema path")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoc(cmd.Context(), cmd.OutOrStdout(), args[0], opts, global)
		},
	}

	cmd.Flags().StringVar(&opts.baseDir, "base-dir", "", "Base directory for resolving imports/includes")
	cmd.Flags().BoolVar(&opts.allowRemote, "allow-remote", false, "Allow loading remote schemas over HTTP/S")
	cmd.Flags().StringVar(&opts.outPath, "out", opts.outPath, "Output path ('-' for stdout)")
	cmd.Flags().StringVar(&opts.title, "title", "", "Override documentation title")
	cmd.Flags().BoolVar(&opts.includeTOC, "toc", opts.includeTOC, "Include a table of contents in supported formats")
	cmd.Flags().StringSliceVar(&opts.sections, "section", nil, "Limit documentation to specific sections (overview,elements,types,constraints)")
	cmd.Flags().StringVar(&opts.format, "format", opts.format, "Output format (markdown, html, asciidoc, json)")
	cmd.Flags().IntVar(&opts.maxDepth, "max-depth", opts.maxDepth, "Maximum nesting depth for element/type traversal")

	return cmd
}

func runDoc(ctx context.Context, stdout io.Writer, schemaPath string, opts *docOptions, global *globalOptions) error {
	format := xsd.DocFormat(strings.ToLower(opts.format))
	if _, ok := xsd.SupportedDocFormats()[format]; !ok {
		return fmt.Errorf("unsupported documentation format %q", opts.format)
	}

	schema, err := xsd.LoadSchemaWithOptions(ctx, xsd.SchemaLoadOptions{
		SchemaPath:         schemaPath,
		BaseDir:            opts.baseDir,
		AllowRemoteSchemas: opts.allowRemote,
	})
	if err != nil {
		return err
	}

	schema.StrictContentModel = global.strictContentModels

	doc, err := xsd.GenerateDoc(schema, format, xsd.DocOptions{
		Title:      opts.title,
		IncludeTOC: opts.includeTOC,
		Sections:   opts.sections,
		MaxDepth:   opts.maxDepth,
	})
	if err != nil {
		return err
	}

	var writer io.Writer = stdout
	var file *os.File
	if opts.outPath != "-" {
		file, err = os.OpenFile(opts.outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", opts.outPath, err)
		}
		defer func() { _ = file.Close() }()
		writer = file
	}

	if _, err := fmt.Fprint(writer, doc); err != nil {
		return fmt.Errorf("failed to write documentation: %w", err)
	}

	if opts.outPath != "-" {
		fmt.Fprintf(stdout, "Documentation written to %s\n", opts.outPath)
	}

	return nil
}
