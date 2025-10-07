package xsd

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// FacetValidator validates a value against a facet constraint
type FacetValidator interface {
	Validate(value string, baseType Type) error
	Name() string
}

// PatternFacet validates against a regular expression pattern
type PatternFacet struct {
	Pattern string
	regex   *regexp.Regexp
}

func (f *PatternFacet) Name() string {
	return "pattern"
}

func (f *PatternFacet) Validate(value string, baseType Type) error {
	if f.regex == nil {
		// Compile XSD regex pattern to Go regex
		// XSD patterns are anchored by default
		pattern := "^" + convertXSDRegex(f.Pattern) + "$"
		var err error
		f.regex, err = regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern: %v", err)
		}
	}
	
	if !f.regex.MatchString(value) {
		return fmt.Errorf("value '%s' does not match pattern '%s'", value, f.Pattern)
	}
	return nil
}

// convertXSDRegex converts XSD regex patterns to Go regex
func convertXSDRegex(pattern string) string {
	// Basic conversion - XSD uses some different syntax than Go
	result := pattern
	
	// Convert character class shortcuts
	result = strings.ReplaceAll(result, `\i`, `[_:A-Za-z]`)           // Initial name char
	result = strings.ReplaceAll(result, `\c`, `[_:A-Za-z0-9.-]`)     // Name char
	result = strings.ReplaceAll(result, `\d`, `[0-9]`)               // Digit
	result = strings.ReplaceAll(result, `\D`, `[^0-9]`)              // Non-digit
	result = strings.ReplaceAll(result, `\s`, `[ \t\n\r]`)           // Whitespace
	result = strings.ReplaceAll(result, `\S`, `[^ \t\n\r]`)          // Non-whitespace
	result = strings.ReplaceAll(result, `\w`, `[A-Za-z0-9_]`)        // Word char
	result = strings.ReplaceAll(result, `\W`, `[^A-Za-z0-9_]`)       // Non-word char
	
	return result
}

// EnumerationFacet validates against a set of allowed values
type EnumerationFacet struct {
	Values []string
}

func (f *EnumerationFacet) Name() string {
	return "enumeration"
}

func (f *EnumerationFacet) Validate(value string, baseType Type) error {
	for _, allowed := range f.Values {
		if value == allowed {
			return nil
		}
	}
	return fmt.Errorf("value '%s' is not in enumeration %v", value, f.Values)
}

// LengthFacet validates exact length
type LengthFacet struct {
	Value int
}

func (f *LengthFacet) Name() string {
	return "length"
}

func (f *LengthFacet) Validate(value string, baseType Type) error {
	length := getLength(value, baseType)
	if length != f.Value {
		return fmt.Errorf("length must be exactly %d, got %d", f.Value, length)
	}
	return nil
}

// MinLengthFacet validates minimum length
type MinLengthFacet struct {
	Value int
}

func (f *MinLengthFacet) Name() string {
	return "minLength"
}

func (f *MinLengthFacet) Validate(value string, baseType Type) error {
	length := getLength(value, baseType)
	if length < f.Value {
		return fmt.Errorf("length must be at least %d, got %d", f.Value, length)
	}
	return nil
}

// MaxLengthFacet validates maximum length
type MaxLengthFacet struct {
	Value int
}

func (f *MaxLengthFacet) Name() string {
	return "maxLength"
}

func (f *MaxLengthFacet) Validate(value string, baseType Type) error {
	length := getLength(value, baseType)
	if length > f.Value {
		return fmt.Errorf("length must be at most %d, got %d", f.Value, length)
	}
	return nil
}

// getLength returns the length of a value based on its type
func getLength(value string, baseType Type) int {
	// For list types, length is number of items
	if strings.Contains(value, " ") {
		// Check if it's a SimpleType with a List component
		if st, ok := baseType.(*SimpleType); ok && st.List != nil {
			return len(strings.Fields(value))
		}
	}
	
	// For hexBinary, length is number of octets (bytes)
	if IsBuiltinType("hexBinary") {
		return len(value) / 2
	}
	
	// For base64Binary, we need to decode to get actual byte length
	if IsBuiltinType("base64Binary") {
		// Approximate - not exact but good enough for validation
		n := len(value)
		// Remove padding
		if strings.HasSuffix(value, "==") {
			n -= 2
		} else if strings.HasSuffix(value, "=") {
			n -= 1
		}
		return n * 3 / 4
	}
	
	// For string types, length is number of characters (runes)
	return len([]rune(value))
}

