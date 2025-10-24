package xsd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// SchemaValidator validates that an XSD schema document conforms to XSD rules
type SchemaValidator struct {
	errors []error
	idMap  map[string]xmldom.Element // Track ID values for uniqueness
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		errors: []error{},
		idMap:  make(map[string]xmldom.Element),
	}
}

// ValidateSchema validates that a schema document conforms to XSD rules
func (sv *SchemaValidator) ValidateSchema(doc xmldom.Document) []error {
	sv.errors = []error{}
	sv.idMap = make(map[string]xmldom.Element)

	if doc == nil {
		return []error{fmt.Errorf("nil document")}
	}

	root := doc.DocumentElement()
	if root == nil {
		return []error{fmt.Errorf("no root element")}
	}

	// Check root element is xs:schema
	if string(root.NamespaceURI()) != XSDNamespace || string(root.LocalName()) != "schema" {
		sv.addError("document root must be xs:schema element")
	}

	// Validate the entire schema tree
	sv.validateElement(root)

	return sv.errors
}

// validateElement recursively validates an element and its children
func (sv *SchemaValidator) validateElement(elem xmldom.Element) {
	if elem == nil {
		return
	}

	// Check for ID attributes
	sv.validateIDAttribute(elem)

	// Validate element-specific rules
	if string(elem.NamespaceURI()) == XSDNamespace {
		switch string(elem.LocalName()) {
		case "schema":
			// Root schema element - no additional validation needed here
		case "simpleType":
			sv.validateSimpleType(elem)
		case "complexType":
			sv.validateComplexType(elem)
		case "element":
			sv.validateElementDecl(elem)
		case "attribute":
			sv.validateAttributeDecl(elem)
		case "restriction":
			sv.validateRestriction(elem)
		case "extension":
			sv.validateExtension(elem)
		case "sequence", "choice", "all":
			sv.validateModelGroup(elem)
		case "group":
			sv.validateGroup(elem)
		case "attributeGroup":
			sv.validateAttributeGroup(elem)
		case "import":
			sv.validateImport(elem)
		case "include":
			sv.validateInclude(elem)
		case "annotation", "documentation", "appinfo":
			// These are always valid
		case "any":
			sv.validateAny(elem)
		case "anyAttribute":
			sv.validateAnyAttribute(elem)
		case "unique", "key", "keyref":
			sv.validateIdentityConstraint(elem)
		case "selector", "field":
			sv.validateXPathElement(elem)
		case "notation":
			sv.validateNotation(elem)
		case "union":
			sv.validateUnion(elem)
		case "list":
			sv.validateList(elem)
		case "enumeration", "pattern", "length", "minLength", "maxLength",
			"minInclusive", "maxInclusive", "minExclusive", "maxExclusive",
			"totalDigits", "fractionDigits", "whiteSpace":
			sv.validateFacet(elem)
		case "simpleContent", "complexContent":
			sv.validateContentModel(elem)
		default:
			sv.addErrorAt(elem, fmt.Sprintf("unknown XSD element: %s", elem.LocalName()))
		}
	}

	// Recursively validate children
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil {
			sv.validateElement(child)
		}
	}
}

// validateIDAttribute validates ID attribute values
func (sv *SchemaValidator) validateIDAttribute(elem xmldom.Element) {
	// Check if id attribute exists
	if !elem.HasAttribute("id") {
		return
	}

	id := elem.GetAttribute("id")
	idStr := string(id)

	// Check for empty ID
	if idStr == "" {
		sv.addErrorAt(elem, "id attribute cannot be empty")
		return
	}

	// Check ID format (must be valid NCName)
	if !isValidNCName(idStr) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid id value '%s': must be a valid NCName", idStr))
		return
	}

	// Check for duplicate IDs
	if existing, exists := sv.idMap[idStr]; exists {
		sv.addErrorAt(elem, fmt.Sprintf("duplicate id value '%s'", idStr))
		if existing != nil {
			sv.addErrorAt(existing, fmt.Sprintf("id '%s' already defined here", idStr))
		}
	} else {
		sv.idMap[idStr] = elem
	}
}

