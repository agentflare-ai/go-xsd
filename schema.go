package xsd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/agentflare-ai/go-xmldom"
)

// XSDNamespace is the XML Schema namespace
const XSDNamespace = "http://www.w3.org/2001/XMLSchema"

// Schema represents a compiled XSD schema
type Schema struct {
	mu                 sync.RWMutex
	TargetNamespace    string
	ElementDecls       map[QName]*ElementDecl
	TypeDefs           map[QName]Type
	AttributeGroups    map[QName]*AttributeGroup
	Groups             map[QName]*ModelGroup
	Imports            []*Import
	ImportedSchemas    map[string]*Schema // Map of imported schemas by location
	SubstitutionGroups map[QName][]QName  // Maps head element to list of substitutable elements
	BlockDefault       map[string]bool    // xs:schema@blockDefault tokens
	FinalDefault       map[string]bool    // xs:schema@finalDefault tokens
	ElementFormDefault    string          // "qualified" or "unqualified" (default)
	AttributeFormDefault  string          // "qualified" or "unqualified" (default)
	doc                xmldom.Document

	// Options
	StrictContentModel bool // If true, use strict content-model validation
}
type QName struct {
	Namespace string
	Local     string
}

// String returns the string representation of a QName
func (q QName) String() string {
	if q.Namespace == "" {
		return q.Local
	}
	return fmt.Sprintf("{%s}%s", q.Namespace, q.Local)
}

// ElementDecl represents an element declaration
type ElementDecl struct {
	Name              QName
	Type              Type
	MinOcc            int // Renamed to avoid conflict with Particle interface method
	MaxOcc            int // -1 for unbounded, renamed to avoid conflict
	Nillable          bool
	Abstract          bool
	SubstitutionGroup QName           // Head element this element can substitute for
	Block             map[string]bool // Disallowed substitutions from @block
	Default           string
	Fixed             string
	Constraints       []*IdentityConstraint // Identity constraints (key, keyref, unique)
}

// Type is the interface for all XSD types
type Type interface {
	Name() QName
	Validate(element xmldom.Element, schema *Schema) []Violation
}

// SimpleType represents an XSD simple type
type SimpleType struct {
	QName       QName
	Base        QName
	Restriction *Restriction
	List        *List
	Union       *Union
}

// ComplexType represents an XSD complex type
type ComplexType struct {
	QName              QName
	Content            Content
	Attributes         []*AttributeDecl
	AttributeGroup     []QName
	AnyAttribute       *AnyAttribute
	Mixed              bool
	Abstract           bool
	Final              map[string]bool // final constraints: extension, restriction
	DerivedByExtension bool            // set true if this type (or its base chain) includes an extension
}

// Content represents element content model
type Content interface {
	Validate(element xmldom.Element, schema *Schema) []Violation
}

// SimpleContent represents simple content in a complex type
type SimpleContent struct {
	Base        QName
	Extension   *Extension
	Restriction *Restriction
}

// ComplexContent represents complex content
type ComplexContent struct {
	Mixed       bool
	Base        QName
	Extension   *Extension
	Restriction *Restriction
}

// ModelGroup represents a group of elements
type ModelGroup struct {
	Kind      ModelGroupKind // sequence, choice, all
	Particles []Particle
	MinOcc    int // Renamed to avoid conflict with method
	MaxOcc    int // Renamed to avoid conflict with method
}

// ModelGroupKind represents the kind of model group
type ModelGroupKind string

const (
	SequenceGroup ModelGroupKind = "sequence"
	ChoiceGroup   ModelGroupKind = "choice"
	AllGroup      ModelGroupKind = "all"
)

// Particle represents a particle in a content model
type Particle interface {
	MinOccurs() int
	MaxOccurs() int
	Validate(element xmldom.Element, schema *Schema) []Violation
}

// ElementRef represents a reference to an element
type ElementRef struct {
	Ref    QName
	MinOcc int // Renamed to avoid conflict with method
	MaxOcc int // Renamed to avoid conflict with method
}

// GroupRef represents a reference to a model group
type GroupRef struct {
	Ref    QName
	MinOcc int
	MaxOcc int
}

// AnyElement represents xs:any wildcard
type AnyElement struct {
	Namespace       string
	ProcessContents string
	MinOcc          int
	MaxOcc          int
}

// AttributeDecl represents an attribute declaration
type AttributeDecl struct {
	Name    QName
	Type    Type
	Use     AttributeUse
	Default string
	Fixed   string
}

// AttributeUse represents attribute use
type AttributeUse string

const (
	OptionalUse   AttributeUse = "optional"
	RequiredUse   AttributeUse = "required"
	ProhibitedUse AttributeUse = "prohibited"
)

// AttributeGroup represents a group of attributes
type AttributeGroup struct {
	Name       QName
	Attributes []*AttributeDecl
}

// Restriction represents a restriction on a type
type Restriction struct {
	Base         QName
	Facets       []FacetValidator
	// For complexContent restrictions
	Content      Content
	Attributes   []*AttributeDecl
	AnyAttribute *AnyAttribute
}

// Facet represents a constraining facet (deprecated - use FacetValidator from facets.go)
 type Facet interface {
 	Validate(value string) error
 }

// parseDerivationSet parses tokens like "extension restriction substitution #all" into a set
func parseDerivationSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, tok := range strings.Fields(s) {
		set[tok] = true
		if tok == "#all" {
			// include common tokens
			set["extension"] = true
			set["restriction"] = true
			set["substitution"] = true
			set["list"] = true
			set["union"] = true
		}
	}
	return set
}

// List represents a list type
type List struct {
	ItemType QName
}

// Union represents a union type
type Union struct {
	MemberTypes []QName
}

// Extension represents type extension
type Extension struct {
	Base         QName
	Attributes   []*AttributeDecl
	Content      Content
	AnyAttribute *AnyAttribute
}

// AnyAttribute represents xs:anyAttribute
type AnyAttribute struct {
	Namespace       string
	ProcessContents string
}

// Import represents an xs:import
type Import struct {
	Namespace      string
	SchemaLocation string
}

// AllowAnyContent is a content model that allows any child elements
type AllowAnyContent struct{}

// Violation represents a validation error
type Violation struct {
	Element   xmldom.Element
	Attribute string
	Code      string
	Message   string
	Expected  []string
	Actual    string
}

// LoadSchema loads and parses an XSD schema from a file
func LoadSchema(filename string) (*Schema, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	doc, err := xmldom.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML file: %w", err)
	}

	// Validate the schema document itself (basic structural checks)
	sv := NewSchemaValidator()
	if errors := sv.ValidateSchema(doc); len(errors) > 0 {
		// Return the first validation error
		return nil, fmt.Errorf("invalid XSD schema: %w", errors[0])
	}

// Use SchemaLoader to load schema with imports/includes
	baseDir := filepath.Dir(filename)
	loader := NewSchemaLoaderSimple(baseDir)
	combined, err := loader.LoadSchemaWithImports(filename)
	if err != nil {
		return nil, err
	}

	// Perform compiled content model determinism checks (UPA)
	if err := combined.ValidateContentModels(); err != nil {
		return nil, fmt.Errorf("invalid XSD schema: %w", err)
	}

	return combined, nil
}

// Parse parses an XSD schema from an XML document
func Parse(doc xmldom.Document) (*Schema, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil document")
	}

	root := doc.DocumentElement()
	if root == nil {
		return nil, fmt.Errorf("no root element")
	}

	// Check if this is an XSD schema
	if string(root.NamespaceURI()) != XSDNamespace || string(root.LocalName()) != "schema" {
		return nil, fmt.Errorf("not an XSD schema document")
	}

	schema := &Schema{
		ElementDecls:       make(map[QName]*ElementDecl),
		TypeDefs:           make(map[QName]Type),
		AttributeGroups:    make(map[QName]*AttributeGroup),
		Groups:             make(map[QName]*ModelGroup),
		ImportedSchemas:    make(map[string]*Schema),
		SubstitutionGroups: make(map[QName][]QName),
		doc:                doc,
	}

// Get target namespace
if tns := root.GetAttribute("targetNamespace"); tns != "" {
	schema.TargetNamespace = string(tns)
}

// Parse schema-level defaults
	if bd := root.GetAttribute("blockDefault"); bd != "" {
		schema.BlockDefault = parseDerivationSet(string(bd))
	}
	if fd := root.GetAttribute("finalDefault"); fd != "" {
		schema.FinalDefault = parseDerivationSet(string(fd))
	}
	// Parse form defaults (defaults are 'unqualified')
	if efd := root.GetAttribute("elementFormDefault"); efd != "" {
		schema.ElementFormDefault = string(efd)
	} else {
		schema.ElementFormDefault = "unqualified"
	}
	if afd := root.GetAttribute("attributeFormDefault"); afd != "" {
		schema.AttributeFormDefault = string(afd)
	} else {
		schema.AttributeFormDefault = "unqualified"
	}

	// Parse schema components
	children := root.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil {
			continue
		}

		if string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "element":
			if err := schema.parseElement(child); err != nil {
				return nil, err
			}
		case "simpleType":
			if err := schema.parseSimpleType(child); err != nil {
				return nil, err
			}
		case "complexType":
			if err := schema.parseComplexType(child); err != nil {
				return nil, err
			}
		case "attributeGroup":
			if err := schema.parseAttributeGroup(child); err != nil {
				return nil, err
			}
		case "group":
			if err := schema.parseGroup(child); err != nil {
				return nil, err
			}
		case "import":
			if err := schema.parseImport(child); err != nil {
				return nil, err
			}
		}
	}

	// Second pass: resolve type references
	schema.resolveReferences()

	return schema, nil
}

// resolveReferences performs a second pass to resolve all type references
func (s *Schema) resolveReferences() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Resolve element type references
	for _, decl := range s.ElementDecls {
		if decl.Type == nil {
			continue
		}

		// Check if it's a placeholder simple type
		if st, ok := decl.Type.(*SimpleType); ok && st.Restriction == nil && st.List == nil && st.Union == nil {
			// Try to resolve the actual type
			if actualType, exists := s.TypeDefs[st.QName]; exists {
				decl.Type = actualType
			}
		}
	}

	// Resolve group references in complex types
	for _, typeDef := range s.TypeDefs {
		if ct, ok := typeDef.(*ComplexType); ok {
			// Check if content is a GroupRef that needs resolution
			if gr, ok := ct.Content.(*GroupRef); ok {
				// Resolve the group reference
				if group, exists := s.Groups[gr.Ref]; exists {
					// Create a copy of the group with updated occurrences
					resolvedGroup := &ModelGroup{
						Kind:      group.Kind,
						Particles: s.resolveParticles(group.Particles),
						MinOcc:    gr.MinOcc,
						MaxOcc:    gr.MaxOcc,
					}
					if gr.MinOcc == 0 && gr.MaxOcc == 0 {
						// Use original if not specified
						resolvedGroup.MinOcc = group.MinOcc
						resolvedGroup.MaxOcc = group.MaxOcc
					}
					ct.Content = resolvedGroup
				}
			}

			// Also resolve particles in existing ModelGroup content
			if mg, ok := ct.Content.(*ModelGroup); ok {
				mg.Particles = s.resolveParticles(mg.Particles)

				// Resolve types for inline ElementDecl particles
				s.resolveInlineElementTypes(mg.Particles)
			}

			// Resolve SimpleContent extensions
			if sc, ok := ct.Content.(*SimpleContent); ok && sc.Extension != nil {
				s.resolveExtension(ct, sc.Extension)
			}

			// Resolve ComplexContent extensions
			if cc, ok := ct.Content.(*ComplexContent); ok && cc.Extension != nil {
				s.resolveExtension(ct, cc.Extension)
			}
		}
	}

	// Also resolve types in anonymous complex types used in element declarations
	for _, elemDecl := range s.ElementDecls {
		if ct, ok := elemDecl.Type.(*ComplexType); ok {
			s.resolveTypesInComplexType(ct)
		}
	}

	// Also resolve particles in standalone groups
	for _, group := range s.Groups {
		group.Particles = s.resolveParticles(group.Particles)
	}

	// Resolve attribute types in attribute groups
	for _, attrGroup := range s.AttributeGroups {
		for _, attr := range attrGroup.Attributes {
			if attr.Type != nil {
				// Check if it's a placeholder type that needs resolution
				if st, ok := attr.Type.(*SimpleType); ok && st.Restriction == nil && st.List == nil && st.Union == nil {
					// Try to resolve the actual type
					if actualType, exists := s.TypeDefs[st.QName]; exists {
						attr.Type = actualType
					}
				}
			}
		}
	}

	// Also resolve attribute types in complex types
	for _, typeDef := range s.TypeDefs {
		if ct, ok := typeDef.(*ComplexType); ok {
			for _, attr := range ct.Attributes {
				if attr.Type != nil {
					// Check if it's a placeholder type that needs resolution
					if st, ok := attr.Type.(*SimpleType); ok && st.Restriction == nil && st.List == nil && st.Union == nil {
						// Try to resolve the actual type
						if actualType, exists := s.TypeDefs[st.QName]; exists {
							attr.Type = actualType
						}
					}
				}
			}
		}
	}

	// Build substitution group registry
	s.buildSubstitutionGroups()
}

// buildSubstitutionGroups builds the substitution group registry
func (s *Schema) buildSubstitutionGroups() {
	// Iterate through all element declarations
	for name, decl := range s.ElementDecls {
		// If element has a substitutionGroup, add it to the registry
		if decl.SubstitutionGroup.Local != "" {
			// Resolve the head element QName if needed
			headQName := decl.SubstitutionGroup
			if headQName.Namespace == "" {
				headQName.Namespace = s.TargetNamespace
			}

			// Add this element to the substitution group for the head element
			s.SubstitutionGroups[headQName] = append(s.SubstitutionGroups[headQName], decl.Name)

			// Debug: log what we're adding
			_ = name // Use the name variable to avoid unused warning
		}
	}

	// Also check imported schemas for their substitution groups
	for _, importedSchema := range s.ImportedSchemas {
		for headQName, members := range importedSchema.SubstitutionGroups {
			// Merge imported substitution groups
			existing := s.SubstitutionGroups[headQName]
			s.SubstitutionGroups[headQName] = append(existing, members...)
		}
	}
}