// MinInclusiveFacet validates minimum value (inclusive)
type MinInclusiveFacet struct {
	Value string
}

func (f *MinInclusiveFacet) Name() string {
	return "minInclusive"
}

func (f *MinInclusiveFacet) Validate(value string, baseType Type) error {
	cmp, err := compareValues(value, f.Value, baseType)
	if err != nil {
		return err
	}
	if cmp < 0 {
		return fmt.Errorf("value must be >= %s, got %s", f.Value, value)
	}
	return nil
}

// MaxInclusiveFacet validates maximum value (inclusive)
type MaxInclusiveFacet struct {
	Value string
}

func (f *MaxInclusiveFacet) Name() string {
	return "maxInclusive"
}

func (f *MaxInclusiveFacet) Validate(value string, baseType Type) error {
	cmp, err := compareValues(value, f.Value, baseType)
	if err != nil {
		return err
	}
	if cmp > 0 {
		return fmt.Errorf("value must be <= %s, got %s", f.Value, value)
	}
	return nil
}

// MinExclusiveFacet validates minimum value (exclusive)
type MinExclusiveFacet struct {
	Value string
}

func (f *MinExclusiveFacet) Name() string {
	return "minExclusive"
}

func (f *MinExclusiveFacet) Validate(value string, baseType Type) error {
	cmp, err := compareValues(value, f.Value, baseType)
	if err != nil {
		return err
	}
	if cmp <= 0 {
		return fmt.Errorf("value must be > %s, got %s", f.Value, value)
	}
	return nil
}

// MaxExclusiveFacet validates maximum value (exclusive)
type MaxExclusiveFacet struct {
	Value string
}

func (f *MaxExclusiveFacet) Name() string {
	return "maxExclusive"
}

func (f *MaxExclusiveFacet) Validate(value string, baseType Type) error {
	cmp, err := compareValues(value, f.Value, baseType)
	if err != nil {
		return err
	}
	if cmp >= 0 {
		return fmt.Errorf("value must be < %s, got %s", f.Value, value)
	}
	return nil
}

// TotalDigitsFacet validates total number of digits
type TotalDigitsFacet struct {
	Value int
}

func (f *TotalDigitsFacet) Name() string {
	return "totalDigits"
}

func (f *TotalDigitsFacet) Validate(value string, baseType Type) error {
	// Remove sign and decimal point
	digits := strings.TrimLeft(value, "+-")
	digits = strings.Replace(digits, ".", "", 1)
	
	// Remove leading zeros
	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		digits = "0"
	}
	
	if len(digits) > f.Value {
		return fmt.Errorf("total digits must be at most %d, got %d", f.Value, len(digits))
	}
	return nil
}

// FractionDigitsFacet validates number of fraction digits
type FractionDigitsFacet struct {
	Value int
}

func (f *FractionDigitsFacet) Name() string {
	return "fractionDigits"
}

func (f *FractionDigitsFacet) Validate(value string, baseType Type) error {
	parts := strings.Split(value, ".")
	if len(parts) == 1 {
		// No fraction part
		if f.Value < 0 {
			return fmt.Errorf("fraction digits must be at least 0")
		}
		return nil
	}
	
	fractionDigits := len(parts[1])
	if fractionDigits > f.Value {
		return fmt.Errorf("fraction digits must be at most %d, got %d", f.Value, fractionDigits)
	}
	return nil
}

// WhiteSpaceFacet handles whitespace normalization
type WhiteSpaceFacet struct {
	Value string // "preserve", "replace", or "collapse"
}

func (f *WhiteSpaceFacet) Name() string {
	return "whiteSpace"
}

func (f *WhiteSpaceFacet) Validate(value string, baseType Type) error {
	// WhiteSpace facet doesn't validate, it normalizes
	// The actual normalization should be done before validation
	return nil
}

// NormalizeWhiteSpace normalizes whitespace according to the facet value
func NormalizeWhiteSpace(value string, whiteSpace string) string {
	switch whiteSpace {
	case "replace":
		// Replace each tab, newline, carriage return with space
		result := strings.ReplaceAll(value, "\t", " ")
		result = strings.ReplaceAll(result, "\n", " ")
		result = strings.ReplaceAll(result, "\r", " ")
		return result
	
	case "collapse":
		// First do replace
		result := NormalizeWhiteSpace(value, "replace")
		// Then collapse sequences of spaces to single space
		result = strings.Join(strings.Fields(result), " ")
		return result
	
	default: // "preserve"
		return value
	}
}

