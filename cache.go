package xsd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/agentflare-ai/go-xmldom"
)

// SchemaCache manages cached XSD schemas using sync.OnceValue per project rules
type SchemaCache struct {
	mu       sync.RWMutex
	schemas  map[string]*schemaEntry
	BasePath string // Base path for resolving relative schema locations
}

// schemaEntry holds a schema and its loader
type schemaEntry struct {
	loader func() (*Schema, error)
	once   sync.Once
	schema *Schema
	err    error
}

// GlobalCache is the singleton schema cache
var GlobalCache = NewSchemaCache("")

// NewSchemaCache creates a new schema cache
func NewSchemaCache(basePath string) *SchemaCache {
	return &SchemaCache{
		schemas:  make(map[string]*schemaEntry),
		BasePath: basePath,
	}
}

// SetBasePath sets the base path for resolving relative schema locations
func (sc *SchemaCache) SetBasePath(path string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.BasePath = path
}

// Get retrieves a schema from cache or loads it if not present
func (sc *SchemaCache) Get(location string) (*Schema, error) {
	// Resolve path
	resolvedPath := sc.resolvePath(location)

	// Check cache
	sc.mu.RLock()
	entry, exists := sc.schemas[resolvedPath]
	sc.mu.RUnlock()

	if exists {
		// Use sync.Once to ensure single loading
		entry.once.Do(func() {
			if entry.loader != nil {
				entry.schema, entry.err = entry.loader()
			}
		})
		return entry.schema, entry.err
	}

	// Create new entry with loader
	entry = &schemaEntry{
		loader: func() (*Schema, error) {
			return sc.loadSchema(resolvedPath)
		},
	}

	// Store in cache
	sc.mu.Lock()
	sc.schemas[resolvedPath] = entry
	sc.mu.Unlock()

	// Load and return
	entry.once.Do(func() {
		entry.schema, entry.err = entry.loader()
	})
	return entry.schema, entry.err
}

// GetOrLoad gets a schema from cache or loads from provided document
func (sc *SchemaCache) GetOrLoad(location string, doc xmldom.Document) (*Schema, error) {
	// Try to get from cache first
	if schema, err := sc.Get(location); err == nil {
		return schema, nil
	}

	// Parse from provided document
	schema, err := Parse(doc)
	if err != nil {
		return nil, err
	}

	// Cache it
	sc.mu.Lock()
	sc.schemas[location] = &schemaEntry{
		schema: schema,
		err:    nil,
	}
	sc.mu.Unlock()

	return schema, nil
}

// Clear removes all cached schemas
func (sc *SchemaCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.schemas = make(map[string]*schemaEntry)
}

// Remove removes a specific schema from cache
func (sc *SchemaCache) Remove(location string) {
	resolvedPath := sc.resolvePath(location)
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.schemas, resolvedPath)
}

// resolvePath resolves a schema location to an absolute path
func (sc *SchemaCache) resolvePath(location string) string {
	if filepath.IsAbs(location) {
		return location
	}
	if sc.BasePath != "" {
		return filepath.Join(sc.BasePath, location)
	}
	// Try to resolve relative to current directory
	abs, err := filepath.Abs(location)
	if err != nil {
		return location
	}
	return abs
}

// loadSchema loads a schema from disk
func (sc *SchemaCache) loadSchema(path string) (*Schema, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file %s: %w", path, err)
	}

	// Parse XML
	decoder := xmldom.NewDecoderFromBytes(data)
	doc, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to parse schema file %s: %w", path, err)
	}

	// Parse schema
	schema, err := Parse(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XSD schema %s: %w", path, err)
	}

	// Handle imports
	for _, imp := range schema.Imports {
		if imp.SchemaLocation != "" {
			// Recursively load imported schemas
			importPath := sc.resolvePath(imp.SchemaLocation)
			if _, err := sc.Get(importPath); err != nil {
				// Log warning but don't fail
				// Imports might be optional or resolved differently
				slog.Warn("failed to load imported schema", "location", importPath, "error", err)
			}
		}
	}

	return schema, nil
}

// PreloadCommonTypes preloads commonly used XSD types for performance
func (sc *SchemaCache) PreloadCommonTypes() {
	// This would load built-in XSD types like xs:string, xs:integer, etc.
	// For now, we'll just ensure they're available when needed
	commonTypes := []string{
		"string", "boolean", "decimal", "float", "double",
		"dateTime", "time", "date", "anyURI", "QName",
		"ID", "IDREF", "IDREFS", "NMTOKEN", "NMTOKENS",
		"integer", "nonNegativeInteger", "positiveInteger",
	}

	// Create simple type definitions for built-ins
	for _, typeName := range commonTypes {
		qname := QName{
			Namespace: XSDNamespace,
			Local:     typeName,
		}
		// These would be properly defined with their restrictions
		// For now, create placeholder types
		st := &SimpleType{
			QName: qname,
		}

		// Store in a special built-in schema
		builtinSchema := &Schema{
			TargetNamespace: XSDNamespace,
			TypeDefs:        map[QName]Type{qname: st},
		}

		sc.mu.Lock()
		sc.schemas["builtin:"+XSDNamespace] = &schemaEntry{
			schema: builtinSchema,
			err:    nil,
		}
		sc.mu.Unlock()
	}
}

