package xsd

import (
	"fmt"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// Validator validates XML documents against XSD schemas
type Validator struct {
	schema        *Schema
	idRefs        map[string]xmldom.Element
	ids           map[string]xmldom.Element
	violations    []Violation
	idConstraints *IdentityConstraintValidator // Identity constraints validator
}

// NewValidator creates a new validator for a schema
func NewValidator(schema *Schema) *Validator {
	v := &Validator{
		schema:        schema,
		idRefs:        make(map[string]xmldom.Element),
		ids:           make(map[string]xmldom.Element),
		violations:    make([]Violation, 0),
		idConstraints: NewIdentityConstraintValidator(),
	}

	// Collect all identity constraints from element declarations
	for _, decl := range schema.ElementDecls {
		for _, constraint := range decl.Constraints {
			v.idConstraints.AddConstraint(constraint)
		}
	}

	return v
}

// Validate validates an XML document against the schema
func (v *Validator) Validate(doc xmldom.Document) []Violation {
	if doc == nil {
		return []Violation{{
			Code:    "xsd-null-document",
			Message: "Document is null",
		}}
	}

	root := doc.DocumentElement()
	if root == nil {
		return []Violation{{
			Code:    "xsd-no-root",
			Message: "Document has no root element",
		}}
	}

	// Reset state
	v.violations = make([]Violation, 0)
	v.ids = make(map[string]xmldom.Element)
	v.idRefs = make(map[string]xmldom.Element)

	// Collect all IDs and IDREFs first
	v.collectIDsAndRefs(root)

	// Validate root element
	v.validateElement(root, nil)

	// Check IDREF constraints
	v.validateIDREFs()

	// Validate identity constraints (key, keyref, unique)
	if v.idConstraints != nil {
		idViolations := v.idConstraints.Validate(doc)
		v.violations = append(v.violations, idViolations...)
	}

	return v.violations
}

// collectIDsAndRefs collects all ID and IDREF attributes in the document
func (v *Validator) collectIDsAndRefs(elem xmldom.Element) {
	// Check all attributes for ID and IDREF types
	attrs := elem.Attributes()
	for i := uint(0); i < attrs.Length(); i++ {
		attr := attrs.Item(i)
		if attr != nil {
			attrName := string(attr.LocalName())
			attrValue := string(attr.NodeValue())

			// Check if this attribute is of type ID
			// For simplicity, we'll check common ID attribute names
			// In a full implementation, we'd check the schema type
			if attrName == "id" || attrName == "ID" {
				if _, exists := v.ids[attrValue]; exists {
					v.addViolation(elem, attrName, "cvc-id.2",
						fmt.Sprintf("Duplicate ID value '%s'", attrValue), nil, attrValue)
				} else {
					v.ids[attrValue] = elem
				}
			}

			// Check common IDREF attribute names
			idrefAttrs := []string{"target", "ref", "idref", "IDREF", "idlocation", "sendid"}
			for _, refName := range idrefAttrs {
				if attrName == refName && attrValue != "" {
					v.idRefs[attrValue] = elem
					break
				}
			}
		}
	}

	// Recurse through children
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		if child := children.Item(i); child != nil {
			v.collectIDsAndRefs(child)
		}
	}
}

