package xsd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

const simpleSchema = `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/test"
           elementFormDefault="qualified">
  <xs:element name="root" type="xs:string"/>
</xs:schema>`

func TestLoadSchemaWithOptionsDirectoryLookup(t *testing.T) {
	t.Run("repoNameSchema", func(t *testing.T) {
		dir := t.TempDir()
		schemaPath := filepath.Join(dir, filepath.Base(dir)+".xsd")
		if err := os.WriteFile(schemaPath, []byte(simpleSchema), 0644); err != nil {
			t.Fatalf("write schema: %v", err)
		}

		schema, err := LoadSchemaWithOptions(context.Background(), SchemaLoadOptions{
			SchemaPath: dir,
		})
		if err != nil {
			t.Fatalf("LoadSchemaWithOptions failed: %v", err)
		}
		if schema == nil {
			t.Fatalf("expected schema, got nil")
		}
	})

	t.Run("fallbackSchemaDotXsd", func(t *testing.T) {
		dir := t.TempDir()
		schemaPath := filepath.Join(dir, "schema.xsd")
		if err := os.WriteFile(schemaPath, []byte(simpleSchema), 0644); err != nil {
			t.Fatalf("write schema: %v", err)
		}

		schema, err := LoadSchemaWithOptions(context.Background(), SchemaLoadOptions{
			SchemaPath: dir,
		})
		if err != nil {
			t.Fatalf("LoadSchemaWithOptions failed: %v", err)
		}
		if schema == nil {
			t.Fatalf("expected schema, got nil")
		}
	})

	t.Run("openaiSchema", func(t *testing.T) {
		dir := t.TempDir()
		schemaPath := filepath.Join(dir, "openai.xsd")
		if err := os.WriteFile(schemaPath, []byte(simpleSchema), 0644); err != nil {
			t.Fatalf("write schema: %v", err)
		}

		if _, err := LoadSchemaWithOptions(context.Background(), SchemaLoadOptions{
			SchemaPath: dir,
		}); err != nil {
			t.Fatalf("LoadSchemaWithOptions failed: %v", err)
		}
	})

	t.Run("missingSchema", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := LoadSchemaWithOptions(context.Background(), SchemaLoadOptions{
			SchemaPath: dir,
		}); err == nil {
			t.Fatalf("expected error when schema missing")
		}
	})
}
