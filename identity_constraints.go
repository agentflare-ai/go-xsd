package xsd

import (
	"fmt"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// IdentityConstraintKind represents the type of identity constraint
type IdentityConstraintKind string

const (
	KeyConstraint    IdentityConstraintKind = "key"
	KeyRefConstraint IdentityConstraintKind = "keyref"
	UniqueConstraint IdentityConstraintKind = "unique"
)

// IdentityConstraint represents an identity constraint (key, keyref, or unique)
type IdentityConstraint struct {
	Name     string
	Kind     IdentityConstraintKind
	Selector *Selector
	Fields   []*Field
	Refer    QName // For keyref, refers to a key or unique constraint
}

// Selector represents the xs:selector element
type Selector struct {
	XPath string // XPath expression to select nodes
}

// Field represents the xs:field element
type Field struct {
	XPath string // XPath expression to select field value
}

// IdentityConstraintValidator validates identity constraints in XML documents
type IdentityConstraintValidator struct {
	constraints map[string]*IdentityConstraint         // Map of constraint name to constraint
	keyValues   map[string]map[string][]xmldom.Element // constraint name -> concatenated field values -> elements
}

// NewIdentityConstraintValidator creates a new identity constraint validator
func NewIdentityConstraintValidator() *IdentityConstraintValidator {
	return &IdentityConstraintValidator{
		constraints: make(map[string]*IdentityConstraint),
		keyValues:   make(map[string]map[string][]xmldom.Element),
	}
}

// AddConstraint adds an identity constraint to the validator
func (v *IdentityConstraintValidator) AddConstraint(constraint *IdentityConstraint) {
	v.constraints[constraint.Name] = constraint
	v.keyValues[constraint.Name] = make(map[string][]xmldom.Element)
}

// Validate validates all identity constraints in the document
func (v *IdentityConstraintValidator) Validate(doc xmldom.Document) []Violation {
	violations := []Violation{}

	// First pass: collect all key values
	for name, constraint := range v.constraints {
		if constraint.Kind == KeyConstraint || constraint.Kind == UniqueConstraint {
			selectedNodes := v.evaluateSelector(doc, constraint.Selector)

			for _, node := range selectedNodes {
				fieldValues := v.extractFieldValues(node, constraint.Fields)
				if len(fieldValues) == 0 {
					continue // Skip if no field values found
				}

				// Concatenate field values to create a unique key
				keyValue := strings.Join(fieldValues, "|")

				// Check for duplicates
				if existingNodes, exists := v.keyValues[name][keyValue]; exists {
					// For key and unique, duplicates are not allowed
					violations = append(violations, Violation{
						Element: node,
						Code:    "cvc-identity-constraint.4.1",
						Message: fmt.Sprintf("Duplicate %s constraint '%s' value: %s",
							constraint.Kind, name, keyValue),
					})
					// Still add it to track all duplicates
					v.keyValues[name][keyValue] = append(existingNodes, node)
				} else {
					v.keyValues[name][keyValue] = []xmldom.Element{node}
				}

				// For key constraints, all fields must be non-null
				if constraint.Kind == KeyConstraint {
					for i, fieldValue := range fieldValues {
						if fieldValue == "" {
							violations = append(violations, Violation{
								Element: node,
								Code:    "cvc-identity-constraint.4.2.2",
								Message: fmt.Sprintf("Key constraint '%s' field %d cannot be null",
									name, i+1),
							})
						}
					}
				}
			}
		}
	}

	// Second pass: validate keyrefs
	for name, constraint := range v.constraints {
		if constraint.Kind == KeyRefConstraint {
			selectedNodes := v.evaluateSelector(doc, constraint.Selector)

			// Find the referenced key/unique constraint
			referencedConstraint, exists := v.constraints[constraint.Refer.Local]
			if !exists {
				violations = append(violations, Violation{
					Code: "src-identity-constraint.2.2.2",
					Message: fmt.Sprintf("Keyref '%s' refers to unknown constraint '%s'",
						name, constraint.Refer.Local),
				})
				continue
			}

			for _, node := range selectedNodes {
				fieldValues := v.extractFieldValues(node, constraint.Fields)
				if len(fieldValues) == 0 {
					continue
				}

				keyValue := strings.Join(fieldValues, "|")

				// Check if this keyref value exists in the referenced constraint
				if _, exists := v.keyValues[constraint.Refer.Local][keyValue]; !exists {
					violations = append(violations, Violation{
						Element: node,
						Code:    "cvc-identity-constraint.4.3",
						Message: fmt.Sprintf("Keyref '%s' value '%s' does not match any %s '%s'",
							name, keyValue, referencedConstraint.Kind, constraint.Refer.Local),
					})
				}
			}
		}
	}

	return violations
}

// evaluateSelector evaluates the selector XPath to find matching nodes
func (v *IdentityConstraintValidator) evaluateSelector(doc xmldom.Document, selector *Selector) []xmldom.Element {
	if selector == nil || selector.XPath == "" {
		return nil
	}

	// Simplified XPath evaluation for common patterns
	// This handles basic paths like "employee", ".//employee", "department/employee"
	return v.evaluateSimpleXPath(doc.DocumentElement(), selector.XPath)
}

// extractFieldValues extracts field values from a node using field XPaths
func (v *IdentityConstraintValidator) extractFieldValues(node xmldom.Element, fields []*Field) []string {
	values := make([]string, 0, len(fields))

	for _, field := range fields {
		value := v.evaluateFieldXPath(node, field.XPath)
		values = append(values, value)
	}

	return values
}

// evaluateSimpleXPath evaluates a simplified XPath expression
func (v *IdentityConstraintValidator) evaluateSimpleXPath(root xmldom.Element, xpath string) []xmldom.Element {
	results := []xmldom.Element{}

	// Remove leading/trailing whitespace
	xpath = strings.TrimSpace(xpath)

	// Remove namespace prefixes from the XPath for simplicity
	// Convert "ex:element" to "element"
	xpath = removeNamespacePrefixes(xpath)

	// Handle absolute paths (starting with /)
	if strings.HasPrefix(xpath, "/") {
		// For now, treat absolute paths as starting from root
		xpath = strings.TrimPrefix(xpath, "/")
	}

	// Handle descendant-or-self axis (.// or //)
	searchDescendants := false
	if strings.HasPrefix(xpath, ".//") {
		searchDescendants = true
		xpath = strings.TrimPrefix(xpath, ".//")
	} else if strings.HasPrefix(xpath, "//") {
		searchDescendants = true
		xpath = strings.TrimPrefix(xpath, "//")
	}

	// Split path into steps
	steps := strings.Split(xpath, "/")

	if searchDescendants {
		// Search all descendants
		v.findMatchingDescendants(root, steps[0], &results)

		// For multi-step paths after //, continue from each result
		if len(steps) > 1 {
			newResults := []xmldom.Element{}
			for _, elem := range results {
				v.findMatchingChildren(elem, steps[1:], &newResults)
			}
			results = newResults
		}
	} else {
		// Direct path navigation
		v.findMatchingChildren(root, steps, &results)
	}

	return results
}

// findMatchingChildren finds child elements matching the path steps
func (v *IdentityConstraintValidator) findMatchingChildren(elem xmldom.Element, steps []string, results *[]xmldom.Element) {
	if len(steps) == 0 {
		*results = append(*results, elem)
		return
	}

	currentStep := steps[0]
	remainingSteps := steps[1:]

	// Handle self (.)
	if currentStep == "." {
		v.findMatchingChildren(elem, remainingSteps, results)
		return
	}

	// Find matching children
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil && string(child.LocalName()) == currentStep {
			if len(remainingSteps) == 0 {
				*results = append(*results, child)
			} else {
				v.findMatchingChildren(child, remainingSteps, results)
			}
		}
	}
}

