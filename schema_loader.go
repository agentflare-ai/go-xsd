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
		ElementDecls:       make(map[QName]*ElementDecl),
		TypeDefs:           make(map[QName]Type),
		AttributeGroups:    make(map[QName]*AttributeGroup),
		Groups:             make(map[QName]*ModelGroup),
		ImportedSchemas:    make(map[string]*Schema),
		SubstitutionGroups: make(map[QName][]QName),
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

	// Merge substitution groups
	for headQName, members := range source.SubstitutionGroups {
		// Append members to existing substitution group (avoiding duplicates)
		existing := target.SubstitutionGroups[headQName]
		for _, member := range members {
			// Check if member already exists
			found := false
			for _, existingMember := range existing {
				if existingMember == member {
					found = true
					break
				}
			}
			if !found {
				existing = append(existing, member)
			}
		}
		target.SubstitutionGroups[headQName] = existing
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

// ExtractNamespaces extracts all xmlns namespace declarations from an XML document
func ExtractNamespaces(doc xmldom.Document) map[string]string {
	namespaces := make(map[string]string)

	root := doc.DocumentElement()
	if root == nil {
		return namespaces
	}

	// Get all attributes on the root element
	attrs := root.Attributes()
	for i := uint(0); i < attrs.Length(); i++ {
		attr := attrs.Item(i)
		if attr == nil {
			continue
		}

		attrNS := string(attr.NamespaceURI())
		attrLocal := string(attr.LocalName())
		attrName := string(attr.NodeName())
		attrValue := string(attr.NodeValue())

		// Check for xmlns declarations
		// xmlns="..." (default namespace)
		if attrName == "xmlns" {
			namespaces[""] = attrValue
			continue
		}

		// xmlns:prefix="..." (prefixed namespace)
		// The xmldom library may represent this with namespace="xmlns" or "http://www.w3.org/2000/xmlns/"
		if attrNS == "http://www.w3.org/2000/xmlns/" || attrNS == "xmlns" {
			prefix := attrLocal
			namespaces[prefix] = attrValue
			continue
		}

		// Also check for xmlns: prefix in the attribute name as a fallback
		if strings.HasPrefix(attrName, "xmlns:") {
			prefix := strings.TrimPrefix(attrName, "xmlns:")
			namespaces[prefix] = attrValue
		}
	}

	return namespaces
}

// Schema path patterns for namespace resolution
// These patterns define how to construct potential schema URLs from a namespace URI
// Variables: {ns} = namespace URI, {prefix} = element prefix, {base} = base domain
var schemaPathPatterns = []string{
	"{ns}.xsd",          // e.g., xsd.agentml.dev/agentml -> xsd.agentml.dev/agentml.xsd
	"{ns}",              // Try the namespace URI directly
	"{ns}/{prefix}.xsd", // e.g., xsd.agentml.dev/agentml -> xsd.agentml.dev/agentml/agentml.xsd
	"{base}/schema.xsd", // e.g., xsd.agentml.dev/agentml -> xsd.agentml.dev/schema.xsd
	"{ns}/schema.xsd",   // e.g., xsd.agentml.dev/agentml -> xsd.agentml.dev/agentml/schema.xsd
	"{ns}/index.xsd",    // e.g., xsd.agentml.dev/agentml -> xsd.agentml.dev/agentml/index.xsd
}

// TryLoadSchemaWithFallbacks attempts to load a schema from a namespace URI with fallback strategies
func (sl *SchemaLoader) TryLoadSchemaWithFallbacks(namespaceURI, prefix string) (*Schema, error) {
	if namespaceURI == "" {
		return nil, fmt.Errorf("empty namespace URI")
	}

	// Skip built-in XML/XSD namespaces
	if namespaceURI == XSDNamespace ||
		namespaceURI == "http://www.w3.org/2001/XMLSchema-instance" ||
		namespaceURI == "http://www.w3.org/XML/1998/namespace" ||
		namespaceURI == "http://www.w3.org/2000/xmlns/" {
		return nil, fmt.Errorf("skipping built-in namespace: %s", namespaceURI)
	}

	// Determine namespace characteristics
	isURL := strings.HasPrefix(namespaceURI, "http://") || strings.HasPrefix(namespaceURI, "https://")
	looksLikeDomain := !isURL && strings.Contains(namespaceURI, ".") &&
		(strings.Contains(namespaceURI, "/") || !strings.Contains(namespaceURI, ":"))

	// Extract base domain from namespace (e.g., xsd.agentml.dev/agentml -> xsd.agentml.dev)
	baseDomain := namespaceURI
	if strings.Contains(namespaceURI, "/") {
		parts := strings.Split(namespaceURI, "/")
		baseDomain = parts[0]
	}

	// Build schema path attempts from patterns
	var attempts []string
	schemes := []string{"https://", "http://"}

	for _, pattern := range schemaPathPatterns {
		// Replace pattern variables
		path := strings.ReplaceAll(pattern, "{ns}", namespaceURI)
		path = strings.ReplaceAll(path, "{prefix}", prefix)
		path = strings.ReplaceAll(path, "{base}", baseDomain)

		// Skip if prefix is required but not provided
		if strings.Contains(pattern, "{prefix}") && prefix == "" {
			continue
		}

		// Generate URLs based on namespace type
		if isURL {
			// Already a full URL, use as-is
			attempts = append(attempts, path)
		} else if looksLikeDomain {
			// Domain-like namespace - try both https and http
			for _, scheme := range schemes {
				attempts = append(attempts, scheme+path)
			}
		} else {
			// Local path
			attempts = append(attempts, path)
		}
	}

	// Try loading from each attempt
	var lastErr error
	for _, location := range attempts {
		fmt.Fprintf(os.Stderr, "DEBUG: Trying to load schema from: %s\n", location)
		schema, err := sl.loadSchemaRecursive(location, namespaceURI)
		if err == nil && schema != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: Successfully loaded schema from: %s\n", location)
			return schema, nil
		}
		lastErr = err
	}

	// Fallback: Try common local paths relative to base directory
	if sl.BaseDir != "" {
		localPaths := []string{
			filepath.Join(sl.BaseDir, "agentml.xsd"),
			filepath.Join(sl.BaseDir, "agentml", "agentml.xsd"),
			filepath.Join(sl.BaseDir, "../agentml/agentml.xsd"),
		}

		for _, localPath := range localPaths {
			absPath, err := filepath.Abs(localPath)
			if err != nil {
				continue
			}

			// Check if file exists
			if _, err := os.Stat(absPath); err == nil {
				schema, err := sl.loadSchemaRecursive(absPath, namespaceURI)
				if err == nil && schema != nil {
					return schema, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("failed to load schema for namespace %s: %w", namespaceURI, lastErr)
}

// LoadSchemasFromDocument extracts xmlns declarations from a document and loads schemas for them
func (sl *SchemaLoader) LoadSchemasFromDocument(doc xmldom.Document) (*Schema, error) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	// Extract namespaces
	namespaces := ExtractNamespaces(doc)

	// Get root element tag name for default namespace
	var rootTagName string
	if root := doc.DocumentElement(); root != nil {
		rootTagName = string(root.LocalName())
	}

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
	for prefix, nsURI := range namespaces {
		// If prefix is empty (default namespace), use the root element's tag name
		prefixToUse := prefix
		if prefixToUse == "" && rootTagName != "" {
			prefixToUse = rootTagName
		}

		schema, err := sl.TryLoadSchemaWithFallbacks(nsURI, prefixToUse)
		if err != nil {
			// Log but continue - not all namespaces may have loadable schemas
			fmt.Printf("Info: Could not load schema for namespace %s (prefix: %s): %v\n", nsURI, prefixToUse, err)
			continue
		}

		// Successfully loaded a schema
		successCount++

		// Use the first successfully loaded schema as main
		if mainSchema == nil {
			mainSchema = schema
			sl.combined.TargetNamespace = schema.TargetNamespace
		}

		// Store in loaded map
		sl.loaded[nsURI] = schema

		// Merge into combined schema
		if err := sl.mergeSchema(schema, nsURI); err != nil {
			return nil, fmt.Errorf("failed to merge schema for %s: %w", nsURI, err)
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("could not load any schemas from document namespaces")
	}

	// Resolve all references in the combined schema
	sl.combined.resolveReferences()

	// Debug: Print what we loaded
	fmt.Fprintf(os.Stderr, "DEBUG: Combined schema has:\n")
	fmt.Fprintf(os.Stderr, "  - Target namespace: %s\n", sl.combined.TargetNamespace)
	fmt.Fprintf(os.Stderr, "  - %d element declarations\n", len(sl.combined.ElementDecls))
	fmt.Fprintf(os.Stderr, "  - Element names: ")
	count := 0
	for qname := range sl.combined.ElementDecls {
		if count > 0 {
			fmt.Fprintf(os.Stderr, ", ")
		}
		fmt.Fprintf(os.Stderr, "%s", qname.Local)
		count++
		if count >= 10 {
			fmt.Fprintf(os.Stderr, ", ...")
			break
		}
	}
	fmt.Fprintf(os.Stderr, "\n")

	return sl.combined, nil
}
