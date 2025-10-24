package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/agentflare-ai/go-xsd"
)

func main() {
	var (
		testSuiteDir = flag.String("suite", "/tmp/xsd-test-suite", "Path to W3C XSD test suite")
		pattern      = flag.String("pattern", "msMeta/*_w3c.xml", "Pattern for test metadata files")
		verbose      = flag.Bool("verbose", false, "Print detailed test results")
		outputFile   = flag.String("output", "", "Output file for report (default: stdout)")
		testFile     = flag.String("file", "", "Run a specific test metadata file")
		limit        = flag.Int("limit", 0, "Limit number of tests to run (0 = no limit)")
		analyze      = flag.Bool("analyze", false, "Generate failure analysis report")
		autoDownload = flag.Bool("auto-download", false, "Automatically download W3C test suite if not found")
		forceDownload = flag.Bool("force-download", false, "Force re-download even if cached (implies --auto-download)")
	)

	flag.Parse()

	// Force download implies auto download
	if *forceDownload {
		*autoDownload = true
		// Remove cache marker to force fresh download
		markerPath := filepath.Join(*testSuiteDir, downloadMarker)
		os.Remove(markerPath)
	}

	// Ensure test suite exists (download if needed)
	downloaded, err := ensureTestSuite(*testSuiteDir, *autoDownload)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if downloaded {
		fmt.Printf("Note: Downloaded test suite is cached for %v\n\n", cacheDuration)
	}

	// Create test runner
	runner := xsd.NewW3CTestRunner(*testSuiteDir)
	runner.Verbose = *verbose

	// Run tests
	if *testFile != "" {
		// Run single test file
		fmt.Printf("Running test file: %s\n", *testFile)
		if err := runner.RunMetadataFile(*testFile); err != nil {
			log.Fatalf("Failed to run test file: %v", err)
		}
	} else {
		// Run all tests matching pattern
		fmt.Printf("Running W3C XSD conformance tests from: %s\n", *testSuiteDir)
		fmt.Printf("Test pattern: %s\n", *pattern)

		if err := runner.RunAllTests(*pattern); err != nil {
			log.Fatalf("Failed to run tests: %v", err)
		}
	}

	// Apply limit if specified
	if *limit > 0 && len(runner.Results) > *limit {
		runner.Results = runner.Results[:*limit]
		fmt.Printf("\nLimited results to %d tests\n", *limit)
	}

	// Generate report
	report := runner.GenerateReport()

	// Generate failure analysis if requested
	if *analyze {
		categories := xsd.AnalyzeTestFailures(runner.Results)
		failureReport := xsd.GenerateFailureReport(categories)
		report = report + "\n\n" + failureReport
	}

	// Output report
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(report), 0644); err != nil {
			log.Fatalf("Failed to write report: %v", err)
		}
		fmt.Printf("\nReport written to: %s\n", *outputFile)
	} else {
		fmt.Println("\n" + report)
	}

	// Exit with non-zero status if tests failed
	failedCount := 0
	for _, result := range runner.Results {
		if !result.Passed {
			failedCount++
		}
	}

	if failedCount > 0 {
		os.Exit(1)
	}
}
