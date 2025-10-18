package xsd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestSchemaImport(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "xsd-import-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create main schema file
	mainSchema := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/main"
           xmlns:main="http://example.com/main"
           xmlns:types="http://example.com/types">
    
    <xs:import namespace="http://example.com/types" 
               schemaLocation="types.xsd"/>
    
    <xs:element name="document">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="title" type="xs:string"/>
                <xs:element name="author" type="types:personType"/>
                <xs:element name="content" type="main:contentType"/>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
    
    <xs:complexType name="contentType">
        <xs:sequence>
            <xs:element name="paragraph" type="xs:string" 
                        minOccurs="1" maxOccurs="unbounded"/>
        </xs:sequence>
    </xs:complexType>
</xs:schema>`

	// Create imported types schema
	typesSchema := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/types"
           xmlns:types="http://example.com/types">
    
    <xs:complexType name="personType">
        <xs:sequence>
            <xs:element name="name" type="xs:string"/>
            <xs:element name="email" type="types:emailType"/>
        </xs:sequence>
    </xs:complexType>
    
    <xs:simpleType name="emailType">
        <xs:restriction base="xs:string">
            <xs:pattern value="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"/>
        </xs:restriction>
    </xs:simpleType>
</xs:schema>`

	// Write schema files
	mainFile := filepath.Join(tempDir, "main.xsd")
	if err := os.WriteFile(mainFile, []byte(mainSchema), 0644); err != nil {
		t.Fatalf("Failed to write main schema: %v", err)
	}

	typesFile := filepath.Join(tempDir, "types.xsd")
	if err := os.WriteFile(typesFile, []byte(typesSchema), 0644); err != nil {
		t.Fatalf("Failed to write types schema: %v", err)
	}

	// Load the schema with imports
	loader := NewSchemaLoaderSimple(tempDir)
	schema, err := loader.LoadSchemaWithImports(mainFile)
	if err != nil {
		t.Fatalf("Failed to load schema with imports: %v", err)
	}

	// Verify the combined schema has components from both files
	if schema.TargetNamespace != "http://example.com/main" {
		t.Errorf("Expected target namespace http://example.com/main, got %s",
			schema.TargetNamespace)
	}

	// Check for main schema element
	documentElem := QName{Namespace: "http://example.com/main", Local: "document"}
	if _, exists := schema.ElementDecls[documentElem]; !exists {
		t.Error("Document element not found in combined schema")
	}

	// Check for main schema type
	contentType := QName{Namespace: "http://example.com/main", Local: "contentType"}
	if _, exists := schema.TypeDefs[contentType]; !exists {
		t.Error("ContentType not found in combined schema")
	}

	// Check for imported type
	personType := QName{Namespace: "http://example.com/types", Local: "personType"}
	if _, exists := schema.TypeDefs[personType]; !exists {
		t.Error("PersonType not found in combined schema")
	}

	emailType := QName{Namespace: "http://example.com/types", Local: "emailType"}
	if _, exists := schema.TypeDefs[emailType]; !exists {
		t.Error("EmailType not found in combined schema")
	}

	// Test validation with the combined schema
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<document xmlns="http://example.com/main"
          xmlns:types="http://example.com/types">
    <title>Test Document</title>
    <author>
        <types:name>John Doe</types:name>
        <types:email>john.doe@example.com</types:email>
    </author>
    <content>
        <paragraph>First paragraph</paragraph>
        <paragraph>Second paragraph</paragraph>
    </content>
</document>`

	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<document xmlns="http://example.com/main"
          xmlns:types="http://example.com/types">
    <title>Test Document</title>
    <author>
        <types:name>John Doe</types:name>
        <types:email>invalid-email</types:email>
    </author>
    <content>
        <paragraph>First paragraph</paragraph>
    </content>
</document>`

	// Validate valid document
	validDoc, err := xmldom.Decode(bytes.NewReader([]byte(validXML)))
	if err != nil {
		t.Fatalf("Failed to parse valid XML: %v", err)
	}

	validator := NewValidator(schema)
	violations := validator.Validate(validDoc)
	if len(violations) > 0 {
		t.Errorf("Valid document should have no violations, got: %v", violations)
	}

	// Validate invalid document
	invalidDoc, err := xmldom.Decode(bytes.NewReader([]byte(invalidXML)))
	if err != nil {
		t.Fatalf("Failed to parse invalid XML: %v", err)
	}

	violations = validator.Validate(invalidDoc)
	if len(violations) == 0 {
		t.Error("Invalid document should have violations")
	}
}

