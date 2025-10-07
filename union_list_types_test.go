package xsd

import (
	"bytes"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestUnionTypes(t *testing.T) {
	// Schema with union types
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		
		<!-- Union of integer and string -->
		<xs:simpleType name="intOrString">
			<xs:union memberTypes="xs:integer xs:string"/>
		</xs:simpleType>
		
		<!-- Union of boolean and specific string values -->
		<xs:simpleType name="boolOrStatus">
			<xs:union>
				<xs:simpleType>
					<xs:restriction base="xs:boolean"/>
				</xs:simpleType>
				<xs:simpleType>
					<xs:restriction base="xs:string">
						<xs:enumeration value="pending"/>
						<xs:enumeration value="unknown"/>
					</xs:restriction>
				</xs:simpleType>
			</xs:union>
		</xs:simpleType>
		
		<!-- Union of date and dateTime -->
		<xs:simpleType name="dateOrDateTime">
			<xs:union memberTypes="xs:date xs:dateTime"/>
		</xs:simpleType>
		
		<xs:element name="data">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="value" type="ex:intOrString"/>
					<xs:element name="status" type="ex:boolOrStatus"/>
					<xs:element name="timestamp" type="ex:dateOrDateTime"/>
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
			name: "valid - integer value in union",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<data xmlns="http://example.com">
				<value>42</value>
				<status>true</status>
				<timestamp>2024-01-15</timestamp>
			</data>`,
			wantError: false,
		},
		{
			name: "valid - string value in union",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<data xmlns="http://example.com">
				<value>hello world</value>
				<status>false</status>
				<timestamp>2024-01-15T10:30:00</timestamp>
			</data>`,
			wantError: false,
		},
		{
			name: "valid - enum string in boolOrStatus",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<data xmlns="http://example.com">
				<value>123</value>
				<status>pending</status>
				<timestamp>2024-01-15</timestamp>
			</data>`,
			wantError: false,
		},
		{
			name: "valid - different union member types",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<data xmlns="http://example.com">
				<value>test string</value>
				<status>unknown</status>
				<timestamp>2024-01-15T14:30:00Z</timestamp>
			</data>`,
			wantError: false,
		},
		{
			name: "invalid - status not in union",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<data xmlns="http://example.com">
				<value>42</value>
				<status>invalid</status>
				<timestamp>2024-01-15</timestamp>
			</data>`,
			wantError: true,
			errorMsg:  "not valid against any member type",
		},
		{
			name: "invalid - bad date format",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<data xmlns="http://example.com">
				<value>42</value>
				<status>true</status>
				<timestamp>not-a-date</timestamp>
			</data>`,
			wantError: true,
			errorMsg:  "not valid against any member type",
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

func TestListTypes(t *testing.T) {
	// Schema with list types
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		
		<!-- List of integers -->
		<xs:simpleType name="integerList">
			<xs:list itemType="xs:integer"/>
		</xs:simpleType>
		
		<!-- List of tokens -->
		<xs:simpleType name="tokenList">
			<xs:list itemType="xs:token"/>
		</xs:simpleType>
		
		<!-- List of dates -->
		<xs:simpleType name="dateList">
			<xs:list itemType="xs:date"/>
		</xs:simpleType>
		
		<!-- List with length restriction -->
		<xs:simpleType name="limitedIntList">
			<xs:restriction>
				<xs:simpleType>
					<xs:list itemType="xs:integer"/>
				</xs:simpleType>
				<xs:length value="3"/>
			</xs:restriction>
		</xs:simpleType>
		
		<xs:element name="lists">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="numbers" type="ex:integerList"/>
					<xs:element name="tokens" type="ex:tokenList"/>
					<xs:element name="dates" type="ex:dateList"/>
					<xs:element name="triple" type="ex:limitedIntList" minOccurs="0"/>
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
			name: "valid - single item lists",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers>42</numbers>
				<tokens>hello</tokens>
				<dates>2024-01-15</dates>
			</lists>`,
			wantError: false,
		},
		{
			name: "valid - multiple item lists",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers>1 2 3 4 5</numbers>
				<tokens>hello world test</tokens>
				<dates>2024-01-15 2024-02-20 2024-03-25</dates>
			</lists>`,
			wantError: false,
		},
		{
			name: "valid - empty lists",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers></numbers>
				<tokens></tokens>
				<dates></dates>
			</lists>`,
			wantError: false,
		},
		{
			name: "valid - list with exact length",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers>10 20</numbers>
				<tokens>a b</tokens>
				<dates>2024-01-01 2024-12-31</dates>
				<triple>100 200 300</triple>
			</lists>`,
			wantError: false,
		},
		{
			name: "invalid - non-integer in integer list",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers>1 2 abc 4</numbers>
				<tokens>hello</tokens>
				<dates>2024-01-15</dates>
			</lists>`,
			wantError: true,
			errorMsg:  "list item",
		},
		{
			name: "invalid - bad date in date list",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers>1 2 3</numbers>
				<tokens>hello</tokens>
				<dates>2024-01-15 not-a-date 2024-03-25</dates>
			</lists>`,
			wantError: true,
			errorMsg:  "list item",
		},
		{
			name: "invalid - wrong length for limited list",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<lists xmlns="http://example.com">
				<numbers>1 2 3</numbers>
				<tokens>hello</tokens>
				<dates>2024-01-15</dates>
				<triple>100 200</triple>
			</lists>`,
			wantError: true,
			errorMsg:  "length",
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

func TestComplexUnionList(t *testing.T) {
	// Schema with complex union and list combinations
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		
		<!-- List of unions -->
		<xs:simpleType name="flexibleList">
			<xs:list>
				<xs:simpleType>
					<xs:union memberTypes="xs:integer xs:boolean xs:token"/>
				</xs:simpleType>
			</xs:list>
		</xs:simpleType>
		
		<!-- Union containing a list -->
		<xs:simpleType name="singleOrList">
			<xs:union>
				<xs:simpleType>
					<xs:restriction base="xs:integer"/>
				</xs:simpleType>
				<xs:simpleType>
					<xs:list itemType="xs:integer"/>
				</xs:simpleType>
			</xs:union>
		</xs:simpleType>
		
		<xs:element name="complex">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="flexible" type="ex:flexibleList"/>
					<xs:element name="numbers" type="ex:singleOrList"/>
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
			name: "valid - mixed types in flexible list",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<complex xmlns="http://example.com">
				<flexible>42 true hello false 100</flexible>
				<numbers>42</numbers>
			</complex>`,
			wantError: false,
		},
		{
			name: "valid - list in union",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<complex xmlns="http://example.com">
				<flexible>1 2 3</flexible>
				<numbers>10 20 30</numbers>
			</complex>`,
			wantError: false,
		},
		{
			name: "valid - single value where list allowed",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<complex xmlns="http://example.com">
				<flexible>test</flexible>
				<numbers>999</numbers>
			</complex>`,
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
