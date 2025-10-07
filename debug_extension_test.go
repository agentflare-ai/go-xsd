package xsd

import (
	"strings"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestExtensionParsing(t *testing.T) {
	schemaDoc := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/vehicle"
           xmlns:v="http://example.com/vehicle">

  <xs:complexType name="VehicleType">
    <xs:sequence>
      <xs:element name="brand" type="xs:string"/>
      <xs:element name="year" type="xs:int"/>
    </xs:sequence>
  </xs:complexType>

  <xs:complexType name="CarType">
    <xs:complexContent>
      <xs:extension base="v:VehicleType">
        <xs:sequence>
          <xs:element name="doors" type="xs:int"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:element name="car" type="v:CarType"/>
</xs:schema>`

	schemaDocParsed, err := xmldom.Decode(strings.NewReader(schemaDoc))
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	schema, err := Parse(schemaDocParsed)
	if err != nil {
		t.Fatalf("Failed to parse XSD schema: %v", err)
	}

	// Check if CarType was parsed
	carTypeQName := QName{Namespace: "http://example.com/vehicle", Local: "CarType"}
	carType, exists := schema.TypeDefs[carTypeQName]
	if !exists {
		t.Fatal("CarType not found in TypeDefs")
	}

	t.Logf("CarType found: %T", carType)

	if ct, ok := carType.(*ComplexType); ok {
		t.Logf("CarType.Content: %T", ct.Content)

		if cc, ok := ct.Content.(*ComplexContent); ok {
			t.Logf("ComplexContent found")
			if cc.Extension != nil {
				t.Logf("Extension base: %v", cc.Extension.Base)
				t.Logf("Extension.Content: %T", cc.Extension.Content)

				if mg, ok := cc.Extension.Content.(*ModelGroup); ok {
					t.Logf("Extension has ModelGroup with %d particles", len(mg.Particles))
					for i, p := range mg.Particles {
						t.Logf("  Particle %d: %T", i, p)
					}
				}
			}
		}
	}

	// Test validation
	instanceDoc := `<?xml version="1.0"?>
<car xmlns="http://example.com/vehicle">
  <brand>Toyota</brand>
  <year>2022</year>
  <doors>4</doors>
</car>`

	instance, err := xmldom.Decode(strings.NewReader(instanceDoc))
	if err != nil {
		t.Fatalf("Failed to parse instance: %v", err)
	}

	validator := NewValidator(schema)
	violations := validator.Validate(instance)

	if len(violations) > 0 {
		t.Errorf("Validation failed with %d violations:", len(violations))
		for _, v := range violations {
			t.Logf("  - %s: %s", v.Code, v.Message)
		}
	} else {
		t.Log("✓ Validation passed")
	}
}
