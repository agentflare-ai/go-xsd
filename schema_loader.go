package xsd

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/agentflare-ai/go-xmldom"
)

// SchemaLoaderFunc is a function that loads a schema for a given namespace attribute node
// The attr parameter is the xmlns attribute node from which the namespace URI can be extracted
type SchemaLoaderFunc func(attr xmldom.Attr) (*Schema, error)

// PatternLoader associates a pattern with a loader function
type PatternLoader struct {
	Pattern string           // Regex pattern to match against namespace
	Loader  SchemaLoaderFunc // Function to load the schema
	regex   *regexp.Regexp   // Compiled regex pattern (internal)
}

// SchemaLoaderConfig configures a SchemaLoader with dependency injection
type SchemaLoaderConfig struct {
	// Base directory for resolving relative paths
	BaseDir string

	// HTTP client for remote loading (optional, defaults to http.DefaultClient)
	HTTPClient *http.Client

	// Pattern-based loaders for namespace resolution
	Loaders []PatternLoader
}

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

	// HTTP client for remote loading
	httpClient *http.Client

	// Pattern-based loaders for namespace resolution
	loaders []*PatternLoader

	mu sync.Mutex
}

// NewSchemaLoader creates a new schema loader with the given configuration
func NewSchemaLoader(config SchemaLoaderConfig) (*SchemaLoader, error) {
	loader := &SchemaLoader{
		BaseDir:    config.BaseDir,
		loaded:     make(map[string]*Schema),
		loading:    make(map[string]bool),
		httpClient: config.HTTPClient,
		loaders:    make([]*PatternLoader, 0, len(config.Loaders)),
	}

	// Use default HTTP client if not provided
	if loader.httpClient == nil {
		loader.httpClient = http.DefaultClient
	}

	// Compile regex patterns for each loader
	for i := range config.Loaders {
		regex, err := regexp.Compile(config.Loaders[i].Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", config.Loaders[i].Pattern, err)
		}
		config.Loaders[i].regex = regex
		loader.loaders = append(loader.loaders, &config.Loaders[i])
	}

	return loader, nil
}

// NewSchemaLoaderSimple creates a simple schema loader with just a base directory
// This is a convenience function for when you don't need custom loaders
func NewSchemaLoaderSimple(baseDir string) *SchemaLoader {
	loader, _ := NewSchemaLoader(SchemaLoaderConfig{
		BaseDir: baseDir,
		Loaders: []PatternLoader{},
	})
	return loader
}

// LoadSchemaForNamespace is deprecated - use LoadSchemasFromNamespaces instead
// This method cannot provide the xmlns attribute node that loaders need
func (sl *SchemaLoader) LoadSchemaForNamespace(namespace string) (*Schema, error) {
	return nil, fmt.Errorf("LoadSchemaForNamespace is not supported - use LoadSchemasFromNamespaces with ExtractNamespaces instead (namespace loaders require attribute nodes)")
}

// LoadSchemaWithImports loads a schema and all its imports/includes
func (sl *SchemaLoader) LoadSchemaWithImports(location string) (*Schema, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Initialize combined schema
	sl.combined = &Schema{
		ElementDecls:       make(map[QName]*ElementDecl),
		TypeDefs:           make(map[QName]Type),
		AttributeGroups:    make(map[QName]*AttributeGroup),
		Groups:             make(map[QName]*ModelGroup),
		ImportedSchemas:    make(map[string]*Schema),
		SubstitutionGroups: make(map[QName][]QName),
	}

	// Load the main schema
	mainSchema, err := sl.loadSchemaRecursive(location)
	if err != nil {
		return nil, err
	}

	// Set the target namespace from the main schema
	sl.combined.TargetNamespace = mainSchema.TargetNamespace
	sl.combined.doc = mainSchema.doc

	// Merge the main schema (treat as include since same namespace)
	if err := sl.mergeSchema(mainSchema, location); err != nil {
		return nil, fmt.Errorf("failed to merge main schema: %w", err)
	}

	// Merge all loaded schemas into the combined schema
	for loc, schema := range sl.loaded {
		if err := sl.mergeSchema(schema, loc); err != nil {
			return nil, fmt.Errorf("failed to merge schema %s: %w", loc, err)
		}
	}

	// Resolve all references in the combined schema
	sl.combined.resolveReferences()

	return sl.combined, nil
}

