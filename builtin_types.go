package xsd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// BuiltinType represents a built-in XSD type
type BuiltinType struct {
	Name      string
	Validator func(value string) error
}

var builtinTypes = map[string]*BuiltinType{}

func init() {
	// Register all built-in types
	registerBuiltinTypes()
}

func registerBuiltinTypes() {
	// Primitive types
	builtinTypes["string"] = &BuiltinType{"string", validateString}
	builtinTypes["boolean"] = &BuiltinType{"boolean", validateBoolean}
	builtinTypes["decimal"] = &BuiltinType{"decimal", validateDecimal}
	builtinTypes["float"] = &BuiltinType{"float", validateFloat}
	builtinTypes["double"] = &BuiltinType{"double", validateDouble}
	builtinTypes["duration"] = &BuiltinType{"duration", validateDuration}
	builtinTypes["dateTime"] = &BuiltinType{"dateTime", validateDateTime}
	builtinTypes["time"] = &BuiltinType{"time", validateTime}
	builtinTypes["date"] = &BuiltinType{"date", validateDate}
	builtinTypes["gYearMonth"] = &BuiltinType{"gYearMonth", validateGYearMonth}
	builtinTypes["gYear"] = &BuiltinType{"gYear", validateGYear}
	builtinTypes["gMonthDay"] = &BuiltinType{"gMonthDay", validateGMonthDay}
	builtinTypes["gDay"] = &BuiltinType{"gDay", validateGDay}
	builtinTypes["gMonth"] = &BuiltinType{"gMonth", validateGMonth}
	builtinTypes["hexBinary"] = &BuiltinType{"hexBinary", validateHexBinary}
	builtinTypes["base64Binary"] = &BuiltinType{"base64Binary", validateBase64Binary}
	builtinTypes["anyURI"] = &BuiltinType{"anyURI", validateAnyURI}
	builtinTypes["QName"] = &BuiltinType{"QName", validateQName}
	builtinTypes["NOTATION"] = &BuiltinType{"NOTATION", validateNOTATION}

	// Derived types - strings
	builtinTypes["normalizedString"] = &BuiltinType{"normalizedString", validateNormalizedString}
	builtinTypes["token"] = &BuiltinType{"token", validateToken}
	builtinTypes["language"] = &BuiltinType{"language", validateLanguage}
	builtinTypes["Name"] = &BuiltinType{"Name", validateName}
	builtinTypes["NCName"] = &BuiltinType{"NCName", validateNCName}
	builtinTypes["ID"] = &BuiltinType{"ID", validateID}
	builtinTypes["IDREF"] = &BuiltinType{"IDREF", validateIDREF}
	builtinTypes["IDREFS"] = &BuiltinType{"IDREFS", validateIDREFS}
	builtinTypes["ENTITY"] = &BuiltinType{"ENTITY", validateENTITY}
	builtinTypes["ENTITIES"] = &BuiltinType{"ENTITIES", validateENTITIES}
	builtinTypes["NMTOKEN"] = &BuiltinType{"NMTOKEN", validateNMTOKEN}
	builtinTypes["NMTOKENS"] = &BuiltinType{"NMTOKENS", validateNMTOKENS}

	// Derived types - numeric
	builtinTypes["integer"] = &BuiltinType{"integer", validateInteger}
	builtinTypes["nonPositiveInteger"] = &BuiltinType{"nonPositiveInteger", validateNonPositiveInteger}
	builtinTypes["negativeInteger"] = &BuiltinType{"negativeInteger", validateNegativeInteger}
	builtinTypes["long"] = &BuiltinType{"long", validateLong}
	builtinTypes["int"] = &BuiltinType{"int", validateInt}
	builtinTypes["short"] = &BuiltinType{"short", validateShort}
	builtinTypes["byte"] = &BuiltinType{"byte", validateByte}
	builtinTypes["nonNegativeInteger"] = &BuiltinType{"nonNegativeInteger", validateNonNegativeInteger}
	builtinTypes["unsignedLong"] = &BuiltinType{"unsignedLong", validateUnsignedLong}
	builtinTypes["unsignedInt"] = &BuiltinType{"unsignedInt", validateUnsignedInt}
	builtinTypes["unsignedShort"] = &BuiltinType{"unsignedShort", validateUnsignedShort}
	builtinTypes["unsignedByte"] = &BuiltinType{"unsignedByte", validateUnsignedByte}
	builtinTypes["positiveInteger"] = &BuiltinType{"positiveInteger", validatePositiveInteger}
}