// isSubstitutableFor checks if actualElement can substitute for expectedElement
func (s *Schema) isSubstitutableFor(actualElement, expectedElement QName) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	//fmt.Printf("[DEBUG] isSubstitutableFor: actual=%+v, expected=%+v\n", actualElement, expectedElement)

	// Check if actualElement is in the substitution group of expectedElement
	if members, exists := s.SubstitutionGroups[expectedElement]; exists {
		//fmt.Printf("[DEBUG] Found %d members in substitution group\n", len(members))
for _, member := range members {
	//fmt.Printf("[DEBUG]   Checking member: %+v\n", member)
	if member == actualElement {
		//fmt.Printf("[DEBUG]   MATCH FOUND!\n")
				// Enforce head element block constraints
if headDecl := s.ElementDecls[expectedElement]; headDecl != nil {
					if headDecl.Block != nil {
						if headDecl.Block["substitution"] || headDecl.Block["#all"] {
return false
						}
					}
					if s.BlockDefault != nil {
						if s.BlockDefault["substitution"] || s.BlockDefault["#all"] {
return false
						}
					}
				}

				// Verify type compatibility: substituting element's type must be derived from head element's type
				actualDecl := s.ElementDecls[actualElement]
				expectedDecl := s.ElementDecls[expectedElement]

				if actualDecl == nil || expectedDecl == nil {
					// If we can't verify types, allow substitution (backward compatibility)
					return true
				}

				// Enforce final constraints on head type (e.g. final="extension") before compatibility fallback
				if headType, ok := expectedDecl.Type.(*ComplexType); ok {
					if (headType.Final != nil && (headType.Final["extension"] || headType.Final["#all"])) || (s.FinalDefault != nil && (s.FinalDefault["extension"] || s.FinalDefault["#all"])) {
if s.hasExtensionInDerivation(actualDecl.Type, expectedDecl.Type) {
							return false
						}
					}
				}

				// Both declarations exist - check type compatibility
compatible := s.isTypeCompatible(actualDecl.Type, expectedDecl.Type)
				if compatible {
					return true
				}

				// Compatibility check failed - allow substitution if either:
				// 1. Head element has no type (common for abstract elements)
				// 2. Both types exist (backward compatibility)
				return expectedDecl.Type == nil || (actualDecl.Type != nil && expectedDecl.Type != nil)
			}
		}
	}

	// Also check imported schemas
	for _, importedSchema := range s.ImportedSchemas {
		if importedSchema.isSubstitutableFor(actualElement, expectedElement) {
			return true
		}
	}

	return false
}

// isTypeCompatible checks if actualType is the same as or derives from expectedType
// Note: This function assumes the caller already holds a read lock on the schema
func (s *Schema) isTypeCompatible(actualType, expectedType Type) bool {
	visited := make(map[QName]bool, 8) // Pre-allocate for typical depth
return s.isTypeCompatibleWithCycleDetection(actualType, expectedType, visited)
}

// hasExtensionInDerivation checks if deriving actualType from expectedType uses any extension step
func (s *Schema) hasExtensionInDerivation(actualType, expectedType Type) bool {
	visited := make(map[QName]bool, 8)
	return s.hasExtensionWithCycleDetection(actualType, expectedType, visited)
}

func (s *Schema) hasExtensionWithCycleDetection(actualType, expectedType Type, visited map[QName]bool) bool {
	if actualType == nil || expectedType == nil {
		return false
	}
	if actualType.Name() == expectedType.Name() {
		return false
	}
	if visited[actualType.Name()] {
		return false
	}
	visited[actualType.Name()] = true

	// Quick check: if the type was marked as derived by extension during resolution
	if ct, ok := actualType.(*ComplexType); ok && ct.DerivedByExtension {
		return true
	}

	switch actual := actualType.(type) {
	case *ComplexType:
		if actual.Content != nil {
			if cc, ok := actual.Content.(*ComplexContent); ok {
				if cc.Extension != nil && cc.Extension.Base.Local != "" {
					// If base equals expected, we've used extension
					if cc.Extension.Base == expectedType.Name() {
						return true
					}
					if baseType := s.TypeDefs[cc.Extension.Base]; baseType != nil {
						// Extension step occurred along the path
						return true
					}
				}
				if cc.Restriction != nil && cc.Restriction.Base.Local != "" {
					if baseType := s.TypeDefs[cc.Restriction.Base]; baseType != nil {
						return s.hasExtensionWithCycleDetection(baseType, expectedType, visited)
					}
				}
			}
			if sc, ok := actual.Content.(*SimpleContent); ok {
				if sc.Extension != nil && sc.Extension.Base.Local != "" {
					if sc.Extension.Base == expectedType.Name() {
						return true
					}
					if baseType := s.TypeDefs[sc.Extension.Base]; baseType != nil {
						return true
					}
				}
				if sc.Restriction != nil && sc.Restriction.Base.Local != "" {
					if baseType := s.TypeDefs[sc.Restriction.Base]; baseType != nil {
						return s.hasExtensionWithCycleDetection(baseType, expectedType, visited)
					}
				}
			}
		}
	}
	return false
}

// isTypeCompatibleWithCycleDetection checks type compatibility with cycle detection
func (s *Schema) isTypeCompatibleWithCycleDetection(actualType, expectedType Type, visited map[QName]bool) bool {
	if actualType == nil || expectedType == nil {
		return false
	}

	actualName := actualType.Name()
	expectedName := expectedType.Name()

	// Same type is always compatible
	if actualName == expectedName {
		return true
	}

	// Cycle detection: prevent infinite recursion on circular type definitions
	if visited[actualName] {
		return false
	}
	visited[actualName] = true

	// Check if actualType derives from expectedType
	switch actual := actualType.(type) {
	case *ComplexType:
		// Check for extension or restriction in complex content
		if actual.Content != nil {
			if cc, ok := actual.Content.(*ComplexContent); ok {
				if cc.Extension != nil && cc.Extension.Base.Local != "" {
					// Note: No additional lock needed - caller already holds read lock
					baseType := s.TypeDefs[cc.Extension.Base]
					if baseType != nil {
						return s.isTypeCompatibleWithCycleDetection(baseType, expectedType, visited)
					}
				}
				if cc.Restriction != nil && cc.Restriction.Base.Local != "" {
					baseType := s.TypeDefs[cc.Restriction.Base]
					if baseType != nil {
						return s.isTypeCompatibleWithCycleDetection(baseType, expectedType, visited)
					}
				}
			}
			if sc, ok := actual.Content.(*SimpleContent); ok {
				if sc.Extension != nil && sc.Extension.Base.Local != "" {
					baseType := s.TypeDefs[sc.Extension.Base]
					if baseType != nil {
						return s.isTypeCompatibleWithCycleDetection(baseType, expectedType, visited)
					}
				}
				if sc.Restriction != nil && sc.Restriction.Base.Local != "" {
					baseType := s.TypeDefs[sc.Restriction.Base]
					if baseType != nil {
						return s.isTypeCompatibleWithCycleDetection(baseType, expectedType, visited)
					}
				}
			}
		}

	case *SimpleType:
		// Check for restriction
		if actual.Restriction != nil && actual.Restriction.Base.Local != "" {
			baseType := s.TypeDefs[actual.Restriction.Base]
			if baseType != nil {
				return s.isTypeCompatibleWithCycleDetection(baseType, expectedType, visited)
			}
		}
	}

	return false
}

// parseElement parses an element declaration
func (s *Schema) parseElement(elem xmldom.Element) error {
	return s.parseElementWithContext(elem, true)
}

// parseElementWithContext parses an element declaration with context about whether it's global
func (s *Schema) parseElementWithContext(elem xmldom.Element, isGlobal bool) error {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		// Could be a reference
		return nil
	}

	decl := &ElementDecl{
		Name: QName{
			Namespace: s.TargetNamespace,
			Local:     name,
		},
		MinOcc:      1,
		MaxOcc:      1,
		Constraints: make([]*IdentityConstraint, 0),
	}

	// Parse attributes
	if min := string(elem.GetAttribute("minOccurs")); min != "" {
		if min == "0" {
			decl.MinOcc = 0
		} else if val, err := strconv.Atoi(min); err == nil {
			decl.MinOcc = val
		}
	}

	if max := string(elem.GetAttribute("maxOccurs")); max != "" {
		if max == "unbounded" {
			decl.MaxOcc = -1
		} else if val, err := strconv.Atoi(max); err == nil {
			decl.MaxOcc = val
		}
	}

	if nillable := string(elem.GetAttribute("nillable")); nillable == "true" {
		decl.Nillable = true
	}

	if abstract := string(elem.GetAttribute("abstract")); abstract == "true" {
		decl.Abstract = true
	}

// Parse substitutionGroup attribute
	if substGroup := string(elem.GetAttribute("substitutionGroup")); substGroup != "" {
		decl.SubstitutionGroup = s.parseQName(substGroup)
	}
	// Parse block attribute
	if block := string(elem.GetAttribute("block")); block != "" {
		decl.Block = parseDerivationSet(block)
	}

	decl.Default = string(elem.GetAttribute("default"))
	decl.Fixed = string(elem.GetAttribute("fixed"))

	// Parse type
	if typeName := string(elem.GetAttribute("type")); typeName != "" {
		decl.Type = s.resolveType(typeName)
	}

	// Parse child elements for inline type definitions and identity constraints
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "key":
			if constraint := s.parseIdentityConstraint(child, KeyConstraint); constraint != nil {
				decl.Constraints = append(decl.Constraints, constraint)
			}
		case "keyref":
			if constraint := s.parseIdentityConstraint(child, KeyRefConstraint); constraint != nil {
				decl.Constraints = append(decl.Constraints, constraint)
			}
		case "unique":
			if constraint := s.parseIdentityConstraint(child, UniqueConstraint); constraint != nil {
				decl.Constraints = append(decl.Constraints, constraint)
			}
		case "simpleType":
			// Parse inline simple type
			st := s.parseInlineSimpleType(child)
			if st != nil {
				decl.Type = st
			}
		case "complexType":
			// Parse inline complex type
			ct := s.parseInlineComplexType(child)
			if ct != nil {
				decl.Type = ct
			}
		}
	}

	// Only register globally if this is a top-level element
	if isGlobal {
		s.mu.Lock()
		s.ElementDecls[decl.Name] = decl
		s.mu.Unlock()
	}

	return nil
}

// parseInlineElement parses an inline element declaration within a model group
// and returns the ElementDecl without registering it globally
func (s *Schema) parseInlineElement(elem xmldom.Element) *ElementDecl {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil
	}

	// Determine namespace based on form attribute or elementFormDefault
	form := string(elem.GetAttribute("form"))
	ns := ""
	if form == "qualified" {
		ns = s.TargetNamespace
	} else if form == "unqualified" {
		ns = ""
	} else {
		if s.ElementFormDefault == "qualified" {
			ns = s.TargetNamespace
		} else {
			ns = ""
		}
	}

	decl := &ElementDecl{
		Name: QName{
			Namespace: ns,
			Local:     name,
		},
		MinOcc:      s.parseOccurs(elem, "minOccurs", 1),
		MaxOcc:      s.parseOccurs(elem, "maxOccurs", 1),
		Constraints: make([]*IdentityConstraint, 0),
	}

	// Parse attributes
	if nillable := string(elem.GetAttribute("nillable")); nillable == "true" {
		decl.Nillable = true
	}

	if abstract := string(elem.GetAttribute("abstract")); abstract == "true" {
		decl.Abstract = true
	}

	// Parse substitutionGroup attribute (for inline elements too)
	if substGroup := string(elem.GetAttribute("substitutionGroup")); substGroup != "" {
		decl.SubstitutionGroup = s.parseQName(substGroup)
	}

	decl.Default = string(elem.GetAttribute("default"))
	decl.Fixed = string(elem.GetAttribute("fixed"))

	// Parse type
	if typeName := string(elem.GetAttribute("type")); typeName != "" {
		decl.Type = s.resolveType(typeName)
	}

	// Parse child elements for inline type definitions
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "simpleType":
			// Parse inline simple type
			st := s.parseInlineSimpleType(child)
			if st != nil {
				decl.Type = st
			}
		case "complexType":
			// Parse inline complex type
			ct := s.parseInlineComplexType(child)
			if ct != nil {
				decl.Type = ct
			}
		case "key":
			if constraint := s.parseIdentityConstraint(child, KeyConstraint); constraint != nil {
				decl.Constraints = append(decl.Constraints, constraint)
			}
		case "keyref":
			if constraint := s.parseIdentityConstraint(child, KeyRefConstraint); constraint != nil {
				decl.Constraints = append(decl.Constraints, constraint)
			}
		case "unique":
			if constraint := s.parseIdentityConstraint(child, UniqueConstraint); constraint != nil {
				decl.Constraints = append(decl.Constraints, constraint)
			}
		}
	}

	return decl
}

// parseInlineSimpleType parses an inline (anonymous) simple type definition
func (s *Schema) parseInlineSimpleType(elem xmldom.Element) *SimpleType {
	st := &SimpleType{
		QName: QName{
			Namespace: s.TargetNamespace,
			Local:     "_anonymous",
		},
	}

	// Parse restriction, list, or union
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "restriction":
			st.Restriction = s.parseRestriction(child)
		case "list":
			st.List = s.parseList(child)
		case "union":
			st.Union = s.parseUnion(child)
		}
	}

	return st
}

// parseSimpleType parses a simple type definition
func (s *Schema) parseSimpleType(elem xmldom.Element) error {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil // Anonymous type
	}

	st := &SimpleType{
		QName: QName{
			Namespace: s.TargetNamespace,
			Local:     name,
		},
	}

	// Parse restriction, list, or union
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "restriction":
			st.Restriction = s.parseRestriction(child)
		case "list":
			st.List = s.parseList(child)
		case "union":
			st.Union = s.parseUnion(child)
		}
	}

	s.mu.Lock()
	s.TypeDefs[st.QName] = st
	s.mu.Unlock()

	return nil
}