// validateElement validates an element against the schema
func (v *Validator) validateElement(elem xmldom.Element, parentType Type) {
	elemNS := string(elem.NamespaceURI())
	elemLocal := string(elem.LocalName())

	// Find element declaration
	qname := QName{Namespace: elemNS, Local: elemLocal}

	v.schema.mu.RLock()
	decl, found := v.schema.ElementDecls[qname]
	v.schema.mu.RUnlock()

	if !found && elemNS == "" && v.schema.TargetNamespace != "" {
		// Try with target namespace
		qname.Namespace = v.schema.TargetNamespace
		v.schema.mu.RLock()
		decl, found = v.schema.ElementDecls[qname]
		v.schema.mu.RUnlock()
	}

	if !found {
		// Check if this is allowed by xs:any
		if parentType == nil {
			v.addViolation(elem, "", "cvc-elt.1",
				fmt.Sprintf("Cannot find declaration for element '%s'", elemLocal),
				nil, elemLocal)
		}
		// Continue to check children even if element not found
		// This allows detecting errors in nested elements
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			if child := children.Item(i); child != nil {
				v.validateElement(child, nil)
			}
		}
		return
	}

	// Check if element is abstract (cannot be used directly)
	if decl.Abstract {
		v.addViolation(elem, "", "cvc-elt.2",
			fmt.Sprintf("Element '%s' is abstract and cannot be used directly in instance documents", elemLocal),
			nil, elemLocal)
	}

	// Check if element's type is abstract
	if decl.Type != nil {
		if ct, ok := decl.Type.(*ComplexType); ok && ct.Abstract {
			v.addViolation(elem, "", "cvc-type.2",
				fmt.Sprintf("Element '%s' has abstract type '%s' which cannot be used directly",
					elemLocal, ct.QName.Local),
				nil, ct.QName.Local)
		}
	}

	// Validate nillable and xsi:nil
	xsiNil := elem.GetAttributeNS("http://www.w3.org/2001/XMLSchema-instance", "nil")
	if xsiNil != "" {
		xsiNilValue := string(xsiNil)

		// Check if element is nillable
		if !decl.Nillable {
			v.addViolation(elem, "xsi:nil", "cvc-elt.3.1",
				fmt.Sprintf("Element '%s' has xsi:nil='%s' but is not nillable", elemLocal, xsiNilValue),
				nil, xsiNilValue)
		}

		// If xsi:nil="true", element must be empty
		if xsiNilValue == "true" || xsiNilValue == "1" {
			content := strings.TrimSpace(getElementTextContent(elem))
			children := elem.Children()

			if content != "" || children.Length() > 0 {
				v.addViolation(elem, "xsi:nil", "cvc-elt.3.2.2",
					fmt.Sprintf("Element '%s' has xsi:nil='true' but has content", elemLocal),
					nil, content)
			}
		}
	}

	// Validate fixed and default values
	fixedDefaultViolations := ValidateElementFixedDefault(elem, decl)
	v.violations = append(v.violations, fixedDefaultViolations...)

	// Validate against type (but skip content validation for ComplexType,
	// as that will be done in validateChildren to avoid duplication)
	if decl.Type != nil {
		if _, isComplexType := decl.Type.(*ComplexType); !isComplexType {
			violations := decl.Type.Validate(elem, v.schema)
			v.violations = append(v.violations, violations...)
		}

		// Validate element value against built-in type and facets
		content := getElementTextContent(elem)
		// Apply default value if element is empty
		if content == "" && decl.Default != "" {
			content = decl.Default
		}
		if content != "" {
			// Validate built-in type
			if err := v.validateBuiltinType(content, decl.Type); err != nil {
				v.addViolation(elem, "", "cvc-datatype-valid.1",
					err.Error(), nil, content)
			}

			// Validate facets for simple types
			if st, ok := decl.Type.(*SimpleType); ok {
				if err := v.validateSimpleTypeFacets(content, st); err != nil {
					v.addViolation(elem, "", "cvc-facet-valid",
						err.Error(), nil, content)
				}
			}
		}
	}

	// Validate attributes
	v.validateAttributes(elem, decl.Type)

	// Validate children
	v.validateChildren(elem, decl.Type)
}

