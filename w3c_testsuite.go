package xsd

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentflare-ai/go-xmldom"
)

// W3CTestSet represents a test set from the W3C XSD test suite
type W3CTestSet struct {
	XMLName     xml.Name       `xml:"testSet"`
	Contributor string         `xml:"contributor,attr"`
	Name        string         `xml:"name,attr"`
	TestGroups  []W3CTestGroup `xml:"testGroup"`
}

// W3CTestGroup represents a test group containing related tests
type W3CTestGroup struct {
	Name          string            `xml:"name,attr"`
	Annotation    *W3CAnnotation    `xml:"annotation"`
	DocReference  *W3CDocReference  `xml:"documentationReference"`
	SchemaTests   []W3CSchemaTest   `xml:"schemaTest"`
	InstanceTests []W3CInstanceTest `xml:"instanceTest"`
}

// W3CAnnotation contains test documentation
type W3CAnnotation struct {
	Documentation string `xml:"documentation"`
}

// W3CDocReference links to specification documentation
type W3CDocReference struct {
	Href string `xml:"href,attr"`
}

// W3CSchemaTest tests whether a schema is valid or invalid
type W3CSchemaTest struct {
	Name           string           `xml:"name,attr"`
	SchemaDocument W3CSchemaDoc     `xml:"schemaDocument"`
	Expected       W3CExpected      `xml:"expected"`
	Current        W3CCurrentStatus `xml:"current"`
}

// W3CInstanceTest tests whether an instance validates against a schema
type W3CInstanceTest struct {
	Name             string           `xml:"name,attr"`
	InstanceDocument W3CInstanceDoc   `xml:"instanceDocument"`
	Expected         W3CExpected      `xml:"expected"`
	Current          W3CCurrentStatus `xml:"current"`
}

// W3CSchemaDoc references a schema document
type W3CSchemaDoc struct {
	Href string `xml:"href,attr"`
}

// W3CInstanceDoc references an instance document
type W3CInstanceDoc struct {
	Href string `xml:"href,attr"`
}

// W3CExpected indicates expected validity
type W3CExpected struct {
	Validity string `xml:"validity,attr"` // "valid", "invalid", or "notKnown"
}

// W3CCurrentStatus tracks test acceptance status
type W3CCurrentStatus struct {
	Status string `xml:"status,attr"` // "accepted", "disputed", etc.
	Date   string `xml:"date,attr"`
}

// W3CTestResult captures the result of running a test
type W3CTestResult struct {
	TestSet      string
	TestGroup    string
	TestName     string
	TestType     string // "schema" or "instance"
	Expected     string // "valid", "invalid", "notKnown"
	Actual       string // "valid", "invalid", "error"
	Passed       bool
	Error        error
	SchemaPath   string
	InstancePath string
}

// W3CTestRunner runs W3C XSD conformance tests
type W3CTestRunner struct {
	TestSuiteDir string
	Results      []W3CTestResult
	Verbose      bool
}

// NewW3CTestRunner creates a test runner for the W3C test suite
func NewW3CTestRunner(testSuiteDir string) *W3CTestRunner {
	return &W3CTestRunner{
		TestSuiteDir: testSuiteDir,
		Results:      []W3CTestResult{},
	}
}

// LoadTestSet loads a W3C test set from an XML file
func (r *W3CTestRunner) LoadTestSet(metadataPath string) (*W3CTestSet, error) {
	file, err := os.Open(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open test metadata: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read test metadata: %w", err)
	}

	var testSet W3CTestSet
	if err := xml.Unmarshal(data, &testSet); err != nil {
		return nil, fmt.Errorf("failed to parse test metadata: %w", err)
	}

	return &testSet, nil
}