// GetBuiltinType returns a built-in type validator
func GetBuiltinType(name string) *BuiltinType {
	// Strip namespace prefix if present
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[idx+1:]
	}
	return builtinTypes[name]
}

// IsBuiltinType checks if a type is a built-in XSD type
func IsBuiltinType(name string) bool {
	return GetBuiltinType(name) != nil
}

// Primitive type validators

func validateString(value string) error {
	// All strings are valid
	return nil
}

func validateBoolean(value string) error {
	switch value {
	case "true", "false", "1", "0":
		return nil
	default:
		return fmt.Errorf("invalid boolean value: %s", value)
	}
}

func validateDecimal(value string) error {
	// Decimal can have optional sign, digits, optional decimal point and more digits
	// Examples: -1.23, 12.00, +100, 210.
	pattern := regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid decimal value: %s", value)
	}

	// Try parsing with big.Float for arbitrary precision
	_, _, err := new(big.Float).Parse(value, 10)
	if err != nil {
		return fmt.Errorf("invalid decimal value: %s", value)
	}
	return nil
}

func validateFloat(value string) error {
	// Special values
	switch value {
	case "INF", "+INF", "-INF", "NaN":
		return nil
	}

	// Try parsing as float32
	_, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return fmt.Errorf("invalid float value: %s", value)
	}
	return nil
}

func validateDouble(value string) error {
	// Special values
	switch value {
	case "INF", "+INF", "-INF", "NaN":
		return nil
	}

	// Try parsing as float64
	_, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid double value: %s", value)
	}
	return nil
}

func validateDuration(value string) error {
	// Duration pattern: P[nY][nM][nD][T[nH][nM][n[.n]S]]
	// Examples: P1Y2M3DT10H30M, PT1H, P1Y, -P1D
	pattern := regexp.MustCompile(`^-?P(\d+Y)?(\d+M)?(\d+D)?(T(\d+H)?(\d+M)?(\d+(\.\d+)?S)?)?$`)
	if !pattern.MatchString(value) && value != "P0Y" && value != "PT0S" && value != "P" {
		return fmt.Errorf("invalid duration value: %s", value)
	}

	// Must have at least one component
	if value == "P" || value == "-P" || value == "PT" || value == "-PT" {
		return fmt.Errorf("duration must have at least one time component: %s", value)
	}

	return nil
}

func validateDateTime(value string) error {
	// DateTime pattern: CCYY-MM-DDThh:mm:ss[.sss][Z|(+|-)hh:mm]
	// Parse according to XML Schema dateTime format
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05.999Z",
		"2006-01-02T15:04:05.999-07:00",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}

	return fmt.Errorf("invalid dateTime value: %s", value)
}

func validateTime(value string) error {
	// Time pattern: hh:mm:ss[.sss][Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid time value: %s", value)
	}

	// Validate hour, minute, second ranges
	parts := strings.Split(value, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	secondPart := parts[2]
	if idx := strings.IndexAny(secondPart, ".Z+-"); idx >= 0 {
		secondPart = secondPart[:idx]
	}
	second, _ := strconv.Atoi(secondPart)

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
		return fmt.Errorf("invalid time value: %s", value)
	}

	return nil
}