// isValidNCName checks if a string is a valid NCName (non-colonized name)
func isValidNCName(s string) bool {
	if s == "" {
		return false
	}

	// NCName pattern: cannot start with digit, cannot contain colons
	// Must start with letter or underscore, can contain letters, digits, ., -, _
	ncNamePattern := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9._\-]*$`)
	return ncNamePattern.MatchString(s)
}

// validateSimpleType validates xs:simpleType element
func (sv *SchemaValidator) validateSimpleType(elem xmldom.Element) {
	name := elem.GetAttribute("name")

	// Check if name is required (global simpleType)
	parent := elem.ParentNode()
	if parent != nil && string(parent.LocalName()) == "schema" {
		// Global simpleType must have name
		if name == "" {
			sv.addErrorAt(elem, "global simpleType must have a name attribute")
		} else if !isValidNCName(string(name)) {
			sv.addErrorAt(elem, fmt.Sprintf("invalid simpleType name '%s': must be a valid NCName", name))
		}
	} else {
		// Local simpleType must not have name
		if name != "" {
			sv.addErrorAt(elem, "local simpleType must not have a name attribute")
		}
	}

	// Check for required child (restriction, list, or union)
	restrictionCount := 0
	listCount := 0
	unionCount := 0

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.NamespaceURI()) == XSDNamespace {
			switch string(child.LocalName()) {
			case "restriction":
				restrictionCount++
			case "list":
				listCount++
			case "union":
				unionCount++
			}
		}
	}

	count := restrictionCount + listCount + unionCount

	if count == 0 {
		sv.addErrorAt(elem, "simpleType must have exactly one of: restriction, list, or union")
	} else if count > 1 {
		sv.addErrorAt(elem, "simpleType cannot have more than one of: restriction, list, or union")
	}
}

// validateComplexType validates xs:complexType element
func (sv *SchemaValidator) validateComplexType(elem xmldom.Element) {
	name := elem.GetAttribute("name")

	// Check if name is required (global complexType)
	parent := elem.ParentNode()
	if parent != nil && string(parent.LocalName()) == "schema" {
		// Global complexType must have name
		if name == "" {
			sv.addErrorAt(elem, "global complexType must have a name attribute")
		} else if !isValidNCName(string(name)) {
			sv.addErrorAt(elem, fmt.Sprintf("invalid complexType name '%s': must be a valid NCName", name))
		}
	} else {
		// Local complexType must not have name
		if name != "" {
			sv.addErrorAt(elem, "local complexType must not have a name attribute")
		}
	}

	// Validate mixed attribute
	mixed := elem.GetAttribute("mixed")
	if mixed != "" && string(mixed) != "true" && string(mixed) != "false" {
		sv.addErrorAt(elem, fmt.Sprintf("invalid mixed value '%s': must be 'true' or 'false'", mixed))
	}

	// Validate abstract attribute
	abstract := elem.GetAttribute("abstract")
	if abstract != "" && string(abstract) != "true" && string(abstract) != "false" {
		sv.addErrorAt(elem, fmt.Sprintf("invalid abstract value '%s': must be 'true' or 'false'", abstract))
	}
}

// validateElementDecl validates xs:element element
func (sv *SchemaValidator) validateElementDecl(elem xmldom.Element) {
	name := elem.GetAttribute("name")
	ref := elem.GetAttribute("ref")

	// Either name or ref, but not both
	if name != "" && ref != "" {
		sv.addErrorAt(elem, "element cannot have both 'name' and 'ref' attributes")
	}

	// Check if name is required (global element)
	parent := elem.ParentNode()
	if parent != nil && string(parent.LocalName()) == "schema" {
		// Global element must have name
		if name == "" && ref == "" {
			sv.addErrorAt(elem, "global element must have a name attribute")
		}
	}

	// Validate name if present
	if name != "" && !isValidNCName(string(name)) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid element name '%s': must be a valid NCName", name))
	}

	// Validate minOccurs and maxOccurs
	sv.validateOccurrences(elem)

	// Validate type and element content
	typeAttr := elem.GetAttribute("type")
	hasInlineType := false

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.NamespaceURI()) == XSDNamespace {
			if string(child.LocalName()) == "simpleType" || string(child.LocalName()) == "complexType" {
				hasInlineType = true
				break
			}
		}
	}

	if typeAttr != "" && hasInlineType {
		sv.addErrorAt(elem, "element cannot have both 'type' attribute and inline type definition")
	}
}

// validateAttributeDecl validates xs:attribute element
func (sv *SchemaValidator) validateAttributeDecl(elem xmldom.Element) {
	name := elem.GetAttribute("name")
	ref := elem.GetAttribute("ref")

	// Either name or ref, but not both
	if name != "" && ref != "" {
		sv.addErrorAt(elem, "attribute cannot have both 'name' and 'ref' attributes")
	}

	// Validate name if present
	if name != "" && !isValidNCName(string(name)) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid attribute name '%s': must be a valid NCName", name))
	}

	// Validate use attribute
	use := elem.GetAttribute("use")
	if use != "" {
		useStr := string(use)
		if useStr != "optional" && useStr != "required" && useStr != "prohibited" {
			sv.addErrorAt(elem, fmt.Sprintf("invalid use value '%s': must be 'optional', 'required', or 'prohibited'", use))
		}
	}

	// Validate default and fixed are mutually exclusive
	defaultAttr := elem.GetAttribute("default")
	fixed := elem.GetAttribute("fixed")
	if defaultAttr != "" && fixed != "" {
		sv.addErrorAt(elem, "attribute cannot have both 'default' and 'fixed' attributes")
	}
}

// validateOccurrences validates minOccurs and maxOccurs attributes
func (sv *SchemaValidator) validateOccurrences(elem xmldom.Element) {
	minOccurs := elem.GetAttribute("minOccurs")
	maxOccurs := elem.GetAttribute("maxOccurs")

	minVal := 1
	maxVal := 1

	if minOccurs != "" {
		minStr := string(minOccurs)
		if !isNonNegativeInteger(minStr) {
			sv.addErrorAt(elem, fmt.Sprintf("invalid minOccurs value '%s': must be non-negative integer", minStr))
			return
		}
		_, err := fmt.Sscanf(minStr, "%d", &minVal)
		if err != nil {
			sv.addErrorAt(elem, fmt.Sprintf("invalid minOccurs value '%s': must be a valid integer", minStr))
			return
		}
	}

	if maxOccurs != "" {
		maxStr := string(maxOccurs)
		if maxStr != "unbounded" {
			if !isNonNegativeInteger(maxStr) {
				sv.addErrorAt(elem, fmt.Sprintf("invalid maxOccurs value '%s': must be non-negative integer or 'unbounded'", maxStr))
				return
			}
			_, err := fmt.Sscanf(maxStr, "%d", &maxVal)
			if err != nil {
				sv.addErrorAt(elem, fmt.Sprintf("invalid maxOccurs value '%s': must be a valid integer", maxStr))
				return
			}

			// Check min <= max
			if minVal > maxVal {
				sv.addErrorAt(elem, fmt.Sprintf("minOccurs (%d) cannot be greater than maxOccurs (%d)", minVal, maxVal))
			}
		}
	}
}

// isNonNegativeInteger checks if string is a non-negative integer
func isNonNegativeInteger(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// validateRestriction validates xs:restriction element
func (sv *SchemaValidator) validateRestriction(elem xmldom.Element) {
	base := elem.GetAttribute("base")
	if base == "" {
		// Check for inline base type
		hasInlineBase := false
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			child := children.Item(i)
			if child != nil && string(child.NamespaceURI()) == XSDNamespace {
				if string(child.LocalName()) == "simpleType" {
					hasInlineBase = true
					break
				}
			}
		}

		if !hasInlineBase {
			sv.addErrorAt(elem, "restriction must have either 'base' attribute or inline simpleType")
		}
	}
}

// validateExtension validates xs:extension element
func (sv *SchemaValidator) validateExtension(elem xmldom.Element) {
	base := elem.GetAttribute("base")
	if base == "" {
		sv.addErrorAt(elem, "extension must have 'base' attribute")
	}
}

// validateModelGroup validates xs:sequence, xs:choice, xs:all elements
func (sv *SchemaValidator) validateModelGroup(elem xmldom.Element) {
	sv.validateOccurrences(elem)

	// Special rules for xs:all
	if string(elem.LocalName()) == "all" {
		minOccurs := elem.GetAttribute("minOccurs")
		maxOccurs := elem.GetAttribute("maxOccurs")

		// For xs:all, minOccurs must be 0 or 1
		if minOccurs != "" && string(minOccurs) != "0" && string(minOccurs) != "1" {
			sv.addErrorAt(elem, "xs:all minOccurs must be 0 or 1")
		}

		// For xs:all, maxOccurs must be 1
		if maxOccurs != "" && string(maxOccurs) != "1" {
			sv.addErrorAt(elem, "xs:all maxOccurs must be 1")
		}

		// Children of xs:all must have maxOccurs 0 or 1 (in XSD 1.0)
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			child := children.Item(i)
			if child != nil && string(child.LocalName()) == "element" {
				childMax := child.GetAttribute("maxOccurs")
				if childMax != "" && string(childMax) != "0" && string(childMax) != "1" {
					sv.addErrorAt(child, "elements within xs:all must have maxOccurs of 0 or 1 (XSD 1.0)")
				}
			}
		}
	}
}

// validateGroup validates xs:group element
func (sv *SchemaValidator) validateGroup(elem xmldom.Element) {
	name := elem.GetAttribute("name")
	ref := elem.GetAttribute("ref")

	// Either name or ref, but not both
	if name != "" && ref != "" {
		sv.addErrorAt(elem, "group cannot have both 'name' and 'ref' attributes")
	}

	// Global group must have name
	parent := elem.ParentNode()
	if parent != nil && string(parent.LocalName()) == "schema" {
		if name == "" && ref == "" {
			sv.addErrorAt(elem, "global group must have a name attribute")
		}
	}

	// Group reference must have ref
	if parent != nil && string(parent.LocalName()) != "schema" && ref == "" && name == "" {
		sv.addErrorAt(elem, "group reference must have 'ref' attribute")
	}

	// Validate name if present
	if name != "" && !isValidNCName(string(name)) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid group name '%s': must be a valid NCName", name))
	}
}

// validateAttributeGroup validates xs:attributeGroup element
func (sv *SchemaValidator) validateAttributeGroup(elem xmldom.Element) {
	name := elem.GetAttribute("name")
	ref := elem.GetAttribute("ref")

	// Either name or ref, but not both
	if name != "" && ref != "" {
		sv.addErrorAt(elem, "attributeGroup cannot have both 'name' and 'ref' attributes")
	}

	// Validate name if present
	if name != "" && !isValidNCName(string(name)) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid attributeGroup name '%s': must be a valid NCName", name))
	}
}

// validateImport validates xs:import element
func (sv *SchemaValidator) validateImport(elem xmldom.Element) {
	// namespace attribute is optional but common
	// schemaLocation is optional
	// No specific validation needed beyond basic structure
}

// validateInclude validates xs:include element
func (sv *SchemaValidator) validateInclude(elem xmldom.Element) {
	schemaLocation := elem.GetAttribute("schemaLocation")
	if schemaLocation == "" {
		sv.addErrorAt(elem, "include must have 'schemaLocation' attribute")
	}
}

// validateAny validates xs:any element
func (sv *SchemaValidator) validateAny(elem xmldom.Element) {
	sv.validateOccurrences(elem)

	// Validate processContents
	processContents := elem.GetAttribute("processContents")
	if processContents != "" {
		pc := string(processContents)
		if pc != "strict" && pc != "lax" && pc != "skip" {
			sv.addErrorAt(elem, fmt.Sprintf("invalid processContents value '%s': must be 'strict', 'lax', or 'skip'", pc))
		}
	}
}

// validateAnyAttribute validates xs:anyAttribute element
func (sv *SchemaValidator) validateAnyAttribute(elem xmldom.Element) {
	// Validate processContents
	processContents := elem.GetAttribute("processContents")
	if processContents != "" {
		pc := string(processContents)
		if pc != "strict" && pc != "lax" && pc != "skip" {
			sv.addErrorAt(elem, fmt.Sprintf("invalid processContents value '%s': must be 'strict', 'lax', or 'skip'", pc))
		}
	}
}

// validateIdentityConstraint validates xs:unique, xs:key, xs:keyref elements
func (sv *SchemaValidator) validateIdentityConstraint(elem xmldom.Element) {
	name := elem.GetAttribute("name")
	if name == "" {
		sv.addErrorAt(elem, fmt.Sprintf("%s must have 'name' attribute", elem.LocalName()))
	} else if !isValidNCName(string(name)) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid %s name '%s': must be a valid NCName", elem.LocalName(), name))
	}

	// For keyref, check refer attribute
	if string(elem.LocalName()) == "keyref" {
		refer := elem.GetAttribute("refer")
		if refer == "" {
			sv.addErrorAt(elem, "keyref must have 'refer' attribute")
		}
	}

	// Check for required selector and field children
	hasSelector := false
	fieldCount := 0

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.NamespaceURI()) == XSDNamespace {
			switch string(child.LocalName()) {
			case "selector":
				hasSelector = true
			case "field":
				fieldCount++
			}
		}
	}

	if !hasSelector {
		sv.addErrorAt(elem, fmt.Sprintf("%s must have a selector child element", elem.LocalName()))
	}
	if fieldCount == 0 {
		sv.addErrorAt(elem, fmt.Sprintf("%s must have at least one field child element", elem.LocalName()))
	}
}

// validateXPathElement validates xs:selector and xs:field elements
func (sv *SchemaValidator) validateXPathElement(elem xmldom.Element) {
	xpath := elem.GetAttribute("xpath")
	if xpath == "" {
		sv.addErrorAt(elem, fmt.Sprintf("%s must have 'xpath' attribute", elem.LocalName()))
	}
}

// validateNotation validates xs:notation element
func (sv *SchemaValidator) validateNotation(elem xmldom.Element) {
	name := elem.GetAttribute("name")
	if name == "" {
		sv.addErrorAt(elem, "notation must have 'name' attribute")
	} else if !isValidNCName(string(name)) {
		sv.addErrorAt(elem, fmt.Sprintf("invalid notation name '%s': must be a valid NCName", name))
	}

	// Must have either public or system
	public := elem.GetAttribute("public")
	system := elem.GetAttribute("system")
	if public == "" && system == "" {
		sv.addErrorAt(elem, "notation must have either 'public' or 'system' attribute")
	}
}

// validateUnion validates xs:union element
func (sv *SchemaValidator) validateUnion(elem xmldom.Element) {
	memberTypes := elem.GetAttribute("memberTypes")

	// Check for inline simpleTypes
	hasInlineTypes := false
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.NamespaceURI()) == XSDNamespace {
			if string(child.LocalName()) == "simpleType" {
				hasInlineTypes = true
				break
			}
		}
	}

	// Must have either memberTypes or inline simpleTypes
	if memberTypes == "" && !hasInlineTypes {
		sv.addErrorAt(elem, "union must have either 'memberTypes' attribute or inline simpleType elements")
	}
}

// validateList validates xs:list element
func (sv *SchemaValidator) validateList(elem xmldom.Element) {
	itemType := elem.GetAttribute("itemType")

	// Check for inline simpleType
	hasInlineType := false
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.NamespaceURI()) == XSDNamespace {
			if string(child.LocalName()) == "simpleType" {
				hasInlineType = true
				break
			}
		}
	}

	// Must have either itemType or inline simpleType, but not both
	if itemType == "" && !hasInlineType {
		sv.addErrorAt(elem, "list must have either 'itemType' attribute or inline simpleType element")
	} else if itemType != "" && hasInlineType {
		sv.addErrorAt(elem, "list cannot have both 'itemType' attribute and inline simpleType element")
	}
}

// validateFacet validates facet elements
func (sv *SchemaValidator) validateFacet(elem xmldom.Element) {
	value := elem.GetAttribute("value")
	if value == "" {
		sv.addErrorAt(elem, fmt.Sprintf("%s facet must have 'value' attribute", elem.LocalName()))
	}

	// Validate fixed attribute if present
	fixed := elem.GetAttribute("fixed")
	if fixed != "" && string(fixed) != "true" && string(fixed) != "false" {
		sv.addErrorAt(elem, fmt.Sprintf("invalid fixed value '%s': must be 'true' or 'false'", fixed))
	}
}

// validateContentModel validates xs:simpleContent and xs:complexContent
func (sv *SchemaValidator) validateContentModel(elem xmldom.Element) {
	// Check for required child (restriction or extension)
	hasRestriction := false
	hasExtension := false

	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.NamespaceURI()) == XSDNamespace {
			switch string(child.LocalName()) {
			case "restriction":
				hasRestriction = true
			case "extension":
				hasExtension = true
			}
		}
	}

	if !hasRestriction && !hasExtension {
		sv.addErrorAt(elem, fmt.Sprintf("%s must have either restriction or extension child", elem.LocalName()))
	} else if hasRestriction && hasExtension {
		sv.addErrorAt(elem, fmt.Sprintf("%s cannot have both restriction and extension children", elem.LocalName()))
	}
}

// addError adds an error to the list
func (sv *SchemaValidator) addError(msg string) {
	sv.errors = append(sv.errors, fmt.Errorf("%s", msg))
}

// addErrorAt adds an error with element context
func (sv *SchemaValidator) addErrorAt(elem xmldom.Element, msg string) {
	// Try to get element location info
	name := elem.GetAttribute("name")
	if name == "" {
		name = elem.GetAttribute("ref")
	}

	location := fmt.Sprintf("<%s", elem.LocalName())
	if name != "" {
		location += fmt.Sprintf(" name='%s'", name)
	}
	location += ">"

	sv.errors = append(sv.errors, fmt.Errorf("%s: %s", location, msg))
}

// ValidateSchemaFile validates an XSD schema file and returns any errors
func ValidateSchemaFile(filename string) []error {
	doc, err := xmldom.Decode(strings.NewReader("")) // We'll need to load the file
	if err != nil {
		return []error{fmt.Errorf("failed to parse schema file: %w", err)}
	}

	validator := NewSchemaValidator()
	return validator.ValidateSchema(doc)
}
