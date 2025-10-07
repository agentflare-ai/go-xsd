package xsd

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/agentflare-ai/go-xmldom"
)

// SchemaLoader handles loading schemas with import/include support
type SchemaLoader struct {
	// Base directory for resolving relative paths
	BaseDir string

	// Map of loaded schemas by location
	loaded map[string]*Schema

	// Map of schemas being loaded (for cycle detection)
	loading map[string]bool

	// Combined schema with all imports/includes merged
	combined *Schema

	// Whether to allow remote schema loading
	AllowRemote bool

	// HTTP client for remote loading
	httpClient *http.Client

	mu sync.Mutex
}

// NewSchemaLoader creates a new schema loader
func NewSchemaLoader(baseDir string) *SchemaLoader {
	return &SchemaLoader{
		BaseDir:     baseDir,
		loaded:      make(map[string]*Schema),
		loading:     make(map[string]bool),
		AllowRemote: false, // Disabled by default for security
		httpClient:  &http.Client{},
	}
}

// LoadSchemaWithImports loads a schema and all its imports/includes
func (sl *SchemaLoader) LoadSchemaWithImports(location string) (*Schema, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Initialize combined schema
	sl.combined = &Schema{
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
		AttributeGroups: make(map[QName]*AttributeGroup),
		Groups:          make(map[QName]*ModelGroup),
		ImportedSchemas: make(map[string]*Schema),
	}

	// Load the main schema
	mainSchema, err := sl.loadSchemaRecursive(location, "")
	if err != nil {
		return nil, err
	}

	// Set the target namespace from the main schema
	sl.combined.TargetNamespace = mainSchema.TargetNamespace
	sl.combined.doc = mainSchema.doc

	// Merge all loaded schemas into the combined schema
	for location, schema := range sl.loaded {
		if err := sl.mergeSchema(schema, location); err != nil {
			return nil, fmt.Errorf("failed to merge schema %s: %w", location, err)
		}
	}

	// Resolve all references in the combined schema
	sl.combined.resolveReferences()

	return sl.combined, nil
}

// loadSchemaRecursive loads a schema and processes its imports/includes
func (sl *SchemaLoader) loadSchemaRecursive(location, namespace string) (*Schema, error) {
	// Resolve the location to an absolute path/URL
	absLocation, err := sl.resolveLocation(location)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve location %s: %w", location, err)
	}

	// Check if already loaded
	if schema, ok := sl.loaded[absLocation]; ok {
		return schema, nil
	}

	// Check for circular dependencies
	if sl.loading[absLocation] {
		return nil, fmt.Errorf("circular dependency detected: %s", absLocation)
	}

	// Mark as loading
	sl.loading[absLocation] = true
	defer func() {
		delete(sl.loading, absLocation)
	}()

	// Load the schema document
	doc, err := sl.loadDocument(absLocation)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema from %s: %w", absLocation, err)
	}

	// Parse the schema
	schema, err := Parse(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema from %s: %w", absLocation, err)
	}

	// Store the loaded schema
	sl.loaded[absLocation] = schema

	// Process imports
	for _, imp := range schema.Imports {
		if imp.SchemaLocation != "" {
			// Resolve relative to current schema location
			impLocation := sl.resolveRelative(imp.SchemaLocation, absLocation)

			// Load the imported schema
			_, err := sl.loadSchemaRecursive(impLocation, imp.Namespace)
			if err != nil {
				// Import failures are often non-fatal
				// Log the error but continue
				fmt.Printf("Warning: failed to import %s: %v\n", imp.SchemaLocation, err)
			}
		}
	}

	// Process includes (xs:include)
	includes := sl.findIncludes(doc)
	for _, includeLocation := range includes {
		// Resolve relative to current schema location
		incLocation := sl.resolveRelative(includeLocation, absLocation)

		// Load the included schema
		_, err := sl.loadSchemaRecursive(incLocation, schema.TargetNamespace)
		if err != nil {
			return nil, fmt.Errorf("failed to include %s: %w", includeLocation, err)
		}
	}

	return schema, nil
}

// findIncludes finds all xs:include elements in the document
func (sl *SchemaLoader) findIncludes(doc xmldom.Document) []string {
	var includes []string

	root := doc.DocumentElement()
	if root == nil {
		return includes
	}

	children := root.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil {
			continue
		}

		if string(child.NamespaceURI()) == XSDNamespace &&
			string(child.LocalName()) == "include" {
			if location := child.GetAttribute("schemaLocation"); location != "" {
				includes = append(includes, string(location))
			}
		}
	}

	return includes
}