// validateAttributes validates element attributes
func (v *Validator) validateAttributes(elem xmldom.Element, elemType Type) {
	// Get expected attributes from type
	var expectedAttrs []*AttributeDecl
	var anyAttr *AnyAttribute

	if ct, ok := elemType.(*ComplexType); ok {
		expectedAttrs = ct.Attributes
		anyAttr = ct.AnyAttribute

		// Also resolve attribute groups
		groupAttrs := v.schema.ResolveAttributeGroups(ct)
		expectedAttrs = append(expectedAttrs, groupAttrs...)
	}

	// Build map of expected attributes
	expected := make(map[string]*AttributeDecl)
	for _, attr := range expectedAttrs {
		expected[attr.Name.Local] = attr
	}

	// Check all attributes on element
	attrs := elem.Attributes()
	for i := uint(0); i < attrs.Length(); i++ {
		attr := attrs.Item(i)
		if attr == nil {
			continue
		}

		attrLocal := string(attr.LocalName())
		attrNS := string(attr.NamespaceURI())

		// Skip namespace declarations
		// The xmldom library reports xmlns:prefix with namespace="xmlns"
		// and xmlns without prefix with local="xmlns"
		if attrNS == "http://www.w3.org/2000/xmlns/" || attrNS == "xmlns" || attrLocal == "xmlns" {
			continue
		}

		// Check if attribute is expected
		if decl, ok := expected[attrLocal]; ok {
			// Validate fixed and default values
			fixedDefaultViolations := ValidateAttributeFixedDefault(attr, decl, elem)
			v.violations = append(v.violations, fixedDefaultViolations...)

			// Validate attribute value against type
			if decl.Type != nil {
				attrValue := string(attr.NodeValue())
				typeViolations := v.validateAttributeType(elem, attrLocal, attrValue, decl.Type)
				v.violations = append(v.violations, typeViolations...)
			}
			delete(expected, attrLocal) // Mark as found
		} else {
			// Check if allowed by anyAttribute
			if anyAttr != nil {
				// Validate against anyAttribute wildcard
				wildcardViolations := ValidateAnyAttribute(attr, anyAttr, v.schema)
				for _, wv := range wildcardViolations {
					// Set element context if not set
					if wv.Element == nil {
						wv.Element = elem
					}
					wv.Attribute = attrLocal
					v.violations = append(v.violations, wv)
				}
			} else {
				// No anyAttribute, so this is not allowed
				suggestions := v.suggestAttribute(attrLocal, expectedAttrs)
				v.addViolation(elem, attrLocal, "cvc-complex-type.3.2.2",
					fmt.Sprintf("Attribute '%s' is not allowed to appear in element '%s'",
						attrLocal, elem.LocalName()),
					suggestions, attrLocal)
			}
		}
	}

	// Check for required attributes that are missing
	for name, decl := range expected {
		// Check fixed value for missing attributes
		if decl.Fixed != "" {
			// Missing attribute with fixed value - validate that it would have the fixed value
			fixedDefaultViolations := ValidateAttributeFixedDefault(nil, decl, elem)
			v.violations = append(v.violations, fixedDefaultViolations...)
		}

		if decl.Use == RequiredUse {
			v.addViolation(elem, name, "cvc-complex-type.4",
				fmt.Sprintf("Required attribute '%s' is missing", name),
				[]string{name}, "")
		}
		// Note: Default values for missing attributes are typically applied
		// during document processing, not during validation
	}
}

// validateAttributeType validates an attribute value against its type
func (v *Validator) validateAttributeType(elem xmldom.Element, attrName string, value string, attrType Type) []Violation {
	var violations []Violation

	// Check if type is a SimpleType
	simpleType, isSimple := attrType.(*SimpleType)
	if !isSimple {
		// Not a simple type - skip validation for now
		// (Complex types shouldn't be used for attributes in standard XSD)
		return violations
	}

	// If it has a restriction, validate against facets
	if simpleType.Restriction != nil {
		for _, facet := range simpleType.Restriction.Facets {
			err := facet.Validate(value, simpleType)
			if err != nil {
				// Create violation based on facet type
				code := "cvc-datatype-valid.1.2.1"
				message := fmt.Sprintf("Attribute '%s': %s", attrName, err.Error())

				// Add expected values for enumeration failures
				var expected []string
				if enumFacet, isEnum := facet.(*EnumerationFacet); isEnum {
					expected = enumFacet.Values
				}

				violations = append(violations, Violation{
					Element:   elem,
					Code:      code,
					Message:   message,
					Attribute: attrName,
					Expected:  expected,
					Actual:    value,
				})
			}
		}
	}

	// TODO: Handle List and Union types

	return violations
}

