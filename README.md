# go-xsd

[![CI](https://github.com/agentflare-ai/go-xsd/workflows/CI/badge.svg)](https://github.com/agentflare-ai/go-xsd/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agentflare-ai/go-xsd)](https://goreportcard.com/report/github.com/agentflare-ai/go-xsd)
[![GoDoc](https://godoc.org/github.com/agentflare-ai/go-xsd?status.svg)](https://godoc.org/github.com/agentflare-ai/go-xsd)
[![codecov](https://codecov.io/gh/agentflare-ai/go-xsd/branch/main/graph/badge.svg)](https://codecov.io/gh/agentflare-ai/go-xsd)

A comprehensive XML Schema Definition (XSD) validator for Go that provides W3C-compliant XML validation against XSD schemas.

## Features

### Core Validation
- **Full XSD 1.0 Support**: Comprehensive validation according to W3C XML Schema specifications
- **Schema Loading**: Load schemas with automatic import/include resolution and circular dependency protection
- **Built-in Types**: All standard XSD built-in types (string, int, date, etc.)
- **Complex Types**: Support for sequences, choices, all groups, and nested content models
  - ComplexContent with extension/restriction
  - SimpleContent with extension/restriction
  - Mixed content models
- **Simple Types**: Full support for restrictions, lists, unions, and all standard facets
- **Type Derivation**: Proper type compatibility checking for extensions and restrictions

### Advanced Features
- **Identity Constraints**: key, keyref, and unique constraints with XPath selectors
  - Proper ID/IDREF type detection (not just name-based)
  - Type-aware validation including derived types
- **Wildcards**: `<xs:any>` and `<xs:anyAttribute>` with namespace constraint validation
  - Namespace constraints: ##any, ##other, ##targetNamespace, ##local
  - ProcessContents modes: strict, lax, skip
- **Substitution Groups**: Element substitution with type compatibility verification
  - Validates that substituting element's type derives from head element's type
  - Cycle detection prevents infinite recursion
- **Fixed/Default Values**: Validation of fixed and default attribute/element values
- **Attribute Groups**: Full support for attribute group references and resolution
- **Model Groups**: Support for named group definitions and references

### Performance & Safety
- **Schema Caching**: Efficient schema reuse with LRU caching
- **Thread-Safe**: Concurrent schema validation support with proper locking
- **Cycle Detection**: Protects against infinite recursion in type hierarchies
- **Memory Efficient**: Optimized allocations in hot paths, GC-friendly design

## Installation

```bash
go get github.com/agentflare-ai/go-xsd
```

## Quick Start

### Basic Validation

```go
package main

import (
    "fmt"
    "log"

    "github.com/agentflare-ai/go-xsd"
    "github.com/agentflare-ai/go-xmldom"
)

func main() {
    // Load schema
    loader := xsd.NewSchemaLoader("./schemas")
    schema, err := loader.LoadSchemaWithImports("myschema.xsd")
    if err != nil {
        log.Fatal(err)
    }

    // Parse XML document
    doc, err := xmldom.ParseFile("document.xml")
    if err != nil {
        log.Fatal(err)
    }

    // Validate
    validator := xsd.NewValidator(schema)
    violations := validator.Validate(doc)

    if len(violations) > 0 {
        for _, v := range violations {
            if v.Attribute != "" {
                fmt.Printf("[%s] Attribute '%s': %s\n", v.Code, v.Attribute, v.Message)
            } else {
                fmt.Printf("[%s] %s\n", v.Code, v.Message)
            }
        }
    } else {
        fmt.Println("Document is valid!")
    }
}
```

### Schema Loading with Imports

```go
loader := xsd.NewSchemaLoader("./schemas")
loader.AllowRemote = true // Enable remote schema loading (disabled by default)

schema, err := loader.LoadSchemaWithImports("schema.xsd")
if err != nil {
    log.Fatal(err)
}

// The loader automatically resolves all imports and includes
```

### Using the Schema Cache

```go
// Create a cache for frequently used schemas
cache := xsd.NewSchemaCache(100) // Max 100 schemas

// Load with caching
schema, err := cache.GetOrLoad("./schemas/myschema.xsd", func(path string) (*xsd.Schema, error) {
    loader := xsd.NewSchemaLoader(filepath.Dir(path))
    return loader.LoadSchemaWithImports(filepath.Base(path))
})
```

### Advanced Example: Type-Safe Validation

```go
// Schema with ID/IDREF constraints and substitution groups
schemaXML := `
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <!-- Base vehicle element -->
  <xs:element name="vehicle" type="VehicleType"/>

  <!-- Car can substitute for vehicle -->
  <xs:element name="car" type="CarType" substitutionGroup="vehicle"/>

  <xs:complexType name="VehicleType">
    <xs:sequence>
      <xs:element name="brand" type="xs:string"/>
    </xs:sequence>
    <xs:attribute name="id" type="xs:ID" use="required"/>
  </xs:complexType>

  <xs:complexType name="CarType">
    <xs:complexContent>
      <xs:extension base="VehicleType">
        <xs:sequence>
          <xs:element name="doors" type="xs:int"/>
        </xs:sequence>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
</xs:schema>`

// The validator will:
// - Check that 'car' type derives from 'vehicle' type (type compatibility)
// - Validate 'id' attribute as xs:ID type (not just by name)
// - Detect duplicate ID values
// - Ensure proper type structure with extensions
```

## API Overview

### Core Types

#### SchemaLoader

Handles loading XSD schemas from files or URLs with automatic import/include resolution:

```go
type SchemaLoader struct {
    BaseDir     string      // Base directory for relative paths
    AllowRemote bool        // Allow loading remote schemas
}

func NewSchemaLoader(baseDir string) *SchemaLoader
func (sl *SchemaLoader) LoadSchemaWithImports(path string) (*Schema, error)
```

#### Schema

Represents a compiled XSD schema:

```go
type Schema struct {
    TargetNamespace    string
    ElementDecls       map[QName]*ElementDecl
    TypeDefs           map[QName]Type
    SubstitutionGroups map[QName][]QName
}
```

#### Validator

Validates XML documents against schemas:

```go
type Validator struct {
    // Internal state
}

func NewValidator(schema *Schema) *Validator
func (v *Validator) Validate(doc xmldom.Document) []Violation
```

#### Violation

Represents a validation error:

```go
type Violation struct {
    Element   xmldom.Element  // The violating element
    Attribute string         // Attribute name (if violation is attribute-related)
    Code      string         // W3C error code (e.g., "cvc-complex-type.2.4.d")
    Message   string         // Human-readable error message
    Expected  []string       // Expected values (for enumerations)
    Actual    string         // Actual value that failed validation
}
```

Error codes follow W3C XML Schema validation error conventions (cvc-* codes).

### Type System

The library supports all XSD types:

- **Simple Types**: string, int, boolean, date, time, decimal, etc.
- **Complex Types**: Elements with child content and attributes
- **Lists**: Space-separated lists of values
- **Unions**: Values that can be one of several types
- **Restrictions**: Types with facet constraints (minLength, maxLength, pattern, etc.)

### Identity Constraints

Support for XSD identity constraints:

```go
// Automatically validated when present in schema
<xs:key name="productKey">
    <xs:selector xpath="product"/>
    <xs:field xpath="@id"/>
</xs:key>

<xs:keyref name="orderProductRef" refer="productKey">
    <xs:selector xpath="order/item"/>
    <xs:field xpath="@productId"/>
</xs:keyref>
```

## Advanced Features

### Custom Type Validation

The library validates all standard XSD facets:

- Length constraints: `minLength`, `maxLength`, `length`
- Numeric bounds: `minInclusive`, `maxInclusive`, `minExclusive`, `maxExclusive`
- Total digits and fraction digits
- Pattern matching (regular expressions)
- Enumeration restrictions
- Whitespace handling

### Substitution Groups

Elements can be substituted based on their substitution group membership:

```go
// Schema automatically handles substitution group validation
<xs:element name="vehicle" type="VehicleType"/>
<xs:element name="car" type="CarType" substitutionGroup="vehicle"/>
<xs:element name="truck" type="TruckType" substitutionGroup="vehicle"/>
```

### Wildcards with Namespace Processing

Support for `<xs:any>` elements with proper namespace constraint validation:

```go
<xs:any namespace="##any" processContents="lax"/>
<xs:any namespace="##other" processContents="strict"/>
<xs:any namespace="http://example.com ##targetNamespace" processContents="skip"/>
```

## Testing

### Unit Tests

Run the test suite:

```bash
go test -v ./...
```

Run tests with coverage:

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### W3C Conformance Tests

The library is tested against the official [W3C XML Schema Test Suite](https://www.w3.org/XML/2004/xml-schema-test-suite/):

```bash
# Auto-download and run W3C conformance tests
go run ./cmd/w3c_test --auto-download --analyze

# View detailed results
go run ./cmd/w3c_test --auto-download --verbose
```

**CI Integration**: W3C tests run automatically in CI/CD before each release to ensure standards compliance.

### Test Coverage

The library includes comprehensive tests including:
- **W3C Conformance Suite**: Official W3C test cases (runs in CI)
- **Identity Constraints**: key, keyref, and unique validation
- **Substitution Groups**: Element substitution and type checking
- **Wildcard Validation**: xs:any and xs:anyAttribute handling
- **Fixed/Default Values**: Attribute and element defaults
- **Union and List Types**: Complex type composition

## Error Codes

Validation errors use W3C XML Schema validation error codes (Component Validation Constraint codes):

| Code | Description |
|------|-------------|
| `cvc-complex-type.2.4.a` | Missing required element |
| `cvc-complex-type.2.4.b` | Element not allowed by content model |
| `cvc-complex-type.2.4.d` | Unexpected element |
| `cvc-datatype-valid.1` | Invalid value for datatype |
| `cvc-enumeration-valid` | Value not in enumeration |
| `cvc-id.1` | ID value must be unique |
| `cvc-id.2` | Duplicate ID value |
| `cvc-wildcard.2` | Element not allowed by namespace constraint |
| `cvc-attribute.3` | Attribute value invalid |
| `cvc-complex-type.3.2.2` | Attribute not allowed |
| `cvc-complex-type.4` | Attribute required but missing |

Full list follows W3C XML Schema 1.0 Part 1: Structures specification.

## Project Structure

```
go-xsd/
├── schema.go              # Core schema types and structures
├── schema_loader.go       # Schema loading with imports
├── schema_validator.go    # Schema-to-schema validation
├── validator.go           # Document validation engine
├── builtin_types.go       # XSD built-in type definitions
├── facets.go             # Facet validation
├── identity_constraints.go # Key/keyref/unique validation
├── wildcards.go          # Any/anyAttribute support
├── union_list_types.go   # Union and list type handling
├── fixed_default.go      # Fixed/default value validation
├── cache.go              # Schema caching with LRU
├── diagnostic.go         # Violation reporting and formatting
├── fixes.plan.md         # Development roadmap and tracking
└── cmd/
    ├── validate/         # CLI validation tool
    └── w3c_test/         # W3C test suite runner
```

## Command-Line Tools

### validate

Validate an XML document against a schema:

```bash
go run ./cmd/validate schema.xsd document.xml
```

### w3c_test

Run W3C XSD test suite:

```bash
# Auto-download and run (downloads once, cached for 7 days)
go run ./cmd/w3c_test --auto-download

# Run with specific suite directory
go run ./cmd/w3c_test -suite /path/to/w3c/testsuite

# Force fresh download (bypasses cache)
go run ./cmd/w3c_test --force-download

# Run specific test file
go run ./cmd/w3c_test --auto-download -file msMeta/test_w3c.xml

# Generate failure analysis report
go run ./cmd/w3c_test --auto-download -analyze
```

**Note:** The test suite (≈50MB) is automatically downloaded from W3C and cached locally. The cache expires after 7 days to avoid hammering W3C servers with repeated downloads.

## Recent Improvements (2025)

### Validation Engine Enhancements
- ✅ **SimpleContent Validation**: Full validation of text content in elements with simpleContent
- ✅ **List/Union Attribute Validation**: Proper validation of attributes with list or union types
- ✅ **Type-Aware ID/IDREF Detection**: Schema-based ID/IDREF validation instead of name-based heuristics
- ✅ **Substitution Group Type Checking**: Verifies type compatibility for substitution group members
- ✅ **Wildcard Namespace Constraints**: Proper error reporting for namespace constraint violations

### Performance & Reliability
- ✅ **Cycle Detection**: Added infinite recursion protection for circular type definitions
- ✅ **Memory Optimization**: Reduced allocations in hot paths (ID/IDREF collection)
- ✅ **GC Efficiency**: Pre-allocated maps and reusable singleton objects

See [fixes.plan.md](fixes.plan.md) for detailed tracking of improvements and remaining enhancements.

## Requirements

- Go 1.21 or later
- [go-xmldom](https://github.com/agentflare-ai/go-xmldom) for XML parsing

## License

Copyright © 2025 AgentFlare AI

## Contributing

This is a private repository. For questions or issues, please contact the maintainers.

## Related Projects

- [go-xmldom](https://github.com/agentflare-ai/go-xmldom) - XML DOM and XPath library for Go
- [go-scxml](https://github.com/agentflare-ai/go-scxml) - State Chart XML (SCXML) implementation

## References

- [W3C XML Schema Part 1: Structures](https://www.w3.org/TR/xmlschema-1/)
- [W3C XML Schema Part 2: Datatypes](https://www.w3.org/TR/xmlschema-2/)
- [XML Schema Test Suite](https://www.w3.org/XML/2004/xml-schema-test-suite/)