// resolveLocation resolves a location to an absolute path or URL
func (sl *SchemaLoader) resolveLocation(location string) (string, error) {
	// Check if it's already an absolute path
	if filepath.IsAbs(location) {
		return location, nil
	}

	// Check if it's a URL
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		if !sl.AllowRemote {
			return "", fmt.Errorf("remote schema loading is disabled")
		}
		return location, nil
	}

	// Resolve relative to base directory
	if sl.BaseDir != "" {
		return filepath.Abs(filepath.Join(sl.BaseDir, location))
	}

	// Resolve relative to current directory
	return filepath.Abs(location)
}

// resolveRelative resolves a relative location based on a base location
func (sl *SchemaLoader) resolveRelative(relative, base string) string {
	// If relative is already absolute, return as-is
	if filepath.IsAbs(relative) {
		return relative
	}

	// If relative is a URL, return as-is
	if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
		return relative
	}

	// If base is a URL, resolve as URL
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		baseURL, err := url.Parse(base)
		if err != nil {
			return relative
		}
		relURL, err := baseURL.Parse(relative)
		if err != nil {
			return relative
		}
		return relURL.String()
	}

	// Resolve as file path
	baseDir := filepath.Dir(base)
	return filepath.Join(baseDir, relative)
}

// loadDocument loads an XML document from a location
func (sl *SchemaLoader) loadDocument(location string) (xmldom.Document, error) {
	var reader io.ReadCloser
	var err error

	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		// Load from URL
		resp, err := sl.httpClient.Get(location)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", location, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, location)
		}
		reader = resp.Body
	} else {
		// Load from file
		file, err := os.Open(location)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", location, err)
		}
		reader = file
	}

	defer reader.Close()

	// Parse the XML document
	doc, err := xmldom.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	return doc, nil
}

// mergeSchema merges a schema into the combined schema
func (sl *SchemaLoader) mergeSchema(source *Schema, location string) error {
	// Store as imported schema
	sl.combined.ImportedSchemas[location] = source

	// Determine if this is an import or include
	isInclude := source.TargetNamespace == sl.combined.TargetNamespace

	if isInclude {
		// xs:include: merge all components directly
		sl.mergeComponents(source, sl.combined)
	} else {
		// xs:import: merge with namespace preservation
		sl.mergeWithNamespace(source, sl.combined)
	}

	return nil
}

// mergeComponents merges schema components (for xs:include)
func (sl *SchemaLoader) mergeComponents(source, target *Schema) {
	// Merge element declarations
	for qname, elem := range source.ElementDecls {
		if _, exists := target.ElementDecls[qname]; !exists {
			target.ElementDecls[qname] = elem
		}
	}

	// Merge type definitions
	for qname, typ := range source.TypeDefs {
		if _, exists := target.TypeDefs[qname]; !exists {
			target.TypeDefs[qname] = typ
		}
	}

	// Merge attribute groups
	for qname, ag := range source.AttributeGroups {
		if _, exists := target.AttributeGroups[qname]; !exists {
			target.AttributeGroups[qname] = ag
		}
	}

	// Merge model groups
	for qname, mg := range source.Groups {
		if _, exists := target.Groups[qname]; !exists {
			target.Groups[qname] = mg
		}
	}

	// Append imports (for transitive imports)
	target.Imports = append(target.Imports, source.Imports...)
}

// mergeWithNamespace merges schema components preserving namespaces (for xs:import)
func (sl *SchemaLoader) mergeWithNamespace(source, target *Schema) {
	// For imports, we keep components in their original namespace
	// The components are already namespaced correctly
	sl.mergeComponents(source, target)

	// Additionally, we might need to handle namespace mappings
	// This is where cross-namespace type resolution happens
	// The resolver will look in ImportedSchemas when needed
}

// LoadSchemaFromString loads a schema from a string with import/include support
func LoadSchemaFromString(content string, baseDir string) (*Schema, error) {
	// Create a temporary file to establish a base location
	tempFile, err := os.CreateTemp("", "schema-*.xsd")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write the content
	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tempFile.Close()

	// Load with imports
	loader := NewSchemaLoader(baseDir)
	return loader.LoadSchemaWithImports(tempFile.Name())
}

// LoadSchemaWithImports is a convenience function
func LoadSchemaWithImports(location string) (*Schema, error) {
	baseDir := filepath.Dir(location)
	loader := NewSchemaLoader(baseDir)
	return loader.LoadSchemaWithImports(location)
}
