package xsd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestAnyElementValidation(t *testing.T) {
	// Create a schema with xs:any wildcard
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="container">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="header" type="xs:string"/>
					<xs:any namespace="##other" processContents="lax" minOccurs="0" maxOccurs="unbounded"/>
					<xs:element name="footer" type="xs:string"/>
				</xs:sequence>
			</xs:complexType>
		</xs:element>
	</xs:schema>`

	doc, err := xmldom.Decode(bytes.NewReader([]byte(schemaXML)))
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	schema, err := Parse(doc)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid - no wildcard elements",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<header>Title</header>
				<footer>End</footer>
			</container>`,
			wantError: false,
		},
		{
			name: "valid - other namespace element",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<header>Title</header>
				<other:element xmlns:other="http://other.com">Content</other:element>
				<footer>End</footer>
			</container>`,
			wantError: false,
		},
		{
			name: "invalid - same namespace element",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<header>Title</header>
				<extra>Not allowed</extra>
				<footer>End</footer>
			</container>`,
			wantError: true,
			errorMsg:  "not allowed by the namespace constraint",
		},
		{
			name: "valid - multiple other namespace elements",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<header>Title</header>
				<ns1:elem1 xmlns:ns1="http://ns1.com">First</ns1:elem1>
				<ns2:elem2 xmlns:ns2="http://ns2.com">Second</ns2:elem2>
				<footer>End</footer>
			</container>`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := xmldom.Decode(bytes.NewReader([]byte(tt.xml)))
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			validator := NewValidator(schema)
			violations := validator.Validate(doc)

			hasError := len(violations) > 0
			if hasError != tt.wantError {
				t.Errorf("Expected error=%v but got %v violations: %v",
					tt.wantError, len(violations), violations)
			}

			if tt.wantError && tt.errorMsg != "" && hasError {
				found := false
				for _, v := range violations {
					if contains(v.Message, tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s' but got: %v",
						tt.errorMsg, violations)
				}
			}
		})
	}
}

func TestAnyAttributeValidation(t *testing.T) {
	// Create a schema with xs:anyAttribute wildcard
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="element">
			<xs:complexType>
				<xs:simpleContent>
					<xs:extension base="xs:string">
						<xs:attribute name="id" type="xs:string" use="required"/>
						<xs:attribute name="type" type="xs:string"/>
						<xs:anyAttribute namespace="##local" processContents="skip"/>
					</xs:extension>
				</xs:simpleContent>
			</xs:complexType>
		</xs:element>
	</xs:schema>`

	doc, err := xmldom.Decode(bytes.NewReader([]byte(schemaXML)))
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	schema, err := Parse(doc)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid - only declared attributes",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<element xmlns="http://example.com" id="elem1" type="string">Content</element>`,
			wantError: false,
		},
		{
			name: "valid - with local namespace attribute",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<element xmlns="http://example.com" id="elem1" custom="value">Content</element>`,
			wantError: false,
		},
		{
			name: "invalid - with qualified namespace attribute",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<element xmlns="http://example.com" xmlns:other="http://other.com" 
			         id="elem1" other:attr="value">Content</element>`,
			wantError: true,
			errorMsg:  "not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := xmldom.Decode(bytes.NewReader([]byte(tt.xml)))
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			validator := NewValidator(schema)
			violations := validator.Validate(doc)

			hasError := len(violations) > 0
			if hasError != tt.wantError {
				t.Errorf("Expected error=%v but got %v violations: %v",
					tt.wantError, len(violations), violations)
			}

			if tt.wantError && tt.errorMsg != "" && hasError {
				found := false
				for _, v := range violations {
					if contains(v.Message, tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing '%s' but got: %v",
						tt.errorMsg, violations)
				}
			}
		})
	}
}

