package xsd

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDocMarkdown(t *testing.T) {
	schemaPath := filepath.Join("testdata", "doc", "simple_schema.xsd")
	schema, err := LoadSchemaWithOptions(context.Background(), SchemaLoadOptions{
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatalf("failed to load schema: %v", err)
	}

	doc, err := GenerateDoc(schema, DocFormatMarkdown, DocOptions{
		Title:      "",
		IncludeTOC: true,
	})
	if err != nil {
		t.Fatalf("GenerateDoc failed: %v", err)
	}

	if !strings.Contains(doc, "# Agentml Schema") {
		t.Fatalf("missing derived title with namespace: %s", doc)
	}
	if !strings.Contains(doc, "AgentML 1.0 core schema") {
		t.Fatalf("missing top-level schema documentation")
	}
	if !strings.Contains(doc, "## 1 Overview") {
		t.Fatalf("missing numbered Overview section")
	}
	if !strings.Contains(doc, "## 2 Elements") {
		t.Fatalf("missing numbered Elements section")
	}
	if !strings.Contains(doc, "## 3 Types") {
		t.Fatalf("missing numbered Types section")
	}
	if !strings.Contains(doc, "### 2 &lt;") {
		t.Fatalf("missing numbered element headings with angle-bracket names")
	}
	if !strings.Contains(doc, "- [2 Elements](#elements)") {
		t.Fatalf("TOC does not include numbered Elements entry")
	}
}

func TestSupportedDocFormats(t *testing.T) {
	formats := SupportedDocFormats()
	if _, ok := formats[DocFormatMarkdown]; !ok {
		t.Fatalf("markdown format not advertised")
	}
	if len(formats) != 1 {
		t.Fatalf("unexpected formats: %v", formats)
	}
}
