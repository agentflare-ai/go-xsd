package xsd

import (
	"fmt"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// ProcessContentsMode defines how wildcard content should be processed
type ProcessContentsMode string

const (
	// StrictProcess requires the element/attribute to be validated against its declaration
	StrictProcess ProcessContentsMode = "strict"
	// LaxProcess validates if a declaration is found, otherwise allows it
	LaxProcess ProcessContentsMode = "lax"
	// SkipProcess allows the element/attribute without validation
	SkipProcess ProcessContentsMode = "skip"
)

// WildcardNamespaceConstraint represents namespace constraints for wildcards
type WildcardNamespaceConstraint struct {
	Mode       string   // "##any", "##other", "##targetNamespace", "##local", or space-separated list
	Namespaces []string // Explicit list of allowed namespaces (when not using ##modes)
}

// ParseNamespaceConstraint parses a namespace attribute value into a constraint
func ParseNamespaceConstraint(value string) *WildcardNamespaceConstraint {
	if value == "" {
		value = "##any" // Default
	}

	constraint := &WildcardNamespaceConstraint{
		Mode: value,
	}

	// If it's not a special mode, parse as space-separated list
	if !strings.HasPrefix(value, "##") {
		constraint.Namespaces = strings.Fields(value)
		constraint.Mode = "list"
	}

	return constraint
}

// Matches checks if a namespace matches this constraint
func (c *WildcardNamespaceConstraint) Matches(namespace, targetNamespace string) bool {
	switch c.Mode {
	case "##any":
		return true
	case "##other":
		return namespace != targetNamespace
	case "##targetNamespace":
		return namespace == targetNamespace
	case "##local":
		return namespace == ""
	case "list":
		// Check explicit namespace list
		for _, ns := range c.Namespaces {
			if ns == namespace {
				return true
			}
		}
		// Also check for ##targetNamespace or ##local in the list
		for _, ns := range c.Namespaces {
			if ns == "##targetNamespace" && namespace == targetNamespace {
				return true
			}
			if ns == "##local" && namespace == "" {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// ValidateAnyElement validates an element against xs:any wildcard constraints
func ValidateAnyElement(elem xmldom.Element, wildcard *AnyElement, schema *Schema) []Violation {
	violations := []Violation{}

	elemNS := string(elem.NamespaceURI())
	elemName := string(elem.LocalName())

	// Parse namespace constraint
	nsConstraint := ParseNamespaceConstraint(wildcard.Namespace)

	// Check namespace constraint
	if !nsConstraint.Matches(elemNS, schema.TargetNamespace) {
		violations = append(violations, Violation{
			Element: elem,
			Code:    "cvc-wildcard.2",
			Message: fmt.Sprintf("Element '{%s}%s' is not allowed by the namespace constraint '%s'",
				elemNS, elemName, wildcard.Namespace),
		})
		return violations
	}

	// Handle processContents
	mode := ProcessContentsMode(wildcard.ProcessContents)
	if mode == "" {
		mode = StrictProcess // Default
	}

	switch mode {
	case StrictProcess:
		// Must validate against element declaration
		qname := QName{Namespace: elemNS, Local: elemName}
		if decl, found := schema.ElementDecls[qname]; found {
			// Validate element against its declaration
			if decl.Type != nil {
				typeViolations := decl.Type.Validate(elem, schema)
				violations = append(violations, typeViolations...)
			}
		} else {
			// In strict mode, element must have a declaration
			violations = append(violations, Violation{
				Element: elem,
				Code:    "cvc-assess-elt.1.1.1",
				Message: fmt.Sprintf("No element declaration found for '{%s}%s' (processContents='strict')",
					elemNS, elemName),
			})
		}

	case LaxProcess:
		// Validate if declaration is found, otherwise allow
		qname := QName{Namespace: elemNS, Local: elemName}
		if decl, found := schema.ElementDecls[qname]; found {
			// Found declaration, validate against it
			if decl.Type != nil {
				typeViolations := decl.Type.Validate(elem, schema)
				violations = append(violations, typeViolations...)
			}
		}
		// If no declaration found, that's OK in lax mode

	case SkipProcess:
		// No validation required
		return nil

	default:
		violations = append(violations, Violation{
			Element: elem,
			Code:    "cvc-wildcard.3",
			Message: fmt.Sprintf("Invalid processContents value: '%s'", wildcard.ProcessContents),
		})
	}

	return violations
}

// ValidateAnyAttribute validates attributes against xs:anyAttribute wildcard constraints
func ValidateAnyAttribute(attr xmldom.Node, wildcard *AnyAttribute, schema *Schema) []Violation {
	violations := []Violation{}

	attrNS := string(attr.NamespaceURI())
	attrName := string(attr.LocalName())

	// Skip namespace declarations
	if attrNS == "http://www.w3.org/2000/xmlns/" || attrName == "xmlns" {
		return nil
	}

	// Skip xsi attributes (like xsi:type, xsi:nil, etc.)
	if attrNS == "http://www.w3.org/2001/XMLSchema-instance" {
		return nil
	}

	// Parse namespace constraint
	nsConstraint := ParseNamespaceConstraint(wildcard.Namespace)

	// Check namespace constraint
	if !nsConstraint.Matches(attrNS, schema.TargetNamespace) {
		violations = append(violations, Violation{
			Code: "cvc-wildcard-attribute.2",
			Message: fmt.Sprintf("Attribute '{%s}%s' is not allowed by the anyAttribute namespace constraint '%s'",
				attrNS, attrName, wildcard.Namespace),
		})
		return violations
	}

	// Handle processContents
	mode := ProcessContentsMode(wildcard.ProcessContents)
	if mode == "" {
		mode = StrictProcess // Default
	}

	switch mode {
	case StrictProcess:
		// For attributes, we would need global attribute declarations (not commonly used)
		// For now, we'll allow it if namespace matches
		// In a full implementation, we'd look up global attribute declarations

	case LaxProcess:
		// Similar to strict but more permissive

	case SkipProcess:
		// No validation required

	default:
		violations = append(violations, Violation{
			Code:    "cvc-wildcard-attribute.3",
			Message: fmt.Sprintf("Invalid processContents value for anyAttribute: '%s'", wildcard.ProcessContents),
		})
	}

	return violations
}

// MatchesWildcard checks if an element matches a wildcard's namespace constraint
func MatchesWildcard(elem xmldom.Element, namespace string, targetNamespace string) bool {
	elemNS := string(elem.NamespaceURI())
	constraint := ParseNamespaceConstraint(namespace)
	return constraint.Matches(elemNS, targetNamespace)
}

// CountWildcardMatches counts how many elements match a wildcard
func CountWildcardMatches(elements []xmldom.Element, wildcard *AnyElement, targetNamespace string) int {
	count := 0
	for _, elem := range elements {
		if MatchesWildcard(elem, wildcard.Namespace, targetNamespace) {
			count++
		}
	}
	return count
}

// ValidateWildcardOccurrences validates that wildcard occurrences are within bounds
func ValidateWildcardOccurrences(matchCount int, wildcard *AnyElement) []Violation {
	violations := []Violation{}

	minOcc := wildcard.MinOcc
	maxOcc := wildcard.MaxOcc

	if matchCount < minOcc {
		violations = append(violations, Violation{
			Code:    "cvc-complex-type.2.4.b",
			Message: fmt.Sprintf("Expected at least %d wildcard element(s), found %d", minOcc, matchCount),
		})
	}

	if maxOcc != -1 && matchCount > maxOcc {
		violations = append(violations, Violation{
			Code:    "cvc-complex-type.2.4.d",
			Message: fmt.Sprintf("Expected at most %d wildcard element(s), found %d", maxOcc, matchCount),
		})
	}

	return violations
}
