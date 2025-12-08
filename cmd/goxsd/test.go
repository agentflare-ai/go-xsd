package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/agentflare-ai/go-xsd"
	"github.com/spf13/cobra"
)

type testOptions struct {
	suiteDir      string
	pattern       string
	metadataFile  string
	autoDownload  bool
	forceDownload bool
	limit         int
	analyze       bool
	grep          string
	verbose       bool
	outputPath    string
}

func newTestCommand(global *globalOptions) *cobra.Command {
	opts := &testOptions{
		suiteDir: "/tmp/xsd-test-suite",
		pattern:  "msMeta/*_w3c.xml",
	}

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run the official W3C XSD test suite",
		Long:  "Download (if requested) and execute the W3C XML Schema conformance suite, summarizing failures and optionally generating analysis reports.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTests(cmd.Context(), cmd.OutOrStdout(), opts, global)
		},
	}

	cmd.Flags().StringVar(&opts.suiteDir, "suite", opts.suiteDir, "Path to the W3C XSD test suite directory")
	cmd.Flags().StringVar(&opts.pattern, "pattern", opts.pattern, "Glob for metadata files to execute")
	cmd.Flags().StringVar(&opts.metadataFile, "file", "", "Run a single metadata file instead of pattern")
	cmd.Flags().BoolVar(&opts.autoDownload, "auto-download", false, "Automatically download the W3C test suite if missing")
	cmd.Flags().BoolVar(&opts.forceDownload, "force-download", false, "Force re-download of the W3C test suite (implies --auto-download)")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Limit the number of executed tests (0 = all)")
	cmd.Flags().BoolVar(&opts.analyze, "analyze", false, "Generate a categorized failure analysis summary")
	cmd.Flags().StringVar(&opts.grep, "grep", "", "Only run tests whose set/group/name contains this substring (case-insensitive)")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Print each test result as it executes")
	cmd.Flags().StringVar(&opts.outputPath, "output", "", "Write the final test report to a file instead of stdout")

	return cmd
}

func runTests(ctx context.Context, stdout io.Writer, opts *testOptions, global *globalOptions) error {
	run, err := xsd.RunW3CTestSuite(ctx, xsd.W3CTestOptions{
		SuiteDir:           opts.suiteDir,
		Pattern:            opts.pattern,
		MetadataFile:       opts.metadataFile,
		AutoDownload:       opts.autoDownload,
		ForceDownload:      opts.forceDownload,
		Grep:               opts.grep,
		Limit:              opts.limit,
		StrictContentModel: global.strictContentModels,
		Verbose:            opts.verbose,
		AnalyzeFailures:    opts.analyze,
	})
	if err != nil {
		return err
	}

	if run.Downloaded {
		fmt.Fprintf(stdout, "Note: downloaded W3C suite cached for %s\n\n", xsd.W3CTestCacheDuration)
	}

	if opts.verbose {
		for _, line := range run.LogLines {
			if line != "" {
				fmt.Fprintln(stdout, line)
			}
		}
	}

	if opts.outputPath != "" {
		if err := os.WriteFile(opts.outputPath, []byte(run.Report), 0o644); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		fmt.Fprintf(stdout, "Report written to %s\n", opts.outputPath)
	} else {
		fmt.Fprintln(stdout, run.Report)
	}

	if run.FailureAnalysis != "" {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, run.FailureAnalysis)
	}

	if run.FailedCount > 0 {
		return &exitError{code: 1, silent: true}
	}

	return nil
}