// loadSchemaRecursive loads a schema and processes its imports/includes
func (sl *SchemaLoader) loadSchemaRecursive(location string) (*Schema, error) {
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
			_, err := sl.loadSchemaRecursive(impLocation)
			if err != nil {
				// Import failures are often non-fatal
				// Log the error but continue
				slog.Error("failed to import schema", "location", imp.SchemaLocation, "error", err)
			}
		}
	}

	// Process includes (xs:include)
	includes := sl.findIncludes(doc)
	for _, includeLocation := range includes {
		// Resolve relative to current schema location
		incLocation := sl.resolveRelative(includeLocation, absLocation)

		// Load the included schema
		_, err := sl.loadSchemaRecursive(incLocation)
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
// TODO: Refactor to use a Loader interface that can be registered for different protocols
// This would allow extensibility for custom protocols (e.g., file://, https://, custom://)
// Example:
//
//	type Loader interface {
//	    CanLoad(uri string) bool
//	    Load(uri string) (io.ReadCloser, error)
//	}
//
// Then we could register loaders: RegisterLoader(&HTTPLoader{}, &FileLoader{}, &CustomLoader{})
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

	// Merge substitution groups
	for headQName, members := range source.SubstitutionGroups {
		// Append members to existing substitution group (avoiding duplicates)
		existing := target.SubstitutionGroups[headQName]
		for _, member := range members {
			// Check if member already exists
			if !slices.Contains(existing, member) {
				existing = append(existing, member)
			}
		}
		target.SubstitutionGroups[headQName] = existing
	}

	// Append imports (for transitive imports), avoiding duplicates
	for _, imp := range source.Imports {
		// Check if this import already exists
		found := false
		for _, existing := range target.Imports {
			if existing.Namespace == imp.Namespace && existing.SchemaLocation == imp.SchemaLocation {
				found = true
				break
			}
		}
		if !found {
			target.Imports = append(target.Imports, imp)
		}
	}
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
	loader := NewSchemaLoaderSimple(baseDir)
	return loader.LoadSchemaWithImports(tempFile.Name())
}

// LoadSchemaWithImports is a convenience function
func LoadSchemaWithImports(location string) (*Schema, error) {
	baseDir := filepath.Dir(location)
	loader := NewSchemaLoaderSimple(baseDir)
	return loader.LoadSchemaWithImports(location)
}

// NamespaceAttr holds a namespace URI and its attribute node
type NamespaceAttr struct {
	Prefix string      // Namespace prefix (empty for default namespace)
	URI    string      // Namespace URI
	Attr   xmldom.Attr // The xmlns attribute node
}

