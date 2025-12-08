package xsd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	w3cTestSuiteURL = "https://www.w3.org/XML/2004/xml-schema-test-suite/xmlschema2006-11-06/xsts-2007-06-20.tar.gz"
	downloadMarker  = ".w3c_test_suite_downloaded"
	cacheDuration   = 7 * 24 * time.Hour
)

// W3CTestCacheDuration exposes the cache lifetime for the downloaded suite.
const W3CTestCacheDuration = cacheDuration

// W3CTestOptions controls the behavior of RunW3CTestSuite.
type W3CTestOptions struct {
	SuiteDir           string
	Pattern            string
	MetadataFile       string
	AutoDownload       bool
	ForceDownload      bool
	Limit              int
	Grep               string
	StrictContentModel bool
	Verbose            bool
	AnalyzeFailures    bool
}

// W3CTestRun captures the outcome of RunW3CTestSuite.
type W3CTestRun struct {
	Downloaded      bool
	Results         []W3CTestResult
	Report          string
	FailureAnalysis string
	TestCount       int
	PassedCount     int
	FailedCount     int
	LogLines        []string
}

// RunW3CTestSuite orchestrates downloading (if needed) and executing the W3C test suite.
func RunW3CTestSuite(ctx context.Context, opts W3CTestOptions) (*W3CTestRun, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if opts.SuiteDir == "" {
		return nil, fmt.Errorf("suite directory is required")
	}

	if opts.ForceDownload {
		if err := os.Remove(filepath.Join(opts.SuiteDir, downloadMarker)); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to reset suite cache: %w", err)
		}
	}

	downloaded, err := ensureTestSuite(ctx, opts.SuiteDir, opts.AutoDownload || opts.ForceDownload)
	if err != nil {
		return nil, err
	}

	runner := NewW3CTestRunner(opts.SuiteDir)
	runner.StrictContentModel = opts.StrictContentModel
	runner.Grep = opts.Grep
	runner.Verbose = opts.Verbose

	var verboseBuf strings.Builder
	if opts.Verbose {
		runner.LogWriter = &verboseBuf
	} else {
		runner.LogWriter = io.Discard
	}

	if opts.MetadataFile != "" {
		if err := runner.RunMetadataFile(opts.MetadataFile); err != nil {
			return nil, err
		}
	} else {
		pattern := opts.Pattern
		if pattern == "" {
			pattern = "msMeta/*_w3c.xml"
		}
		if err := runner.RunAllTests(pattern); err != nil {
			return nil, err
		}
	}

	if opts.Limit > 0 && len(runner.Results) > opts.Limit {
		runner.Results = runner.Results[:opts.Limit]
	}

	report := runner.GenerateReport()

	var failureReport string
	if opts.AnalyzeFailures {
		categories := AnalyzeTestFailures(runner.Results)
		failureReport = GenerateFailureReport(categories)
	}

	logLines := []string{}
	if opts.Verbose {
		logLines = strings.Split(strings.TrimSpace(verboseBuf.String()), "\n")
		if len(logLines) == 1 && logLines[0] == "" {
			logLines = nil
		}
	}

	run := &W3CTestRun{
		Downloaded:      downloaded,
		Results:         runner.Results,
		Report:          report,
		FailureAnalysis: failureReport,
		TestCount:       len(runner.Results),
		LogLines:        logLines,
	}

	for _, result := range runner.Results {
		if result.Passed {
			run.PassedCount++
		} else {
			run.FailedCount++
		}
	}

	return run, nil
}

func ensureTestSuite(ctx context.Context, dir string, autoDownload bool) (bool, error) {
	markerPath := filepath.Join(dir, downloadMarker)

	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		if markerStat, err := os.Stat(markerPath); err == nil {
			if age := time.Since(markerStat.ModTime()); age < cacheDuration {
				return false, nil
			}
		}
	}

	if !autoDownload {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return false, fmt.Errorf("test suite not found at %s\n\nUse --auto-download to fetch automatically or download manually from %s", dir, w3cTestSuiteURL)
		}
		return false, nil
	}

	if err := downloadAndExtract(ctx, w3cTestSuiteURL, dir); err != nil {
		return false, err
	}

	if err := os.WriteFile(markerPath, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		return true, fmt.Errorf("failed to write suite marker: %w", err)
	}

	return true, nil
}

func downloadAndExtract(ctx context.Context, url, destDir string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "go-xsd-test-runner/1.0 (https://github.com/agentflare-ai/go-xsd)")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download test suite: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status downloading test suite: %s", resp.Status)
	}

	tempDir := destDir + ".tmp"
	if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temp dir: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	progress := &progressReader{reader: resp.Body, total: resp.ContentLength}
	gzReader, err := gzip.NewReader(progress)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var commonPrefix string
	firstFile := true

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		if firstFile && header.Typeflag == tar.TypeDir {
			if idx := strings.Index(header.Name, "/"); idx > 0 {
				commonPrefix = header.Name[:idx+1]
			}
			firstFile = false
		}

		targetPath := header.Name
		if commonPrefix != "" && strings.HasPrefix(header.Name, commonPrefix) {
			targetPath = header.Name[len(commonPrefix):]
			if targetPath == "" {
				continue
			}
		}

		target := filepath.Join(tempDir, targetPath)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(tempDir)) {
			return fmt.Errorf("illegal path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", target, err)
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			file.Close()
		}
	}

	if err := os.RemoveAll(destDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old suite dir: %w", err)
	}
	if err := os.Rename(tempDir, destDir); err != nil {
		return fmt.Errorf("failed to finalize suite dir: %w", err)
	}

	return nil
}

type progressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	lastPrint time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	now := time.Now()
	if now.Sub(pr.lastPrint) > 500*time.Millisecond || err == io.EOF {
		pr.lastPrint = now
		if pr.total > 0 {
			percent := float64(pr.current) / float64(pr.total) * 100
			fmt.Printf("\rProgress: %.1f%% (%d / %d bytes)", percent, pr.current, pr.total)
		} else {
			fmt.Printf("\rProgress: %d bytes", pr.current)
		}
	}

	return n, err
}
