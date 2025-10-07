package xsd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FailureCategory represents a category of test failures
type FailureCategory struct {
	Name        string
	Description string
	Count       int
	Examples    []W3CTestResult
}

// AnalyzeTestFailures categorizes test failures by XSD feature
func AnalyzeTestFailures(results []W3CTestResult) map[string]*FailureCategory {
	categories := make(map[string]*FailureCategory)
	
	// Initialize categories
	initCategories(categories)
	
	for _, result := range results {
		if !result.Passed {
			categorizeFailure(result, categories)
		}
	}
	
	return categories
}

func initCategories(cats map[string]*FailureCategory) {
	cats["duplicate-id"] = &FailureCategory{
		Name:        "Duplicate ID Validation",
		Description: "Failures due to missing duplicate ID detection",
	}
	cats["facet-validation"] = &FailureCategory{
		Name:        "Facet Validation",
		Description: "Failures due to missing facet constraints (pattern, length, enumeration, etc.)",
	}
	cats["type-validation"] = &FailureCategory{
		Name:        "Built-in Type Validation",
		Description: "Failures validating built-in XSD types (int, date, boolean, etc.)",
	}
	cats["namespace"] = &FailureCategory{
		Name:        "Namespace Handling",
		Description: "Failures related to namespace resolution and validation",
	}
	cats["identity-constraint"] = &FailureCategory{
		Name:        "Identity Constraints",
		Description: "Failures with key, keyref, unique constraints",
	}
	cats["complex-derivation"] = &FailureCategory{
		Name:        "Complex Type Derivation",
		Description: "Failures with restriction/extension of complex types",
	}
	cats["attribute-use"] = &FailureCategory{
		Name:        "Attribute Use/Validation",
		Description: "Failures with required/prohibited/optional attributes",
	}
	cats["content-model"] = &FailureCategory{
		Name:        "Content Model Validation",
		Description: "Failures validating element content against model groups",
	}
	cats["schema-composition"] = &FailureCategory{
		Name:        "Schema Composition",
		Description: "Failures with import, include, redefine",
	}
	cats["wildcards"] = &FailureCategory{
		Name:        "Wildcards",
		Description: "Failures with xs:any and xs:anyAttribute",
	}
	cats["substitution"] = &FailureCategory{
		Name:        "Substitution Groups",
		Description: "Failures with substitution group validation",
	}
	cats["notation"] = &FailureCategory{
		Name:        "Notations",
		Description: "Failures with NOTATION type",
	}
	cats["fixed-default"] = &FailureCategory{
		Name:        "Fixed/Default Values",
		Description: "Failures with fixed and default attribute/element values",
	}
	cats["simple-type"] = &FailureCategory{
		Name:        "Simple Type Definition",
		Description: "Failures defining or validating simple types",
	}
	cats["mixed-content"] = &FailureCategory{
		Name:        "Mixed Content",
		Description: "Failures with mixed content models",
	}
	cats["other"] = &FailureCategory{
		Name:        "Other/Unknown",
		Description: "Uncategorized failures",
	}
}

