package xsd

import (
	"fmt"
	"os"
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
	doc                xmldom.Document
}

// QName represents a qualified XML name
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
	SubstitutionGroup QName // Head element this element can substitute for
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
	QName          QName
	Content        Content
	Attributes     []*AttributeDecl
	AttributeGroup []QName
	AnyAttribute   *AnyAttribute
	Mixed          bool
	Abstract       bool
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
	Base   QName
	Facets []FacetValidator
}

// Facet represents a constraining facet (deprecated - use FacetValidator from facets.go)
type Facet interface {
	Validate(value string) error
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

	// Validate the schema document itself
	sv := NewSchemaValidator()
	if errors := sv.ValidateSchema(doc); len(errors) > 0 {
		// Return the first validation error
		return nil, fmt.Errorf("invalid XSD schema: %w", errors[0])
	}

	return Parse(doc)
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
		SubstitutionGroups: make(map[QName][]QName),
		doc:                doc,
	}

	// Get target namespace
	if tns := root.GetAttribute("targetNamespace"); tns != "" {
		schema.TargetNamespace = string(tns)
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

	// Check if actualElement is in the substitution group of expectedElement
	if members, exists := s.SubstitutionGroups[expectedElement]; exists {
		for _, member := range members {
			if member == actualElement {
				// TODO: Add type compatibility check here
				return true
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

	decl := &ElementDecl{
		Name: QName{
			Namespace: s.TargetNamespace,
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
		QName: QName{
			Namespace: s.TargetNamespace,
			Local:     "_anonymous",
		},
		Attributes: make([]*AttributeDecl, 0),
	}

	if mixed := string(elem.GetAttribute("mixed")); mixed == "true" {
		ct.Mixed = true
	}

	if abstract := string(elem.GetAttribute("abstract")); abstract == "true" {
		ct.Abstract = true
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
		QName: QName{
			Namespace: s.TargetNamespace,
			Local:     name,
		},
		Attributes: make([]*AttributeDecl, 0),
	}

	if mixed := string(elem.GetAttribute("mixed")); mixed == "true" {
		ct.Mixed = true
	}

	if abstract := string(elem.GetAttribute("abstract")); abstract == "true" {
		ct.Abstract = true
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
		Facets: make([]FacetValidator, 0),
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

	attr := &AttributeDecl{
		Name: QName{
			Namespace: s.TargetNamespace,
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
		case "attributeGroup":
			// Handle attribute group references in extensions
			if ref := string(child.GetAttribute("ref")); ref != "" {
				// We'll resolve these later
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
				}

				// Fallback: assume prefix refers to target namespace
				// This is common in schemas where xmlns:t="targetNamespace"
				// TODO: Improve namespace prefix resolution by actually reading xmlns attributes
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

// resolveInlineElementTypes resolves placeholder types for inline ElementDecl particles
func (s *Schema) resolveInlineElementTypes(particles []Particle) {
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
			// Recursively resolve nested model groups
			s.resolveInlineElementTypes(pt.Particles)
		}
	}
}

// resolveParticles recursively resolves GroupRef particles with cycle detection
func (s *Schema) resolveParticles(particles []Particle) []Particle {
	return s.resolveParticlesWithVisited(particles, make(map[QName]bool))
}

// resolveParticlesWithVisited recursively resolves GroupRef particles with cycle detection
func (s *Schema) resolveParticlesWithVisited(particles []Particle, visited map[QName]bool) []Particle {
	var resolved []Particle

	for _, p := range particles {
		switch pt := p.(type) {
		case *GroupRef:
			// Check for cycles
			if visited[pt.Ref] {
				// Cycle detected - keep the unresolved reference
				resolved = append(resolved, pt)
				continue
			}

			// Mark as visited
			visited[pt.Ref] = true

			// Resolve group reference
			if group, exists := s.Groups[pt.Ref]; exists {
				// Inline the group's particles
				resolvedGroup := &ModelGroup{
					Kind:      group.Kind,
					Particles: s.resolveParticlesWithVisited(group.Particles, visited), // Recursive resolution with visited tracking
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
			delete(visited, pt.Ref)

		case *ModelGroup:
			// Recursively resolve nested groups
			pt.Particles = s.resolveParticlesWithVisited(pt.Particles, visited)
			resolved = append(resolved, pt)
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
	// Simple content validation
	return nil
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
		violations = mg.validateSequence(childElements, schema)
	case ChoiceGroup:
		violations = mg.validateChoice(childElements, schema)
	case AllGroup:
		violations = mg.validateAll(childElements, schema)
	}

	return violations
}

func (mg *ModelGroup) validateSequence(children []xmldom.Element, schema *Schema) []Violation {
	var violations []Violation
	childIndex := 0
	particleIndex := 0

	// Process children
	for childIndex < len(children) && particleIndex < len(mg.Particles) {
		child := children[childIndex]
		particle := mg.Particles[particleIndex]

		// Special handling for nested model groups (group references)
		if nestedGroup, isModelGroup := particle.(*ModelGroup); isModelGroup {
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
			violations = append(violations, nestedViolations...)
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
				// Child doesn't match wildcard
				if wildcard.MinOcc == 0 {
					// Wildcard is optional, skip to next particle
					// Check if element matches next particle
					if particleIndex+1 < len(mg.Particles) {
						nextParticle := mg.Particles[particleIndex+1]
						if mg.elementMatchesParticle(child, nextParticle, schema) {
							// Element matches next particle, skip wildcard
							particleIndex++
							continue
						}
					}
					// Element doesn't match next particle either, report namespace constraint violation
					violations = append(violations, Violation{
						Element: child,
						Code:    "cvc-wildcard.2",
						Message: fmt.Sprintf("Element '%s' is not allowed by the namespace constraint '%s'",
							child.LocalName(), wildcard.Namespace),
					})
					childIndex++
					particleIndex++
				} else {
					// Required wildcard doesn't match
					violations = append(violations, Violation{
						Element: child,
						Code:    "cvc-wildcard.2",
						Message: fmt.Sprintf("Element '%s' is not allowed by the namespace constraint '%s'",
							child.LocalName(), wildcard.Namespace),
					})
					childIndex++
				}
			}
		} else {
			// Regular particle
			matched, consumed := mg.matchParticle(particle, children[childIndex:], schema)

			if consumed > 0 {
				// Particle consumed some children

				// If this is an inline ElementDecl, validate the matched elements
				if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl && elemDecl.Type != nil {
					// Validate each matched element against its type
					for i := 0; i < consumed; i++ {
						childElem := children[childIndex+i]

						// Check if actual element is different (substitution group)
						actualQName := QName{
							Namespace: string(childElem.NamespaceURI()),
							Local:     string(childElem.LocalName()),
						}

						// If element is substituted, use the actual element's type
						var typeToValidate Type = elemDecl.Type
						if actualQName != elemDecl.Name {
							if actualDecl, exists := schema.ElementDecls[actualQName]; exists && actualDecl.Type != nil {
								typeToValidate = actualDecl.Type
							}
						}

						typeViolations := typeToValidate.Validate(childElem, schema)
						violations = append(violations, typeViolations...)
					}
				} else if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
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
					violations = append(violations, Violation{
						Element: child,
						Code:    "cvc-complex-type.2.4.d",
						Message: fmt.Sprintf("Unexpected element '%s'", child.LocalName()),
					})
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
		violations = append(violations, Violation{
			Element: children[childIndex],
			Code:    "cvc-complex-type.2.4.d",
			Message: fmt.Sprintf("Unexpected element '%s'", children[childIndex].LocalName()),
		})
		childIndex++
	}

	return violations
}

func (mg *ModelGroup) validateChoice(children []xmldom.Element, schema *Schema) []Violation {
	var violations []Violation

	// At least one particle must match
	for _, particle := range mg.Particles {
		matched, consumed := mg.matchParticle(particle, children, schema)
		if matched > 0 {
			// Found a match - validate the matched elements
			if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl && elemDecl.Type != nil {
				// Validate each matched element against its type
				for i := 0; i < consumed; i++ {
					childElem := children[i]
					typeViolations := elemDecl.Type.Validate(childElem, schema)
					violations = append(violations, typeViolations...)
				}
			} else if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
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
				if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl && elemDecl.Type != nil {
					typeViolations := elemDecl.Type.Validate(child, schema)
					violations = append(violations, typeViolations...)
				} else if elemRef, isElemRef := particle.(*ElementRef); isElemRef {
					// For ElementRef, look up the global declaration and validate
					if decl, exists := schema.ElementDecls[elemRef.Ref]; exists && decl.Type != nil {
						typeViolations := decl.Type.Validate(child, schema)
						violations = append(violations, typeViolations...)
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
				}
				break // Found a match, stop trying other particles
			}
		}

		if !matched {
			// No particle in the choice matched this child
			// Stop consuming (choice can't match this element)
			break
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
			matched, cons := mg.matchParticle(particle, children[childIndex:], schema)
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

func (mg *ModelGroup) matchParticle(particle Particle, children []xmldom.Element, schema *Schema) (matched int, consumed int) {
	// Handle wildcards specially
	if wildcard, isWildcard := particle.(*AnyElement); isWildcard {
		matched, consumed, _ = mg.matchWildcard(wildcard, children, schema)
		return matched, consumed
	}

	// Handle inline ElementDecl specially - it can match and validate
	if elemDecl, isElemDecl := particle.(*ElementDecl); isElemDecl {
		for i := 0; i < len(children); i++ {
			child := children[i]
			elemQName := QName{
				Namespace: string(child.NamespaceURI()),
				Local:     string(child.LocalName()),
			}
			if elemQName == elemDecl.Name {
				// Element matches the inline declaration
				matched++
				consumed++

				// Validate the element against its type if defined
				if elemDecl.Type != nil {
					// Note: We should collect violations here but this function
					// doesn't return violations. For now, we just match.
					// The actual validation happens in validateSequence/validateChoice/validateAll
				}

				maxOcc := elemDecl.MaxOcc
				if maxOcc != -1 && matched >= maxOcc {
					break
				}
			} else {
				break // Stop at first non-match for sequence
			}
		}
		return matched, consumed
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
		// Resolve and check group
		if group, exists := schema.Groups[p.Ref]; exists {
			violations := group.Validate(elem, schema)
			return len(violations) == 0
		}
	case *AnyElement:
		// Check if element matches the wildcard's namespace constraint
		return MatchesWildcard(elem, p.Namespace, schema.TargetNamespace)
	case *ModelGroup:
		// Nested group validation
		violations := p.Validate(elem, schema)
		return len(violations) == 0
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
		// Restriction validation
	}
	return nil
}

// AllowAnyContent implementation
func (a *AllowAnyContent) Validate(element xmldom.Element, schema *Schema) []Violation {
	// Allow any children - don't report violations for content model
	return nil
}