// SchemaRegistry manages multiple schemas for different namespaces
type SchemaRegistry struct {
	mu            sync.RWMutex
	namespaces    map[string]*Schema
	defaultSchema *Schema
	cache         *SchemaCache
}

// NewSchemaRegistry creates a new schema registry
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		namespaces: make(map[string]*Schema),
		cache:      NewSchemaCache(""),
	}
}

// Register registers a schema for a namespace
func (sr *SchemaRegistry) Register(namespace string, schema *Schema) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.namespaces[namespace] = schema
}

// RegisterFile registers a schema from a file
func (sr *SchemaRegistry) RegisterFile(namespace, location string) error {
	schema, err := sr.cache.Get(location)
	if err != nil {
		return err
	}
	sr.Register(namespace, schema)
	return nil
}

// SetDefault sets the default schema for elements without namespaces
func (sr *SchemaRegistry) SetDefault(schema *Schema) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.defaultSchema = schema
}

// GetForNamespace retrieves the schema for a namespace
func (sr *SchemaRegistry) GetForNamespace(namespace string) (*Schema, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	if namespace == "" && sr.defaultSchema != nil {
		return sr.defaultSchema, true
	}

	schema, ok := sr.namespaces[namespace]
	return schema, ok
}

// ValidateWithRegistry validates a document using multiple namespace schemas
func (sr *SchemaRegistry) Validate(doc xmldom.Document) []Violation {
	violations := []Violation{}

	// Get root element namespace
	root := doc.DocumentElement()
	if root == nil {
		return []Violation{{
			Code:    "xsd-no-root",
			Message: "Document has no root element",
		}}
	}

	rootNS := string(root.NamespaceURI())

	// Get appropriate schema
	schema, ok := sr.GetForNamespace(rootNS)
	if !ok && sr.defaultSchema != nil {
		schema = sr.defaultSchema
	}

	if schema == nil {
		return []Violation{{
			Code:    "xsd-no-schema",
			Message: fmt.Sprintf("No schema found for namespace '%s'", rootNS),
		}}
	}

	// Validate with primary schema
	validator := NewValidator(schema)
	violations = append(violations, validator.Validate(doc)...)

	// Handle mixed namespace validation
	// Validate elements from imported namespaces against their respective schemas
	if len(schema.ImportedSchemas) > 0 {
		violations = append(violations, sr.validateImportedNamespaces(doc, schema)...)
	}

	return violations
}

// validateImportedNamespaces validates elements from imported namespaces
func (sr *SchemaRegistry) validateImportedNamespaces(doc xmldom.Document, primarySchema *Schema) []Violation {
	violations := []Violation{}

	// Build namespace to schema mapping
	nsMap := make(map[string]*Schema)
	for _, importedSchema := range primarySchema.ImportedSchemas {
		if importedSchema.TargetNamespace != "" {
			nsMap[importedSchema.TargetNamespace] = importedSchema
		}
	}

	// Recursively validate elements against their namespace schemas
	root := doc.DocumentElement()
	if root != nil {
		violations = append(violations, sr.validateElementNamespace(root, nsMap, primarySchema)...)
	}

	return violations
}

// validateElementNamespace validates an element against the schema for its namespace
func (sr *SchemaRegistry) validateElementNamespace(elem xmldom.Element, nsMap map[string]*Schema, primarySchema *Schema) []Violation {
	violations := []Violation{}

	elemNS := string(elem.NamespaceURI())

	// If element is from an imported namespace, validate against that schema
	if elemNS != "" && elemNS != primarySchema.TargetNamespace {
		if importedSchema, exists := nsMap[elemNS]; exists {
			// Validate this element against the imported schema
			// Note: We don't need the validator variable here, just validate the type directly
			qname := QName{Namespace: elemNS, Local: string(elem.LocalName())}
			if decl, found := importedSchema.ElementDecls[qname]; found {
				if decl.Type != nil {
					violations = append(violations, decl.Type.Validate(elem, importedSchema)...)
				}
			}
		}
	}

	// Recursively validate children
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		if child := children.Item(i); child != nil {
			violations = append(violations, sr.validateElementNamespace(child, nsMap, primarySchema)...)
		}
	}

	return violations
}