// parseInlineComplexType parses an inline (anonymous) complex type definition
func (s *Schema) parseInlineComplexType(elem xmldom.Element) *ComplexType {
ct := &ComplexType{
		QName:              QName{Namespace: s.TargetNamespace, Local: "_anonymous"},
		Attributes:         make([]*AttributeDecl, 0),
		Final:              make(map[string]bool),
		DerivedByExtension: false,
	}

	if mixed := string(elem.GetAttribute("mixed")); mixed == "true" {
		ct.Mixed = true
	}

	if abstract := string(elem.GetAttribute("abstract")); abstract == "true" {
		ct.Abstract = true
	}
	// Parse final attribute if present
	if finalAttr := string(elem.GetAttribute("final")); finalAttr != "" {
		ct.Final = parseDerivationSet(finalAttr)
	}

	// Parse content and attributes
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "simpleContent":
			sc := s.parseSimpleContent(child)
			ct.Content = sc
			// Transfer attributes from simpleContent extension to the ComplexType
			if sc.Extension != nil {
				ct.Attributes = append(ct.Attributes, sc.Extension.Attributes...)
				// Also handle anyAttribute from extension
				if sc.Extension.AnyAttribute != nil {
					ct.AnyAttribute = sc.Extension.AnyAttribute
				}
			}
		case "complexContent":
			ct.Content = s.parseComplexContent(child)
		case "sequence", "choice", "all":
			ct.Content = s.parseModelGroup(child)
		case "group":
			// Handle group references for content models
			if ref := string(child.GetAttribute("ref")); ref != "" {
				ct.Content = &GroupRef{
					Ref:    s.parseQName(ref),
					MinOcc: s.parseOccurs(child, "minOccurs", 1),
					MaxOcc: s.parseOccurs(child, "maxOccurs", 1),
				}
			}
		case "attribute":
			if attr := s.parseAttribute(child); attr != nil {
				ct.Attributes = append(ct.Attributes, attr)
			}
		case "attributeGroup":
			// Handle attribute group references
			if ref := string(child.GetAttribute("ref")); ref != "" {
				qname := s.parseQName(ref)
				ct.AttributeGroup = append(ct.AttributeGroup, qname)
			}
		case "anyAttribute":
			ct.AnyAttribute = s.parseAnyAttribute(child)
		}
	}

	return ct
}

// parseComplexType parses a complex type definition
func (s *Schema) parseComplexType(elem xmldom.Element) error {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil // Anonymous type
	}

ct := &ComplexType{
		QName:              QName{Namespace: s.TargetNamespace, Local: name},
		Attributes:         make([]*AttributeDecl, 0),
		Final:              make(map[string]bool),
		DerivedByExtension: false,
	}

	if mixed := string(elem.GetAttribute("mixed")); mixed == "true" {
		ct.Mixed = true
	}

	if abstract := string(elem.GetAttribute("abstract")); abstract == "true" {
		ct.Abstract = true
	}
	// Parse final attribute if present
	if finalAttr := string(elem.GetAttribute("final")); finalAttr != "" {
		ct.Final = parseDerivationSet(finalAttr)
	}

	// Parse content and attributes
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "simpleContent":
			sc := s.parseSimpleContent(child)
			ct.Content = sc
			// Transfer attributes from simpleContent extension to the ComplexType
			if sc.Extension != nil {
				ct.Attributes = append(ct.Attributes, sc.Extension.Attributes...)
				// Also handle anyAttribute from extension
				if sc.Extension.AnyAttribute != nil {
					ct.AnyAttribute = sc.Extension.AnyAttribute
				}
			}
		case "complexContent":
			ct.Content = s.parseComplexContent(child)
		case "sequence", "choice", "all":
			ct.Content = s.parseModelGroup(child)
		case "group":
			// Handle group references for content models
			if ref := string(child.GetAttribute("ref")); ref != "" {
				// Create a group reference particle
				ct.Content = &GroupRef{
					Ref:    s.parseQName(ref),
					MinOcc: s.parseOccurs(child, "minOccurs", 1),
					MaxOcc: s.parseOccurs(child, "maxOccurs", 1),
				}
			}
		case "attribute":
			if attr := s.parseAttribute(child); attr != nil {
				ct.Attributes = append(ct.Attributes, attr)
			}
		case "attributeGroup":
			// Handle attribute group references
			if ref := string(child.GetAttribute("ref")); ref != "" {
				qname := s.parseQName(ref)
				ct.AttributeGroup = append(ct.AttributeGroup, qname)
			}
		case "anyAttribute":
			ct.AnyAttribute = s.parseAnyAttribute(child)
		}
	}

	s.mu.Lock()
	s.TypeDefs[ct.QName] = ct
	s.mu.Unlock()

	return nil
}

// Helper methods for parsing various components

func (s *Schema) parseRestriction(elem xmldom.Element) *Restriction {
	r := &Restriction{
		Facets:     make([]FacetValidator, 0),
		Attributes: make([]*AttributeDecl, 0),
	}

	if base := string(elem.GetAttribute("base")); base != "" {
		r.Base = s.parseQName(base)
	}

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		childName := string(child.LocalName())

		// Handle inline simpleType as base
		if childName == "simpleType" && r.Base == (QName{}) {
			// Parse the inline simple type and store it as the base
			st := s.parseInlineSimpleType(child)
			if st != nil {
				// Generate a unique name for this anonymous type
				uniqName := fmt.Sprintf("_restriction_base_%d", i)
				st.QName = QName{
					Namespace: s.TargetNamespace,
					Local:     uniqName,
				}
				// Store the type
				s.mu.Lock()
				s.TypeDefs[st.QName] = st
				s.mu.Unlock()
				// Set as base type
				r.Base = st.QName
			}
			continue
		}

		// Handle complexContent restriction content (sequence/choice/all/group)
		switch childName {
		case "sequence", "choice", "all":
			r.Content = s.parseModelGroup(child)
			continue
		case "group":
			if ref := string(child.GetAttribute("ref")); ref != "" {
				r.Content = &GroupRef{
					Ref:    s.parseQName(ref),
					MinOcc: 1,
					MaxOcc: 1,
				}
			}
			continue
		case "attribute":
			if attr := s.parseAttribute(child); attr != nil {
				r.Attributes = append(r.Attributes, attr)
			}
			continue
		case "anyAttribute":
			r.AnyAttribute = &AnyAttribute{
				Namespace:       string(child.GetAttribute("namespace")),
				ProcessContents: string(child.GetAttribute("processContents")),
			}
			continue
		}

		// Parse facets (for simpleType/simpleContent restrictions)
		value := string(child.GetAttribute("value"))
		facetName := childName

		// Parse the facet using the facet parser
		if facet := ParseFacet(facetName, value); facet != nil {
			// For enumeration facets, combine multiple values
			if facetName == "enumeration" {
				// Check if we already have an enumeration facet
				var found bool
				for _, existing := range r.Facets {
					if enum, ok := existing.(*EnumerationFacet); ok {
						enum.Values = append(enum.Values, value)
						found = true
						break
					}
				}
				if !found {
					r.Facets = append(r.Facets, facet)
				}
			} else {
				r.Facets = append(r.Facets, facet)
			}
		}
	}

	return r
}

func (s *Schema) parseList(elem xmldom.Element) *List {
	list := &List{}

	// Parse itemType attribute if present
	if itemType := string(elem.GetAttribute("itemType")); itemType != "" {
		list.ItemType = s.parseQName(itemType)
	} else {
		// Look for inline simpleType child
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			child := children.Item(i)
			if child == nil || string(child.NamespaceURI()) != XSDNamespace {
				continue
			}

			if string(child.LocalName()) == "simpleType" {
				// Parse the inline simple type and store it
				st := s.parseInlineSimpleType(child)
				if st != nil {
					// Generate a unique name for this anonymous type
					uniqName := fmt.Sprintf("_list_item_%d", i)
					st.QName = QName{
						Namespace: s.TargetNamespace,
						Local:     uniqName,
					}
					// Store the type
					s.mu.Lock()
					s.TypeDefs[st.QName] = st
					s.mu.Unlock()
					// Set as item type
					list.ItemType = st.QName
					break
				}
			}
		}
	}

	return list
}

func (s *Schema) parseUnion(elem xmldom.Element) *Union {
	u := &Union{
		MemberTypes: make([]QName, 0),
	}

	// Parse memberTypes attribute if present
	if memberTypes := string(elem.GetAttribute("memberTypes")); memberTypes != "" {
		types := strings.Fields(memberTypes)
		for _, t := range types {
			u.MemberTypes = append(u.MemberTypes, s.parseQName(t))
		}
	}

	// Parse inline simpleType children
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		if string(child.LocalName()) == "simpleType" {
			// Parse the inline simple type and store it
			st := s.parseInlineSimpleType(child)
			if st != nil {
				// Generate a unique name for this anonymous type
				uniqName := fmt.Sprintf("_union_member_%d", i)
				st.QName = QName{
					Namespace: s.TargetNamespace,
					Local:     uniqName,
				}
				// Store the type
				s.mu.Lock()
				s.TypeDefs[st.QName] = st
				s.mu.Unlock()
				// Add to member types
				u.MemberTypes = append(u.MemberTypes, st.QName)
			}
		}
	}

	return u
}

func (s *Schema) parseSimpleContent(elem xmldom.Element) *SimpleContent {
	sc := &SimpleContent{}

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "extension":
			sc.Extension = s.parseExtension(child)
		case "restriction":
			sc.Restriction = s.parseRestriction(child)
		}
	}

	return sc
}

func (s *Schema) parseComplexContent(elem xmldom.Element) *ComplexContent {
	cc := &ComplexContent{}

	if mixed := string(elem.GetAttribute("mixed")); mixed == "true" {
		cc.Mixed = true
	}

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "extension":
			cc.Extension = s.parseExtension(child)
		case "restriction":
			cc.Restriction = s.parseRestriction(child)
		}
	}

	return cc
}

func (s *Schema) parseModelGroup(elem xmldom.Element) *ModelGroup {
	mg := &ModelGroup{
		MinOcc:    s.parseOccurs(elem, "minOccurs", 1),
		MaxOcc:    s.parseOccurs(elem, "maxOccurs", 1),
		Particles: make([]Particle, 0),
	}

	switch string(elem.LocalName()) {
	case "sequence":
		mg.Kind = SequenceGroup
	case "choice":
		mg.Kind = ChoiceGroup
	case "all":
		mg.Kind = AllGroup
	}

	// Parse particles
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "element":
			// Parse element particle (either declaration or reference)
			if ref := string(child.GetAttribute("ref")); ref != "" {
				// Element reference
				mg.Particles = append(mg.Particles, &ElementRef{
					Ref:    s.parseQName(ref),
					MinOcc: s.parseOccurs(child, "minOccurs", 1),
					MaxOcc: s.parseOccurs(child, "maxOccurs", 1),
				})
			} else if name := string(child.GetAttribute("name")); name != "" {
				// Inline element declaration - parse it without registering globally
				inlineElem := s.parseInlineElement(child)
				if inlineElem != nil {
					// Create an inline element declaration particle
					mg.Particles = append(mg.Particles, inlineElem)
				}
			}
		case "group":
			// Parse group reference
			if ref := string(child.GetAttribute("ref")); ref != "" {
				mg.Particles = append(mg.Particles, &GroupRef{
					Ref:    s.parseQName(ref),
					MinOcc: s.parseOccurs(child, "minOccurs", 1),
					MaxOcc: s.parseOccurs(child, "maxOccurs", 1),
				})
			}
		case "choice", "sequence", "all":
			// Parse nested model group
			nested := s.parseModelGroup(child)
			mg.Particles = append(mg.Particles, nested)
		case "any":
			// Parse xs:any wildcard
			mg.Particles = append(mg.Particles, &AnyElement{
				Namespace:       string(child.GetAttribute("namespace")),
				ProcessContents: string(child.GetAttribute("processContents")),
				MinOcc:          s.parseOccurs(child, "minOccurs", 1),
				MaxOcc:          s.parseOccurs(child, "maxOccurs", 1),
			})
		}
	}

	return mg
}

// parseOccurs parses minOccurs/maxOccurs attributes
func (s *Schema) parseOccurs(elem xmldom.Element, attr string, defaultValue int) int {
	value := string(elem.GetAttribute(xmldom.DOMString(attr)))
	if value == "" {
		return defaultValue
	}
	if value == "unbounded" {
		return -1 // -1 represents unbounded
	}
	// Try to parse as integer
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	return defaultValue
}

func (s *Schema) parseAttribute(elem xmldom.Element) *AttributeDecl {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil // Could be a reference
	}

	// Determine if global or local by inspecting parent
	parent := elem.ParentNode()
	isGlobal := false
	if parent != nil {
		if pe, ok := parent.(xmldom.Element); ok {
			if string(pe.LocalName()) == "schema" {
				isGlobal = true
			}
		}
	}

	// Determine namespace based on form/defaults
	ns := ""
	if isGlobal {
		ns = s.TargetNamespace
	} else {
		form := string(elem.GetAttribute("form"))
		if form == "qualified" {
			ns = s.TargetNamespace
		} else if form == "unqualified" || form == "" {
			if s.AttributeFormDefault == "qualified" {
				ns = s.TargetNamespace
			} else {
				ns = ""
			}
		}
	}

	attr := &AttributeDecl{
		Name: QName{
			Namespace: ns,
			Local:     name,
		},
		Use: OptionalUse,
	}

	if use := string(elem.GetAttribute("use")); use != "" {
		attr.Use = AttributeUse(use)
	}

	attr.Default = string(elem.GetAttribute("default"))
	attr.Fixed = string(elem.GetAttribute("fixed"))

	// Parse type attribute
	if typeName := string(elem.GetAttribute("type")); typeName != "" {
		typeQName := s.parseQName(typeName)
		// Look up the type in the schema
		if t, exists := s.TypeDefs[typeQName]; exists {
			attr.Type = t
		} else {
			// Create a placeholder that will be resolved in second pass
			attr.Type = &SimpleType{QName: typeQName}
		}
	}

	return attr
}

func (s *Schema) parseAnyAttribute(elem xmldom.Element) *AnyAttribute {
	return &AnyAttribute{
		Namespace:       string(elem.GetAttribute("namespace")),
		ProcessContents: string(elem.GetAttribute("processContents")),
	}
}

