package xsd

import (
	"bytes"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestUniqueConstraint(t *testing.T) {
	// Create a schema with a unique constraint on employee ID
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="company">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="employee" maxOccurs="unbounded">
						<xs:complexType>
							<xs:sequence>
								<xs:element name="name" type="xs:string"/>
								<xs:element name="department" type="xs:string"/>
							</xs:sequence>
							<xs:attribute name="id" type="xs:string" use="required"/>
						</xs:complexType>
					</xs:element>
				</xs:sequence>
			</xs:complexType>
			<xs:unique name="uniqueEmployeeId">
				<xs:selector xpath=".//ex:employee"/>
				<xs:field xpath="@id"/>
			</xs:unique>
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
	}{
		{
			name: "unique IDs - valid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<company xmlns="http://example.com">
				<employee id="emp001">
					<name>John Doe</name>
					<department>Engineering</department>
				</employee>
				<employee id="emp002">
					<name>Jane Smith</name>
					<department>Marketing</department>
				</employee>
			</company>`,
			wantError: false,
		},
		{
			name: "duplicate IDs - invalid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<company xmlns="http://example.com">
				<employee id="emp001">
					<name>John Doe</name>
					<department>Engineering</department>
				</employee>
				<employee id="emp001">
					<name>Jane Smith</name>
					<department>Marketing</department>
				</employee>
			</company>`,
			wantError: true,
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
		})
	}
}

func TestKeyConstraint(t *testing.T) {
	// Create a schema with a key constraint
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="database">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="table" maxOccurs="unbounded">
						<xs:complexType>
							<xs:sequence>
								<xs:element name="column" maxOccurs="unbounded">
									<xs:complexType>
										<xs:simpleContent>
											<xs:extension base="xs:string">
												<xs:attribute name="name" type="xs:string" use="required"/>
											</xs:extension>
										</xs:simpleContent>
									</xs:complexType>
								</xs:element>
							</xs:sequence>
							<xs:attribute name="name" type="xs:string" use="required"/>
						</xs:complexType>
					</xs:element>
				</xs:sequence>
			</xs:complexType>
			<xs:key name="tableKey">
				<xs:selector xpath=".//ex:table"/>
				<xs:field xpath="@name"/>
			</xs:key>
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
			name: "unique table names - valid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<database xmlns="http://example.com">
				<table name="users">
					<column name="id">INTEGER</column>
					<column name="name">VARCHAR</column>
				</table>
				<table name="orders">
					<column name="id">INTEGER</column>
					<column name="user_id">INTEGER</column>
				</table>
			</database>`,
			wantError: false,
		},
		{
			name: "duplicate table names - invalid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<database xmlns="http://example.com">
				<table name="users">
					<column name="id">INTEGER</column>
				</table>
				<table name="users">
					<column name="id">INTEGER</column>
				</table>
			</database>`,
			wantError: true,
			errorMsg:  "Duplicate key",
		},
		{
			name: "null key field - invalid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<database xmlns="http://example.com">
				<table>
					<column name="id">INTEGER</column>
				</table>
			</database>`,
			wantError: true,
			errorMsg:  "cannot be null",
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

func TestKeyRefConstraint(t *testing.T) {
	// Create a schema with key and keyref constraints
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="library">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="books">
						<xs:complexType>
							<xs:sequence>
								<xs:element name="book" maxOccurs="unbounded">
									<xs:complexType>
										<xs:sequence>
											<xs:element name="title" type="xs:string"/>
										</xs:sequence>
										<xs:attribute name="isbn" type="xs:string" use="required"/>
									</xs:complexType>
								</xs:element>
							</xs:sequence>
						</xs:complexType>
					</xs:element>
					<xs:element name="loans">
						<xs:complexType>
							<xs:sequence>
								<xs:element name="loan" maxOccurs="unbounded">
									<xs:complexType>
										<xs:sequence>
											<xs:element name="borrower" type="xs:string"/>
										</xs:sequence>
										<xs:attribute name="book" type="xs:string" use="required"/>
									</xs:complexType>
								</xs:element>
							</xs:sequence>
						</xs:complexType>
					</xs:element>
				</xs:sequence>
			</xs:complexType>
			<xs:key name="bookKey">
				<xs:selector xpath=".//ex:book"/>
				<xs:field xpath="@isbn"/>
			</xs:key>
			<xs:keyref name="bookRef" refer="ex:bookKey">
				<xs:selector xpath=".//ex:loan"/>
				<xs:field xpath="@book"/>
			</xs:keyref>
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
			name: "valid references",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<library xmlns="http://example.com">
				<books>
					<book isbn="978-0-123456-47-2">
						<title>XML Schema Guide</title>
					</book>
					<book isbn="978-0-234567-58-3">
						<title>Web Services</title>
					</book>
				</books>
				<loans>
					<loan book="978-0-123456-47-2">
						<borrower>John Doe</borrower>
					</loan>
					<loan book="978-0-234567-58-3">
						<borrower>Jane Smith</borrower>
					</loan>
				</loans>
			</library>`,
			wantError: false,
		},
		{
			name: "invalid reference - non-existent book",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<library xmlns="http://example.com">
				<books>
					<book isbn="978-0-123456-47-2">
						<title>XML Schema Guide</title>
					</book>
				</books>
				<loans>
					<loan book="978-0-999999-99-9">
						<borrower>John Doe</borrower>
					</loan>
				</loans>
			</library>`,
			wantError: true,
			errorMsg:  "does not match",
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

func TestComplexIdentityConstraints(t *testing.T) {
	// Test with multiple fields in constraints
	schemaXML := `<?xml version="1.0" encoding="UTF-8"?>
	<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" 
	           targetNamespace="http://example.com" 
	           xmlns:ex="http://example.com">
		<xs:element name="school">
			<xs:complexType>
				<xs:sequence>
					<xs:element name="class" maxOccurs="unbounded">
						<xs:complexType>
							<xs:sequence>
								<xs:element name="student" maxOccurs="unbounded">
									<xs:complexType>
										<xs:sequence>
											<xs:element name="name" type="xs:string"/>
										</xs:sequence>
										<xs:attribute name="id" type="xs:string" use="required"/>
									</xs:complexType>
								</xs:element>
							</xs:sequence>
							<xs:attribute name="grade" type="xs:integer" use="required"/>
							<xs:attribute name="section" type="xs:string" use="required"/>
						</xs:complexType>
					</xs:element>
				</xs:sequence>
			</xs:complexType>
			<xs:unique name="uniqueClass">
				<xs:selector xpath=".//ex:class"/>
				<xs:field xpath="@grade"/>
				<xs:field xpath="@section"/>
			</xs:unique>
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
	}{
		{
			name: "unique grade-section combination - valid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<school xmlns="http://example.com">
				<class grade="1" section="A">
					<student id="s1"><name>Alice</name></student>
				</class>
				<class grade="1" section="B">
					<student id="s2"><name>Bob</name></student>
				</class>
				<class grade="2" section="A">
					<student id="s3"><name>Charlie</name></student>
				</class>
			</school>`,
			wantError: false,
		},
		{
			name: "duplicate grade-section combination - invalid",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
			<school xmlns="http://example.com">
				<class grade="1" section="A">
					<student id="s1"><name>Alice</name></student>
				</class>
				<class grade="1" section="A">
					<student id="s2"><name>Bob</name></student>
				</class>
			</school>`,
			wantError: true,
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
		})
	}
}

// Helper function
func contains(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 &&
		bytes.Contains([]byte(str), []byte(substr))
}
