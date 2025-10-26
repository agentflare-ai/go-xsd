package xsd

import (
	"strings"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestHeadBlockSubstitution(t *testing.T) {
	schemaDoc := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/bf"
           xmlns:bf="http://example.com/bf"
           elementFormDefault="qualified">

  <xs:complexType name="BaseT">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>

  <xs:complexType name="ChildT">
    <xs:complexContent>
      <xs:extension base="bf:BaseT">
        <xs:sequence>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <!-- Head blocks substitution -->
  <xs:element name="head" type="bf:BaseT" block="substitution"/>
  <xs:element name="child" type="bf:ChildT" substitutionGroup="bf:head"/>

  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="bf:head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	good := `<?xml version="1.0"?>
<root xmlns="http://example.com/bf">
  <head>
    <a>x</a>
  </head>
</root>`

	bad := `<?xml version="1.0"?>
<root xmlns="http://example.com/bf">
  <child>
    <a>x</a>
    <b>y</b>
  </child>
</root>`

	doc, _ := xmldom.Decode(strings.NewReader(schemaDoc))
	schema, err := Parse(doc)
	if err != nil { t.Fatalf("parse schema: %v", err) }
	validator := NewValidator(schema)

	goodDoc, _ := xmldom.Decode(strings.NewReader(good))
	if v := validator.Validate(goodDoc); len(v) != 0 {
		t.Fatalf("expected good to be valid, got %v", v)
	}
	badDoc, _ := xmldom.Decode(strings.NewReader(bad))
	if v := validator.Validate(badDoc); len(v) == 0 {
		t.Fatalf("expected bad to be invalid due to block substitution")
	}
}

func TestFinalExtensionBlocksDerivation(t *testing.T) {
	schemaDoc := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/bf"
           xmlns:bf="http://example.com/bf"
           elementFormDefault="qualified">

  <xs:complexType name="BaseT" final="extension">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>

  <xs:complexType name="ChildT">
    <xs:complexContent>
      <xs:extension base="bf:BaseT">
        <xs:sequence>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:element name="head" type="bf:BaseT"/>
  <xs:element name="child" type="bf:ChildT" substitutionGroup="bf:head"/>

  <xs:element name="root">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="bf:head"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	bad := `<?xml version="1.0"?>
<root xmlns="http://example.com/bf">
  <child>
    <a>x</a>
    <b>y</b>
  </child>
</root>`

	doc, _ := xmldom.Decode(strings.NewReader(schemaDoc))
	schema, err := Parse(doc)
	if err != nil { t.Fatalf("parse schema: %v", err) }
	validator := NewValidator(schema)

	badDoc, _ := xmldom.Decode(strings.NewReader(bad))
	if v := validator.Validate(badDoc); len(v) == 0 {
		t.Fatalf("expected invalid due to final=extension on head type")
	}
}