func (s *Schema) parseIdentityConstraint(elem xmldom.Element, kind IdentityConstraintKind) *IdentityConstraint {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil
	}

	constraint := &IdentityConstraint{
		Name:   name,
		Kind:   kind,
		Fields: make([]*Field, 0),
	}

	// For keyref, get the refer attribute
	if kind == KeyRefConstraint {
		if refer := string(elem.GetAttribute("refer")); refer != "" {
			constraint.Refer = s.parseQName(refer)
		}
	}

	// Parse selector and field elements
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "selector":
			if xpath := string(child.GetAttribute("xpath")); xpath != "" {
				constraint.Selector = &Selector{XPath: xpath}
			}
		case "field":
			if xpath := string(child.GetAttribute("xpath")); xpath != "" {
				constraint.Fields = append(constraint.Fields, &Field{XPath: xpath})
			}
		}
	}

	return constraint
}

func (s *Schema) parseExtension(elem xmldom.Element) *Extension {
	ext := &Extension{
		Base:       s.parseQName(string(elem.GetAttribute("base"))),
		Attributes: make([]*AttributeDecl, 0),
	}

	// Parse extended content
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "attribute":
			if attr := s.parseAttribute(child); attr != nil {
				ext.Attributes = append(ext.Attributes, attr)
			}
		case "sequence", "choice", "all", "group":
			if string(child.LocalName()) == "group" {
				// Handle group reference
				if ref := string(child.GetAttribute("ref")); ref != "" {
					ext.Content = &GroupRef{
						Ref:    s.parseQName(ref),
						MinOcc: 1,
						MaxOcc: 1,
					}
				}
			} else {
				ext.Content = s.parseModelGroup(child)
			}
		case "anyAttribute":
			ext.AnyAttribute = s.parseAnyAttribute(child)
		}
	}

	return ext
}

func (s *Schema) parseAttributeGroup(elem xmldom.Element) error {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil // Could be a reference
	}

	ag := &AttributeGroup{
		Name: QName{
			Namespace: s.TargetNamespace,
			Local:     name,
		},
		Attributes: make([]*AttributeDecl, 0),
	}

	// Parse attributes
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		if string(child.LocalName()) == "attribute" {
			if attr := s.parseAttribute(child); attr != nil {
				ag.Attributes = append(ag.Attributes, attr)
			}
		}
	}

	s.mu.Lock()
	s.AttributeGroups[ag.Name] = ag
	s.mu.Unlock()

	return nil
}

func (s *Schema) parseGroup(elem xmldom.Element) error {
	name := string(elem.GetAttribute("name"))
	if name == "" {
		return nil // Could be a reference
	}

	// Find the model group child
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil || string(child.NamespaceURI()) != XSDNamespace {
			continue
		}

		switch string(child.LocalName()) {
		case "sequence", "choice", "all":
			mg := s.parseModelGroup(child)
			s.mu.Lock()
			s.Groups[QName{Namespace: s.TargetNamespace, Local: name}] = mg
			s.mu.Unlock()
			return nil
		}
	}

	return nil
}

func (s *Schema) parseImport(elem xmldom.Element) error {
	imp := &Import{
		Namespace:      string(elem.GetAttribute("namespace")),
		SchemaLocation: string(elem.GetAttribute("schemaLocation")),
	}

	s.Imports = append(s.Imports, imp)
	return nil
}

func (s *Schema) parseQName(name string) QName {
	if name == "" {
		return QName{}
	}

	// Handle prefixed names
	parts := strings.SplitN(name, ":", 2)
	if len(parts) == 2 {
		prefix := parts[0]
		local := parts[1]

		// Special handling for built-in XML Schema types
		if prefix == "xs" || prefix == "xsd" {
			return QName{
				Namespace: XSDNamespace,
				Local:     local,
			}
		}

		// For other prefixes, try to resolve from the schema document
		if s.doc != nil {
			root := s.doc.DocumentElement()
			if root != nil {
				// Check all attributes for namespace declarations
				attrs := root.Attributes()
				for i := uint(0); i < attrs.Length(); i++ {
					attr := attrs.Item(i)
					if attr == nil {
						continue
					}

					attrName := string(attr.NodeName())
					// Check for xmlns:prefix
					if attrName == "xmlns:"+prefix {
						return QName{
							Namespace: string(attr.NodeValue()),
							Local:     local,
						}
					}
					// xmldom may present namespace declarations without xmlns: prefix
					// Check if this attribute name matches our prefix and has a namespace URI as value
					if attrName == prefix {
						nsValue := string(attr.NodeValue())
						// Heuristic: namespace values typically contain "://" or start with specific patterns
						if strings.Contains(nsValue, "://") || strings.Contains(nsValue, "/") || strings.Contains(nsValue, ".") {
							return QName{
								Namespace: nsValue,
								Local:     local,
							}
						}
					}
				}

				return QName{
					Namespace: s.TargetNamespace,
					Local:     local,
				}
			}
		}

		// If we can't resolve the prefix, it might be an unqualified local name
		// Don't assume target namespace for prefixed names we can't resolve
		return QName{
			Local: name, // Keep the full prefixed name as local
		}
	}

	return QName{
		Namespace: s.TargetNamespace,
		Local:     name,
	}
}

func (s *Schema) resolveType(name string) Type {
	qname := s.parseQName(name)

	s.mu.RLock()
	if t, ok := s.TypeDefs[qname]; ok {
		s.mu.RUnlock()
		return t
	}
	s.mu.RUnlock()

	// Check imported schemas if we have any
	if s.ImportedSchemas != nil {
		for _, importedSchema := range s.ImportedSchemas {
			importedSchema.mu.RLock()
			if t, ok := importedSchema.TypeDefs[qname]; ok {
				importedSchema.mu.RUnlock()
				return t
			}
			importedSchema.mu.RUnlock()
		}
	}

	// Return a placeholder that will be resolved later
	// Store the parsed QName so it can be resolved properly
	return &SimpleType{QName: qname}
}

// ResolveAttributeGroups resolves all attribute group references for a complex type
func (s *Schema) ResolveAttributeGroups(ct *ComplexType) []*AttributeDecl {
	var attrs []*AttributeDecl

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, groupRef := range ct.AttributeGroup {
		if ag, ok := s.AttributeGroups[groupRef]; ok {
			attrs = append(attrs, ag.Attributes...)
		}
	}

	return attrs
}

// resolveTypesInComplexType resolves all types in a complex type
func (s *Schema) resolveTypesInComplexType(ct *ComplexType) {
	// Check if content is a GroupRef that needs resolution
	if gr, ok := ct.Content.(*GroupRef); ok {
		// Resolve the group reference
		if group, exists := s.Groups[gr.Ref]; exists {
			// Create a copy of the group with updated occurrences
			resolvedGroup := &ModelGroup{
				Kind:      group.Kind,
				Particles: s.resolveParticles(group.Particles),
				MinOcc:    gr.MinOcc,
				MaxOcc:    gr.MaxOcc,
			}
			if gr.MinOcc == 0 && gr.MaxOcc == 0 {
				// Use original if not specified
				resolvedGroup.MinOcc = group.MinOcc
				resolvedGroup.MaxOcc = group.MaxOcc
			}
			ct.Content = resolvedGroup
		}
	}

	// Also resolve particles in existing ModelGroup content
	if mg, ok := ct.Content.(*ModelGroup); ok {
		mg.Particles = s.resolveParticles(mg.Particles)

		// Resolve types for inline ElementDecl particles
		s.resolveInlineElementTypes(mg.Particles)
	}

	// Resolve SimpleContent extensions
	if sc, ok := ct.Content.(*SimpleContent); ok && sc.Extension != nil {
		s.resolveExtension(ct, sc.Extension)
	}

	// Resolve ComplexContent extensions
	if cc, ok := ct.Content.(*ComplexContent); ok && cc.Extension != nil {
		s.resolveExtension(ct, cc.Extension)
	}
}

// resolveInlineElementTypes resolves placeholder types for inline ElementDecl particles
func (s *Schema) resolveInlineElementTypes(particles []Particle) {
	s.resolveInlineElementTypesEx(particles, make(map[*ModelGroup]bool))
}

func (s *Schema) resolveInlineElementTypesEx(particles []Particle, visited map[*ModelGroup]bool) {
	for _, p := range particles {
		switch pt := p.(type) {
		case *ElementDecl:
			// Check if this element has a placeholder type that needs resolution
			if pt.Type != nil {
				if st, ok := pt.Type.(*SimpleType); ok && st.Restriction == nil && st.List == nil && st.Union == nil {
					// This is a placeholder - try to resolve the actual type
					if actualType, exists := s.TypeDefs[st.QName]; exists {
						pt.Type = actualType
					} else if st.QName.Namespace == "" && strings.Contains(st.QName.Local, ":") {
						// The QName wasn't resolved properly - try to re-parse it
						resolvedQName := s.parseQName(st.QName.Local)
						if actualType, exists := s.TypeDefs[resolvedQName]; exists {
							pt.Type = actualType
						}
					}
				}
			}
		case *ModelGroup:
			if visited[pt] { continue }
			visited[pt] = true
			// Recursively resolve nested model groups
			s.resolveInlineElementTypesEx(pt.Particles, visited)
			delete(visited, pt)
		}
	}
}

// resolveParticles recursively resolves GroupRef particles with cycle detection
func (s *Schema) resolveParticles(particles []Particle) []Particle {
	return s.resolveParticlesWithVisitedEx(particles, make(map[QName]bool), make(map[*ModelGroup]bool))
}

// resolveParticlesWithVisited recursively resolves GroupRef particles with cycle detection (backward compat)
func (s *Schema) resolveParticlesWithVisited(particles []Particle, visitedRefs map[QName]bool) []Particle {
	return s.resolveParticlesWithVisitedEx(particles, visitedRefs, make(map[*ModelGroup]bool))
}

// resolveParticlesWithVisitedEx resolves particles guarding both GroupRef (QName) and ModelGroup pointer cycles
func (s *Schema) resolveParticlesWithVisitedEx(particles []Particle, visitedRefs map[QName]bool, visitedGroups map[*ModelGroup]bool) []Particle {
	var resolved []Particle

	for _, p := range particles {
		switch pt := p.(type) {
		case *GroupRef:
			// Check for cycles via group references
			if visitedRefs[pt.Ref] {
				// Cycle detected - keep the unresolved reference
				resolved = append(resolved, pt)
				continue
			}

			// Mark as visited
			visitedRefs[pt.Ref] = true

			// Resolve group reference
			if group, exists := s.Groups[pt.Ref]; exists {
				// Inline the group's particles
				resolvedGroup := &ModelGroup{
					Kind:      group.Kind,
					Particles: s.resolveParticlesWithVisitedEx(group.Particles, visitedRefs, visitedGroups), // Recursive resolution with visited tracking
					MinOcc:    pt.MinOcc,
					MaxOcc:    pt.MaxOcc,
				}
				if pt.MinOcc == 0 && pt.MaxOcc == 0 {
					resolvedGroup.MinOcc = group.MinOcc
					resolvedGroup.MaxOcc = group.MaxOcc
				}
				resolved = append(resolved, resolvedGroup)
			} else {
				// Keep unresolved reference
				resolved = append(resolved, pt)
			}

			// Unmark as visited when done (to allow reuse in other branches)
			delete(visitedRefs, pt.Ref)

		case *ModelGroup:
			// Guard against pointer cycles in nested model groups
			if visitedGroups[pt] {
				resolved = append(resolved, pt)
				continue
			}
			visitedGroups[pt] = true
			pt.Particles = s.resolveParticlesWithVisitedEx(pt.Particles, visitedRefs, visitedGroups)
			resolved = append(resolved, pt)
			delete(visitedGroups, pt)
		default:
			// ElementRef, AnyElement, etc. - keep as is
			resolved = append(resolved, p)
		}
	}

	return resolved
}

// resolveExtension resolves type extension/derivation
func (s *Schema) resolveExtension(ct *ComplexType, ext *Extension) {
	// Find base type
	if baseType, exists := s.TypeDefs[ext.Base]; exists {
		if baseCT, ok := baseType.(*ComplexType); ok {
			// Inherit attributes from base type
			baseAttrs := make([]*AttributeDecl, len(baseCT.Attributes))
			copy(baseAttrs, baseCT.Attributes)

			// Add extension's attributes
			ct.Attributes = append(baseAttrs, ext.Attributes...)

			// Inherit attribute groups
			ct.AttributeGroup = append(ct.AttributeGroup, baseCT.AttributeGroup...)

			// Mark derivation by extension on this type and propagate from base
			ct.DerivedByExtension = true || baseCT.DerivedByExtension

			// Handle content model extension
			if ext.Content != nil {
				// Extension adds to base content
				if baseCT.Content != nil {
					// If both are ModelGroups, combine their particles in a sequence
					var particles []Particle

					// Add base content particles
					if baseMG, ok := baseCT.Content.(*ModelGroup); ok {
						// Extract particles from base model group
						particles = append(particles, baseMG.Particles...)
					} else {
						// Base content is not a ModelGroup, add it as-is
						particles = append(particles, baseCT.Content.(Particle))
					}

					// Add extension content particles
					if extMG, ok := ext.Content.(*ModelGroup); ok {
						// Extract particles from extension model group
						particles = append(particles, extMG.Particles...)
					} else if extParticle, ok := ext.Content.(Particle); ok {
						// Extension content is a single particle
						particles = append(particles, extParticle)
					}

					if len(particles) > 0 {
						// Create a sequence containing all particles from base and extension
						sequence := &ModelGroup{
							Kind:      SequenceGroup,
							MinOcc:    1,
							MaxOcc:    1,
							Particles: particles,
						}
						ct.Content = sequence
					} else {
						ct.Content = ext.Content
					}
				} else {
					ct.Content = ext.Content
				}
			} else if baseCT.Content != nil {
				// Just inherit base content
				ct.Content = baseCT.Content
			}

			// Inherit mixed attribute
			if baseCT.Mixed {
				ct.Mixed = true
			}

			// Inherit anyAttribute
			if ct.AnyAttribute == nil && baseCT.AnyAttribute != nil {
				ct.AnyAttribute = baseCT.AnyAttribute
			}
		}
	}
}

// Type interface implementations

func (st *SimpleType) Name() QName {
	return st.QName
}