// findMatchingDescendants finds all descendant elements matching the name
func (v *IdentityConstraintValidator) findMatchingDescendants(elem xmldom.Element, name string, results *[]xmldom.Element) {
	// Check current element
	if string(elem.LocalName()) == name {
		*results = append(*results, elem)
	}

	// Check all children recursively
	children := elem.Children()
	for i := uint(0); i < children.Length(); i++ {
		child := children.Item(i)
		if child != nil {
			v.findMatchingDescendants(child, name, results)
		}
	}
}

// evaluateFieldXPath evaluates a field XPath expression to get a string value
func (v *IdentityConstraintValidator) evaluateFieldXPath(node xmldom.Element, xpath string) string {
	xpath = strings.TrimSpace(xpath)

	// Handle attribute selection (@attributeName)
	if strings.HasPrefix(xpath, "@") {
		attrName := strings.TrimPrefix(xpath, "@")
		attr := node.GetAttribute(xmldom.DOMString(attrName))
		return string(attr)
	}

	// Handle text() function
	if xpath == "." || xpath == "text()" {
		return getElementTextContent(node)
	}

	// Handle child element path
	if !strings.Contains(xpath, "@") && !strings.Contains(xpath, "()") {
		// It's an element path, find the element and get its text content
		elements := v.evaluateSimpleXPath(node, xpath)
		if len(elements) > 0 {
			return getElementTextContent(elements[0])
		}
	}

	// Handle paths like element/@attribute
	if strings.Contains(xpath, "/@") {
		parts := strings.Split(xpath, "/@")
		if len(parts) == 2 {
			elements := v.evaluateSimpleXPath(node, parts[0])
			if len(elements) > 0 {
				attr := elements[0].GetAttribute(xmldom.DOMString(parts[1]))
				return string(attr)
			}
		}
	}

	return ""
}

// removeNamespacePrefixes removes namespace prefixes from XPath expressions
func removeNamespacePrefixes(xpath string) string {
	// Remove namespace prefixes like "ex:" from element names
	parts := strings.Split(xpath, "/")
	for i, part := range parts {
		if idx := strings.Index(part, ":"); idx > 0 && !strings.HasPrefix(part, "@") {
			// Remove prefix but keep the local name
			parts[i] = part[idx+1:]
		}
	}
	return strings.Join(parts, "/")
}