// RunTestSet runs all tests in a test set
func (r *W3CTestRunner) RunTestSet(testSet *W3CTestSet, metadataDir string) {
	for _, group := range testSet.TestGroups {
		// Run schema tests
		for _, test := range group.SchemaTests {
			result := r.runSchemaTest(testSet.Name, group.Name, test, metadataDir)
			r.Results = append(r.Results, result)

			if r.Verbose {
				r.printResult(result)
			}
		}

		// Run instance tests
		for _, test := range group.InstanceTests {
			result := r.runInstanceTest(testSet.Name, group.Name, test, metadataDir)
			r.Results = append(r.Results, result)

			if r.Verbose {
				r.printResult(result)
			}
		}
	}
}

// runSchemaTest validates whether a schema is well-formed
func (r *W3CTestRunner) runSchemaTest(testSet, testGroup string, test W3CSchemaTest, metadataDir string) W3CTestResult {
	result := W3CTestResult{
		TestSet:    testSet,
		TestGroup:  testGroup,
		TestName:   test.Name,
		TestType:   "schema",
		Expected:   test.Expected.Validity,
		SchemaPath: test.SchemaDocument.Href,
	}

	// Resolve schema path relative to metadata directory
	schemaPath := filepath.Join(metadataDir, test.SchemaDocument.Href)
	if !filepath.IsAbs(test.SchemaDocument.Href) {
		schemaPath = filepath.Join(filepath.Dir(metadataDir), test.SchemaDocument.Href)
	}

	// Try to load and parse the schema
	_, err := LoadSchema(schemaPath)

	if err != nil {
		result.Actual = "invalid"
		result.Error = err
	} else {
		result.Actual = "valid"
	}

	result.Passed = (result.Expected == result.Actual)
	return result
}

// runInstanceTest validates an instance document against a schema
func (r *W3CTestRunner) runInstanceTest(testSet, testGroup string, test W3CInstanceTest, metadataDir string) W3CTestResult {
	result := W3CTestResult{
		TestSet:      testSet,
		TestGroup:    testGroup,
		TestName:     test.Name,
		TestType:     "instance",
		Expected:     test.Expected.Validity,
		InstancePath: test.InstanceDocument.Href,
	}

	// For instance tests, we need to find the associated schema
	// Usually the schema test has the same base name
	schemaTestName := strings.TrimSuffix(test.Name, ".v")
	schemaTestName = strings.TrimSuffix(schemaTestName, ".i")

	// Find the schema path from a previous schema test
	var schemaPath string
	for _, prevResult := range r.Results {
		if prevResult.TestGroup == testGroup && prevResult.TestName == schemaTestName {
			schemaPath = prevResult.SchemaPath
			break
		}
	}

	if schemaPath == "" {
		result.Actual = "error"
		result.Error = fmt.Errorf("could not find associated schema for instance test")
		result.Passed = false
		return result
	}

	result.SchemaPath = schemaPath

	// Resolve paths relative to metadata directory
	if !filepath.IsAbs(schemaPath) {
		schemaPath = filepath.Join(filepath.Dir(metadataDir), schemaPath)
	}
	instancePath := filepath.Join(metadataDir, test.InstanceDocument.Href)
	if !filepath.IsAbs(test.InstanceDocument.Href) {
		instancePath = filepath.Join(filepath.Dir(metadataDir), test.InstanceDocument.Href)
	}

	// Load the schema
	schema, err := LoadSchema(schemaPath)
	if err != nil {
		result.Actual = "error"
		result.Error = fmt.Errorf("failed to load schema: %w", err)
		result.Passed = false
		return result
	}

	// Create validator
	validator := NewValidator(schema)

	// Load and validate the instance document
	file, err := os.Open(instancePath)
	if err != nil {
		result.Actual = "error"
		result.Error = fmt.Errorf("failed to open instance file: %w", err)
		result.Passed = false
		return result
	}
	defer file.Close()

	doc, err := xmldom.Decode(file)
	if err != nil {
		result.Actual = "error"
		result.Error = fmt.Errorf("failed to parse instance: %w", err)
		result.Passed = false
		return result
	}

	// Validate
	errors := validator.Validate(doc)

	if len(errors) > 0 {
		result.Actual = "invalid"
		result.Error = fmt.Errorf("%d validation errors: %v", len(errors), errors[0])
	} else {
		result.Actual = "valid"
	}

	result.Passed = (result.Expected == result.Actual)
	return result
}