func (st *SimpleType) Validate(element xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// Get the text content of the element
	content := strings.TrimSpace(string(element.TextContent()))

	// Validate based on the simple type definition
	var err error
	if st.Union != nil {
		err = ValidateUnionType(content, st.Union, schema)
	} else if st.List != nil {
		err = ValidateListType(content, st.List, schema)
	} else if st.Restriction != nil {
		// Validate against restriction
		err = validateSimpleTypeValue(content, st, schema)
	}

	if err != nil {
		violations = append(violations, Violation{
			Element: element,
			Code:    "cvc-datatype-valid.1",
			Message: err.Error(),
		})
	}

	return violations
}

func (ct *ComplexType) Name() QName {
	return ct.QName
}

func (ct *ComplexType) Validate(element xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// Debug: uncomment to see validation flow
	// fmt.Printf("ComplexType.Validate: %s, Content: %T\n", ct.QName, ct.Content)

	// If the complex type has content, validate it
	if ct.Content != nil {
		contentViolations := ct.Content.Validate(element, schema)
		violations = append(violations, contentViolations...)
	}

	return violations
}

// Content interface implementations for use as Particles

func (sc *SimpleContent) MinOccurs() int { return 1 }
func (sc *SimpleContent) MaxOccurs() int { return 1 }
func (sc *SimpleContent) Validate(element xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// Get text content
	content := strings.TrimSpace(string(element.TextContent()))

	// Validate based on extension/restriction
	if sc.Extension != nil {
		// Extension: validate text against base type
		if sc.Extension.Base.Local != "" {
			baseType := schema.TypeDefs[sc.Extension.Base]
			if baseType != nil {
				if st, ok := baseType.(*SimpleType); ok {
					// Validate against simple type
					var err error
					if st.Union != nil {
						err = ValidateUnionType(content, st.Union, schema)
					} else if st.List != nil {
						err = ValidateListType(content, st.List, schema)
					} else if st.Restriction != nil {
						err = validateSimpleTypeValue(content, st, schema)
					}

					if err != nil {
						violations = append(violations, Violation{
							Element: element,
							Code:    "cvc-datatype-valid.1",
							Message: err.Error(),
						})
					}
				}
			}
		}
	} else if sc.Restriction != nil {
		// Restriction: validate text against restricted type with facets
		if sc.Restriction.Base.Local != "" {
			baseType := schema.TypeDefs[sc.Restriction.Base]
			if baseType != nil {
				if st, ok := baseType.(*SimpleType); ok {
					// Validate against base simple type
					var err error
					if st.Union != nil {
						err = ValidateUnionType(content, st.Union, schema)
					} else if st.List != nil {
						err = ValidateListType(content, st.List, schema)
					} else if st.Restriction != nil {
						err = validateSimpleTypeValue(content, st, schema)
					}

					if err != nil {
						violations = append(violations, Violation{
							Element: element,
							Code:    "cvc-datatype-valid.1",
							Message: err.Error(),
						})
					}
				}
			}
		}

		// Validate against restriction facets
		if len(sc.Restriction.Facets) > 0 {
			err := ValidateFacets(content, sc.Restriction.Facets, nil)
			if err != nil {
				violations = append(violations, Violation{
					Element: element,
					Code:    "cvc-facet-valid",
					Message: err.Error(),
				})
			}
		}
	}

	return violations
}

func (cc *ComplexContent) MinOccurs() int { return 1 }
func (cc *ComplexContent) MaxOccurs() int { return 1 }

// Particle interface implementations

func (er *ElementRef) MinOccurs() int { return er.MinOcc }
func (er *ElementRef) MaxOccurs() int { return er.MaxOcc }
func (er *ElementRef) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Validation is handled by the validator
	return nil
}

func (gr *GroupRef) MinOccurs() int { return gr.MinOcc }
func (gr *GroupRef) MaxOccurs() int { return gr.MaxOcc }
func (gr *GroupRef) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Resolve the group from the schema
	schema.mu.RLock()
	group, found := schema.Groups[gr.Ref]
	schema.mu.RUnlock()

	if !found {
		// Group not found - this shouldn't happen in valid schemas
		return []Violation{{
			Code:    "xsd-group-not-found",
			Message: fmt.Sprintf("Group reference '%s' not found in schema", gr.Ref),
			Element: element,
		}}
	}

	// Validate using the resolved group
	return group.Validate(element, schema)
}

func (ae *AnyElement) MinOccurs() int { return ae.MinOcc }
func (ae *AnyElement) MaxOccurs() int { return ae.MaxOcc }
func (ae *AnyElement) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Validate using wildcard validation
	return ValidateAnyElement(element, ae, schema)
}

// Particle interface implementation for inline ElementDecl
func (ed *ElementDecl) MinOccurs() int {
	if ed == nil {
		return 1
	}
	return ed.MinOcc
}
func (ed *ElementDecl) MaxOccurs() int {
	if ed == nil {
		return 1
	}
	return ed.MaxOcc
}
func (ed *ElementDecl) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Validation is handled by the validator
	return nil
}

func (mg *ModelGroup) MinOccurs() int { return mg.MinOcc }
func (mg *ModelGroup) MaxOccurs() int { return mg.MaxOcc }
func (mg *ModelGroup) Validate(element xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// Get child elements
	children := element.Children()
	var childElements []xmldom.Element
	for i := uint(0); i < children.Length(); i++ {
		if child := children.Item(i); child != nil {
			childElements = append(childElements, child)
		}
	}

switch mg.Kind {
case SequenceGroup:
	if schema != nil && schema.StrictContentModel {
		// If the entire sequence group is optional and there are no children, accept
		if mg.MinOccurs() == 0 && len(childElements) == 0 {
			return nil
		}
		violations = mg.validateSequenceStrict(childElements, schema)
	} else {
		violations = mg.validateSequence(childElements, schema)
	}
case ChoiceGroup:
	// Fast-path: single wildcard alternative behaves like allowing that wildcard
	if len(mg.Particles) == 1 {
		if wc, ok := mg.Particles[0].(*AnyElement); ok {
			matchCount := 0
			for _, ch := range childElements {
				if MatchesWildcard(ch, wc.Namespace, schema.TargetNamespace) {
					matchCount++
					violations = append(violations, ValidateAnyElement(ch, wc, schema)...)
				} else {
					// This child cannot be produced by the single wildcard alternative
					violations = append(violations, Violation{ Element: ch, Code: "cvc-complex-type.2.4.d", Message: fmt.Sprintf("Unexpected element '%s'", ch.LocalName()) })
				}
			}
			// Enforce wildcard occurrence bounds
			violations = append(violations, ValidateWildcardOccurrences(matchCount, wc)...)
			break
		}
	}
	if schema != nil && schema.StrictContentModel {
		violations = mg.validateChoiceStrict(childElements, schema)
	} else {
		violations = mg.validateChoice(childElements, schema)
	}
case AllGroup:
	if schema != nil && schema.StrictContentModel {
		violations = mg.validateAllStrict(childElements, schema)
	} else {
		violations = mg.validateAll(childElements, schema)
	}
}

return violations
}

// Strict variants (initially delegate to existing implementations)
func (mg *ModelGroup) validateSequenceStrict(children []xmldom.Element, schema *Schema) []Violation {
	type altMatch struct{ consumed int; violations []Violation }
	type choiceFrame struct{
		particleIndex int
		idxBefore int
		altMatches []altMatch
		altChosen int
		violationsLen int
	}
	var violations []Violation
	idx := 0
	occ := 0
	minG := mg.MinOccurs()
	maxG := mg.MaxOccurs()

	// If the entire sequence group is optional and there are no children, accept
	if minG == 0 && len(children) == 0 {
		return nil
	}

for maxG == -1 || occ < maxG {
		stack := make([]choiceFrame, 0, 4)
		startIdx := idx
		matchedAll := true
		allOptional := true

		for pi := 0; pi < len(mg.Particles); {
			p := mg.Particles[pi]
			var next Particle
			if pi+1 < len(mg.Particles) { next = mg.Particles[pi+1] }

			minP := p.MinOccurs()
			maxP := p.MaxOccurs()
			if minP > 0 { allOptional = false }
			count := 0

			// Special handling for wildcard in sequences: be conservative to avoid over-greedy consumption
			if ae, ok := p.(*AnyElement); ok {
				for (ae.MaxOccurs() == -1 || count < ae.MaxOccurs()) && idx < len(children) {
					// If the next particle is present and the next child matches it, stop consuming wildcard
					if next != nil && mg.elementMatchesParticle(children[idx], next, schema) {
						break
					}
					// Otherwise consume one wildcard element
					violations = append(violations, ValidateAnyElement(children[idx], ae, schema)...)
					count++
					idx++
				}
			} else if choiceGrp, ok := p.(*ModelGroup); ok && choiceGrp.Kind == ChoiceGroup {
				// compute alternatives
				alts := make([]altMatch, 0, len(choiceGrp.Particles))
				for _, alt := range choiceGrp.Particles {
					m, c, v := mg.matchParticle(alt, children[idx:], schema)
					if c > 0 && m >= alt.MinOccurs() {
						// prefer those that allow next to match if required handled indirectly via backtrack
						alts = append(alts, altMatch{consumed: c, violations: v})
					}
				}
				if len(alts) == 0 {
					if minP == 0 { pi++; continue } // optional choice
					// will trigger backtrack/fail below by count<minP
				} else {
					// pick longest first
					best := 0
					for i := 1; i < len(alts); i++ { if alts[i].consumed > alts[best].consumed { best = i } }
					// push frame for backtracking
					stack = append(stack, choiceFrame{particleIndex: pi, idxBefore: idx, altMatches: alts, altChosen: best, violationsLen: len(violations)})
					// apply
					violations = append(violations, alts[best].violations...)
					count++
					idx += alts[best].consumed
					pi++
					continue
				}
				for (maxP == -1 || count < maxP) && idx < len(children) {
					c, v := mg.matchChoiceWithLookahead(choiceGrp, children[idx:], next, schema)
					if c == 0 {
						break
					}
					violations = append(violations, v...)
					count++
					idx += c
				}
			} else {
				for (maxP == -1 || count < maxP) && idx < len(children) {
					m, c, pv := mg.matchParticle(p, children[idx:], schema)
					if m == 0 || c == 0 {
						break
					}
					// Only append violations when we actually consume children
					if c > 0 {
						violations = append(violations, pv...)
					}
					count += m
					idx += c
				}
			}

			if count < minP {
				// backtrack if possible
				btDone := false
				for !btDone && len(stack) > 0 {
					// pop
					frame := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if frame.altChosen+1 < len(frame.altMatches) {
						// try next alt
						idx = frame.idxBefore
						violations = violations[:frame.violationsLen]
						// apply next alt
						nextIdx := frame.altChosen+1
						nextAlt := frame.altMatches[nextIdx]
						violations = append(violations, nextAlt.violations...)
						idx += nextAlt.consumed
						// push updated frame
						frame.altChosen = nextIdx
						stack = append(stack, frame)
						// continue from particle after this choice
						pi = frame.particleIndex + 1
						btDone = true
						continue
					}
				}
				if !btDone {
					matchedAll = false
					break
				}
				// satisfied by backtrack; proceed to next particle
				continue
			}
			// advance particle index when current particle satisfied
			pi++
		}

		if matchedAll {
			occ++
			// Zero-length occurrence: prevent infinite loop (when all particles optional)
			if idx == startIdx || allOptional {
				break
			}
			continue
		}

		// Revert consumption for the failed occurrence and stop
		idx = startIdx
		break
	}

	if occ < minG {
		violations = append(violations, Violation{Code: "cvc-complex-type.2.4.b", Message: fmt.Sprintf("Expected at least %d occurrence(s)", minG)})
	}

	// Any remaining children are unexpected
	for idx < len(children) {
		violations = append(violations, Violation{Element: children[idx], Code: "cvc-complex-type.2.4.d", Message: fmt.Sprintf("Unexpected element '%s'", children[idx].LocalName())})
		idx++
	}
	return violations
}
func (mg *ModelGroup) validateChoiceStrict(children []xmldom.Element, schema *Schema) []Violation {
	var violations []Violation
	i := 0
	reps := 0
	min := mg.MinOccurs()
	max := mg.MaxOccurs()

	// If entire choice is optional and no children, accept
	if min == 0 && len(children) == 0 {
		return nil
	}

	for i < len(children) {
		bestConsumed := 0
		var bestViolations []Violation

		for _, alt := range mg.Particles {
			m, c, v := mg.matchParticle(alt, children[i:], schema)
			if m >= alt.MinOccurs() && c > 0 {
				// Pick the alternative that consumes the most children
				if c > bestConsumed {
					bestConsumed = c
					bestViolations = v
				}
			}
		}

		if bestConsumed == 0 {
			break
		}

		violations = append(violations, bestViolations...)
		i += bestConsumed
		reps++
		if max != -1 && reps >= max {
			break
		}
	}

	if reps < min {
		violations = append(violations, Violation{Code: "cvc-complex-type.2.4.b", Message: fmt.Sprintf("Expected at least %d choice occurrence(s)", min)})
	}
	// leftover children are unexpected here
	for i < len(children) {
		violations = append(violations, Violation{Element: children[i], Code: "cvc-complex-type.2.4.d", Message: fmt.Sprintf("Unexpected element '%s'", children[i].LocalName())})
		i++
	}
	return violations
}
func (mg *ModelGroup) validateAllStrict(children []xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// Track occurrences per particle (index-based)
	counts := make(map[int]int, len(mg.Particles))

	// Helper to get maxOcc for particle (treat -1 as unbounded)
	maxOcc := func(p Particle) int { return p.MaxOccurs() }

	// Try to match each child to one unmatched/under-max particle
	for _, child := range children {
		matched := false
		for i, particle := range mg.Particles {
			// Enforce per-particle max
			m := counts[i]
			mx := maxOcc(particle)
			if mx != -1 && m >= mx {
				continue
			}
			if mg.elementMatchesParticle(child, particle, schema) {
				counts[i] = m + 1
				matched = true

				// Validate matched child's type similarly to non-strict path
				if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl && elemDecl.Type != nil {
					violations = append(violations, elemDecl.Type.Validate(child, schema)...)
				} else if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
					if decl, exists := schema.ElementDecls[elemRef.Ref]; exists && decl.Type != nil {
						violations = append(violations, decl.Type.Validate(child, schema)...)
					}
				} else if wildcard, isWildcard := particle.(*AnyElement); isWildcard {
					violations = append(violations, ValidateAnyElement(child, wildcard, schema)...)
				}
				break
			}
		}
		if !matched {
			violations = append(violations, Violation{
				Element: child,
				Code:    "cvc-complex-type.2.4.d",
				Message: fmt.Sprintf("Unexpected element '%s'", child.LocalName()),
			})
		}
	}

	// Enforce per-particle minOccurs
	for i, particle := range mg.Particles {
		if counts[i] < particle.MinOccurs() {
			violations = append(violations, Violation{
				Code:    "cvc-complex-type.2.4.a",
				Message: "Required element missing in 'all' group",
			})
		}
	}

	return violations
}

