// union_list_types.go - Implementation of XSD union and list simple types validation
// 
// This file provides validation support for:
// - Union types: values that can be valid against any one of multiple member types
// - List types: space-separated lists of values of a specific item type  
// - Proper namespace resolution for type references (fixed parseQName in schema.go)
// - Integration with the overall XSD validation framework
package xsd

import (
	"fmt"
	"strings"
)

// ValidateUnionType validates a value against a union type
// A union type allows a value to be valid against any one of its member types
func ValidateUnionType(value string, union *Union, schema *Schema) error {
	if union == nil || len(union.MemberTypes) == 0 {
		return fmt.Errorf("union type has no member types")
	}

	var lastError error
	// Try to validate against each member type
	for _, memberType := range union.MemberTypes {
		// Resolve the member type
		var resolvedType Type
		if t, exists := schema.TypeDefs[memberType]; exists {
			resolvedType = t
		} else {
			// Check if it's a built-in type
			if validator := GetBuiltinTypeValidator(memberType.Local); validator != nil {
				err := validator(value)
				if err == nil {
					// Valid against this built-in type
					return nil
				}
				lastError = err
				continue
			}
		}

		// Validate against the resolved type
		if resolvedType != nil {
			// Create a dummy element for validation
			if err := validateValueAgainstType(value, resolvedType, schema); err == nil {
				// Valid against this member type
				return nil
			} else {
				lastError = err
			}
		}
	}

	// Not valid against any member type
	if lastError != nil {
		return fmt.Errorf("value '%s' is not valid against any member type of the union: %v", value, lastError)
	}
	return fmt.Errorf("value '%s' is not valid against any member type of the union", value)
}

// ValidateListType validates a value against a list type
// A list type represents a space-separated list of values of the item type
func ValidateListType(value string, list *List, schema *Schema) error {
	if list == nil || list.ItemType == (QName{}) {
		return fmt.Errorf("list type has no item type")
	}

	// Split the value into list items (space-separated)
	items := strings.Fields(value)
	if len(items) == 0 && value != "" {
		// If value is not empty but has no items, it might be all whitespace
		return fmt.Errorf("list value contains only whitespace")
	}

	// Resolve the item type
	var itemType Type
	if t, exists := schema.TypeDefs[list.ItemType]; exists {
		itemType = t
	} else {
		// Check if it's a built-in type
		if validator := GetBuiltinTypeValidator(list.ItemType.Local); validator != nil {
			// Validate each item against the built-in type
			for i, item := range items {
				if err := validator(item); err != nil {
					return fmt.Errorf("list item %d ('%s') is invalid: %v", i+1, item, err)
				}
			}
			return nil
		}
		return fmt.Errorf("unknown item type: %s", list.ItemType)
	}

	// Validate each item against the resolved type
	for i, item := range items {
		if err := validateValueAgainstType(item, itemType, schema); err != nil {
			return fmt.Errorf("list item %d ('%s') is invalid: %v", i+1, item, err)
		}
	}

	return nil
}

// validateValueAgainstType validates a string value against a type
func validateValueAgainstType(value string, t Type, schema *Schema) error {
	switch typ := t.(type) {
	case *SimpleType:
		return validateSimpleTypeValue(value, typ, schema)
	case *ComplexType:
		// Complex types can't be used in unions or lists
		return fmt.Errorf("complex type cannot be used in union or list")
	default:
		return fmt.Errorf("unknown type: %T", t)
	}
}

// validateSimpleTypeValue validates a value against a simple type
func validateSimpleTypeValue(value string, st *SimpleType, schema *Schema) error {
	// Handle restriction
	if st.Restriction != nil {
		// First validate against base type if it exists
		if st.Restriction.Base != (QName{}) {
			// Check built-in base type
			if validator := GetBuiltinTypeValidator(st.Restriction.Base.Local); validator != nil {
				if err := validator(value); err != nil {
					return err
				}
			} else if baseType, exists := schema.TypeDefs[st.Restriction.Base]; exists {
				// Validate against user-defined base type
				if err := validateValueAgainstType(value, baseType, schema); err != nil {
					return err
				}
				
				// Special handling for restrictions on list types
				if baseST, ok := baseType.(*SimpleType); ok && baseST.List != nil {
					// When base is a list, length facets apply to the list itself
					// Count the number of items in the list
					items := strings.Fields(value)
					
					// Apply length facets to the list length
					for _, facet := range st.Restriction.Facets {
						switch f := facet.(type) {
						case *LengthFacet:
							if len(items) != f.Value {
								return fmt.Errorf("list length must be exactly %d, got %d", f.Value, len(items))
							}
						case *MinLengthFacet:
							if len(items) < f.Value {
								return fmt.Errorf("list length must be at least %d, got %d", f.Value, len(items))
							}
						case *MaxLengthFacet:
							if len(items) > f.Value {
								return fmt.Errorf("list length must be at most %d, got %d", f.Value, len(items))
							}
						default:
							// Other facets apply to the value as a whole
							if err := facet.Validate(value, baseType); err != nil {
								return err
							}
						}
					}
					return nil
				}
			}
		}

		// Apply facets normally
		// Determine the base type for facet validation
		var baseType Type
		if st.Restriction.Base.Local != "" {
			// Try to get the base type from schema
			if t, exists := schema.TypeDefs[st.Restriction.Base]; exists {
				baseType = t
			} else {
				// It might be a built-in type
				baseType = &SimpleType{QName: st.Restriction.Base}
			}
		}
		
		for _, facet := range st.Restriction.Facets {
			if err := facet.Validate(value, baseType); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle list
	if st.List != nil {
		return ValidateListType(value, st.List, schema)
	}

	// Handle union
	if st.Union != nil {
		return ValidateUnionType(value, st.Union, schema)
	}

	// If it's just a named type without restriction/list/union,
	// it might be a reference to a built-in type
	if st.QName.Local != "" {
		if validator := GetBuiltinTypeValidator(st.QName.Local); validator != nil {
			return validator(value)
		}
	}

	return nil
}

// GetBuiltinTypeValidator returns a validator function for a built-in XSD type
func GetBuiltinTypeValidator(typeName string) func(string) error {
	// Remove namespace prefix if present
	if idx := strings.Index(typeName, ":"); idx >= 0 {
		typeName = typeName[idx+1:]
	}

	// Get the built-in type
	builtinType := GetBuiltinType(typeName)
	if builtinType != nil {
		return builtinType.Validator
	}

	return nil
}