// validateChildren validates element children against content model
func (v *Validator) validateChildren(elem xmldom.Element, elemType Type) {
	if elemType == nil {
		return
	}

	// Handle ComplexType
	ct, isComplex := elemType.(*ComplexType)
	if !isComplex {
		// Simple type - no element children allowed
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			if child := children.Item(i); child != nil {
				v.addViolation(elem, "", "cvc-complex-type.2.3",
					"Element with simple type cannot have element children",
					nil, "element children")
				return
			}
		}
		return
	}

	// Check if this is a complex type with simple content
	_, hasSimpleContent := ct.Content.(*SimpleContent)
	if hasSimpleContent {
		// Complex type with simple content - text content is allowed, but no element children
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			if child := children.Item(i); child != nil {
				v.addViolation(elem, "", "cvc-complex-type.2.3",
					"Element with simple content cannot have element children",
					nil, "element children")
				return
			}
		}
		// Text content is allowed for simple content, so return here
		return
	}

	// Check for mixed content (only for non-simple content)
	if !ct.Mixed {
		// Check for text content between elements
		nodes := elem.ChildNodes()
		for i := uint(0); i < nodes.Length(); i++ {
			if node := nodes.Item(i); node != nil && node.NodeType() == 3 { // TEXT_NODE = 3
				text := strings.TrimSpace(string(node.NodeValue()))
				if text != "" {
					v.addViolation(elem, "", "cvc-complex-type.2.3",
						"Element cannot have text content (mixed='false')",
						nil, text)
				}
			}
		}
	}

	// Get content model
	if ct.Content != nil {
		// Validate against content model
		violations := ct.Content.Validate(elem, v.schema)
		for _, violation := range violations {
			// Set element if not already set
			if violation.Element == nil {
				violation.Element = elem
			}
			v.violations = append(v.violations, violation)
		}

		// Content model validation handles child validation, so we're done
		// The model group validation already checked element declarations and types
		return
	} else {
		// Empty content - no children allowed
		children := elem.Children()
		if children.Length() > 0 {
			v.addViolation(elem, "", "cvc-complex-type.2.1",
				"Element must be empty",
				nil, "children")
		}
	}

	// Only validate children recursively if there's no content model
	// (This case is for elements with AllowAnyContent or similar)
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		if child := children.Item(i); child != nil {
			v.validateElement(child, elemType)
		}
	}
}

// validateIDREFs validates all collected IDREF references
func (v *Validator) validateIDREFs() {
	for idref, elem := range v.idRefs {
		if _, exists := v.ids[idref]; !exists {
			v.addViolation(elem, "", "cvc-id.1",
				fmt.Sprintf("There is no ID/IDREF binding for IDREF '%s'", idref),
				nil, idref)
		}
	}
}

// matchesNamespace checks if a namespace matches a wildcard pattern
func (v *Validator) matchesNamespace(ns, pattern string) bool {
	if pattern == "##any" {
		return true
	}
	if pattern == "##other" {
		return ns != v.schema.TargetNamespace
	}
	if pattern == "##targetNamespace" {
		return ns == v.schema.TargetNamespace
	}
	if pattern == "##local" {
		return ns == ""
	}
	// Check explicit namespace list
	for _, allowed := range strings.Fields(pattern) {
		if allowed == ns {
			return true
		}
	}
	return false
}

// suggestAttribute suggests similar attribute names
func (v *Validator) suggestAttribute(wrong string, attrs []*AttributeDecl) []string {
	suggestions := []string{}
	wrongLower := strings.ToLower(wrong)

	// Common SCXML attribute corrections
	corrections := map[string]string{
		"sendid":     "id",       // Common mistake in <send>
		"priority":   "",         // Not a valid attribute
		"invokeId":   "invokeid", // Case correction
		"targetType": "type",     // Wrong attribute name
	}

	if correct, ok := corrections[wrong]; ok && correct != "" {
		return []string{correct}
	}

	// Fuzzy matching for typos
	for _, attr := range attrs {
		name := attr.Name.Local
		nameLower := strings.ToLower(name)

		// Exact match ignoring case
		if wrongLower == nameLower {
			suggestions = append(suggestions, name)
			continue
		}

		// Check for common typos
		if levenshteinDistance(wrongLower, nameLower) <= 2 {
			suggestions = append(suggestions, name)
		}
	}

	return suggestions
}