// Existing non-strict implementation
func (mg *ModelGroup) validateSequence(children []xmldom.Element, schema *Schema) []Violation {
	var violations []Violation
	childIndex := 0
	particleIndex := 0

	// Process children
	for childIndex < len(children) && particleIndex < len(mg.Particles) {
		child := children[childIndex]
		particle := mg.Particles[particleIndex]

		// Special handling for group references
		if groupRef, isGroupRef := particle.(*GroupRef); isGroupRef {
			// Resolve the group reference
			if resolvedGroup, exists := schema.Groups[groupRef.Ref]; exists {
				// Treat it as a nested model group
				var nestedViolations []Violation
				var consumed int

				switch resolvedGroup.Kind {
				case ChoiceGroup:
					consumed, nestedViolations = mg.matchChoiceGroup(resolvedGroup, children[childIndex:], schema)
				case SequenceGroup:
					nestedViolations = resolvedGroup.validateSequence(children[childIndex:], schema)
					consumed = mg.countConsumedByGroup(resolvedGroup, children[childIndex:], schema)
				case AllGroup:
					nestedViolations = resolvedGroup.validateAll(children[childIndex:], schema)
					consumed = mg.countConsumedByGroup(resolvedGroup, children[childIndex:], schema)
				}

				childIndex += consumed
				// Only add violations if the group is required OR if it consumed some children
				if groupRef.MinOcc > 0 || consumed > 0 {
					violations = append(violations, nestedViolations...)
				} else {
					// Group is optional and didn't consume - only add real constraint violations
					for _, v := range nestedViolations {
						if v.Code == "cvc-wildcard.2" {
							violations = append(violations, v)
						}
					}
				}
			}
			particleIndex++
		} else if nestedGroup, isModelGroup := particle.(*ModelGroup); isModelGroup {
			// Special handling for nested model groups (inline groups)
			// When a ModelGroup is a particle in a sequence, we need to inline its validation
			// The nested group should consume children according to its own kind (sequence/choice/all)
			var nestedViolations []Violation
			var consumed int

			switch nestedGroup.Kind {
			case ChoiceGroup:
				// For a choice group, keep consuming children that match any particle
				consumed, nestedViolations = mg.matchChoiceGroup(nestedGroup, children[childIndex:], schema)
			case SequenceGroup:
				// For a sequence group, validate in order
				nestedViolations = nestedGroup.validateSequence(children[childIndex:], schema)
				// Count consumed by checking how many children matched
				consumed = mg.countConsumedByGroup(nestedGroup, children[childIndex:], schema)
			case AllGroup:
				// For an all group, similar to sequence
				nestedViolations = nestedGroup.validateAll(children[childIndex:], schema)
				consumed = mg.countConsumedByGroup(nestedGroup, children[childIndex:], schema)
			}

			childIndex += consumed
			// Only add violations if the group is required OR if it consumed some children
			if nestedGroup.MinOccurs() > 0 || consumed > 0 {
				violations = append(violations, nestedViolations...)
			} else {
				// Group is optional and didn't consume - only add violations that are real constraint errors
				// Namespace constraint violations (cvc-wildcard.2) are real errors
				// "Unexpected element" violations (cvc-complex-type.2.4.d) from optional content can be ignored
				for _, v := range nestedViolations {
					if v.Code == "cvc-wildcard.2" {
						// Namespace constraint violation - this is a real error
						violations = append(violations, v)
					}
				}
			}
			particleIndex++
		} else if wildcard, isWildcard := particle.(*AnyElement); isWildcard {
			// Check if child matches wildcard
			if MatchesWildcard(child, wildcard.Namespace, schema.TargetNamespace) {
				// Child matches wildcard, consume as many as possible
				matched, consumed, wildcardViolations := mg.matchWildcard(wildcard, children[childIndex:], schema)
				childIndex += consumed
				violations = append(violations, wildcardViolations...)

				// Check occurrence constraints
				if matched < wildcard.MinOcc {
					violations = append(violations, Violation{
						Code:    "cvc-complex-type.2.4.b",
						Message: fmt.Sprintf("Expected at least %d wildcard match(es)", wildcard.MinOcc),
					})
				}
				particleIndex++
			} else {
				// Child doesn't match wildcard namespace constraint
				childNS := string(child.NamespaceURI())
				childName := string(child.LocalName())

				if wildcard.MinOcc == 0 {
					// Wildcard is optional, skip to next particle without error
					// Check if element matches next particle
					if particleIndex+1 < len(mg.Particles) {
						nextParticle := mg.Particles[particleIndex+1]
						if mg.elementMatchesParticle(child, nextParticle, schema) {
							// Element matches next particle, skip wildcard
							particleIndex++
							continue
						}
					}
					// Element doesn't match next particle either
					// Since wildcard is optional and we're at the end of particles or
					// the element doesn't match the next particle, just skip the wildcard
					// The element will be handled by remaining children logic
					particleIndex++
				} else {
					// Required wildcard doesn't match
					violations = append(violations, Violation{
						Element: child,
						Code:    "cvc-wildcard.2",
						Message: fmt.Sprintf("Element '{%s}%s' is not allowed by the namespace constraint '%s'",
							childNS, childName, wildcard.Namespace),
					})
					childIndex++
				}
			}
		} else {
			// Regular particle
			matched, consumed, particleViolations := mg.matchParticle(particle, children[childIndex:], schema)
			violations = append(violations, particleViolations...)

			if consumed > 0 {
				// Particle consumed some children

				// If this is an ElementRef (not inline ElementDecl), validate the matched elements
				if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
					// For ElementRef, validate each matched element
					for i := 0; i < consumed; i++ {
						childElem := children[childIndex+i]

						// Get the actual element's QName (might be substituted)
						actualQName := QName{
							Namespace: string(childElem.NamespaceURI()),
							Local:     string(childElem.LocalName()),
						}

						// Look up the actual element's declaration (not the referenced one)
						// This handles substitution groups properly
						actualDecl, exists := schema.ElementDecls[actualQName]
						if !exists && actualQName.Namespace == "" {
							// Try with target namespace
							actualQName.Namespace = schema.TargetNamespace
							actualDecl, exists = schema.ElementDecls[actualQName]
						}

						// Get the declaration to use for validation (actual or referenced)
						var declToValidate *ElementDecl
						if exists {
							declToValidate = actualDecl
						} else if refDecl, refExists := schema.ElementDecls[elemRef.Ref]; refExists {
							declToValidate = refDecl
						}

						// Validate fixed and default values
						if declToValidate != nil {
							fixedDefaultViolations := ValidateElementFixedDefault(childElem, declToValidate)
							violations = append(violations, fixedDefaultViolations...)
						}

						// Use the actual element's type if found, otherwise fall back to referenced type
						var typeToValidate Type
						if exists && actualDecl.Type != nil {
							typeToValidate = actualDecl.Type
						} else if refDecl, refExists := schema.ElementDecls[elemRef.Ref]; refExists {
							typeToValidate = refDecl.Type
						}

						if typeToValidate != nil {
							typeViolations := typeToValidate.Validate(childElem, schema)
							violations = append(violations, typeViolations...)
						}
					}
				}

				childIndex += consumed

				// Check occurrence constraints
				minOcc := particle.MinOccurs()
				maxOcc := particle.MaxOccurs()
				if matched < minOcc {
					violations = append(violations, Violation{
						Code:    "cvc-complex-type.2.4.b",
						Message: fmt.Sprintf("Expected at least %d occurrence(s)", minOcc),
					})
				}
				if maxOcc != -1 && matched > maxOcc {
					violations = append(violations, Violation{
						Code:    "cvc-complex-type.2.4.d",
						Message: fmt.Sprintf("Expected at most %d occurrence(s)", maxOcc),
					})
				}
				particleIndex++
			} else {
				// Particle didn't match
				if particle.MinOccurs() == 0 {
					// Optional particle, try next
					particleIndex++
				} else {
					// Required particle didn't match
					// Check if this element would have matched a preceding wildcard but violated namespace constraint
					wildcardViolation := false
					for i := 0; i < particleIndex; i++ {
						if wildcard, isWildcard := mg.Particles[i].(*AnyElement); isWildcard {
							// Check if element doesn't match wildcard's namespace constraint
							if !MatchesWildcard(child, wildcard.Namespace, schema.TargetNamespace) {
								// Element violates wildcard namespace constraint
								childNS := string(child.NamespaceURI())
								childName := string(child.LocalName())
								violations = append(violations, Violation{
									Element: child,
									Code:    "cvc-wildcard.2",
									Message: fmt.Sprintf("Element '{%s}%s' is not allowed by the namespace constraint '%s'",
										childNS, childName, wildcard.Namespace),
								})
								wildcardViolation = true
								break
							}
						}
					}

					// If no wildcard violation was found, report as unexpected element
					if !wildcardViolation {
						violations = append(violations, Violation{
							Element: child,
							Code:    "cvc-complex-type.2.4.d",
							Message: fmt.Sprintf("Unexpected element '%s'", child.LocalName()),
						})
					}
					childIndex++
				}
			}
		}
	}

	// Check remaining particles are optional
	for particleIndex < len(mg.Particles) {
		particle := mg.Particles[particleIndex]
		if particle.MinOccurs() > 0 {
			violations = append(violations, Violation{
				Code:    "cvc-complex-type.2.4.b",
				Message: "Required element missing",
			})
		}
		particleIndex++
	}

	// Check remaining children
	for childIndex < len(children) {
		child := children[childIndex]

		// Check if this element would have matched a wildcard but violated namespace constraint
		wildcardViolation := false
		for _, particle := range mg.Particles {
			if wildcard, isWildcard := particle.(*AnyElement); isWildcard {
				// Check if element doesn't match wildcard's namespace constraint
				if !MatchesWildcard(child, wildcard.Namespace, schema.TargetNamespace) {
					// Element violates wildcard namespace constraint
					childNS := string(child.NamespaceURI())
					childName := string(child.LocalName())
					violations = append(violations, Violation{
						Element: child,
						Code:    "cvc-wildcard.2",
						Message: fmt.Sprintf("Element '{%s}%s' is not allowed by the namespace constraint '%s'",
							childNS, childName, wildcard.Namespace),
					})
					wildcardViolation = true
					break
				}
			}
		}

		// If no wildcard violation was found, report as unexpected element
		if !wildcardViolation {
			violations = append(violations, Violation{
				Element: child,
				Code:    "cvc-complex-type.2.4.d",
				Message: fmt.Sprintf("Unexpected element '%s'", child.LocalName()),
			})
		}
		childIndex++
	}

	return violations
}

func (mg *ModelGroup) validateChoice(children []xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// At least one particle must match
	for _, particle := range mg.Particles {
		matched, consumed, particleViolations := mg.matchParticle(particle, children, schema)
		if matched > 0 {
			// Found a match - collect violations from matchParticle
			violations = append(violations, particleViolations...)

			// For ElementRef, we still need to validate since matchParticle doesn't handle it
			if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
				// For ElementRef, look up the global declaration and validate
				if decl, exists := schema.ElementDecls[elemRef.Ref]; exists && decl.Type != nil {
					for i := 0; i < consumed; i++ {
						childElem := children[i]
						typeViolations := decl.Type.Validate(childElem, schema)
						violations = append(violations, typeViolations...)
					}
				}
			}

			if consumed == len(children) {
				return violations // All children consumed by this choice
			}
		}
	}

	violations = append(violations, Violation{
		Code:    "cvc-complex-type.2.4.a",
		Message: "Content does not match any choice alternative",
	})
	return violations
}

func (mg *ModelGroup) validateAll(children []xmldom.Element, schema *Schema) []Violation {
	// All particles must appear exactly once in any order
	var violations []Violation
	matched := make(map[int]bool)

	for _, child := range children {
		found := false
		for i, particle := range mg.Particles {
			if !matched[i] && mg.elementMatchesParticle(child, particle, schema) {
				matched[i] = true
				found = true

				// Validate the matched element's type
				if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl {
					// Check if element is abstract
					if elemDecl.Abstract {
						violations = append(violations, Violation{
							Element: child,
							Code:    "cvc-elt.2",
							Message: fmt.Sprintf("Element '%s' is abstract and cannot be used directly in instance documents", child.LocalName()),
						})
					}
					if elemDecl.Type != nil {
						typeViolations := elemDecl.Type.Validate(child, schema)
						violations = append(violations, typeViolations...)
					}
				} else if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
					// For ElementRef, look up the global declaration and validate
					if decl, exists := schema.ElementDecls[elemRef.Ref]; exists {
						// Check if element is abstract (only if actual element matches the ref)
						actualQName := QName{Namespace: string(child.NamespaceURI()), Local: string(child.LocalName())}
						if actualQName == elemRef.Ref && decl.Abstract {
							violations = append(violations, Violation{
								Element: child,
								Code:    "cvc-elt.2",
								Message: fmt.Sprintf("Element '%s' is abstract and cannot be used directly in instance documents", child.LocalName()),
							})
						}
						if decl.Type != nil {
							typeViolations := decl.Type.Validate(child, schema)
							violations = append(violations, typeViolations...)
						}
					}
				}

				break
			}
		}
		if !found {
			violations = append(violations, Violation{
				Element: child,
				Code:    "cvc-complex-type.2.4.a",
				Message: fmt.Sprintf("Unexpected element '%s' in 'all' group", child.LocalName()),
			})
		}
	}

	// Check all required particles were found
	for i, particle := range mg.Particles {
		if !matched[i] && particle.MinOccurs() > 0 {
			violations = append(violations, Violation{
				Code:    "cvc-complex-type.2.4.a",
				Message: "Required element missing in 'all' group",
			})
		}
	}

	return violations
}