func categorizeFailure(result W3CTestResult, categories map[string]*FailureCategory) {
	// Categorize based on test group name and test name patterns
	testPath := strings.ToLower(result.TestGroup + "/" + result.TestName)
	
	category := "other"
	
	// Check for specific patterns in test names and paths
	switch {
	// Identity and ID/IDREF
	case strings.Contains(testPath, "identity") || strings.Contains(testPath, "keyref") ||
		strings.Contains(testPath, "unique") || strings.Contains(testPath, "key"):
		category = "identity-constraint"
	case strings.Contains(testPath, "id") && strings.Contains(testPath, "dup"):
		category = "duplicate-id"
	
	// Facets
	case strings.Contains(testPath, "pattern") || strings.Contains(testPath, "regex") ||
		strings.Contains(testPath, "length") || strings.Contains(testPath, "enum") ||
		strings.Contains(testPath, "facet") || strings.Contains(testPath, "whitespace") ||
		strings.Contains(testPath, "fraction") || strings.Contains(testPath, "digit"):
		category = "facet-validation"
	
	// Built-in types
	case strings.Contains(testPath, "datatype") || strings.Contains(testPath, "decimal") ||
		strings.Contains(testPath, "integer") || strings.Contains(testPath, "boolean") ||
		strings.Contains(testPath, "date") || strings.Contains(testPath, "time") ||
		strings.Contains(testPath, "base64") || strings.Contains(testPath, "hex"):
		category = "type-validation"
	
	// Namespaces
	case strings.Contains(testPath, "namespace") || strings.Contains(testPath, "targetns") ||
		strings.Contains(testPath, "qualified") || strings.Contains(testPath, "unqualified") ||
		strings.Contains(testPath, "import"):
		category = "namespace"
	
	// Complex type derivation
	case strings.Contains(testPath, "extension") || strings.Contains(testPath, "restriction") ||
		strings.Contains(testPath, "derive") || strings.Contains(testPath, "complexcontent") ||
		strings.Contains(testPath, "simplecontent"):
		category = "complex-derivation"
	
	// Attributes
	case strings.Contains(testPath, "attribute") && (strings.Contains(testPath, "required") ||
		strings.Contains(testPath, "prohibited") || strings.Contains(testPath, "optional") ||
		strings.Contains(testPath, "use")):
		category = "attribute-use"
	
	// Content models
	case strings.Contains(testPath, "sequence") || strings.Contains(testPath, "choice") ||
		strings.Contains(testPath, "all") || strings.Contains(testPath, "group") ||
		strings.Contains(testPath, "particle") || strings.Contains(testPath, "occurrence"):
		category = "content-model"
	
	// Schema composition
	case strings.Contains(testPath, "include") || strings.Contains(testPath, "redefine") ||
		strings.Contains(testPath, "override"):
		category = "schema-composition"
	
	// Wildcards
	case strings.Contains(testPath, "wildcard") || strings.Contains(testPath, "any") ||
		strings.Contains(testPath, "anyattribute") || strings.Contains(testPath, "processcontents"):
		category = "wildcards"
	
	// Substitution groups
	case strings.Contains(testPath, "substitution") || strings.Contains(testPath, "abstract"):
		category = "substitution"
	
	// Notations
	case strings.Contains(testPath, "notation"):
		category = "notation"
	
	// Fixed and default values
	case strings.Contains(testPath, "fixed") || strings.Contains(testPath, "default"):
		category = "fixed-default"
	
	// Simple types
	case strings.Contains(testPath, "simpletype") || strings.Contains(testPath, "list") ||
		strings.Contains(testPath, "union"):
		category = "simple-type"
	
	// Mixed content
	case strings.Contains(testPath, "mixed"):
		category = "mixed-content"
		
	// Additional checks based on error messages
	default:
		if result.Error != nil {
			errMsg := strings.ToLower(result.Error.Error())
			if strings.Contains(errMsg, "duplicate") && strings.Contains(errMsg, "id") {
				category = "duplicate-id"
			} else if strings.Contains(errMsg, "pattern") || strings.Contains(errMsg, "facet") {
				category = "facet-validation"
			} else if strings.Contains(errMsg, "namespace") {
				category = "namespace"
			} else if strings.Contains(errMsg, "type") {
				category = "type-validation"
			}
		}
	}
	
	// Add to category
	cat := categories[category]
	cat.Count++
	if len(cat.Examples) < 5 { // Keep up to 5 examples per category
		cat.Examples = append(cat.Examples, result)
	}
}

// GenerateFailureReport creates a detailed failure analysis report
func GenerateFailureReport(categories map[string]*FailureCategory) string {
	var report strings.Builder
	
	report.WriteString("XSD Test Failure Analysis Report\n")
	report.WriteString("=================================\n\n")
	
	// Sort categories by failure count
	type catEntry struct {
		key string
		cat *FailureCategory
	}
	var sorted []catEntry
	for k, v := range categories {
		if v.Count > 0 {
			sorted = append(sorted, catEntry{k, v})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].cat.Count > sorted[j].cat.Count
	})
	
	// Calculate total failures
	totalFailures := 0
	for _, entry := range sorted {
		totalFailures += entry.cat.Count
	}
	
	report.WriteString(fmt.Sprintf("Total Failures: %d\n\n", totalFailures))
	report.WriteString("Failures by Category:\n")
	report.WriteString("--------------------\n\n")
	
	// Print each category
	for _, entry := range sorted {
		cat := entry.cat
		percentage := float64(cat.Count) * 100 / float64(totalFailures)
		report.WriteString(fmt.Sprintf("%s: %d failures (%.1f%%)\n", cat.Name, cat.Count, percentage))
		report.WriteString(fmt.Sprintf("  %s\n", cat.Description))
		
		if len(cat.Examples) > 0 {
			report.WriteString("  Examples:\n")
			for i, ex := range cat.Examples {
				if i >= 3 { // Show max 3 examples
					break
				}
				report.WriteString(fmt.Sprintf("    - %s/%s (expected: %s, got: %s)\n",
					ex.TestGroup, ex.TestName, ex.Expected, ex.Actual))
				if ex.SchemaPath != "" {
					report.WriteString(fmt.Sprintf("      Schema: %s\n", filepath.Base(ex.SchemaPath)))
				}
			}
		}
		report.WriteString("\n")
	}
	
	// Add implementation priority recommendations
	report.WriteString("Implementation Priority:\n")
	report.WriteString("-----------------------\n")
	report.WriteString("Based on failure counts, implement features in this order:\n\n")
	
	for i, entry := range sorted {
		if i >= 5 { // Show top 5 priorities
			break
		}
		report.WriteString(fmt.Sprintf("%d. %s (%d failures)\n", i+1, entry.cat.Name, entry.cat.Count))
	}
	
	return report.String()
}