# go-xsd

A comprehensive XML Schema Definition (XSD) validator for Go that provides W3C-compliant XML validation against XSD schemas.

## Features

- **Full XSD 1.0 Support**: Comprehensive validation according to W3C XML Schema specifications
- **Schema Loading**: Load schemas with automatic import/include resolution
- **Built-in Types**: All standard XSD built-in types (string, int, date, etc.)
- **Complex Types**: Support for complex types with sequences, choices, and all content models
- **Simple Types**: Restrictions, lists, unions, and facets
- **Identity Constraints**: key, keyref, and unique constraints
- **Wildcards**: `<xs:any>` and `<xs:anyAttribute>` support with namespace processing
- **Substitution Groups**: Element substitution with proper type checking
- **Fixed/Default Values**: Validation of fixed and default attribute/element values
- **Schema Caching**: Efficient schema reuse with built-in caching
- **Thread-Safe**: Concurrent schema validation support

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
            fmt.Printf("Validation error: %s at %s\n", v.Message, v.Path)
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
    Code    string  // Error code
    Message string  // Human-readable message
    Path    string  // XPath to the violating element
    Line    int     // Line number (if available)
    Column  int     // Column number (if available)
}
```

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

Run the test suite:

```bash
go test -v ./...
```

Run tests with coverage:

```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

The library includes comprehensive tests including:
- W3C test suite validation
- Identity constraint tests
- Substitution group tests
- Wildcard validation tests
- Fixed/default value tests
- Union and list type tests

## Project Structure

```
go-xsd/
├── schema.go              # Core schema types and structures
├── schema_loader.go       # Schema loading with imports
├── validator.go           # Main validation engine
├── builtin_types.go       # XSD built-in type definitions
├── facets.go             # Facet validation
├── identity_constraints.go # Key/keyref/unique validation
├── substitution_groups.go # Element substitution
├── wildcards.go          # Any/anyAttribute support
├── union_list_types.go   # Union and list type handling
├── fixed_default.go      # Fixed/default value validation
├── cache.go              # Schema caching
├── diagnostic.go         # Violation reporting
└── cmd/
    ├── validate/         # CLI validation tool
    ├── test_schema/      # Schema testing utility
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
go run ./cmd/w3c_test path/to/w3c/testsuite
```

## Requirements

- Go 1.24.5 or later
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