// matchChoiceGroup handles a choice group as a particle in a sequence
// It consumes children that match any particle in the choice, respecting occurrence constraints
func (mg *ModelGroup) matchChoiceGroup(choiceGroup *ModelGroup, children []xmldom.Element, schema *Schema) (consumed int, violations []Violation) {
	// For a choice group in a sequence, keep matching children against any of the choice particles
	// until no more children match
	for i := 0; i < len(children); i++ {
		child := children[i]
		matched := false

		// Try to match against any particle in the choice
		for _, particle := range choiceGroup.Particles {
			if mg.elementMatchesParticle(child, particle, schema) {
				matched = true
				consumed++

				// Validate the matched element's type
				if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl && elemDecl.Type != nil {
					typeViolations := elemDecl.Type.Validate(child, schema)
					violations = append(violations, typeViolations...)
				} else if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
					// Look up the element declaration and validate
					actualQName := QName{
						Namespace: string(child.NamespaceURI()),
						Local:     string(child.LocalName()),
					}
					if decl, exists := schema.ElementDecls[actualQName]; exists && decl.Type != nil {
						typeViolations := decl.Type.Validate(child, schema)
						violations = append(violations, typeViolations...)
					} else if decl, exists := schema.ElementDecls[elemRef.Ref]; exists && decl.Type != nil {
						typeViolations := decl.Type.Validate(child, schema)
						violations = append(violations, typeViolations...)
					}
				} else if wildcard, isWildcard := particle.(*AnyElement); isWildcard {
					// Validate wildcard element according to processContents
					wildcardViolations := ValidateAnyElement(child, wildcard, schema)
					violations = append(violations, wildcardViolations...)
				}
				break // Found a match, stop trying other particles
			}
		}

		if !matched {
			// No particle in the choice matched this child
			// Check if there's a wildcard (possibly nested in groups) that explains the failure
			// This provides better error messages
			wildcardFound := false
			var findWildcard func([]Particle) *AnyElement
			findWildcard = func(particles []Particle) *AnyElement {
				for _, p := range particles {
					if wc, ok := p.(*AnyElement); ok {
						return wc
					}
					if nested, ok := p.(*ModelGroup); ok {
						if wc := findWildcard(nested.Particles); wc != nil {
							return wc
						}
					}
				}
				return nil
			}

			if wildcard := findWildcard(choiceGroup.Particles); wildcard != nil {
				// Found a wildcard in the choice (possibly nested)
				// Report namespace constraint violation
				childNS := string(child.NamespaceURI())
				childName := string(child.LocalName())
				violations = append(violations, Violation{
					Element: child,
					Code:    "cvc-wildcard.2",
					Message: fmt.Sprintf("Element '{%s}%s' is not allowed by the namespace constraint '%s'",
						childNS, childName, wildcard.Namespace),
				})
				consumed++ // Consume the invalid element to continue validation
				wildcardFound = true
			}

			if !wildcardFound {
				// Stop consuming (choice can't match this element)
				break
			}
		}
	}

	return consumed, violations
}

// countConsumedByGroup counts how many children a group consumed during validation
func (mg *ModelGroup) countConsumedByGroup(group *ModelGroup, children []xmldom.Element, schema *Schema) int {
	consumed := 0
	switch group.Kind {
	case SequenceGroup:
		// For a sequence, validate and count matched elements
		childIndex := 0
		for _, particle := range group.Particles {
			if childIndex >= len(children) {
				break
			}
			matched, cons, _ := mg.matchParticle(particle, children[childIndex:], schema)
			childIndex += cons
			_ = matched // unused but returned by matchParticle
		}
		consumed = childIndex

	case ChoiceGroup:
		// For a choice, count consecutive matches against any particle
		for i := 0; i < len(children); i++ {
			matched := false
			for _, particle := range group.Particles {
				if mg.elementMatchesParticle(children[i], particle, schema) {
					matched = true
					consumed++
					break
				}
			}
			if !matched {
				break
			}
		}

	case AllGroup:
		// For an all group, count all elements that match any particle
		matchedParticles := make(map[int]bool)
		for i := 0; i < len(children); i++ {
			for j, particle := range group.Particles {
				if !matchedParticles[j] && mg.elementMatchesParticle(children[i], particle, schema) {
					matchedParticles[j] = true
					consumed++
					break
				}
			}
		}
	}

	return consumed
}

func (mg *ModelGroup) matchParticle(particle Particle, children []xmldom.Element, schema *Schema) (matched int, consumed int, violations []Violation) {
	// Handle wildcards specially
	if wildcard, isWildcard := particle.(*AnyElement); isWildcard {
		matched, consumed, violations = mg.matchWildcard(wildcard, children, schema)
		return matched, consumed, violations
	}

	// Handle nested ModelGroups specially
	if nestedGroup, isModelGroup := particle.(*ModelGroup); isModelGroup {
		minG := nestedGroup.MinOccurs()
		maxG := nestedGroup.MaxOccurs()
		occ := 0
		offset := 0
		for maxG == -1 || occ < maxG {
			if offset >= len(children) { break }
			c, ok := mg.matchGroupOnce(nestedGroup, children[offset:], schema)
			if !ok {
				break
			}
			if c == 0 {
				// Zero-length match (all children optional/prohibited) counts as one occurrence; avoid infinite loop
				occ++
				break
			}
			occ++
			offset += c
		}
		matched = occ
		consumed = offset
		if matched < minG {
			violations = append(violations, Violation{ Code: "cvc-complex-type.2.4.b", Message: "Required group missing" })
		}
		return matched, consumed, violations
	}

	// Handle GroupRef similarly to nested model groups
	if gr, isGroupRef := particle.(*GroupRef); isGroupRef {
		// Resolve group reference
		schema.mu.RLock()
		group, exists := schema.Groups[gr.Ref]
		schema.mu.RUnlock()
		if !exists {
			return 0, 0, []Violation{{ Code: "xsd-group-not-found", Message: fmt.Sprintf("Group reference '%s' not found", gr.Ref) }}
		}
		// Use group's structure with occurrences from the reference
		minG := gr.MinOcc
		maxG := gr.MaxOcc
		occ := 0
		offset := 0
		for maxG == -1 || occ < maxG {
			if offset >= len(children) { break }
			c, ok := mg.matchGroupOnce(group, children[offset:], schema)
			if !ok {
				break
			}
			if c == 0 {
				occ++
				break
			}
			offset += c
			occ++
		}
		matched = occ
		consumed = offset
		if matched < minG {
			violations = append(violations, Violation{ Code: "cvc-complex-type.2.4.b", Message: "Required group missing" })
		}
		return matched, consumed, violations
	}

	// Handle inline ElementDecl specially - it can match and validate
	if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl {
		for i := 0; i < len(children); i++ {
			child := children[i]
			elemQName := QName{
				Namespace: string(child.NamespaceURI()),
				Local:     string(child.LocalName()),
			}
			// Check both direct match and substitution groups
			if elemQName == elemDecl.Name || schema.isSubstitutableFor(elemQName, elemDecl.Name) {
				// Element matches the inline declaration (or can substitute for it)
				matched++
				consumed++

				// Check if actual element is different (substitution group)
				actualQName := QName{
					Namespace: string(child.NamespaceURI()),
					Local:     string(child.LocalName()),
				}

				// Get the actual element declaration (for substitution groups)
				var actualDecl *ElementDecl = elemDecl
				if actualQName != elemDecl.Name {
					if foundDecl, exists := schema.ElementDecls[actualQName]; exists {
						actualDecl = foundDecl
					}
				}

				// Validate fixed and default values
				fixedDefaultViolations := ValidateElementFixedDefault(child, actualDecl)
				violations = append(violations, fixedDefaultViolations...)

				// Validate against its type
				if actualDecl.Type != nil {
					typeViolations := actualDecl.Type.Validate(child, schema)
					violations = append(violations, typeViolations...)
				}

				maxOcc := elemDecl.MaxOcc
				if maxOcc != -1 && matched >= maxOcc {
					break
				}
			} else {
				break // Stop at first non-match for sequence
			}
		}
		return matched, consumed, violations
	}

	// Handle ElementRef explicitly to both match and validate types/defaults
	if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
		for i := 0; i < len(children); i++ {
			child := children[i]
			if mg.elementMatchesParticle(child, elemRef, schema) {
				matched++
				consumed++

				// Determine actual declaration: prefer child's own global decl (handles substitution), fallback to referenced decl
				actualQName := QName{Namespace: string(child.NamespaceURI()), Local: string(child.LocalName())}
				declToValidate, exists := schema.ElementDecls[actualQName]
				if !exists && actualQName.Namespace == "" {
					// Try with target namespace when instance has no namespace
					actualQName.Namespace = schema.TargetNamespace
					declToValidate, exists = schema.ElementDecls[actualQName]
				}
				if !exists {
					// Fallback to referenced declaration
					declToValidate = schema.ElementDecls[elemRef.Ref]
				}

				if declToValidate != nil {
					// Validate fixed/default
					violations = append(violations, ValidateElementFixedDefault(child, declToValidate)...)
					// Validate type
					if declToValidate.Type != nil {
						violations = append(violations, declToValidate.Type.Validate(child, schema)...)
					}
				}

				// Enforce maxOccurs
				mx := elemRef.MaxOcc
				if mx != -1 && matched >= mx {
					break
				}
			} else {
				break
			}
		}
		return matched, consumed, violations
	}

	// Count how many children match this particle
	for i := 0; i < len(children); i++ {
		if mg.elementMatchesParticle(children[i], particle, schema) {
			matched++
			consumed++
			maxOcc := particle.MaxOccurs()
			if maxOcc != -1 && matched >= maxOcc {
				break
			}
		} else {
			break // Stop at first non-match for sequence
		}
	}
return
}

// matchGroupOnce tries to match exactly one occurrence of a nested group against the head of children.
// It returns the number of consumed children and whether the match was successful.
func (mg *ModelGroup) matchGroupOnce(group *ModelGroup, children []xmldom.Element, schema *Schema) (consumed int, ok bool) {
	switch group.Kind {
	case SequenceGroup:
		return mg.matchSequenceOnce(group, children, schema)
	case AllGroup:
		return mg.matchAllOnce(group, children, schema)
	case ChoiceGroup:
		return mg.matchChoiceOnce(group, children, schema)
	}
	return 0, false
}

// matchSequenceOnce matches one occurrence of a sequence group
func (mg *ModelGroup) matchSequenceOnce(group *ModelGroup, children []xmldom.Element, schema *Schema) (consumed int, ok bool) {
	idx := 0
	allOptional := true
	for _, p := range group.Particles {
		min := p.MinOccurs()
		max := p.MaxOccurs()
		if min > 0 { allOptional = false }
		count := 0
		for (max == -1 || count < max) && idx < len(children) {
			m, c, _ := mg.matchParticle(p, children[idx:], schema)
			if m == 0 || c == 0 {
				break
			}
			count += m
			idx += c
		}
		if count < min {
			return 0, false
		}
	}
	if idx == 0 && allOptional {
		return 0, true
	}
	if idx == 0 {
		return 0, false
	}
	return idx, true
}

// matchAllOnce matches one occurrence of an all group
func (mg *ModelGroup) matchAllOnce(group *ModelGroup, children []xmldom.Element, schema *Schema) (consumed int, ok bool) {
	matched := make(map[int]int, len(group.Particles))
	used := make([]bool, len(children))
	progress := true
	for progress {
		progress = false
		for i, p := range group.Particles {
			if p.MaxOccurs() != -1 && matched[i] >= p.MaxOccurs() {
				continue
			}
			// find next child not used that matches p
			for ci := 0; ci < len(children); ci++ {
				if used[ci] { continue }
				if mg.elementMatchesParticle(children[ci], p, schema) {
					used[ci] = true
					matched[i]++
					consumed++
					progress = true
					break
				}
			}
		}
	}
	// enforce mins
	allMinZero := true
	for i, p := range group.Particles {
		if p.MinOccurs() > 0 { allMinZero = false }
		if matched[i] < p.MinOccurs() {
			return 0, false
		}
	}
	if consumed == 0 && allMinZero {
		return 0, true
	}
	if consumed == 0 {
		return 0, false
	}
	return consumed, true
}

// matchChoiceOnce matches one occurrence of a choice group (choose one alternative)
func (mg *ModelGroup) matchChoiceOnce(group *ModelGroup, children []xmldom.Element, schema *Schema) (consumed int, ok bool) {
	best := 0
	for _, alt := range group.Particles {
		m, c, _ := mg.matchParticle(alt, children, schema)
		if m >= alt.MinOccurs() && c > 0 {
			if c > best { best = c }
		}
	}
	if best == 0 {
		return 0, false
	}
	return best, true
}

// matchWildcard matches elements against a wildcard
func (mg *ModelGroup) matchWildcard(wildcard *AnyElement, children []xmldom.Element, schema *Schema) (matched int, consumed int, violations []Violation) {
	for i := 0; i < len(children); i++ {
		child := children[i]

		// Check if element matches wildcard namespace constraint
		if !MatchesWildcard(child, wildcard.Namespace, schema.TargetNamespace) {
			// Element doesn't match wildcard
			if matched >= wildcard.MinOcc || wildcard.MinOcc == 0 {
				// We've satisfied min occurrences or wildcard is optional
				break
			}
			// Otherwise this is an error (handled by caller)
			break
		}

		// Element matches wildcard namespace constraint
		// Now validate it according to processContents
		elemViolations := ValidateAnyElement(child, wildcard, schema)
		violations = append(violations, elemViolations...)

		matched++
		consumed++

		// Check max occurrences
		if wildcard.MaxOcc != -1 && matched >= wildcard.MaxOcc {
			break
		}
	}

	return matched, consumed, violations
}