func validateDate(value string) error {
	// Date pattern: CCYY-MM-DD[Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^-?\d{4,}-\d{2}-\d{2}(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid date value: %s", value)
	}

	// Extract date parts
	datePart := value
	// Look for timezone suffix (Z or +/-HH:MM)
	if strings.HasSuffix(value, "Z") {
		datePart = value[:len(value)-1]
	} else {
		// Look for timezone offset at the end (+HH:MM or -HH:MM)
		// Need to find the last occurrence of + or - that's part of timezone
		if len(value) >= 6 {
			// Check if the last 6 chars match timezone pattern
			if (value[len(value)-6] == '+' || value[len(value)-6] == '-') &&
				value[len(value)-3] == ':' {
				datePart = value[:len(value)-6]
			}
		}
	}

	// Handle negative years
	isNegativeYear := strings.HasPrefix(datePart, "-")
	if isNegativeYear {
		// XML Schema allows years before 0001
		// For now, just check the pattern is correct
		return nil
	}

	// Parse the date
	_, err := time.Parse("2006-01-02", datePart)
	if err != nil {
		return fmt.Errorf("invalid date value: %s", value)
	}

	return nil
}

func validateGYearMonth(value string) error {
	// gYearMonth pattern: CCYY-MM[Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^-?\d{4,}-\d{2}(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid gYearMonth value: %s", value)
	}

	// Validate month
	parts := strings.Split(value, "-")
	monthStr := parts[len(parts)-1]
	if idx := strings.IndexAny(monthStr, "Z+-"); idx >= 0 {
		monthStr = monthStr[:idx]
	}
	month, _ := strconv.Atoi(monthStr)
	if month < 1 || month > 12 {
		return fmt.Errorf("invalid month in gYearMonth: %s", value)
	}

	return nil
}

func validateGYear(value string) error {
	// gYear pattern: CCYY[Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^-?\d{4,}(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid gYear value: %s", value)
	}
	return nil
}

func validateGMonthDay(value string) error {
	// gMonthDay pattern: --MM-DD[Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^--\d{2}-\d{2}(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid gMonthDay value: %s", value)
	}

	// Validate month and day
	parts := strings.Split(value[2:], "-") // Skip initial --
	month, _ := strconv.Atoi(parts[0])
	dayStr := parts[1]
	if idx := strings.IndexAny(dayStr, "Z+-"); idx >= 0 {
		dayStr = dayStr[:idx]
	}
	day, _ := strconv.Atoi(dayStr)

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return fmt.Errorf("invalid gMonthDay value: %s", value)
	}

	return nil
}

func validateGDay(value string) error {
	// gDay pattern: ---DD[Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^---\d{2}(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid gDay value: %s", value)
	}

	// Validate day
	dayStr := value[3:]
	if idx := strings.IndexAny(dayStr, "Z+-"); idx >= 0 {
		dayStr = dayStr[:idx]
	}
	day, _ := strconv.Atoi(dayStr)
	if day < 1 || day > 31 {
		return fmt.Errorf("invalid gDay value: %s", value)
	}

	return nil
}