func TestNamespaceConstraints(t *testing.T) {
	tests := []struct {
		name            string
		constraint      string
		elemNamespace   string
		targetNamespace string
		shouldMatch     bool
	}{
		{
			name:            "##any allows everything",
			constraint:      "##any",
			elemNamespace:   "http://any.com",
			targetNamespace: "http://target.com",
			shouldMatch:     true,
		},
		{
			name:            "##other allows different namespace",
			constraint:      "##other",
			elemNamespace:   "http://other.com",
			targetNamespace: "http://target.com",
			shouldMatch:     true,
		},
		{
			name:            "##other rejects target namespace",
			constraint:      "##other",
			elemNamespace:   "http://target.com",
			targetNamespace: "http://target.com",
			shouldMatch:     false,
		},
		{
			name:            "##targetNamespace allows target",
			constraint:      "##targetNamespace",
			elemNamespace:   "http://target.com",
			targetNamespace: "http://target.com",
			shouldMatch:     true,
		},
		{
			name:            "##targetNamespace rejects other",
			constraint:      "##targetNamespace",
			elemNamespace:   "http://other.com",
			targetNamespace: "http://target.com",
			shouldMatch:     false,
		},
		{
			name:            "##local allows no namespace",
			constraint:      "##local",
			elemNamespace:   "",
			targetNamespace: "http://target.com",
			shouldMatch:     true,
		},
		{
			name:            "##local rejects namespace",
			constraint:      "##local",
			elemNamespace:   "http://other.com",
			targetNamespace: "http://target.com",
			shouldMatch:     false,
		},
		{
			name:            "explicit namespace list",
			constraint:      "http://ns1.com http://ns2.com",
			elemNamespace:   "http://ns1.com",
			targetNamespace: "http://target.com",
			shouldMatch:     true,
		},
		{
			name:            "explicit namespace list rejects unlisted",
			constraint:      "http://ns1.com http://ns2.com",
			elemNamespace:   "http://ns3.com",
			targetNamespace: "http://target.com",
			shouldMatch:     false,
		},
		{
			name:            "list with ##targetNamespace",
			constraint:      "http://ns1.com ##targetNamespace",
			elemNamespace:   "http://target.com",
			targetNamespace: "http://target.com",
			shouldMatch:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraint := ParseNamespaceConstraint(tt.constraint)
			matches := constraint.Matches(tt.elemNamespace, tt.targetNamespace)
			if matches != tt.shouldMatch {
				t.Errorf("Expected matches=%v but got %v for constraint '%s' with namespace '%s'",
					tt.shouldMatch, matches, tt.constraint, tt.elemNamespace)
			}
		})
	}
}

func TestProcessContentsMode(t *testing.T) {
	// Create schemas with different processContents modes
	schemaWithStrict := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com">
		<xs:element name="known" type="xs:string"/>
		<xs:element name="container">
			<xs:complexType>
				<xs:sequence>
					<xs:any namespace="##targetNamespace" processContents="strict"/>
				</xs:sequence>
			</xs:complexType>
		</xs:element>
	</xs:schema>`

	schemaWithLax := strings.Replace(schemaWithStrict, "strict", "lax", 1)
	schemaWithSkip := strings.Replace(schemaWithStrict, "strict", "skip", 1)

	tests := []struct {
		name       string
		schemaXML  string
		xml        string
		wantError  bool
		errorCount int
	}{
		{
			name:      "strict mode - known element valid",
			schemaXML: schemaWithStrict,
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<known>Valid</known>
			</container>`,
			wantError: false,
		},
		{
			name:      "strict mode - unknown element invalid",
			schemaXML: schemaWithStrict,
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<unknown>Invalid</unknown>
			</container>`,
			wantError:  true,
			errorCount: 1, // One for no declaration in strict mode
		},
		{
			name:      "lax mode - known element valid",
			schemaXML: schemaWithLax,
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<known>Valid</known>
			</container>`,
			wantError: false,
		},
		{
			name:      "lax mode - unknown element valid",
			schemaXML: schemaWithLax,
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<unknown>Also Valid in Lax</unknown>
			</container>`,
			wantError: false,
		},
		{
			name:      "skip mode - anything valid",
			schemaXML: schemaWithSkip,
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<container xmlns="http://example.com">
				<anything>Always Valid in Skip</anything>
			</container>`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := xmldom.Decode(bytes.NewReader([]byte(tt.schemaXML)))
			if err != nil {
				t.Fatalf("Failed to parse schema: %v", err)
			}

			schema, err := Parse(doc)
			if err != nil {
				t.Fatalf("Failed to parse schema: %v", err)
			}

			doc, err = xmldom.Decode(bytes.NewReader([]byte(tt.xml)))
			if err != nil {
				t.Fatalf("Failed to parse XML: %v", err)
			}

			validator := NewValidator(schema)
			violations := validator.Validate(doc)

			hasError := len(violations) > 0
			if hasError != tt.wantError {
				t.Errorf("Expected error=%v but got %v violations: %v",
					tt.wantError, len(violations), violations)
			}

			if tt.errorCount > 0 && len(violations) != tt.errorCount {
				t.Errorf("Expected %d errors but got %d: %v",
					tt.errorCount, len(violations), violations)
			}
		})
	}
}

// contains function is defined in identity_constraints_test.go