// printResult prints a single test result
func (r *W3CTestRunner) printResult(result W3CTestResult) {
	status := "PASS"
	if !result.Passed {
		status = "FAIL"
	}

	fmt.Printf("[%s] %s/%s/%s: expected=%s, actual=%s",
		status, result.TestSet, result.TestGroup, result.TestName,
		result.Expected, result.Actual)

	if result.Error != nil && !result.Passed {
		fmt.Printf(" (error: %v)", result.Error)
	}
	fmt.Println()
}

// GenerateReport generates a summary report of test results
func (r *W3CTestRunner) GenerateReport() string {
	total := len(r.Results)
	passed := 0
	failed := 0
	errors := 0

	schemaTests := 0
	schemaPassed := 0
	instanceTests := 0
	instancePassed := 0

	failedTests := []W3CTestResult{}

	for _, result := range r.Results {
		if result.Passed {
			passed++
		} else {
			failed++
			failedTests = append(failedTests, result)
		}

		if result.Actual == "error" {
			errors++
		}

		if result.TestType == "schema" {
			schemaTests++
			if result.Passed {
				schemaPassed++
			}
		} else {
			instanceTests++
			if result.Passed {
				instancePassed++
			}
		}
	}

	var report strings.Builder
	report.WriteString("W3C XSD Conformance Test Results\n")
	report.WriteString("=================================\n\n")

	report.WriteString(fmt.Sprintf("Total Tests:     %d\n", total))
	report.WriteString(fmt.Sprintf("Passed:          %d (%.1f%%)\n", passed, float64(passed)*100/float64(total)))
	report.WriteString(fmt.Sprintf("Failed:          %d (%.1f%%)\n", failed, float64(failed)*100/float64(total)))
	report.WriteString(fmt.Sprintf("Errors:          %d\n\n", errors))

	report.WriteString(fmt.Sprintf("Schema Tests:    %d (passed: %d, %.1f%%)\n",
		schemaTests, schemaPassed, float64(schemaPassed)*100/float64(schemaTests)))
	report.WriteString(fmt.Sprintf("Instance Tests:  %d (passed: %d, %.1f%%)\n",
		instanceTests, instancePassed, float64(instancePassed)*100/float64(instanceTests)))

	if len(failedTests) > 0 && len(failedTests) <= 20 {
		report.WriteString("\nFailed Tests (showing first 20):\n")
		report.WriteString("---------------------------------\n")
		for i, result := range failedTests {
			if i >= 20 {
				break
			}
			report.WriteString(fmt.Sprintf("%s/%s/%s: expected=%s, actual=%s\n",
				result.TestSet, result.TestGroup, result.TestName,
				result.Expected, result.Actual))
		}
	}

	return report.String()
}

// RunMetadataFile runs all tests from a single metadata file
func (r *W3CTestRunner) RunMetadataFile(metadataPath string) error {
	testSet, err := r.LoadTestSet(metadataPath)
	if err != nil {
		return err
	}

	r.RunTestSet(testSet, metadataPath)
	return nil
}

// RunAllTests discovers and runs all test metadata files
func (r *W3CTestRunner) RunAllTests(pattern string) error {
	// Find all metadata files matching the pattern
	metadataFiles, err := filepath.Glob(filepath.Join(r.TestSuiteDir, pattern))
	if err != nil {
		return fmt.Errorf("failed to find test files: %w", err)
	}

	fmt.Printf("Found %d test metadata files\n", len(metadataFiles))

	for i, metadataPath := range metadataFiles {
		fmt.Printf("Running test file %d/%d: %s\n", i+1, len(metadataFiles), filepath.Base(metadataPath))

		if err := r.RunMetadataFile(metadataPath); err != nil {
			fmt.Printf("Error running %s: %v\n", metadataPath, err)
			// Continue with other test files
		}
	}

	return nil
}
