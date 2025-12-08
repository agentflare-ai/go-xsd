package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/agentflare-ai/go-xsd"
	"github.com/spf13/cobra"
)

type validateOptions struct {
	baseDir          string
	allowRemote      bool
	outputFormat     string
	exitOnViolation  bool
	color            bool
	includeDiagHints bool
}

func newValidateCommand(global *globalOptions) *cobra.Command {
	opts := &validateOptions{
		outputFormat:    "text",
		exitOnViolation: true,
		color:           true,
	}

	cmd := &cobra.Command{
		Use:   "validate <schema.xsd> <document.xml> [document.xml ...]",
		Short: "Validate XML documents against an XSD schema",
		Long:  "Load an XSD schema (with imports/includes) and validate one or more XML documents, returning structured diagnostics.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("expected schema path followed by at least one XML document path")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd.Context(), cmd.OutOrStdout(), args, opts, global)
		},
	}

	cmd.Flags().StringVar(&opts.baseDir, "base-dir", "", "Base directory for resolving schema imports/includes")
	cmd.Flags().BoolVar(&opts.allowRemote, "allow-remote", false, "Allow loading remote schemas over HTTP/S")
	cmd.Flags().StringVar(&opts.outputFormat, "format", opts.outputFormat, "Output format: text or json")
	cmd.Flags().BoolVar(&opts.exitOnViolation, "exit-on-violation", opts.exitOnViolation, "Exit with non-zero code if any violations are found")
	cmd.Flags().BoolVar(&opts.color, "color", opts.color, "Enable ANSI color in text output")
	cmd.Flags().BoolVar(&opts.includeDiagHints, "include-hints", true, "Include diagnostic hints in JSON output")

	return cmd
}

func runValidate(ctx context.Context, writer io.Writer, args []string, opts *validateOptions, global *globalOptions) error {
	schemaPath := args[0]
	documentPaths := args[1:]

	format := strings.ToLower(opts.outputFormat)
	if format != "text" && format != "json" {
		return fmt.Errorf("unsupported format %q (valid options: text, json)", opts.outputFormat)
	}

	reports, err := xsd.ValidateDocuments(ctx, xsd.ValidateOptions{
		SchemaPath:          schemaPath,
		DocumentPaths:       documentPaths,
		SchemaBaseDir:       opts.baseDir,
		AllowRemoteSchemas:  opts.allowRemote,
		StrictContentModels: global.strictContentModels,
		IncludeDiagnostics:  true,
		IncludeSource:       format == "text",
	})
	if err != nil {
		return err
	}

	switch format {
	case "json":
		return writeValidateJSON(writer, reports, opts.includeDiagHints)
	default:
		totalViolations, err := writeValidateText(writer, reports, opts.color)
		if err != nil {
			return err
		}
		if opts.exitOnViolation && totalViolations > 0 {
			return &exitError{code: 1, silent: true}
		}
		return nil
	}
}

func writeValidateJSON(w io.Writer, reports []xsd.DocumentValidationReport, includeHints bool) error {
	type jsonReport struct {
		Document    string           `json:"document"`
		Violations  []xsd.Violation  `json:"violations"`
		Diagnostics []xsd.Diagnostic `json:"diagnostics"`
	}

	payload := make([]jsonReport, 0, len(reports))
	for _, report := range reports {
		diag := report.Diagnostics
		if !includeHints {
			diag = make([]xsd.Diagnostic, len(report.Diagnostics))
			copy(diag, report.Diagnostics)
			for i := range diag {
				diag[i].Hints = nil
			}
		}
		payload = append(payload, jsonReport{
			Document:    report.DocumentPath,
			Violations:  report.Violations,
			Diagnostics: diag,
		})
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func writeValidateText(w io.Writer, reports []xsd.DocumentValidationReport, color bool) (int, error) {
	formatter := &xsd.ErrorFormatter{
		Color:           color,
		ShowFullElement: false,
		ContextLines:    2,
	}

	totalViolations := 0
	for _, report := range reports {
		if len(report.Diagnostics) == 0 {
			fmt.Fprintf(w, "✅ %s is valid\n", report.DocumentPath)
			continue
		}

		totalViolations += len(report.Diagnostics)
		fmt.Fprintf(w, "Found %d issue(s) in %s:\n\n", len(report.Diagnostics), report.DocumentPath)
		source := string(report.Source)
		for _, diag := range report.Diagnostics {
			fmt.Fprint(w, formatter.Format(diag, source))
			fmt.Fprint(w, "\n\n")
		}
	}

	return totalViolations, nil
}