// ExtractNamespaces extracts all xmlns namespace declarations from an XML document
func ExtractNamespaces(doc xmldom.Document) map[string]NamespaceAttr {
	namespaces := make(map[string]NamespaceAttr)

	root := doc.DocumentElement()
	if root == nil {
		return namespaces
	}

	// Get all attributes on the root element
	attrs := root.Attributes()
	for i := uint(0); i < attrs.Length(); i++ {
		node := attrs.Item(i)
		if node == nil {
			continue
		}

		// Type assert to Attr
		attr, ok := node.(xmldom.Attr)
		if !ok {
			continue
		}

		attrNS := string(attr.NamespaceURI())
		attrLocal := string(attr.LocalName())
		attrName := string(attr.NodeName())
		attrValue := string(attr.NodeValue())

		// Check for xmlns declarations
		// xmlns="..." (default namespace)
		if attrName == "xmlns" {
			namespaces[""] = NamespaceAttr{
				Prefix: "",
				URI:    attrValue,
				Attr:   attr,
			}
			continue
		}

		// xmlns:prefix="..." (prefixed namespace)
		// The xmldom library may represent this with namespace="xmlns" or "http://www.w3.org/2000/xmlns/"
		if attrNS == "http://www.w3.org/2000/xmlns/" || attrNS == "xmlns" {
			prefix := attrLocal
			namespaces[prefix] = NamespaceAttr{
				Prefix: prefix,
				URI:    attrValue,
				Attr:   attr,
			}
			continue
		}

		// Also check for xmlns: prefix in the attribute name as a fallback
		if prefix, found := strings.CutPrefix(attrName, "xmlns:"); found {
			namespaces[prefix] = NamespaceAttr{
				Prefix: prefix,
				URI:    attrValue,
				Attr:   attr,
			}
		}
	}

	return namespaces
}

// LoadSchemasFromNamespaces loads schemas for the given namespaces using configured loaders
// Returns a combined schema with all loaded schemas merged
func (sl *SchemaLoader) LoadSchemasFromNamespaces(namespaces map[string]NamespaceAttr) (*Schema, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Initialize combined schema
	sl.combined = &Schema{
		ElementDecls:       make(map[QName]*ElementDecl),
		TypeDefs:           make(map[QName]Type),
		AttributeGroups:    make(map[QName]*AttributeGroup),
		Groups:             make(map[QName]*ModelGroup),
		ImportedSchemas:    make(map[string]*Schema),
		SubstitutionGroups: make(map[QName][]QName),
	}

	var mainSchema *Schema
	successCount := 0

	// Try to load schema for each namespace
	for _, nsAttr := range namespaces {
		// Skip built-in XML/XSD namespaces
		if nsAttr.URI == XSDNamespace ||
			nsAttr.URI == "http://www.w3.org/2001/XMLSchema-instance" ||
			nsAttr.URI == "http://www.w3.org/XML/1998/namespace" ||
			nsAttr.URI == "http://www.w3.org/2000/xmlns/" {
			continue
		}

		// Try to load using configured loaders
		schema, err := sl.loadSchemaForNamespaceUnlocked(nsAttr.Attr)
		if err != nil {
			// Log but continue - not all namespaces may have loadable schemas
			slog.Info("could not load schema for namespace", "namespace", nsAttr.URI, "error", err)
			continue
		}

		// Successfully loaded a schema
		successCount++

		// Use the first successfully loaded schema as main
		if mainSchema == nil {
			mainSchema = schema
			sl.combined.TargetNamespace = schema.TargetNamespace
		}

		// Merge into combined schema
		if err := sl.mergeSchema(schema, nsAttr.URI); err != nil {
			return nil, fmt.Errorf("failed to merge schema for %s: %w", nsAttr.URI, err)
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("could not load any schemas from document namespaces")
	}

	// Resolve all references in the combined schema
	sl.combined.resolveReferences()
	return sl.combined, nil
}

// loadSchemaForNamespaceUnlocked is the unlocked version of LoadSchemaForNamespace
// Must be called with sl.mu held
func (sl *SchemaLoader) loadSchemaForNamespaceUnlocked(attr xmldom.Attr) (*Schema, error) {
	namespace := string(attr.NodeValue())

	// Check if already loaded
	if schema, ok := sl.loaded[namespace]; ok {
		return schema, nil
	}

	// Try each loader in order
	for _, loader := range sl.loaders {
		if loader.regex.MatchString(namespace) {
			schema, err := loader.Loader(attr)
			if err != nil {
				continue // Try next loader
			}
			if schema != nil {
				sl.loaded[namespace] = schema
				return schema, nil
			}
		}
	}

	return nil, fmt.Errorf("no loader found for namespace: %s", namespace)
}
