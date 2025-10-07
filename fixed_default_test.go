package xsd

import (
	"bytes"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestElementFixedValues(t *testing.T) {
	// Schema with fixed element values
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="config">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="version" type="xs:string" fixed="1.0"/>
					<xs:element name="mode" type="xs:string" fixed="production"/>
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
			name: "valid - correct fixed values",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<config xmlns="http://example.com">
				<version>1.0</version>
				<mode>production</mode>
			</config>`,
			wantError: false,
		},
		{
			name: "invalid - wrong fixed value for version",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<config xmlns="http://example.com">
				<version>2.0</version>
				<mode>production</mode>
			</config>`,
			wantError: true,
			errorMsg:  "must have fixed value '1.0' but has '2.0'",
		},
		{
			name: "invalid - wrong fixed value for mode",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<config xmlns="http://example.com">
				<version>1.0</version>
				<mode>development</mode>
			</config>`,
			wantError: true,
			errorMsg:  "must have fixed value 'production' but has 'development'",
		},
		{
			name: "empty elements - should match fixed",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<config xmlns="http://example.com">
				<version></version>
				<mode></mode>
			</config>`,
			wantError: true,
			errorMsg:  "must have fixed value",
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

func TestElementDefaultValues(t *testing.T) {
	// Schema with default element values
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="settings">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="timeout" type="xs:integer" default="30"/>
					<xs:element name="retries" type="xs:integer" default="3"/>
					<xs:element name="debug" type="xs:boolean" default="false"/>
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
			name: "valid - explicit values",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<settings xmlns="http://example.com">
				<timeout>60</timeout>
				<retries>5</retries>
				<debug>true</debug>
			</settings>`,
			wantError: false,
		},
		{
			name: "valid - empty elements use defaults",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<settings xmlns="http://example.com">
				<timeout></timeout>
				<retries></retries>
				<debug></debug>
			</settings>`,
			wantError: false, // Empty elements should be validated with default values
		},
		{
			name: "valid - mixed explicit and default",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<settings xmlns="http://example.com">
				<timeout>45</timeout>
				<retries></retries>
				<debug>true</debug>
			</settings>`,
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

func TestAttributeFixedValues(t *testing.T) {
	// Schema with fixed attribute values
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="product">
			<xs:complexType>
				<xs:simpleContent>
					<xs:extension base="xs:string">
						<xs:attribute name="currency" type="xs:string" fixed="USD" use="optional"/>
						<xs:attribute name="version" type="xs:string" fixed="2.0" use="required"/>
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
			name: "valid - correct fixed values",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<product xmlns="http://example.com" currency="USD" version="2.0">Widget</product>`,
			wantError: false,
		},
		{
			name: "valid - missing optional fixed attribute",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<product xmlns="http://example.com" version="2.0">Widget</product>`,
			wantError: false, // Optional fixed attribute can be omitted
		},
		{
			name: "invalid - wrong fixed value for currency",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<product xmlns="http://example.com" currency="EUR" version="2.0">Widget</product>`,
			wantError: true,
			errorMsg:  "must have fixed value 'USD' but has 'EUR'",
		},
		{
			name: "invalid - wrong fixed value for version",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<product xmlns="http://example.com" currency="USD" version="1.0">Widget</product>`,
			wantError: true,
			errorMsg:  "must have fixed value '2.0' but has '1.0'",
		},
		{
			name: "invalid - missing required fixed attribute",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<product xmlns="http://example.com" currency="USD">Widget</product>`,
			wantError: true,
			errorMsg:  "Required attribute 'version' is missing",
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

func TestAttributeDefaultValues(t *testing.T) {
	// Schema with default attribute values
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="item">
			<xs:complexType>
				<xs:simpleContent>
					<xs:extension base="xs:string">
						<xs:attribute name="quantity" type="xs:integer" default="1"/>
						<xs:attribute name="taxable" type="xs:boolean" default="true"/>
						<xs:attribute name="category" type="xs:string" default="general"/>
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
			name: "valid - explicit values",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<item xmlns="http://example.com" quantity="5" taxable="false" category="electronics">Laptop</item>`,
			wantError: false,
		},
		{
			name: "valid - missing attributes use defaults",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<item xmlns="http://example.com">Book</item>`,
			wantError: false, // Missing attributes should use default values
		},
		{
			name: "valid - partial attributes",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<item xmlns="http://example.com" quantity="2">Pen</item>`,
			wantError: false,
		},
		{
			name: "valid - override one default",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<item xmlns="http://example.com" category="office">Stapler</item>`,
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

func TestFixedAndDefaultCombination(t *testing.T) {
	// Schema with both fixed and default values
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="system">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="name" type="xs:string"/>
					<xs:element name="version" type="xs:string" fixed="1.0.0"/>
					<xs:element name="environment" type="xs:string" default="development"/>
				</xs:sequence>
				<xs:attribute name="id" type="xs:string" use="required"/>
				<xs:attribute name="type" type="xs:string" fixed="standard"/>
				<xs:attribute name="priority" type="xs:integer" default="5"/>
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
			name: "valid - all correct",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<system xmlns="http://example.com" id="sys1" type="standard" priority="10">
				<name>TestSystem</name>
				<version>1.0.0</version>
				<environment>production</environment>
			</system>`,
			wantError: false,
		},
		{
			name: "valid - using defaults",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<system xmlns="http://example.com" id="sys2" type="standard">
				<name>TestSystem</name>
				<version>1.0.0</version>
				<environment></environment>
			</system>`,
			wantError: false,
		},
		{
			name: "invalid - wrong fixed element value",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<system xmlns="http://example.com" id="sys3" type="standard">
				<name>TestSystem</name>
				<version>2.0.0</version>
				<environment>staging</environment>
			</system>`,
			wantError: true,
			errorMsg:  "must have fixed value '1.0.0' but has '2.0.0'",
		},
		{
			name: "invalid - wrong fixed attribute value",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<system xmlns="http://example.com" id="sys4" type="premium">
				<name>TestSystem</name>
				<version>1.0.0</version>
				<environment>production</environment>
			</system>`,
			wantError: true,
			errorMsg:  "must have fixed value 'standard' but has 'premium'",
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
