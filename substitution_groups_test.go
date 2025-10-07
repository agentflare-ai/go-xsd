package xsd

import (
	"strings"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

func TestSubstitutionGroups(t *testing.T) {
	// Define schema with substitution group
	schemaDoc := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/vehicle"
           xmlns:v="http://example.com/vehicle"
           elementFormDefault="qualified">

  <!-- Base element (head of substitution group) -->
  <xs:element name="vehicle" type="v:VehicleType"/>

  <!-- Elements that can substitute for vehicle -->
  <xs:element name="car" type="v:CarType" substitutionGroup="v:vehicle"/>
  <xs:element name="truck" type="v:TruckType" substitutionGroup="v:vehicle"/>
  <xs:element name="motorcycle" type="v:MotorcycleType" substitutionGroup="v:vehicle"/>

  <!-- Types -->
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

  <xs:complexType name="TruckType">
    <xs:complexContent>
      <xs:extension base="v:VehicleType">
        <xs:sequence>
          <xs:element name="payloadCapacity" type="xs:int"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <xs:complexType name="MotorcycleType">
    <xs:complexContent>
      <xs:extension base="v:VehicleType">
        <xs:sequence>
          <xs:element name="engineCC" type="xs:int"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>

  <!-- Container element that expects vehicle -->
  <xs:element name="fleet">
    <xs:complexType>
      <xs:sequence>
        <xs:element ref="v:vehicle" maxOccurs="unbounded"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

	tests := []struct {
		name        string
		xml         string
		expectError bool
		description string
	}{
		{
			name: "valid - base vehicle element",
			xml: `<?xml version="1.0"?>
<fleet xmlns="http://example.com/vehicle">
  <vehicle>
    <brand>Generic</brand>
    <year>2020</year>
  </vehicle>
</fleet>`,
			expectError: false,
			description: "Using the head element directly should work",
		},
		{
			name: "valid - car substitutes for vehicle",
			xml: `<?xml version="1.0"?>
<fleet xmlns="http://example.com/vehicle">
  <car>
    <brand>Toyota</brand>
    <year>2022</year>
    <doors>4</doors>
  </car>
</fleet>`,
			expectError: false,
			description: "Car element should substitute for vehicle",
		},
		{
			name: "valid - truck substitutes for vehicle",
			xml: `<?xml version="1.0"?>
<fleet xmlns="http://example.com/vehicle">
  <truck>
    <brand>Ford</brand>
    <year>2021</year>
    <payloadCapacity>2000</payloadCapacity>
  </truck>
</fleet>`,
			expectError: false,
			description: "Truck element should substitute for vehicle",
		},
		{
			name: "valid - mixed substitution group members",
			xml: `<?xml version="1.0"?>
<fleet xmlns="http://example.com/vehicle">
  <vehicle>
    <brand>Generic</brand>
    <year>2020</year>
  </vehicle>
  <car>
    <brand>Honda</brand>
    <year>2023</year>
    <doors>2</doors>
  </car>
  <truck>
    <brand>Chevy</brand>
    <year>2022</year>
    <payloadCapacity>3000</payloadCapacity>
  </truck>
  <motorcycle>
    <brand>Harley</brand>
    <year>2021</year>
    <engineCC>1200</engineCC>
  </motorcycle>
</fleet>`,
			expectError: false,
			description: "Multiple different substitution group members should work together",
		},
	}

	// Parse schema
	schemaDocParsed, err := xmldom.Decode(strings.NewReader(schemaDoc))
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	schema, err := Parse(schemaDocParsed)
	if err != nil {
		t.Fatalf("Failed to parse XSD schema: %v", err)
	}

	// Verify substitution groups were built
	vehicleQName := QName{Namespace: "http://example.com/vehicle", Local: "vehicle"}
	if members, exists := schema.SubstitutionGroups[vehicleQName]; exists {
		if len(members) != 3 {
			t.Errorf("Expected 3 substitution group members for vehicle, got %d", len(members))
		}
		t.Logf("Substitution group for 'vehicle': %v", members)
	} else {
		t.Error("Substitution group for 'vehicle' was not built")
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse instance document
			instanceDoc, err := xmldom.Decode(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Failed to parse instance document: %v", err)
			}

			// Validate
			validator := NewValidator(schema)
			violations := validator.Validate(instanceDoc)

			hasError := len(violations) > 0
			if hasError != tt.expectError {
				t.Errorf("Expected error=%v but got %d violations: %v",
					tt.expectError, len(violations), violations)
				for _, v := range violations {
					t.Logf("  - %s: %s", v.Code, v.Message)
				}
			}

			if !tt.expectError && len(violations) == 0 {
				t.Logf("✓ %s", tt.description)
			}
		})
	}
}

func TestSubstitutionGroupRegistry(t *testing.T) {
	schemaDoc := `<?xml version="1.0"?>
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
           targetNamespace="http://example.com/test"
           xmlns:t="http://example.com/test"
           elementFormDefault="qualified">

  <xs:element name="base" type="xs:string"/>
  <xs:element name="derived1" type="xs:string" substitutionGroup="t:base"/>
  <xs:element name="derived2" type="xs:string" substitutionGroup="t:base"/>
  <xs:element name="derived3" type="xs:string" substitutionGroup="t:derived1"/>
</xs:schema>`

	schemaDocParsed, err := xmldom.Decode(strings.NewReader(schemaDoc))
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	schema, err := Parse(schemaDocParsed)
	if err != nil {
		t.Fatalf("Failed to parse XSD schema: %v", err)
	}

	// Test that substitution groups were registered
	baseQName := QName{Namespace: "http://example.com/test", Local: "base"}
	members, exists := schema.SubstitutionGroups[baseQName]
	if !exists {
		t.Fatal("Substitution group for 'base' was not registered")
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 direct members for 'base', got %d: %v", len(members), members)
	}

	// Test that derived1 also has a substitution group
	derived1QName := QName{Namespace: "http://example.com/test", Local: "derived1"}
	members2, exists2 := schema.SubstitutionGroups[derived1QName]
	if !exists2 {
		t.Fatal("Substitution group for 'derived1' was not registered")
	}

	if len(members2) != 1 {
		t.Errorf("Expected 1 member for 'derived1', got %d", len(members2))
	}
}
