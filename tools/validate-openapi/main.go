// Package main provides a CLI tool to validate OpenAPI 3.x specifications
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	// Minimum number of arguments required (program name + spec file).
	minArgsRequired = 2
)

func main() {
	if len(os.Args) < minArgsRequired {
		fmt.Fprintf(os.Stderr, "Usage: %s <openapi-spec-file>\n", os.Args[0])
		os.Exit(1)
	}

	specFile := os.Args[1]

	// Load and validate OpenAPI specification
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	// Validate the document
	ctx := context.Background()
	if err := doc.Validate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "OpenAPI spec validation failed: %v\n", err)
		os.Exit(1)
	}

	// Additional checks
	checkForCommonIssues(doc)

	//nolint:forbidigo // CLI tool requires stdout output
	fmt.Printf("✓ OpenAPI spec is valid: %s\n", specFile)
	//nolint:forbidigo // CLI tool requires stdout output
	fmt.Printf("  Title: %s\n", doc.Info.Title)
	//nolint:forbidigo // CLI tool requires stdout output
	fmt.Printf("  Version: %s\n", doc.Info.Version)
	//nolint:forbidigo // CLI tool requires stdout output
	fmt.Printf("  Paths: %d\n", len(doc.Paths.Map()))
	//nolint:forbidigo // CLI tool requires stdout output
	fmt.Printf("  Components/Schemas: %d\n", len(doc.Components.Schemas))
}

// checkForCommonIssues performs additional validation checks.
//
//nolint:cyclop // OpenAPI validation tool requires checking multiple common issues
func checkForCommonIssues(doc *openapi3.T) {
	warnings := []string{}

	// Check for paths without operationId
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.OperationID == "" {
				warnings = append(warnings, fmt.Sprintf("Missing operationId: %s %s", method, path))
			}
		}
	}

	// Check for responses without descriptions
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			for status, response := range operation.Responses.Map() {
				if response.Value != nil && response.Value.Description == nil {
					warnings = append(warnings, fmt.Sprintf("Missing response description: %s %s [%s]", method, path, status))
				}
			}
		}
	}

	if len(warnings) > 0 {
		//nolint:forbidigo // CLI tool requires stdout output
		fmt.Println("\nWarnings:")
		for _, warning := range warnings {
			//nolint:forbidigo // CLI tool requires stdout output
			fmt.Printf("  ⚠ %s\n", warning)
		}
	}
}
