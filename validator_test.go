package xsd

import (
	"bytes"
	"testing"

	"github.com/agentflare-ai/go-xmldom"
)

// Test attribute validation
func TestAttributeValidation(t *testing.T) {
	// Create a simple schema with specific attribute requirements
	schema := &Schema{
		TargetNamespace: "http://test.com",
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
	}

	// Define element with required and optional attributes
	elemType := &ComplexType{
		QName: QName{Namespace: schema.TargetNamespace, Local: "test"},
		Attributes: []*AttributeDecl{
			{Name: QName{Local: "required"}, Use: RequiredUse},
			{Name: QName{Local: "optional"}, Use: OptionalUse},
		},
		Content: &AllowAnyContent{},
	}
	schema.ElementDecls[elemType.QName] = &ElementDecl{
		Name: elemType.QName,
		Type: elemType,
	}
	schema.TypeDefs[elemType.QName] = elemType

	tests := []struct {
		name      string
		xml       string
		wantError bool
		errorCode string
	}{
		{
			name:      "valid with all attributes",
			xml:       `<test xmlns="http://test.com" required="val" optional="val2"/>`,
			wantError: false,
		},
		{
			name:      "valid with only required",
			xml:       `<test xmlns="http://test.com" required="val"/>`,
			wantError: false,
		},
		{
			name:      "missing required attribute",
			xml:       `<test xmlns="http://test.com" optional="val"/>`,
			wantError: true,
			errorCode: "cvc-complex-type.4",
		},
		{
			name:      "unknown attribute",
			xml:       `<test xmlns="http://test.com" required="val" unknown="val"/>`,
			wantError: true,
			errorCode: "cvc-complex-type.3.2.2",
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

			if tt.wantError {
				if len(violations) == 0 {
					t.Errorf("Expected validation error but got none")
				} else if tt.errorCode != "" && violations[0].Code != tt.errorCode {
					t.Errorf("Expected error code %s but got %s", tt.errorCode, violations[0].Code)
				}
			} else {
				if len(violations) > 0 {
					t.Errorf("Expected no validation errors but got: %v", violations)
				}
			}
		})
	}
}