// levenshteinDistance calculates edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// addViolation adds a validation violation
func (v *Validator) addViolation(elem xmldom.Element, attr, code, message string, expected []string, actual string) {
	v.violations = append(v.violations, Violation{
		Element:   elem,
		Attribute: attr,
		Code:      code,
		Message:   message,
		Expected:  expected,
		Actual:    actual,
	})
}

// ValidateChildSequence validates a sequence of child elements
func ValidateChildSequence(elem xmldom.Element, expected []string) []Violation {
	violations := []Violation{}

	children := elem.Children()
	childIndex := 0

	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child == nil {
			continue
		}

		childName := string(child.LocalName())

		if childIndex >= len(expected) {
			violations = append(violations, Violation{
				Element:  child,
				Code:     "cvc-complex-type.2.4.d",
				Message:  fmt.Sprintf("Element '%s' is not expected here", childName),
				Expected: []string{},
				Actual:   childName,
			})
			continue
		}

		if childName != expected[childIndex] {
			violations = append(violations, Violation{
				Element:  child,
				Code:     "cvc-complex-type.2.4.a",
				Message:  fmt.Sprintf("Element name '%s' is invalid", childName),
				Expected: expected[childIndex:],
				Actual:   childName,
			})
		}

		childIndex++
	}

	// Check for missing required elements
	if childIndex < len(expected) {
		violations = append(violations, Violation{
			Element:  elem,
			Code:     "cvc-complex-type.2.4.b",
			Message:  fmt.Sprintf("Missing required elements: %v", expected[childIndex:]),
			Expected: expected[childIndex:],
		})
	}

	return violations
}

// validateBuiltinType validates a value against built-in XSD type constraints
func (v *Validator) validateBuiltinType(value string, elemType Type) error {
	if elemType == nil {
		return nil
	}

	// Get the base type name
	typeName := elemType.Name()

	// Check if it's a built-in type
	if builtinType := GetBuiltinType(typeName.Local); builtinType != nil {
		return builtinType.Validator(value)
	}

	// If it's a simple type, check its base type
	if st, ok := elemType.(*SimpleType); ok && st.Restriction != nil {
		if baseType := GetBuiltinType(st.Restriction.Base.Local); baseType != nil {
			return baseType.Validator(value)
		}
	}

	return nil
}

// validateSimpleTypeFacets validates a value against simple type facet constraints
func (v *Validator) validateSimpleTypeFacets(value string, simpleType *SimpleType) error {
	if simpleType == nil {
		return nil
	}

	// Handle union types
	if simpleType.Union != nil {
		return ValidateUnionType(value, simpleType.Union, v.schema)
	}

	// Handle list types
	if simpleType.List != nil {
		return ValidateListType(value, simpleType.List, v.schema)
	}

	// Handle restriction types
	if simpleType.Restriction != nil && len(simpleType.Restriction.Facets) > 0 {
		// Get the base type for context
		var baseType Type
		if simpleType.Restriction.Base.Local != "" {
			v.schema.mu.RLock()
			baseType = v.schema.TypeDefs[simpleType.Restriction.Base]
			v.schema.mu.RUnlock()
		}

		return ValidateFacets(value, simpleType.Restriction.Facets, baseType)
	}

	return nil
}

// getElementTextContent extracts text content from an element
func getElementTextContent(elem xmldom.Element) string {
	var content strings.Builder
	nodes := elem.ChildNodes()
	for i := uint(0); i < nodes.Length(); i++ {
		if node := nodes.Item(i); node != nil && node.NodeType() == 3 { // TEXT_NODE
			content.WriteString(string(node.NodeValue()))
		}
	}
	return content.String()
}
