package xsd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// SchemaLoadOptions configures how schemas should be resolved from disk/remote locations.
type SchemaLoadOptions struct {
	SchemaPath         string
	BaseDir            string
	AllowRemoteSchemas bool
}

// ValidateOptions controls schema + document validation runs used by both the CLI and Go consumers.
type ValidateOptions struct {
	SchemaPath          string
	DocumentPaths       []string
	SchemaBaseDir       string
	AllowRemoteSchemas  bool
	StrictContentModels bool
	IncludeDiagnostics  bool
	IncludeSource       bool
}

// DocumentValidationReport captures validation results for a single XML document.
type DocumentValidationReport struct {
	DocumentPath string
	Source       []byte
	Violations   []Violation
	Diagnostics  []Diagnostic
}

// LoadSchemaWithOptions loads an XSD schema (with imports/includes) using the same logic as the CLI.
func LoadSchemaWithOptions(ctx context.Context, opts SchemaLoadOptions) (*Schema, error) {
	if opts.SchemaPath == "" {
		return nil, fmt.Errorf("schema path is required")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if isRemote(opts.SchemaPath) && !opts.AllowRemoteSchemas {
		return nil, fmt.Errorf("remote schema loading disabled: %s", opts.SchemaPath)
	}

	location := opts.SchemaPath
	var err error
	if !isRemote(location) {
		location, err = resolveSchemaLocation(location, opts.BaseDir)
		if err != nil {
			return nil, err
		}
	}

	baseDir := opts.BaseDir
	if !isRemote(location) {
		if !filepath.IsAbs(location) {
			abs, err := filepath.Abs(location)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve schema path: %w", err)
			}
			location = abs
		}
		if baseDir == "" {
			baseDir = filepath.Dir(location)
		}
	}

	loader, err := NewSchemaLoader(SchemaLoaderConfig{BaseDir: baseDir})
	if err != nil {
		return nil, err
	}

	if !opts.AllowRemoteSchemas {
		loader.httpClient = &http.Client{
			Transport: disallowRemoteTransport{},
		}
	}

	return loader.LoadSchemaWithImports(location)
}

// ValidateDocuments validates one or more XML documents against a schema.
func ValidateDocuments(ctx context.Context, opts ValidateOptions) ([]DocumentValidationReport, error) {
	if opts.SchemaPath == "" {
		return nil, fmt.Errorf("schema path is required")
	}
	if len(opts.DocumentPaths) == 0 {
		return nil, fmt.Errorf("at least one XML document path is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	schema, err := LoadSchemaWithOptions(ctx, SchemaLoadOptions{
		SchemaPath:         opts.SchemaPath,
		BaseDir:            opts.SchemaBaseDir,
		AllowRemoteSchemas: opts.AllowRemoteSchemas,
	})
	if err != nil {
		return nil, err
	}

	schema.StrictContentModel = opts.StrictContentModels

	reports := make([]DocumentValidationReport, 0, len(opts.DocumentPaths))

	for _, docPath := range opts.DocumentPaths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		source, err := os.ReadFile(docPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", docPath, err)
		}

		decoder := xmldom.NewDecoderFromBytes(source)
		doc, err := decoder.Decode()
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", docPath, err)
		}

		validator := NewValidator(schema)
		violations := validator.Validate(doc)

		report := DocumentValidationReport{
			DocumentPath: docPath,
			Violations:   violations,
		}

		if opts.IncludeSource {
			report.Source = append([]byte(nil), source...)
		}

		if opts.IncludeDiagnostics {
			converter := NewDiagnosticConverter(docPath, string(source))
			report.Diagnostics = converter.Convert(violations)
		}

		reports = append(reports, report)
	}

	return reports, nil
}

type disallowRemoteTransport struct{}

func (disallowRemoteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("remote schema loading disabled for %s", req.URL.String())
}

func isRemote(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func resolveSchemaLocation(inputPath, baseDir string) (string, error) {
	path := inputPath
	if baseDir != "" && !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return findDefaultSchema(path)
	}

	if err != nil {
		return "", err
	}

	return path, nil
}

func findDefaultSchema(dir string) (string, error) {
	dir = filepath.Clean(dir)
	repoName := filepath.Base(dir)
	candidates := []string{
		filepath.Join(dir, repoName+".xsd"),
		filepath.Join(dir, "schema.xsd"),
		filepath.Join(dir, "openai.xsd"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no schema found in %s (looked for %s)", dir, strings.Join(candidates, ", "))
}
