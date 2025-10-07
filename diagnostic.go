package xsd

import (
	"fmt"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// Diagnostic represents a rustc-style validation diagnostic
type Diagnostic struct {
	Severity  Severity  `json:"severity"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Position  Position  `json:"position"`
	Tag       string    `json:"tag"`
	Attribute string    `json:"attribute,omitempty"`
	SpecRef   string    `json:"spec_ref,omitempty"`
	Hints     []string  `json:"hints,omitempty"`
	Related   []Related `json:"related,omitempty"`
}

// Severity represents the severity level of a diagnostic
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Position contains source position information for a node
type Position struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Offset int64  `json:"offset"`
}

// Related points to a related location in the source
type Related struct {
	Label    string   `json:"label"`
	Position Position `json:"position"`
}

// DiagnosticConverter converts XSD violations to rustc-style diagnostics
type DiagnosticConverter struct {
	fileName string
	source   string
}

// NewDiagnosticConverter creates a new converter
func NewDiagnosticConverter(fileName, source string) *DiagnosticConverter {
	return &DiagnosticConverter{
		fileName: fileName,
		source:   source,
	}
}

// Convert converts XSD violations to rustc-style diagnostics
func (dc *DiagnosticConverter) Convert(violations []Violation) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(violations))

	for _, v := range violations {
		diag := dc.convertViolation(v)
		diagnostics = append(diagnostics, diag)
	}

	return diagnostics
}

// convertViolation converts a single violation to a diagnostic
func (dc *DiagnosticConverter) convertViolation(v Violation) Diagnostic {
	diag := Diagnostic{
		Severity:  dc.getSeverity(v.Code),
		Code:      dc.mapErrorCode(v.Code),
		Message:   dc.formatMessage(v),
		Position:  dc.getPosition(v.Element, v.Attribute),
		Tag:       dc.getTag(v.Element),
		Attribute: v.Attribute,
		SpecRef:   dc.getSpecRef(v.Code),
		Hints:     dc.generateHints(v),
	}

	// Add related information if available
	if len(v.Expected) > 0 {
		diag.Related = dc.generateRelated(v)
	}

	return diag
}

// getSeverity determines the severity based on error code
func (dc *DiagnosticConverter) getSeverity(code string) Severity {
	// Most XSD violations are errors
	if strings.HasPrefix(code, "cvc-") {
		return SeverityError
	}
	if strings.HasPrefix(code, "xsd-warn-") {
		return SeverityWarning
	}
	return SeverityError
}

// mapErrorCode maps XSD error codes to user-friendly codes
func (dc *DiagnosticConverter) mapErrorCode(xsdCode string) string {
	// Map common XSD constraint violation codes to SCXML validator codes
	codeMap := map[string]string{
		"cvc-complex-type.3.2.2": "E200", // Invalid attribute
		"cvc-complex-type.2.4.a": "E201", // Invalid child element
		"cvc-complex-type.2.4.b": "E202", // Missing required element
		"cvc-complex-type.2.4.d": "E203", // Unexpected element
		"cvc-complex-type.4":     "E204", // Missing required attribute
		"cvc-id.1":               "E205", // IDREF binding error
		"cvc-id.2":               "E206", // Duplicate ID
		"cvc-elt.1":              "E207", // Element not declared
		"cvc-type.3.1.3":         "E208", // Invalid value for type
		"cvc-enumeration-valid":  "E209", // Value not in enumeration
		"cvc-pattern-valid":      "E210", // Pattern mismatch
		"xsd-null-document":      "E001", // Null document
		"xsd-no-root":            "E002", // No root element
	}

	if mapped, ok := codeMap[xsdCode]; ok {
		return mapped
	}

	// Default: use XSD code with E prefix
	return "E" + strings.ReplaceAll(xsdCode, ".", "_")
}

// formatMessage creates a user-friendly message
func (dc *DiagnosticConverter) formatMessage(v Violation) string {
	msg := v.Message

	// Enhance specific messages
	switch v.Code {
	case "cvc-complex-type.3.2.2":
		if v.Attribute == "sendid" {
			msg = fmt.Sprintf("Invalid attribute 'sendid' on element '%s'. Did you mean 'id'?",
				dc.getTag(v.Element))
		}
	case "cvc-complex-type.2.4.a":
		if len(v.Expected) > 0 {
			msg = fmt.Sprintf("Invalid element '%s'. Expected one of: %s",
				v.Actual, strings.Join(v.Expected, ", "))
		}
	case "cvc-id.1":
		// Extract IDREF name from message if possible
		if strings.Contains(v.Message, "IDREF") {
			msg = fmt.Sprintf("Referenced ID '%s' does not exist in document", v.Actual)
		}
	}

	return msg
}

// getPosition gets the position of an element or attribute
func (dc *DiagnosticConverter) getPosition(elem xmldom.Element, attrName string) Position {
	if elem == nil {
		return Position{File: dc.fileName}
	}

	// Try to get attribute position if specified
	if attrName != "" {
		if attr := elem.GetAttributeNode(xmldom.DOMString(attrName)); attr != nil {
			line, col, offset := attr.Position()
			if line > 0 {
				return Position{
					File:   dc.fileName,
					Line:   line,
					Column: col,
					Offset: offset,
				}
			}
		}
	}

	// Fall back to element position
	line, col, offset := elem.Position()
	return Position{
		File:   dc.fileName,
		Line:   line,
		Column: col,
		Offset: offset,
	}
}

// getTag gets the tag name of an element
func (dc *DiagnosticConverter) getTag(elem xmldom.Element) string {
	if elem == nil {
		return ""
	}
	return string(elem.LocalName())
}

// getSpecRef returns specification reference for error codes
func (dc *DiagnosticConverter) getSpecRef(code string) string {
	// Map error codes to relevant SCXML spec sections
	specMap := map[string]string{
		"cvc-complex-type.3.2.2": "W3C SCXML 1.0 §3.14", // Send element
		"cvc-complex-type.2.4.a": "W3C SCXML 1.0 §3",    // Structure
		"cvc-id.1":               "W3C SCXML 1.0 §3.14", // ID/IDREF
		"cvc-id.2":               "W3C SCXML 1.0 §3.14", // Unique IDs
	}

	if ref, ok := specMap[code]; ok {
		return ref
	}

	return "W3C XML Schema 1.1"
}

// generateHints creates helpful hints based on the violation
func (dc *DiagnosticConverter) generateHints(v Violation) []string {
	hints := []string{}

	switch v.Code {
	case "cvc-complex-type.3.2.2":
		// Invalid attribute hints
		if v.Attribute == "sendid" {
			hints = append(hints,
				"The <send> element uses 'id' attribute, not 'sendid'",
				"Use 'id' to specify the identifier for the send request",
				"The 'id' value can be referenced by <cancel> using 'sendid' attribute")
		} else if v.Attribute == "priority" {
			hints = append(hints,
				"The 'priority' attribute is not part of standard SCXML",
				"Transition selection is based on document order",
				"Consider restructuring your transitions if priority is needed")
		} else if len(v.Expected) > 0 {
			hints = append(hints, fmt.Sprintf("Did you mean: %s?", strings.Join(v.Expected, " or ")))
		}

	case "cvc-complex-type.2.4.a":
		// Invalid child element hints
		if v.Actual == "transition" && v.Element != nil {
			parent := dc.getTag(v.Element)
			if parent == "scxml" {
				hints = append(hints,
					"Transitions cannot be placed directly under <scxml>",
					"Transitions must be children of <state>, <parallel>, or <final> elements",
					"Consider wrapping this transition in a state")
			}
		}
		if len(v.Expected) > 0 {
			hints = append(hints,
				fmt.Sprintf("Valid children are: %s", strings.Join(v.Expected, ", ")))
		}

	case "cvc-id.1":
		// IDREF hints
		hints = append(hints,
			fmt.Sprintf("Ensure there is an element with id='%s' in the document", v.Actual),
			"Check for typos in the ID reference",
			"IDs are case-sensitive")

	case "cvc-id.2":
		// Duplicate ID hints
		hints = append(hints,
			"Each id attribute value must be unique within the document",
			"Consider using a more descriptive identifier")

	case "cvc-complex-type.4":
		// Missing required attribute
		if len(v.Expected) == 1 {
			hints = append(hints, fmt.Sprintf("Add required attribute: %s=\"...\"", v.Expected[0]))
		}

	case "cvc-enumeration-valid":
		// Enumeration hints
		if len(v.Expected) > 0 {
			hints = append(hints,
				fmt.Sprintf("Valid values are: %s", strings.Join(v.Expected, ", ")))
		}
	}

	// Add general hint if no specific ones
	if len(hints) == 0 && len(v.Expected) > 0 {
		hints = append(hints, fmt.Sprintf("Expected: %s", strings.Join(v.Expected, ", ")))
	}

	return hints
}

// generateRelated creates related position information
func (dc *DiagnosticConverter) generateRelated(v Violation) []Related {
	related := []Related{}

	// For duplicate IDs, we might want to show where it was first defined
	// This would require tracking in the validator

	// For IDREF errors, we could show similar IDs that exist
	// This would also require additional tracking

	return related
}

// ErrorFormatter provides rustc-style error formatting
type ErrorFormatter struct {
	Color           bool
	ShowFullElement bool
	ContextLines    int
}

// Format formats a diagnostic in rustc style
func (ef *ErrorFormatter) Format(diag Diagnostic, source string) string {
	var sb strings.Builder

	// Error header
	severity := string(diag.Severity)
	if ef.Color {
		switch diag.Severity {
		case SeverityError:
			severity = "\033[31;1merror\033[0m" // Red
		case SeverityWarning:
			severity = "\033[33;1mwarning\033[0m" // Yellow
		case SeverityInfo:
			severity = "\033[36;1minfo\033[0m" // Cyan
		}
	}

	sb.WriteString(fmt.Sprintf("%s[%s]: %s\n", severity, diag.Code, diag.Message))

	// Location
	location := fmt.Sprintf(" --> %s:%d:%d",
		diag.Position.File, diag.Position.Line, diag.Position.Column)
	sb.WriteString(location + "\n")

	// Source context
	if source != "" && diag.Position.Line > 0 {
		lines := strings.Split(source, "\n")
		if diag.Position.Line <= len(lines) {
			// Line number gutter
			lineNum := fmt.Sprintf("%4d | ", diag.Position.Line)
			sb.WriteString(lineNum)

			// Source line
			sourceLine := lines[diag.Position.Line-1]
			sb.WriteString(sourceLine + "\n")

			// Error pointer
			sb.WriteString("     | ")
			if diag.Position.Column > 0 {
				sb.WriteString(strings.Repeat(" ", diag.Position.Column-1))
				if ef.Color {
					sb.WriteString("\033[31;1m^\033[0m") // Red caret
				} else {
					sb.WriteString("^")
				}

				// Underline the problematic part
				if diag.Attribute != "" {
					underlineLen := len(diag.Attribute)
					sb.WriteString(strings.Repeat("~", underlineLen))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Hints
	if len(diag.Hints) > 0 {
		sb.WriteString("     |\n")
		for _, hint := range diag.Hints {
			sb.WriteString("     = help: " + hint + "\n")
		}
	}

	// Spec reference
	if diag.SpecRef != "" {
		sb.WriteString("     = note: see " + diag.SpecRef + "\n")
	}

	// Related information
	for _, rel := range diag.Related {
		sb.WriteString(fmt.Sprintf("\n     %s\n", rel.Label))
		sb.WriteString(fmt.Sprintf("      --> %s:%d:%d\n",
			rel.Position.File, rel.Position.Line, rel.Position.Column))
	}

	return sb.String()
}
