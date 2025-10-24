package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// W3C XSD Test Suite URL
	testSuiteURL = "https://www.w3.org/XML/2004/xml-schema-test-suite/xmlschema2006-11-06/xsts-2007-06-20.tar.gz"

	// Cache duration - don't re-download if younger than this
	cacheDuration = 7 * 24 * time.Hour

	// Marker file to track download timestamp
	downloadMarker = ".w3c_test_suite_downloaded"
)

// downloadInfo stores metadata about the downloaded test suite
type downloadInfo struct {
	DownloadedAt time.Time
	URL          string
}

// ensureTestSuite ensures the test suite exists, downloading if necessary
// Returns true if download occurred, false if using cached version
func ensureTestSuite(dir string, autoDownload bool) (bool, error) {
	// Check if directory exists and has marker file
	markerPath := filepath.Join(dir, downloadMarker)

	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		// Directory exists, check if it's recent enough
		if markerStat, err := os.Stat(markerPath); err == nil {
			age := time.Since(markerStat.ModTime())
			if age < cacheDuration {
				// Cache is still valid
				return false, nil
			}
			fmt.Printf("Test suite cache is %v old (max %v), will re-download if auto-download enabled\n",
				age.Round(time.Hour), cacheDuration)
		}
	}

	// Test suite doesn't exist or is stale
	if !autoDownload {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return false, fmt.Errorf("test suite not found at %s\n\nTo download automatically, use: --auto-download\n\nOr download manually from:\n%s", dir, testSuiteURL)
		}
		// Directory exists but is stale - that's ok, use it anyway
		return false, nil
	}

	// Download the test suite
	fmt.Printf("Downloading W3C XSD Test Suite...\n")
	fmt.Printf("Source: %s\n", testSuiteURL)
	fmt.Printf("Destination: %s\n", dir)

	if err := downloadAndExtract(testSuiteURL, dir); err != nil {
		return false, fmt.Errorf("failed to download test suite: %w", err)
	}

	// Create marker file
	if err := os.WriteFile(markerPath, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		fmt.Printf("Warning: failed to create marker file: %v\n", err)
	}

	fmt.Printf("✓ Test suite downloaded successfully\n\n")
	return true, nil
}

// downloadAndExtract downloads a tar.gz file and extracts it
func downloadAndExtract(url, destDir string) error {
	// Create HTTP client with timeout and proper headers
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set proper User-Agent to be respectful
	req.Header.Set("User-Agent", "go-xsd-test-runner/1.0 (https://github.com/agentflare-ai/go-xsd)")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Create temp directory for extraction
	tempDir := destDir + ".tmp"
	if err := os.RemoveAll(tempDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean temp dir: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Show progress
	fmt.Printf("Downloading (size: %d bytes)...\n", resp.ContentLength)

	// Create progress reader
	progressReader := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
	}

	// Decompress gzip
	gzReader, err := gzip.NewReader(progressReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Extract tar
	tarReader := tar.NewReader(gzReader)

	// Detect common prefix (strip top-level directory if all files share it)
	var commonPrefix string
	firstFile := true

	extractedFiles := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// Detect common prefix from first directory entry
		if firstFile && header.Typeflag == tar.TypeDir {
			// Extract the first directory component
			if idx := strings.Index(header.Name, "/"); idx > 0 {
				commonPrefix = header.Name[:idx+1]
			}
			firstFile = false
		}

		// Strip common prefix if present
		targetPath := header.Name
		if commonPrefix != "" && filepath.HasPrefix(header.Name, commonPrefix) {
			targetPath = header.Name[len(commonPrefix):]
			// Skip if this is just the top-level directory itself
			if targetPath == "" {
				continue
			}
		}

		// Security check: prevent path traversal
		target := filepath.Join(tempDir, targetPath)
		if !filepath.HasPrefix(filepath.Clean(target), filepath.Clean(tempDir)) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", target, err)
			}

			// Create file
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			file.Close()
			extractedFiles++
		}
	}

	fmt.Printf("\n✓ Extracted %d files\n", extractedFiles)

	// Move temp directory to final location
	if err := os.RemoveAll(destDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old directory: %w", err)
	}

	if err := os.Rename(tempDir, destDir); err != nil {
		return fmt.Errorf("failed to move directory: %w", err)
	}

	return nil
}

// progressReader wraps an io.Reader to show download progress
type progressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	lastPrint time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	// Print progress every 500ms
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