// Test ID/IDREF validation
func TestIDREFValidation(t *testing.T) {
	schema := &Schema{
		TargetNamespace: "http://test.com",
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
	}

	// Define elements with ID and IDREF attributes
	elemType := &ComplexType{
		QName:   QName{Namespace: schema.TargetNamespace, Local: "root"},
		Content: &AllowAnyContent{},
	}
	schema.ElementDecls[elemType.QName] = &ElementDecl{
		Name: elemType.QName,
		Type: elemType,
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
	}{
		{
			name: "valid IDREF",
			xml: `<root xmlns="http://test.com">
				<elem id="id1"/>
				<ref target="id1"/>
			</root>`,
			wantError: false,
		},
		{
			name: "invalid IDREF",
			xml: `<root xmlns="http://test.com">
				<elem id="id1"/>
				<ref target="id2"/>
			</root>`,
			wantError: true,
		},
		{
			name: "duplicate ID",
			xml: `<root xmlns="http://test.com">
				<elem id="id1"/>
				<elem id="id1"/>
			</root>`,
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

// Test content model validation (sequence)
func TestSequenceValidation(t *testing.T) {
	schema := &Schema{
		TargetNamespace: "http://test.com",
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
	}

	// Define a sequence content model: must have A then B
	seqType := &ComplexType{
		QName: QName{Namespace: schema.TargetNamespace, Local: "seq"},
		Content: &ModelGroup{
			Kind: SequenceGroup,
			Particles: []Particle{
				&ElementRef{Ref: QName{Namespace: schema.TargetNamespace, Local: "A"}, MinOcc: 1, MaxOcc: 1},
				&ElementRef{Ref: QName{Namespace: schema.TargetNamespace, Local: "B"}, MinOcc: 1, MaxOcc: 1},
			},
			MinOcc: 1,
			MaxOcc: 1,
		},
	}
	schema.ElementDecls[seqType.QName] = &ElementDecl{
		Name: seqType.QName,
		Type: seqType,
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
	}{
		{
			name:      "valid sequence",
			xml:       `<seq xmlns="http://test.com"><A/><B/></seq>`,
			wantError: false,
		},
		{
			name:      "wrong order",
			xml:       `<seq xmlns="http://test.com"><B/><A/></seq>`,
			wantError: true,
		},
		{
			name:      "missing element",
			xml:       `<seq xmlns="http://test.com"><A/></seq>`,
			wantError: true,
		},
		{
			name:      "extra element",
			xml:       `<seq xmlns="http://test.com"><A/><B/><C/></seq>`,
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

// Test choice validation
func TestChoiceValidation(t *testing.T) {
	schema := &Schema{
		TargetNamespace: "http://test.com",
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
	}

	// Define a choice content model: must have either A or B
	choiceType := &ComplexType{
		QName: QName{Namespace: schema.TargetNamespace, Local: "choice"},
		Content: &ModelGroup{
			Kind: ChoiceGroup,
			Particles: []Particle{
				&ElementRef{Ref: QName{Namespace: schema.TargetNamespace, Local: "A"}, MinOcc: 1, MaxOcc: 1},
				&ElementRef{Ref: QName{Namespace: schema.TargetNamespace, Local: "B"}, MinOcc: 1, MaxOcc: 1},
			},
			MinOcc: 1,
			MaxOcc: 1,
		},
	}
	schema.ElementDecls[choiceType.QName] = &ElementDecl{
		Name: choiceType.QName,
		Type: choiceType,
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
	}{
		{
			name:      "valid choice A",
			xml:       `<choice xmlns="http://test.com"><A/></choice>`,
			wantError: false,
		},
		{
			name:      "valid choice B",
			xml:       `<choice xmlns="http://test.com"><B/></choice>`,
			wantError: false,
		},
		{
			name:      "both elements",
			xml:       `<choice xmlns="http://test.com"><A/><B/></choice>`,
			wantError: true,
		},
		{
			name:      "neither element",
			xml:       `<choice xmlns="http://test.com"></choice>`,
			wantError: true,
		},
		{
			name:      "wrong element",
			xml:       `<choice xmlns="http://test.com"><C/></choice>`,
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

// Test occurrence constraints
func TestOccurrenceValidation(t *testing.T) {
	schema := &Schema{
		TargetNamespace: "http://test.com",
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
	}

	// Define element with occurrence constraints
	occType := &ComplexType{
		QName: QName{Namespace: schema.TargetNamespace, Local: "occ"},
		Content: &ModelGroup{
			Kind: SequenceGroup,
			Particles: []Particle{
				&ElementRef{Ref: QName{Namespace: schema.TargetNamespace, Local: "A"}, MinOcc: 2, MaxOcc: 4},
				&ElementRef{Ref: QName{Namespace: schema.TargetNamespace, Local: "B"}, MinOcc: 0, MaxOcc: -1}, // unbounded
			},
			MinOcc: 1,
			MaxOcc: 1,
		},
	}
	schema.ElementDecls[occType.QName] = &ElementDecl{
		Name: occType.QName,
		Type: occType,
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
	}{
		{
			name:      "minimum A elements",
			xml:       `<occ xmlns="http://test.com"><A/><A/></occ>`,
			wantError: false,
		},
		{
			name:      "maximum A elements",
			xml:       `<occ xmlns="http://test.com"><A/><A/><A/><A/></occ>`,
			wantError: false,
		},
		{
			name:      "too few A elements",
			xml:       `<occ xmlns="http://test.com"><A/></occ>`,
			wantError: true,
		},
		{
			name:      "too many A elements",
			xml:       `<occ xmlns="http://test.com"><A/><A/><A/><A/><A/></occ>`,
			wantError: true,
		},
		{
			name:      "unbounded B elements",
			xml:       `<occ xmlns="http://test.com"><A/><A/><B/><B/><B/><B/><B/></occ>`,
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
		})
	}
}

// Test mixed content validation
func TestMixedContentValidation(t *testing.T) {
	schema := &Schema{
		TargetNamespace: "http://test.com",
		ElementDecls:    make(map[QName]*ElementDecl),
		TypeDefs:        make(map[QName]Type),
	}

	// Define elements with and without mixed content
	mixedType := &ComplexType{
		QName:   QName{Namespace: schema.TargetNamespace, Local: "mixed"},
		Mixed:   true,
		Content: &AllowAnyContent{},
	}
	schema.ElementDecls[mixedType.QName] = &ElementDecl{
		Name: mixedType.QName,
		Type: mixedType,
	}

	notMixedType := &ComplexType{
		QName:   QName{Namespace: schema.TargetNamespace, Local: "notmixed"},
		Mixed:   false,
		Content: &AllowAnyContent{},
	}
	schema.ElementDecls[notMixedType.QName] = &ElementDecl{
		Name: notMixedType.QName,
		Type: notMixedType,
	}

	tests := []struct {
		name      string
		xml       string
		wantError bool
	}{
		{
			name:      "mixed content allowed",
			xml:       `<mixed xmlns="http://test.com">Text <child/> more text</mixed>`,
			wantError: false,
		},
		{
			name:      "no mixed content allowed - empty",
			xml:       `<notmixed xmlns="http://test.com"></notmixed>`,
			wantError: false,
		},
		{
			name:      "no mixed content allowed - whitespace only",
			xml:       `<notmixed xmlns="http://test.com">   </notmixed>`,
			wantError: false,
		},
		{
			name:      "no mixed content allowed - text present",
			xml:       `<notmixed xmlns="http://test.com">Some text</notmixed>`,
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
