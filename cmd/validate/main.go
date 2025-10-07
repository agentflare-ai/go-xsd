package main

import (
	"fmt"
	"log"
	"os"

	"github.com/agentflare-ai/go-xmldom"
	"github.com/agentflare-ai/go-xsd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: validate <xml-file> [xsd-file]")
		os.Exit(1)
	}

	xmlFile := os.Args[1]
	xsdFile := ""
	if len(os.Args) > 2 {
		xsdFile = os.Args[2]
	} else {
		// Default to SCXML schema
		xsdFile = "platform/xsd/scxml.xsd"
	}

	// Read XML file
	xmlData, err := os.ReadFile(xmlFile)
	if err != nil {
		log.Fatalf("Failed to read XML file: %v", err)
	}

	// Parse XML document
	decoder := xmldom.NewDecoderFromBytes(xmlData)
	doc, err := decoder.Decode()
	if err != nil {
		log.Fatalf("Failed to parse XML: %v", err)
	}

	// Load XSD schema
	cache := xsd.NewSchemaCache("")
	schema, err := cache.Get(xsdFile)
	if err != nil {
		// For testing, create a mock schema with basic SCXML structure
		fmt.Printf("Warning: Could not load XSD schema from %s: %v\n", xsdFile, err)
		fmt.Println("Using mock SCXML schema for testing...")
		schema = createMockSCXMLSchema()
	}

	// Debug: show what elements are in the schema
	fmt.Println("Debug: Elements in schema:")
	for name := range schema.ElementDecls {
		fmt.Printf("  - %s\n", name.Local)
	}
	fmt.Println()

	// Validate document
	validator := xsd.NewValidator(schema)
	violations := validator.Validate(doc)

	// Convert to diagnostics
	converter := xsd.NewDiagnosticConverter(xmlFile, string(xmlData))
	diagnostics := converter.Convert(violations)

	// Print results
	if len(diagnostics) == 0 {
		fmt.Printf("✅ %s is valid!\n", xmlFile)
		os.Exit(0)
	}

	// Format and print errors
	formatter := &xsd.ErrorFormatter{
		Color:           true,
		ShowFullElement: false,
		ContextLines:    2,
	}

	fmt.Printf("Found %d validation issues in %s:\n\n", len(diagnostics), xmlFile)
	for _, diag := range diagnostics {
		fmt.Print(formatter.Format(diag, string(xmlData)))
		fmt.Println()
	}

	os.Exit(1)
}

// createMockSCXMLSchema creates a basic SCXML schema for testing
func createMockSCXMLSchema() *xsd.Schema {
	schema := &xsd.Schema{
		TargetNamespace: "http://www.w3.org/2005/07/scxml",
		ElementDecls:    make(map[xsd.QName]*xsd.ElementDecl),
		TypeDefs:        make(map[xsd.QName]xsd.Type),
	}

	// Define scxml root element
	scxmlType := &xsd.ComplexType{
		QName: xsd.QName{Namespace: schema.TargetNamespace, Local: "scxml"},
		Attributes: []*xsd.AttributeDecl{
			{Name: xsd.QName{Local: "version"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "initial"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "name"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "datamodel"}, Use: xsd.OptionalUse},
		},
		// Allow any child elements and namespace attributes
		AnyAttribute: &xsd.AnyAttribute{Namespace: "##any", ProcessContents: "lax"},
		Content:      &MockContent{}, // Allow any children
	}
	schema.ElementDecls[scxmlType.QName] = &xsd.ElementDecl{
		Name: scxmlType.QName,
		Type: scxmlType,
	}
	schema.TypeDefs[scxmlType.QName] = scxmlType

	// Define state element
	stateType := &xsd.ComplexType{
		QName: xsd.QName{Namespace: schema.TargetNamespace, Local: "state"},
		Attributes: []*xsd.AttributeDecl{
			{Name: xsd.QName{Local: "id"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "initial"}, Use: xsd.OptionalUse},
		},
		Content: &MockContent{}, // Allow children
	}
	schema.ElementDecls[stateType.QName] = &xsd.ElementDecl{
		Name: stateType.QName,
		Type: stateType,
	}
	schema.TypeDefs[stateType.QName] = stateType

	// Define send element - NOTE: uses 'id' not 'sendid'!
	sendType := &xsd.ComplexType{
		QName: xsd.QName{Namespace: schema.TargetNamespace, Local: "send"},
		Attributes: []*xsd.AttributeDecl{
			{Name: xsd.QName{Local: "id"}, Use: xsd.OptionalUse}, // Correct attribute
			{Name: xsd.QName{Local: "event"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "eventexpr"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "target"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "targetexpr"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "type"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "typeexpr"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "delay"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "delayexpr"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "namelist"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "idlocation"}, Use: xsd.OptionalUse},
		},
	}
	schema.ElementDecls[sendType.QName] = &xsd.ElementDecl{
		Name: sendType.QName,
		Type: sendType,
	}
	schema.TypeDefs[sendType.QName] = sendType

	// Define transition element - NOTE: no 'priority' attribute!
	transitionType := &xsd.ComplexType{
		QName: xsd.QName{Namespace: schema.TargetNamespace, Local: "transition"},
		Attributes: []*xsd.AttributeDecl{
			{Name: xsd.QName{Local: "event"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "cond"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "target"}, Use: xsd.OptionalUse},
			{Name: xsd.QName{Local: "type"}, Use: xsd.OptionalUse},
		},
	}
	schema.ElementDecls[transitionType.QName] = &xsd.ElementDecl{
		Name: transitionType.QName,
		Type: transitionType,
	}
	schema.TypeDefs[transitionType.QName] = transitionType

	// Define cancel element - uses 'sendid' to reference a send's 'id'
	cancelType := &xsd.ComplexType{
		QName: xsd.QName{Namespace: schema.TargetNamespace, Local: "cancel"},
		Attributes: []*xsd.AttributeDecl{
			{Name: xsd.QName{Local: "sendid"}, Use: xsd.OptionalUse}, // References send id
			{Name: xsd.QName{Local: "sendidexpr"}, Use: xsd.OptionalUse},
		},
	}
	schema.ElementDecls[cancelType.QName] = &xsd.ElementDecl{
		Name: cancelType.QName,
		Type: cancelType,
	}
	schema.TypeDefs[cancelType.QName] = cancelType

	// Add other common elements
	// Add finalize for invoke
	for _, elemName := range []string{"datamodel", "data", "onentry", "onexit", "log", "assign", "script", "invoke", "parallel", "final", "history", "initial", "raise", "if", "else", "elseif", "foreach", "param", "content", "donedata", "finalize"} {
		qname := xsd.QName{Namespace: schema.TargetNamespace, Local: elemName}
		elemType := &xsd.ComplexType{
			QName:        qname,
			Attributes:   []*xsd.AttributeDecl{},
			AnyAttribute: &xsd.AnyAttribute{Namespace: "##any", ProcessContents: "lax"},
			Content:      &MockContent{}, // Allow children
		}
		schema.ElementDecls[qname] = &xsd.ElementDecl{
			Name: qname,
			Type: elemType,
		}
	}

	return schema
}

// MockContent allows any child elements
type MockContent struct{}

func (m *MockContent) Validate(element xmldom.Element, schema *xsd.Schema) []xsd.Violation {
	// Allow any children - don't report violations
	return nil
}
