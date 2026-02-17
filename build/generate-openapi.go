// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

const (
	swaggerSpecPath = "templates/swagger/v1_json.tmpl"
	openapi3OutPath = "templates/swagger/v1_openapi3_json.tmpl"

	appSubUrlVar = "{{AppSubUrl | JSEscape}}"
	appVerVar    = "{{AppVer | JSEscape}}"

	// Placeholders used during conversion (must not appear in the real spec)
	appSubUrlPlaceholder = "FORGEJO_APP_SUB_URL_PLACEHOLDER"
	appVerPlaceholder    = "0.0.0-forgejo-placeholder"
)

var (
	appSubUrlRe = regexp.MustCompile(regexp.QuoteMeta(appSubUrlVar))
	appVerRe    = regexp.MustCompile(regexp.QuoteMeta(appVerVar))
)

func main() {
	// Read the Swagger 2.0 template
	data, err := os.ReadFile(swaggerSpecPath)
	if err != nil {
		log.Fatalf("reading swagger spec: %v", err)
	}

	// Strip Go template variables so we have valid JSON for parsing
	cleaned := appSubUrlRe.ReplaceAll(data, nil)
	cleaned = appVerRe.ReplaceAll(cleaned, []byte(appVerPlaceholder))

	// Parse as Swagger 2.0
	var swagger2 openapi2.T
	if err := json.Unmarshal(cleaned, &swagger2); err != nil {
		log.Fatalf("parsing swagger 2.0: %v", err)
	}

	// Convert to OpenAPI 3.0
	oas3, err := openapi2conv.ToV3(&swagger2)
	if err != nil {
		log.Fatalf("converting to openapi 3.0: %v", err)
	}

	// Ensure a servers entry exists with the base path including the placeholder
	oas3.Servers = openapi3.Servers{
		{URL: appSubUrlPlaceholder + "/api/v1"},
	}

	// Fix "type: file" schemas left over from Swagger 2.0 conversion.
	// In OAS3, file responses use type: string + format: binary.
	fixFileSchemas(oas3)

	// Marshal to JSON with indentation
	out, err := json.MarshalIndent(oas3, "", "  ")
	if err != nil {
		log.Fatalf("marshaling openapi 3.0: %v", err)
	}

	// Re-inject Go template variables
	result := strings.ReplaceAll(string(out), appSubUrlPlaceholder, appSubUrlVar)
	result = strings.ReplaceAll(result, appVerPlaceholder, appVerVar)

	// Ensure trailing newline
	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	if err := os.WriteFile(openapi3OutPath, []byte(result), 0o644); err != nil {
		log.Fatalf("writing openapi 3.0 spec: %v", err)
	}

	fmt.Printf("Generated %s\n", openapi3OutPath)
}

// fixFileSchemas walks the OAS3 spec and replaces any schema with
// "type": "file" (invalid in OAS3) with "type": "string", "format": "binary".
func fixFileSchemas(doc *openapi3.T) {
	for _, pathItem := range doc.Paths {
		for _, op := range []*openapi3.Operation{
			pathItem.Get, pathItem.Post, pathItem.Put, pathItem.Patch,
			pathItem.Delete, pathItem.Head, pathItem.Options, pathItem.Trace,
		} {
			if op == nil {
				continue
			}
			for _, resp := range op.Responses {
				if resp.Value == nil {
					continue
				}
				for _, mediaType := range resp.Value.Content {
					fixSchema(mediaType.Schema)
				}
			}
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				for _, mediaType := range op.RequestBody.Value.Content {
					fixSchema(mediaType.Schema)
				}
			}
		}
	}
}

func fixSchema(ref *openapi3.SchemaRef) {
	if ref == nil || ref.Value == nil {
		return
	}
	if ref.Value.Type == "file" {
		ref.Value.Type = "string"
		ref.Value.Format = "binary"
	}
}