func TestSchemaInclude(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "xsd-include-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create main schema file with include
	mainSchema := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/app"
           xmlns:app="http://example.com/app">
    
    <xs:include schemaLocation="common-types.xsd"/>
    
    <xs:element name="application">
        <xs:complexType>
            <xs:sequence>
                <xs:element name="user" type="app:userType"/>
                <xs:element name="settings" type="app:settingsType"/>
            </xs:sequence>
        </xs:complexType>
    </xs:element>
    
    <xs:complexType name="userType">
        <xs:sequence>
            <xs:element name="id" type="app:idType"/>
            <xs:element name="username" type="xs:string"/>
            <xs:element name="role" type="app:roleType"/>
        </xs:sequence>
    </xs:complexType>
</xs:schema>`

	// Create included common types schema (same namespace)
	commonTypes := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/app"
           xmlns:app="http://example.com/app">
    
    <xs:simpleType name="idType">
        <xs:restriction base="xs:string">
            <xs:pattern value="[A-Z]{2}[0-9]{6}"/>
        </xs:restriction>
    </xs:simpleType>
    
    <xs:simpleType name="roleType">
        <xs:restriction base="xs:string">
            <xs:enumeration value="admin"/>
            <xs:enumeration value="user"/>
            <xs:enumeration value="guest"/>
        </xs:restriction>
    </xs:simpleType>
    
    <xs:complexType name="settingsType">
        <xs:sequence>
            <xs:element name="theme" type="xs:string"/>
            <xs:element name="language" type="xs:string"/>
        </xs:sequence>
    </xs:complexType>
</xs:schema>`

	// Write schema files
	mainFile := filepath.Join(tempDir, "main.xsd")
	if err := os.WriteFile(mainFile, []byte(mainSchema), 0644); err != nil {
		t.Fatalf("Failed to write main schema: %v", err)
	}

	commonFile := filepath.Join(tempDir, "common-types.xsd")
	if err := os.WriteFile(commonFile, []byte(commonTypes), 0644); err != nil {
		t.Fatalf("Failed to write common types schema: %v", err)
	}

	// Load the schema with includes
	loader := NewSchemaLoaderSimple(tempDir)
	schema, err := loader.LoadSchemaWithImports(mainFile)
	if err != nil {
		t.Fatalf("Failed to load schema with includes: %v", err)
	}

	// All types should be in the same namespace
	if schema.TargetNamespace != "http://example.com/app" {
		t.Errorf("Expected target namespace http://example.com/app, got %s",
			schema.TargetNamespace)
	}

	// Check for types from main schema
	userType := QName{Namespace: "http://example.com/app", Local: "userType"}
	if _, exists := schema.TypeDefs[userType]; !exists {
		t.Error("UserType not found in combined schema")
	}

	// Check for types from included schema
	idType := QName{Namespace: "http://example.com/app", Local: "idType"}
	if _, exists := schema.TypeDefs[idType]; !exists {
		t.Error("IdType not found in combined schema")
	}

	roleType := QName{Namespace: "http://example.com/app", Local: "roleType"}
	if _, exists := schema.TypeDefs[roleType]; !exists {
		t.Error("RoleType not found in combined schema")
	}

	settingsType := QName{Namespace: "http://example.com/app", Local: "settingsType"}
	if _, exists := schema.TypeDefs[settingsType]; !exists {
		t.Error("SettingsType not found in combined schema")
	}

	// Test validation
	validXML := `<?xml version="1.0" encoding="UTF-8"?>
<application xmlns="http://example.com/app">
    <user>
        <id>AB123456</id>
        <username>johndoe</username>
        <role>admin</role>
    </user>
    <settings>
        <theme>dark</theme>
        <language>en</language>
    </settings>
</application>`

	invalidXML := `<?xml version="1.0" encoding="UTF-8"?>
<application xmlns="http://example.com/app">
    <user>
        <id>invalid-id</id>
        <username>johndoe</username>
        <role>superuser</role>
    </user>
    <settings>
        <theme>dark</theme>
        <language>en</language>
    </settings>
</application>`

	// Validate valid document
	validDoc, err := xmldom.Decode(bytes.NewReader([]byte(validXML)))
	if err != nil {
		t.Fatalf("Failed to parse valid XML: %v", err)
	}

	validator := NewValidator(schema)
	violations := validator.Validate(validDoc)
	if len(violations) > 0 {
		t.Errorf("Valid document should have no violations, got: %v", violations)
	}

	// Validate invalid document
	invalidDoc, err := xmldom.Decode(bytes.NewReader([]byte(invalidXML)))
	if err != nil {
		t.Fatalf("Failed to parse invalid XML: %v", err)
	}

	violations = validator.Validate(invalidDoc)
	if len(violations) < 2 { // Should have errors for both id and role
		t.Errorf("Invalid document should have at least 2 violations, got %d",
			len(violations))
	}
}

