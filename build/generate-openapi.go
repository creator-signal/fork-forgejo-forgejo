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
	"sort"
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
		{URL: appSubUrlPlaceholder + "/api"},
	}

	// Prefix all paths with /v1 so the merged spec can use /api as the base
	prefixedPaths := make(openapi3.Paths, len(oas3.Paths))
	for path, item := range oas3.Paths {
		prefixedPaths["/v1"+path] = item
	}
	oas3.Paths = prefixedPaths

	// Fix "type: file" schemas left over from Swagger 2.0 conversion.
	// In OAS3, file responses use type: string + format: binary.
	fixFileSchemas(oas3)

	// OAS3 post-processing: enrich the spec with details that Swagger 2.0
	// and go-swagger cannot express.
	addURIFormats(oas3)
	addDeprecatedFlags(oas3)
	extractSharedEnums(oas3)

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

// addURIFormats sets format: uri on string properties whose names indicate they
// hold URLs. This information is lost in Swagger 2.0 (go-swagger doesn't emit
// format annotations for URL fields) but is valuable for code generators.
func addURIFormats(doc *openapi3.T) {
	if doc.Components == nil {
		return
	}
	for _, schemaRef := range doc.Components.Schemas {
		if schemaRef.Value == nil {
			continue
		}
		for propName, propRef := range schemaRef.Value.Properties {
			if propRef == nil || propRef.Value == nil || propRef.Ref != "" {
				continue
			}
			prop := propRef.Value
			if prop.Type != "string" || prop.Format != "" {
				continue
			}
			if isURLProperty(propName) {
				prop.Format = "uri"
			}
		}
	}
}

// isURLProperty returns true if the property name indicates it holds a URL.
func isURLProperty(name string) bool {
	if strings.HasSuffix(name, "_url") {
		return true
	}
	switch name {
	case "url", "html_url", "clone_url":
		return true
	}
	return false
}

// addDeprecatedFlags sets deprecated: true on schema properties whose
// description contains "deprecated". In OAS3, deprecated is a first-class
// boolean; Swagger 2.0 could only express it as text in the description.
func addDeprecatedFlags(doc *openapi3.T) {
	if doc.Components == nil {
		return
	}
	for _, schemaRef := range doc.Components.Schemas {
		if schemaRef.Value == nil {
			continue
		}
		for _, propRef := range schemaRef.Value.Properties {
			if propRef == nil || propRef.Value == nil || propRef.Ref != "" {
				continue
			}
			desc := strings.ToLower(propRef.Value.Description)
			if strings.Contains(desc, "deprecated") {
				propRef.Value.Deprecated = true
			}
		}
	}
}

type enumUsage struct {
	schemaName string
	propName   string
	propRef    *openapi3.SchemaRef
	inItems    bool // true if enum is on .Items, not the prop itself
}

// extractSharedEnums finds identical enum arrays used by multiple schema
// properties, creates a standalone named schema for each, and replaces the
// inline enums with $ref pointers. This produces proper enum types for code
// generators instead of anonymous inline enums repeated on each field.
func extractSharedEnums(doc *openapi3.T) {
	if doc.Components == nil {
		return
	}

	enumGroups := map[string][]enumUsage{}

	for schemaName, schemaRef := range doc.Components.Schemas {
		if schemaRef.Value == nil {
			continue
		}
		for propName, propRef := range schemaRef.Value.Properties {
			if propRef == nil || propRef.Value == nil || propRef.Ref != "" {
				continue
			}
			if len(propRef.Value.Enum) > 1 && propRef.Value.Type == "string" {
				key := enumKey(propRef.Value.Enum)
				enumGroups[key] = append(enumGroups[key], enumUsage{schemaName, propName, propRef, false})
			}
			// Check array items
			if propRef.Value.Type == "array" && propRef.Value.Items != nil &&
				propRef.Value.Items.Value != nil && propRef.Value.Items.Ref == "" &&
				len(propRef.Value.Items.Value.Enum) > 1 && propRef.Value.Items.Value.Type == "string" {
				key := enumKey(propRef.Value.Items.Value.Enum)
				enumGroups[key] = append(enumGroups[key], enumUsage{schemaName, propName, propRef, true})
			}
		}
	}

	// Only extract enums used by 2+ fields
	for key, usages := range enumGroups {
		if len(usages) < 2 {
			continue
		}

		// Derive a name from the enum values and usage context
		enumName := deriveEnumName(usages)
		// Avoid collisions with existing schemas
		if _, exists := doc.Components.Schemas[enumName]; exists {
			enumName += "Type"
		}
		if _, exists := doc.Components.Schemas[enumName]; exists {
			continue // still collides, skip
		}

		// Get the enum values from the first usage
		var enumValues []any
		if usages[0].inItems {
			enumValues = usages[0].propRef.Value.Items.Value.Enum
		} else {
			enumValues = usages[0].propRef.Value.Enum
		}

		// Create the standalone enum schema
		doc.Components.Schemas[enumName] = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: "string",
				Enum: enumValues,
			},
		}

		ref := "#/components/schemas/" + enumName

		// Replace inline enums with $ref
		for _, usage := range usages {
			if usage.inItems {
				usage.propRef.Value.Items = &openapi3.SchemaRef{Ref: ref}
			} else {
				// Preserve the property description and other metadata by
				// wrapping in allOf: the $ref provides the type+enum, the
				// inline schema provides description/deprecated/etc.
				old := usage.propRef.Value
				if old.Description == "" && !old.Deprecated && old.Format == "" {
					// Simple case: just replace with a $ref
					usage.propRef.Ref = ref
					usage.propRef.Value = nil
				} else {
					// Has metadata: use allOf to combine $ref with metadata
					usage.propRef.Value = &openapi3.Schema{
						AllOf: openapi3.SchemaRefs{
							{Ref: ref},
						},
						Description: old.Description,
						Deprecated:  old.Deprecated,
					}
					// Clear enum from the wrapper (it's in the $ref now)
				}
			}
		}

		_ = key // used as map key
	}
}