func validateGMonth(value string) error {
	// gMonth pattern: --MM[Z|(+|-)hh:mm]
	pattern := regexp.MustCompile(`^--\d{2}(Z|[+-]\d{2}:\d{2})?$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid gMonth value: %s", value)
	}

	// Validate month
	monthStr := value[2:]
	if idx := strings.IndexAny(monthStr, "Z+-"); idx >= 0 {
		monthStr = monthStr[:idx]
	}
	month, _ := strconv.Atoi(monthStr)
	if month < 1 || month > 12 {
		return fmt.Errorf("invalid gMonth value: %s", value)
	}

	return nil
}

func validateHexBinary(value string) error {
	// Must be even number of hex digits
	if len(value)%2 != 0 {
		return fmt.Errorf("hexBinary must have even number of characters: %s", value)
	}

	// Try decoding
	_, err := hex.DecodeString(value)
	if err != nil {
		return fmt.Errorf("invalid hexBinary value: %s", value)
	}
	return nil
}

func validateBase64Binary(value string) error {
	// Try decoding
	_, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("invalid base64Binary value: %s", value)
	}
	return nil
}

func validateAnyURI(value string) error {
	// Any string is a valid anyURI in XSD 1.1
	// Could add more strict validation if needed
	return nil
}

func validateQName(value string) error {
	// QName is NCName or NCName:NCName
	parts := strings.Split(value, ":")
	if len(parts) > 2 {
		return fmt.Errorf("invalid QName: too many colons: %s", value)
	}

	for _, part := range parts {
		if err := validateNCName(part); err != nil {
			return fmt.Errorf("invalid QName: %s", value)
		}
	}

	return nil
}

func validateNOTATION(value string) error {
	// NOTATION is like QName
	return validateQName(value)
}

// String derived type validators

func validateNormalizedString(value string) error {
	// No carriage return, line feed, or tab
	for _, r := range value {
		if r == '\r' || r == '\n' || r == '\t' {
			return fmt.Errorf("normalizedString cannot contain CR, LF, or TAB")
		}
	}
	return nil
}

func validateToken(value string) error {
	// No CR, LF, TAB, leading/trailing spaces, or multiple spaces
	if err := validateNormalizedString(value); err != nil {
		return err
	}

	if strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		return fmt.Errorf("token cannot have leading or trailing spaces")
	}

	if strings.Contains(value, "  ") {
		return fmt.Errorf("token cannot have multiple consecutive spaces")
	}

	return nil
}

func validateLanguage(value string) error {
	// Language tag pattern (simplified)
	// Format: primary-subtag ("-" subtag)*
	pattern := regexp.MustCompile(`^[a-zA-Z]{1,8}(-[a-zA-Z0-9]{1,8})*$`)
	if !pattern.MatchString(value) {
		return fmt.Errorf("invalid language tag: %s", value)
	}
	return nil
}

func validateName(value string) error {
	if value == "" {
		return fmt.Errorf("Name cannot be empty")
	}

	// Name starts with letter, underscore, or colon
	// Continues with letters, digits, '.', '-', '_', ':'
	first := rune(value[0])
	if !unicode.IsLetter(first) && first != '_' && first != ':' {
		return fmt.Errorf("Name must start with letter, underscore, or colon: %s", value)
	}

	for _, r := range value[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
			r != '.' && r != '-' && r != '_' && r != ':' {
			return fmt.Errorf("invalid character in Name: %s", string(r))
		}
	}

	return nil
}

func validateNCName(value string) error {
	// NCName is Name without colons
	if err := validateName(value); err != nil {
		return err
	}

	if strings.Contains(value, ":") {
		return fmt.Errorf("NCName cannot contain colons: %s", value)
	}

	return nil
}

func validateID(value string) error {
	// ID is an NCName
	return validateNCName(value)
}

func validateIDREF(value string) error {
	// IDREF is an NCName
	return validateNCName(value)
}

func validateIDREFS(value string) error {
	// IDREFS is a space-separated list of IDREF
	if value == "" {
		return fmt.Errorf("IDREFS cannot be empty")
	}

	ids := strings.Fields(value)
	for _, id := range ids {
		if err := validateIDREF(id); err != nil {
			return err
		}
	}
	return nil
}

func validateENTITY(value string) error {
	// ENTITY is an NCName
	return validateNCName(value)
}

func validateENTITIES(value string) error {
	// ENTITIES is a space-separated list of ENTITY
	if value == "" {
		return fmt.Errorf("ENTITIES cannot be empty")
	}

	entities := strings.Fields(value)
	for _, entity := range entities {
		if err := validateENTITY(entity); err != nil {
			return err
		}
	}
	return nil
}

func validateNMTOKEN(value string) error {
	// NMTOKEN is Name characters (but can start with any name char)
	if value == "" {
		return fmt.Errorf("NMTOKEN cannot be empty")
	}

	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
			r != '.' && r != '-' && r != '_' && r != ':' {
			return fmt.Errorf("invalid character in NMTOKEN: %s", string(r))
		}
	}

	return nil
}

func validateNMTOKENS(value string) error {
	// NMTOKENS is a space-separated list of NMTOKEN
	if value == "" {
		return fmt.Errorf("NMTOKENS cannot be empty")
	}

	tokens := strings.Fields(value)
	for _, token := range tokens {
		if err := validateNMTOKEN(token); err != nil {
			return err
		}
	}
	return nil
}

// Numeric derived type validators

func validateInteger(value string) error {
	// Integer is decimal with no fractional part
	if _, ok := new(big.Int).SetString(value, 10); !ok {
		return fmt.Errorf("invalid integer value: %s", value)
	}
	return nil
}

func validateNonPositiveInteger(value string) error {
	i := new(big.Int)
	if _, ok := i.SetString(value, 10); !ok {
		return fmt.Errorf("invalid nonPositiveInteger value: %s", value)
	}
	if i.Cmp(big.NewInt(0)) > 0 {
		return fmt.Errorf("nonPositiveInteger must be <= 0: %s", value)
	}
	return nil
}

func validateNegativeInteger(value string) error {
	i := new(big.Int)
	if _, ok := i.SetString(value, 10); !ok {
		return fmt.Errorf("invalid negativeInteger value: %s", value)
	}
	if i.Cmp(big.NewInt(0)) >= 0 {
		return fmt.Errorf("negativeInteger must be < 0: %s", value)
	}
	return nil
}

func validateLong(value string) error {
	_, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid long value: %s", value)
	}
	return nil
}

func validateInt(value string) error {
	v, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid int value: %s", value)
	}
	if v < math.MinInt32 || v > math.MaxInt32 {
		return fmt.Errorf("int value out of range: %s", value)
	}
	return nil
}

func validateShort(value string) error {
	v, err := strconv.ParseInt(value, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid short value: %s", value)
	}
	if v < math.MinInt16 || v > math.MaxInt16 {
		return fmt.Errorf("short value out of range: %s", value)
	}
	return nil
}

func validateByte(value string) error {
	v, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid byte value: %s", value)
	}
	if v < math.MinInt8 || v > math.MaxInt8 {
		return fmt.Errorf("byte value out of range: %s", value)
	}
	return nil
}

func validateNonNegativeInteger(value string) error {
	i := new(big.Int)
	if _, ok := i.SetString(value, 10); !ok {
		return fmt.Errorf("invalid nonNegativeInteger value: %s", value)
	}
	if i.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("nonNegativeInteger must be >= 0: %s", value)
	}
	return nil
}

func validateUnsignedLong(value string) error {
	_, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid unsignedLong value: %s", value)
	}
	return nil
}

func validateUnsignedInt(value string) error {
	v, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid unsignedInt value: %s", value)
	}
	if v > math.MaxUint32 {
		return fmt.Errorf("unsignedInt value out of range: %s", value)
	}
	return nil
}

func validateUnsignedShort(value string) error {
	v, err := strconv.ParseUint(value, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid unsignedShort value: %s", value)
	}
	if v > math.MaxUint16 {
		return fmt.Errorf("unsignedShort value out of range: %s", value)
	}
	return nil
}

func validateUnsignedByte(value string) error {
	v, err := strconv.ParseUint(value, 10, 8)
	if err != nil {
		return fmt.Errorf("invalid unsignedByte value: %s", value)
	}
	if v > math.MaxUint8 {
		return fmt.Errorf("unsignedByte value out of range: %s", value)
	}
	return nil
}

func validatePositiveInteger(value string) error {
	i := new(big.Int)
	if _, ok := i.SetString(value, 10); !ok {
		return fmt.Errorf("invalid positiveInteger value: %s", value)
	}
	if i.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("positiveInteger must be > 0: %s", value)
	}
	return nil
}