func (mg *ModelGroup) matchChoiceWithLookahead(choiceGroup *ModelGroup, children []xmldom.Element, next Particle, schema *Schema) (consumed int, violations []Violation) {
	bestConsumed := 0
	var bestViolations []Violation
	// First pass: prefer alts that allow the following required particle to match
	for _, alt := range choiceGroup.Particles {
	_, c, v := mg.matchParticle(alt, children, schema)
		if c == 0 { continue }
		// If next is required, check that it can match after this alt
		nextRequired := next != nil && next.MinOccurs() > 0
		if nextRequired {
			if c < len(children) {
				if mg.elementMatchesParticle(children[c], next, schema) {
					if c > bestConsumed { bestConsumed = c; bestViolations = v }
				}
			} else {
				// If no more children after alt but next is required, skip this alt
				continue
			}
		} else {
			// Next optional or absent - accept
			if c > bestConsumed { bestConsumed = c; bestViolations = v }
		}
	}
	if bestConsumed == 0 {
		// Fallback: pick longest alt regardless of next
		for _, alt := range choiceGroup.Particles {
			_, c, v := mg.matchParticle(alt, children, schema)
			if c > bestConsumed { bestConsumed = c; bestViolations = v }
		}
	}
	return bestConsumed, bestViolations
}

func (mg *ModelGroup) elementMatchesParticle(elem xmldom.Element, particle Particle, schema *Schema) bool {
	switch p := particle.(type) {
case *ElementDecl:
		// Inline element declaration - check if element matches
		elemQName := QName{
			Namespace: string(elem.NamespaceURI()),
			Local:     string(elem.LocalName()),
		}
		// Direct match
		if elemQName == p.Name {
			return true
		}
		// Check substitution groups - can this element substitute for the expected element?
		return schema.isSubstitutableFor(elemQName, p.Name)
case *ElementRef:
		// Check if element matches the reference
		elemQName := QName{
			Namespace: string(elem.NamespaceURI()),
			Local:     string(elem.LocalName()),
		}
		// Direct match
		if elemQName == p.Ref {
			return true
		}
		// Check substitution groups
		return schema.isSubstitutableFor(elemQName, p.Ref)
case *GroupRef:
		// Resolve and check if this element falls in the group's FIRST set
		if group, exists := schema.Groups[p.Ref]; exists {
			return schema.elementInFirstSet(elem, group)
		}
case *ModelGroup:
		// Check using FIRST set of the nested group
		return schema.elementInFirstSet(elem, p)
	case *AnyElement:
		// Check namespace constraint only for matching predicate; detailed validation is handled elsewhere
		return MatchesWildcard(elem, p.Namespace, schema.TargetNamespace)
	}
	return false
}

// Facet implementations moved to facets.go

// ComplexContent Validate implementation
func (cc *ComplexContent) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Validate complex content based on extension/restriction
	if cc.Extension != nil {
		// Extension validation
		if cc.Extension.Content != nil {
			return cc.Extension.Content.Validate(element, schema)
		}
	} else if cc.Restriction != nil {
		// Restriction validation - validate against the restricted content model
		if cc.Restriction.Content != nil {
			return cc.Restriction.Content.Validate(element, schema)
		}
	}
	return nil
}

// AllowAnyContent implementation
func (a *AllowAnyContent) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Allow any children - don't report violations for content model
	return nil
}

// ValidateContentModels performs determinism (UPA) checks on compiled content models
func (s *Schema) ValidateContentModels() error {
	visited := make(map[*ModelGroup]bool)
	visitedRefs := make(map[QName]bool)
	// Check complex types
	for _, t := range s.TypeDefs {
		if ct, ok := t.(*ComplexType); ok {
			if mg, ok := ct.Content.(*ModelGroup); ok {
				if err := s.validateGroupUPAWithVisited(mg, visited, visitedRefs); err != nil {
					return err
				}
			}
			if cc, ok := ct.Content.(*ComplexContent); ok {
				var content Content
				if cc.Extension != nil { content = cc.Extension.Content }
				if cc.Restriction != nil { content = cc.Restriction.Content }
				if mg, ok := content.(*ModelGroup); ok {
					if err := s.validateGroupUPAWithVisited(mg, visited, visitedRefs); err != nil { return err }
				}
			}
		}
	}
	// Check named groups
	for _, g := range s.Groups {
		if err := s.validateGroupUPAWithVisited(g, visited, visitedRefs); err != nil {
			return err
		}
	}
	return nil
}

type firstSet struct {
	names map[QName]bool
	wildcards []*WildcardNamespaceConstraint
}

func (fs *firstSet) addName(q QName) {
	if fs.names == nil { fs.names = make(map[QName]bool) }
	fs.names[q] = true
}

func (fs *firstSet) addWildcard(ns string) {
	wc := ParseNamespaceConstraint(ns)
	fs.wildcards = append(fs.wildcards, wc)
}

func (s *Schema) validateGroupUPAWithVisited(mg *ModelGroup, visited map[*ModelGroup]bool, visitedRefs map[QName]bool) error {
	if visited[mg] { return nil }
	visited[mg] = true
	// Recurse into nested groups first
	for _, p := range mg.Particles {
		if ng, ok := p.(*ModelGroup); ok {
			if err := s.validateGroupUPAWithVisited(ng, visited, visitedRefs); err != nil { return err }
		}
		if gr, ok := p.(*GroupRef); ok {
			if visitedRefs[gr.Ref] { continue }
			visitedRefs[gr.Ref] = true
			s.mu.RLock(); g := s.Groups[gr.Ref]; s.mu.RUnlock()
			if g != nil { if err := s.validateGroupUPAWithVisited(g, visited, visitedRefs); err != nil { return err } }
			delete(visitedRefs, gr.Ref)
		}
	}
	// Enforce determinism
	switch mg.Kind {
	case ChoiceGroup:
		// Build first sets for direct particles
		sets := make([]*firstSet, len(mg.Particles))
		for i, p := range mg.Particles {
			sets[i] = s.buildFirstSet(p, make(map[*ModelGroup]bool), make(map[QName]bool))
		}
		// Pairwise overlap check
		for i := 0; i < len(sets); i++ {
			for j := i+1; j < len(sets); j++ {
				if s.firstSetsOverlap(sets[i], sets[j]) {
					return fmt.Errorf("content model is not determinisitic (UPA): overlapping choice alternatives")
				}
			}
		}
	case SequenceGroup:
		// For sequences, if a particle can be absent (nullable), its FIRST set must not overlap with
		// the FIRST set of any particle that can follow via a chain of nullable particles.
		// Compute FIRST for each particle once
		firsts := make([]*firstSet, len(mg.Particles))
		for i, p := range mg.Particles {
			firsts[i] = s.buildFirstSet(p, make(map[*ModelGroup]bool), make(map[QName]bool))
		}
		// Helper to check nullability of a particle
		isNull := func(p Particle) bool { return s.isNullable(p) }
		for i := 0; i < len(mg.Particles); i++ {
			// Only relevant when p_i is nullable
			if !isNull(mg.Particles[i]) { continue }
			// Build suffix FIRST reachable by skipping p_i and subsequent nullable particles
			suffixFirst := &firstSet{names: make(map[QName]bool)}
			for j := i+1; j < len(mg.Particles); j++ {
				// Union FIRST of p_j into suffixFirst
				for q := range firsts[j].names { suffixFirst.names[q] = true }
				suffixFirst.wildcards = append(suffixFirst.wildcards, firsts[j].wildcards...)
				// If current p_j is not nullable, stop
				if !isNull(mg.Particles[j]) { break }
			}
			// Check overlap between FIRST(p_i) and suffixFirst
			if s.firstSetsOverlap(firsts[i], suffixFirst) {
				return fmt.Errorf("content model is not determinisitic (UPA): sequence ambiguity due to nullable particles")
			}
		}
	}
	return nil
}

func (s *Schema) buildFirstSet(p Particle, visitedGroups map[*ModelGroup]bool, visitedRefs map[QName]bool) *firstSet {
	fs := &firstSet{}
	switch pt := p.(type) {
	case *ElementDecl:
		// expected name and its substitution members
		for _, q := range s.allowedNamesFor(pt.Name) { fs.addName(q) }
	case *ElementRef:
		for _, q := range s.allowedNamesFor(pt.Ref) { fs.addName(q) }
	case *AnyElement:
		fs.addWildcard(pt.Namespace)
	case *GroupRef:
		if visitedRefs[pt.Ref] { return fs }
		visitedRefs[pt.Ref] = true
		s.mu.RLock(); g := s.Groups[pt.Ref]; s.mu.RUnlock()
		if g != nil {
			gfs := s.buildFirstSet(g, visitedGroups, visitedRefs)
			for q := range gfs.names { fs.addName(q) }
			fs.wildcards = append(fs.wildcards, gfs.wildcards...)
		}
		delete(visitedRefs, pt.Ref)
	case *ModelGroup:
		if visitedGroups[pt] { return fs }
		visitedGroups[pt] = true
		if pt.Kind == ChoiceGroup || pt.Kind == AllGroup {
			for _, cp := range pt.Particles {
				cfs := s.buildFirstSet(cp, visitedGroups, visitedRefs)
				for q := range cfs.names { fs.addName(q) }
				fs.wildcards = append(fs.wildcards, cfs.wildcards...)
			}
		} else if pt.Kind == SequenceGroup {
			// First set of sequence: from first particle(s), considering nullability
			for _, cp := range pt.Particles {
				cfs := s.buildFirstSet(cp, visitedGroups, visitedRefs)
				for q := range cfs.names { fs.addName(q) }
				fs.wildcards = append(fs.wildcards, cfs.wildcards...)
				if !s.isNullable(cp) { break }
			}
		}
		delete(visitedGroups, pt)
	}
	return fs
}

func (s *Schema) isNullable(p Particle) bool {
	return s.isNullableEx(p, make(map[*ModelGroup]bool), make(map[QName]bool))
}

func (s *Schema) isNullableEx(p Particle, visitedGroups map[*ModelGroup]bool, visitedRefs map[QName]bool) bool {
	switch pt := p.(type) {
	case *ElementDecl:
		return pt.MinOcc == 0
	case *ElementRef:
		return pt.MinOcc == 0
	case *AnyElement:
		return pt.MinOcc == 0
	case *ModelGroup:
		if visitedGroups[pt] { return false }
		visitedGroups[pt] = true
		switch pt.Kind {
		case SequenceGroup, AllGroup:
			for _, cp := range pt.Particles { if !s.isNullableEx(cp, visitedGroups, visitedRefs) { delete(visitedGroups, pt); return false } }
			delete(visitedGroups, pt); return true
		case ChoiceGroup:
			for _, cp := range pt.Particles { if s.isNullableEx(cp, visitedGroups, visitedRefs) { delete(visitedGroups, pt); return true } }
			delete(visitedGroups, pt); return false
		}
	case *GroupRef:
		if visitedRefs[pt.Ref] { return false }
		visitedRefs[pt.Ref] = true
		// Nullable if reference allows zero or the referenced group is nullable
		if pt.MinOcc == 0 { delete(visitedRefs, pt.Ref); return true }
		if g, ok := s.Groups[pt.Ref]; ok { res := s.isNullableEx(g, visitedGroups, visitedRefs); delete(visitedRefs, pt.Ref); return res }
		delete(visitedRefs, pt.Ref); return false
	}
	return false
}

func (s *Schema) allowedNamesFor(expected QName) []QName {
	// include expected and all members of its substitution group (transitively)
	var out []QName
	seen := make(map[QName]bool)
	queue := []QName{expected}
	for len(queue) > 0 {
		h := queue[0]
		queue = queue[1:]
		if seen[h] { continue }
		seen[h] = true
		out = append(out, h)
		if members, ok := s.SubstitutionGroups[h]; ok {
			for _, m := range members {
				if !seen[m] { queue = append(queue, m) }
			}
		}
	}
	return out
}

func (s *Schema) firstSetsOverlap(a, b *firstSet) bool {
	// name-name overlap
	for q := range a.names { if b.names[q] { return true } }
	for q := range b.names { if a.names[q] { return true } }
	// name-wildcard overlap
	for q := range a.names {
		for _, wc := range b.wildcards { if wc.Matches(q.Namespace, s.TargetNamespace) { return true } }
	}
	for q := range b.names {
		for _, wc := range a.wildcards { if wc.Matches(q.Namespace, s.TargetNamespace) { return true } }
	}
	// wildcard-wildcard overlap (coarse)
	for _, wc1 := range a.wildcards {
		for _, wc2 := range b.wildcards { if wildcardsOverlap(wc1, wc2, s.TargetNamespace) { return true } }
	}
	return false
}

func (s *Schema) elementInFirstSet(elem xmldom.Element, group *ModelGroup) bool {
	elemQName := QName{Namespace: string(elem.NamespaceURI()), Local: string(elem.LocalName())}
	fs := s.buildFirstSet(group, make(map[*ModelGroup]bool), make(map[QName]bool))
	// name match
	if fs.names[elemQName] { return true }
	// Also allow if element can substitute any expected name
	for q := range fs.names {
		if s.isSubstitutableFor(elemQName, q) { return true }
	}
	// wildcard match
	for _, wc := range fs.wildcards {
		if wc.Matches(elemQName.Namespace, s.TargetNamespace) { return true }
	}
	return false
}

func wildcardsOverlap(c1, c2 *WildcardNamespaceConstraint, targetNS string) bool {
	// Trivial any
	if c1.Mode == "##any" || c2.Mode == "##any" { return true }
	// Other vs other overlaps (assume there exist other namespaces)
	if c1.Mode == "##other" && c2.Mode == "##other" { return true }
	// If one is other and the other includes at least one non-target namespace, assume overlap
	if c1.Mode == "##other" && c2.Mode == "list" {
		for _, ns := range c2.Namespaces { if ns != targetNS && ns != "##targetNamespace" && ns != "##local" { return true } }
	}
	if c2.Mode == "##other" && c1.Mode == "list" {
		for _, ns := range c1.Namespaces { if ns != targetNS && ns != "##targetNamespace" && ns != "##local" { return true } }
	}
	// list-list: check common tokens
	if c1.Mode == "list" && c2.Mode == "list" {
		set := make(map[string]bool, len(c1.Namespaces))
		for _, ns := range c1.Namespaces { set[ns] = true }
		for _, ns := range c2.Namespaces { if set[ns] { return true } }
		// map ##targetNamespace token to actual targetNS string
		if set["##targetNamespace"] {
			for _, ns := range c2.Namespaces { if ns == targetNS || ns == "##targetNamespace" { return true } }
		}
		if set["##local"] {
			for _, ns := range c2.Namespaces { if ns == "##local" { return true } }
		}
		// symmetric cases covered by initial set
	}
	// other mixed cases: conservative false
	return false
}