// compareValues compares two values based on their type
func compareValues(v1, v2 string, baseType Type) (int, error) {
	// Get the base type name for comparison
	typeName := ""
	if baseType != nil {
		typeName = baseType.Name().Local
	}
	
	// Numeric comparisons
	if isNumericType(typeName) {
		// Use big.Float for arbitrary precision
		f1 := new(big.Float)
		f2 := new(big.Float)
		
		if _, _, err := f1.Parse(v1, 10); err != nil {
			// Try parsing as integer
			i1 := new(big.Int)
			if _, ok := i1.SetString(v1, 10); !ok {
				return 0, fmt.Errorf("invalid numeric value: %s", v1)
			}
			f1.SetInt(i1)
		}
		
		if _, _, err := f2.Parse(v2, 10); err != nil {
			// Try parsing as integer
			i2 := new(big.Int)
			if _, ok := i2.SetString(v2, 10); !ok {
				return 0, fmt.Errorf("invalid numeric value: %s", v2)
			}
			f2.SetInt(i2)
		}
		
		return f1.Cmp(f2), nil
	}
	
	// Date/time comparisons
	if isDateTimeType(typeName) {
		// For simplicity, use string comparison (not fully correct for all cases)
		// A complete implementation would parse and compare as time values
		return strings.Compare(v1, v2), nil
	}
	
	// String comparison as fallback
	return strings.Compare(v1, v2), nil
}

func isNumericType(typeName string) bool {
	numericTypes := []string{
		"decimal", "integer", "float", "double",
		"nonPositiveInteger", "negativeInteger",
		"long", "int", "short", "byte",
		"nonNegativeInteger", "positiveInteger",
		"unsignedLong", "unsignedInt", "unsignedShort", "unsignedByte",
	}
	
	for _, t := range numericTypes {
		if typeName == t {
			return true
		}
	}
	return false
}

func isDateTimeType(typeName string) bool {
	dateTimeTypes := []string{
		"dateTime", "date", "time",
		"gYear", "gYearMonth", "gMonth", "gMonthDay", "gDay",
	}
	
	for _, t := range dateTimeTypes {
		if typeName == t {
			return true
		}
	}
	return false
}

// ParseFacet parses a facet element and returns the appropriate FacetValidator
func ParseFacet(name string, value string) FacetValidator {
	switch name {
	case "pattern":
		return &PatternFacet{Pattern: value}
	case "enumeration":
		return &EnumerationFacet{Values: []string{value}}
	case "length":
		if v, err := strconv.Atoi(value); err == nil {
			return &LengthFacet{Value: v}
		}
	case "minLength":
		if v, err := strconv.Atoi(value); err == nil {
			return &MinLengthFacet{Value: v}
		}
	case "maxLength":
		if v, err := strconv.Atoi(value); err == nil {
			return &MaxLengthFacet{Value: v}
		}
	case "minInclusive":
		return &MinInclusiveFacet{Value: value}
	case "maxInclusive":
		return &MaxInclusiveFacet{Value: value}
	case "minExclusive":
		return &MinExclusiveFacet{Value: value}
	case "maxExclusive":
		return &MaxExclusiveFacet{Value: value}
	case "totalDigits":
		if v, err := strconv.Atoi(value); err == nil {
			return &TotalDigitsFacet{Value: v}
		}
	case "fractionDigits":
		if v, err := strconv.Atoi(value); err == nil {
			return &FractionDigitsFacet{Value: v}
		}
	case "whiteSpace":
		return &WhiteSpaceFacet{Value: value}
	}
	return nil
}

// ValidateFacets validates a value against a list of facets
func ValidateFacets(value string, facets []FacetValidator, baseType Type) error {
	// First apply whitespace normalization if specified
	for _, f := range facets {
		if ws, ok := f.(*WhiteSpaceFacet); ok {
			value = NormalizeWhiteSpace(value, ws.Value)
			break
		}
	}
	
	// Then validate against all facets
	for _, f := range facets {
		if err := f.Validate(value, baseType); err != nil {
			return fmt.Errorf("%s constraint violated: %v", f.Name(), err)
		}
	}
	
	return nil
}

// CombineEnumerations combines multiple enumeration facets
func CombineEnumerations(facets []FacetValidator) []string {
	var values []string
	for _, f := range facets {
		if enum, ok := f.(*EnumerationFacet); ok {
			values = append(values, enum.Values...)
		}
	}
	return values
}