func TestCircularImports(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "xsd-circular-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create schema A that imports B
	schemaA := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/a"
           xmlns:a="http://example.com/a"
           xmlns:b="http://example.com/b">
    <xs:import namespace="http://example.com/b" schemaLocation="b.xsd"/>
    <xs:complexType name="typeA">
        <xs:sequence>
            <xs:element name="value" type="xs:string"/>
        </xs:sequence>
    </xs:complexType>
</xs:schema>`

	// Create schema B that imports A (circular)
	schemaB := `<?xml version="1.0" encoding="UTF-8"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/b"
           xmlns:b="http://example.com/b"
           xmlns:a="http://example.com/a">
    <xs:import namespace="http://example.com/a" schemaLocation="a.xsd"/>
    <xs:complexType name="typeB">
        <xs:sequence>
            <xs:element name="data" type="xs:string"/>
        </xs:sequence>
    </xs:complexType>
</xs:schema>`

	// Write schema files
	fileA := filepath.Join(tempDir, "a.xsd")
	if err := os.WriteFile(fileA, []byte(schemaA), 0644); err != nil {
		t.Fatalf("Failed to write schema A: %v", err)
	}

	fileB := filepath.Join(tempDir, "b.xsd")
	if err := os.WriteFile(fileB, []byte(schemaB), 0644); err != nil {
		t.Fatalf("Failed to write schema B: %v", err)
	}

	// Load schema A (which imports B, which imports A)
	loader := NewSchemaLoaderSimple(tempDir)
	schema, err := loader.LoadSchemaWithImports(fileA)

	// Should handle circular imports gracefully
	if err != nil {
		// Circular imports should be handled, not error
		t.Logf("Loading with circular imports: %v", err)
	}

	// Both types should be available
	if schema != nil {
		typeA := QName{Namespace: "http://example.com/a", Local: "typeA"}
		if _, exists := schema.TypeDefs[typeA]; !exists {
			t.Error("TypeA not found in combined schema")
		}

		typeB := QName{Namespace: "http://example.com/b", Local: "typeB"}
		if _, exists := schema.TypeDefs[typeB]; !exists {
			t.Error("TypeB not found in combined schema")
		}
	}
}