// enumKey returns a canonical string key for an enum value set, for grouping.
func enumKey(values []any) string {
	strs := make([]string, len(values))
	for i, v := range values {
		strs[i] = fmt.Sprintf("%v", v)
	}
	sort.Strings(strs)
	return strings.Join(strs, "|")
}

// Known Go type names for enum types. These are not present in the Swagger 2.0
// definitions (go-swagger doesn't emit standalone string type definitions), so
// we map them explicitly using the const name prefix from x-go-enum-desc.
var knownEnumTypes = map[string]string{
	"CommitStatus":     "CommitStatusState",
	"State":            "StateType",
	"ReviewState":      "ReviewStateType",
	"NotifySubject":    "NotifySubjectType",
	"IssueFormField":   "IssueFormFieldType",
	"ObjectFormatName": "ObjectFormatName",
}

// deriveEnumName picks a name for an extracted enum schema based on the
// Go const name prefix from x-go-enum-desc, with a fallback to the property name.
func deriveEnumName(usages []enumUsage) string {
	// Try to extract the Go type name from x-go-enum-desc.
	// Format: "value ConstName  ConstName description\n..."
	// e.g. "pending CommitStatusPending  CommitStatusPending is for..."
	for _, u := range usages {
		if u.propRef.Value == nil {
			continue
		}
		desc, ok := u.propRef.Value.Extensions["x-go-enum-desc"]
		if !ok {
			continue
		}
		s, ok := desc.(string)
		if !ok {
			continue
		}
		parts := strings.Fields(s)
		if len(parts) < 2 {
			continue
		}
		constName := parts[1]

		// Try to strip the enum value from the const name to get the prefix
		var vals []any
		if u.inItems {
			vals = u.propRef.Value.Items.Value.Enum
		} else {
			vals = u.propRef.Value.Enum
		}
		for _, v := range vals {
			vs := fmt.Sprintf("%v", v)
			// Case-insensitive suffix matching: the enum value may be
			// lowercase ("pending"), UPPER ("APPROVED"), or mixed ("sha1"),
			// while the Go const uses PascalCase ("Pending", "Approved", "SHA1").
			lowerConst := strings.ToLower(constName)
			lowerVal := strings.ToLower(vs)
			if strings.HasSuffix(lowerConst, lowerVal) && len(lowerVal) < len(lowerConst) {
				prefix := constName[:len(constName)-len(vs)]
				// Check if we have a known Go type name for this prefix
				if goType, ok := knownEnumTypes[prefix]; ok {
					return goType
				}
				return prefix
			}
		}
	}

	// Fallback: use the most common property name, PascalCased
	nameCounts := map[string]int{}
	for _, u := range usages {
		nameCounts[u.propName]++
	}
	bestName := ""
	bestCount := 0
	for name, count := range nameCounts {
		if count > bestCount || (count == bestCount && name < bestName) {
			bestName = name
			bestCount = count
		}
	}
	result := ""
	for _, p := range strings.Split(bestName, "_") {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result + "Enum"
}
