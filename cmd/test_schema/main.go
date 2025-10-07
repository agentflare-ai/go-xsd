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
		fmt.Println("Usage: test_schema_validation <schema.xsd>")
		os.Exit(1)
	}

	filename := os.Args[1]

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	doc, err := xmldom.Decode(file)
	if err != nil {
		log.Fatalf("Failed to parse XML: %v", err)
	}

	sv := xsd.NewSchemaValidator()
	errors := sv.ValidateSchema(doc)

	if len(errors) == 0 {
		fmt.Println("Schema is valid!")
	} else {
		fmt.Printf("Schema has %d validation errors:\n", len(errors))
		for i, err := range errors {
			fmt.Printf("%d. %v\n", i+1, err)
		}
	}
}
