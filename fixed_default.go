package xsd

import (
	"fmt"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// ValidateFixedValue validates that an element or attribute has the required fixed value
func ValidateFixedValue(value, fixed string, isElement bool, name string) *Violation {
	if fixed != "" && value != fixed {
		if isElement {
			return &Violation{
				Code:    "cvc-elt.5.2.2",
				Message: fmt.Sprintf("Element '%s' must have fixed value '%s' but has '%s'", name, fixed, value),
			}
		}
		return &Violation{
			Code:    "cvc-attribute.4",
			Message: fmt.Sprintf("Attribute '%s' must have fixed value '%s' but has '%s'", name, fixed, value),
		}
	}
	return nil
}

// ApplyDefaultValue applies a default value to an element or attribute if it's empty
func ApplyDefaultValue(elem xmldom.Element, defaultValue string) string {
	if elem == nil {
		return defaultValue
	}

	content := strings.TrimSpace(string(elem.TextContent()))
	if content == "" && defaultValue != "" {
		return defaultValue
	}
	return content
}

// ValidateElementFixedDefault validates fixed and default values for an element
func ValidateElementFixedDefault(elem xmldom.Element, decl *ElementDecl) []Violation {
	var violations []Violation

	if decl == nil {
		return violations
	}

	// Get element content
	content := strings.TrimSpace(string(elem.TextContent()))

	// Check fixed value
	if decl.Fixed != "" {
		// If element has children elements, we can't validate fixed value on mixed content
		hasChildElements := false
		children := elem.Children()
		for i := uint(0); i < children.Length(); i++ {
			if children.Item(i) != nil {
				hasChildElements = true
				break
			}
		}

		if !hasChildElements {
			// Only validate fixed value for simple content
			if violation := ValidateFixedValue(content, decl.Fixed, true, decl.Name.Local); violation != nil {
				violation.Element = elem
				violations = append(violations, *violation)
			}
		}
	}

	// Note: Default values are typically applied during parsing/processing,
	// not during validation. The validator checks the content after defaults are applied.

	return violations
}

// ValidateAttributeFixedDefault validates fixed and default values for an attribute
func ValidateAttributeFixedDefault(attr xmldom.Node, decl *AttributeDecl, elem xmldom.Element) []Violation {
	var violations []Violation

	if decl == nil {
		return violations
	}

	// Get attribute value
	var value string
	if attr != nil {
		value = string(attr.NodeValue())
	} else if decl.Default != "" {
		// If attribute is not present and has a default, use the default value
		value = decl.Default
	}

	// Check fixed value
	if decl.Fixed != "" {
		if attr == nil && decl.Use != RequiredUse {
			// If attribute is not present but has a fixed value,
			// it's considered to have the fixed value
			value = decl.Fixed
		}

		if value != decl.Fixed {
			violation := &Violation{
				Element: elem,
				Code:    "cvc-attribute.4",
				Message: fmt.Sprintf("Attribute '%s' must have fixed value '%s' but has '%s'",
					decl.Name.Local, decl.Fixed, value),
			}
			violations = append(violations, *violation)
		}
	}

	return violations
}

// HasDefaultValue checks if an element or attribute declaration has a default value
func HasDefaultValue(decl interface{}) (string, bool) {
	switch d := decl.(type) {
	case *ElementDecl:
		if d.Default != "" {
			return d.Default, true
		}
	case *AttributeDecl:
		if d.Default != "" {
			return d.Default, true
		}
	}
	return "", false
}

// HasFixedValue checks if an element or attribute declaration has a fixed value
func HasFixedValue(decl interface{}) (string, bool) {
	switch d := decl.(type) {
	case *ElementDecl:
		if d.Fixed != "" {
			return d.Fixed, true
		}
	case *AttributeDecl:
		if d.Fixed != "" {
			return d.Fixed, true
		}
	}
	return "", false
}
