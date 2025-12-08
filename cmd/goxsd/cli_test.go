package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommandJSON(t *testing.T) {
	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	schema := filepath.Join("testdata", "simple.xsd")
	valid := filepath.Join("testdata", "valid.xml")

	cmd.SetArgs([]string{
		"validate",
		"--format=json",
		"--exit-on-violation=false",
		schema,
		valid,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("validate command failed: %v\noutput: %s", err, output.String())
	}

	var docs []struct {
		Document    string        `json:"document"`
		Violations  []interface{} `json:"violations"`
		Diagnostics []interface{} `json:"diagnostics"`
	}
	if err := json.Unmarshal(output.Bytes(), &docs); err != nil {
		t.Fatalf("failed to parse json output: %v\n%s", err, output.String())
	}
	if len(docs) != 1 || len(docs[0].Violations) != 0 {
		t.Fatalf("expected no violations, got %+v", docs)
	}
}

func TestDocCommandMarkdown(t *testing.T) {
	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	schema := filepath.Join("testdata", "simple.xsd")
	cmd.SetArgs([]string{
		"doc",
		schema,
		"--out=-",
		"--format=markdown",
		"--title=CLI Doc",
		"--section=overview",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("doc command failed: %v\noutput: %s", err, output.String())
	}

	doc := output.String()
	if !strings.Contains(doc, "# CLI Doc") {
		t.Fatalf("expected custom title, got:\n%s", doc)
	}
	if !strings.Contains(doc, "## Overview") {
		t.Fatalf("expected overview section, got:\n%s", doc)
	}
}
