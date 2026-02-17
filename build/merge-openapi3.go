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

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	openapi3SpecPath = "templates/swagger/v1_openapi3_json.tmpl"
	forgejoSpecPath  = "public/assets/forgejo/api.v1.yml"

	appSubUrlVar = "{{AppSubUrl | JSEscape}}"
	appVerVar    = "{{AppVer | JSEscape}}"

	appSubUrlPlaceholder = "FORGEJO_APP_SUB_URL_PLACEHOLDER"
	appVerPlaceholder    = "0.0.0-forgejo-placeholder"
)

var (
	appSubUrlRe = regexp.MustCompile(regexp.QuoteMeta(appSubUrlVar))
	appVerRe    = regexp.MustCompile(regexp.QuoteMeta(appVerVar))
)

func main() {
	// --- Load the auto-converted OAS3 spec (from generate-openapi.go output) ---
	data, err := os.ReadFile(openapi3SpecPath)
	if err != nil {
		log.Fatalf("reading openapi3 spec: %v", err)
	}

	// Replace Go template variables with placeholders so we have valid JSON for parsing
	cleaned := appSubUrlRe.ReplaceAll(data, []byte(appSubUrlPlaceholder))
	cleaned = appVerRe.ReplaceAll(cleaned, []byte(appVerPlaceholder))

	baseDoc, err := openapi3.NewLoader().LoadFromData(cleaned)
	if err != nil {
		log.Fatalf("parsing openapi3 spec: %v", err)
	}

	// --- Load the hand-written Forgejo spec ---
	forgejoDoc, err := openapi3.NewLoader().LoadFromFile(forgejoSpecPath)
	if err != nil {
		log.Fatalf("parsing forgejo spec: %v", err)
	}

	// --- Collect existing operationIds from the base spec ---
	existingOpIDs := make(map[string]bool)
	for _, item := range baseDoc.Paths {
		for _, op := range []*openapi3.Operation{
			item.Get, item.Post, item.Put, item.Patch,
			item.Delete, item.Head, item.Options, item.Trace,
		} {
			if op != nil && op.OperationID != "" {
				existingOpIDs[op.OperationID] = true
			}
		}
	}

	// --- Merge Forgejo paths into the base spec ---
	// Forgejo spec paths are under /api/forgejo/v1; we prefix them with /forgejo/v1
	// (the base spec already uses /api as the server URL).
	// Prefix conflicting operationIds with "forgejo" to avoid duplicates.
	for path, item := range forgejoDoc.Paths {
		for _, op := range []*openapi3.Operation{
			item.Get, item.Post, item.Put, item.Patch,
			item.Delete, item.Head, item.Options, item.Trace,
		} {
			if op != nil && op.OperationID != "" && existingOpIDs[op.OperationID] {
				op.OperationID = "forgejo" + strings.ToUpper(op.OperationID[:1]) + op.OperationID[1:]
			}
		}
		mergedPath := "/forgejo/v1" + path
		baseDoc.Paths[mergedPath] = item
	}

	// --- Merge Forgejo schemas into the base spec ---
	if forgejoDoc.Components != nil && forgejoDoc.Components.Schemas != nil {
		if baseDoc.Components == nil {
			baseDoc.Components = &openapi3.Components{}
		}
		if baseDoc.Components.Schemas == nil {
			baseDoc.Components.Schemas = make(openapi3.Schemas)
		}

		// Build a set of schema names that need renaming (conflicts)
		renames := make(map[string]string)
		for name := range forgejoDoc.Components.Schemas {
			if _, exists := baseDoc.Components.Schemas[name]; exists {
				renames[name] = "Forgejo" + name
			}
		}

		// Add Forgejo schemas (renaming conflicts)
		for name, schema := range forgejoDoc.Components.Schemas {
			targetName := name
			if renamed, ok := renames[name]; ok {
				targetName = renamed
			}
			baseDoc.Components.Schemas[targetName] = schema
		}

		// Update $ref pointers in Forgejo paths to use renamed schemas
		if len(renames) > 0 {
			updateRefs(forgejoDoc, renames, baseDoc)
		}
	}

	// --- Merge Forgejo parameters into the base spec ---
	if forgejoDoc.Components != nil && forgejoDoc.Components.Parameters != nil {
		if baseDoc.Components.Parameters == nil {
			baseDoc.Components.Parameters = make(openapi3.ParametersMap)
		}
		for name, param := range forgejoDoc.Components.Parameters {
			baseDoc.Components.Parameters[name] = param
		}
	}

	// --- Merge Forgejo responses into the base spec ---
	if forgejoDoc.Components != nil && forgejoDoc.Components.Responses != nil {
		if baseDoc.Components.Responses == nil {
			baseDoc.Components.Responses = make(openapi3.Responses)
		}
		for name, resp := range forgejoDoc.Components.Responses {
			baseDoc.Components.Responses[name] = resp
		}
	}

	// --- Marshal and re-inject template variables ---
	out, err := json.MarshalIndent(baseDoc, "", "  ")
	if err != nil {
		log.Fatalf("marshaling merged spec: %v", err)
	}

	result := strings.ReplaceAll(string(out), appSubUrlPlaceholder, appSubUrlVar)
	result = strings.ReplaceAll(result, appVerPlaceholder, appVerVar)

	if !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	if err := os.WriteFile(openapi3SpecPath, []byte(result), 0o644); err != nil {
		log.Fatalf("writing merged spec: %v", err)
	}

	fmt.Printf("Merged Forgejo spec into %s\n", openapi3SpecPath)
}

// updateRefs rewrites $ref pointers in the Forgejo paths (now merged into baseDoc)
// to account for renamed schemas.
func updateRefs(forgejoDoc *openapi3.T, renames map[string]string, baseDoc *openapi3.T) {
	// Only need to fix refs in the Forgejo-originated paths
	for path := range forgejoDoc.Paths {
		mergedPath := "/forgejo/v1" + path
		item := baseDoc.Paths[mergedPath]
		if item == nil {
			continue
		}
		for _, op := range []*openapi3.Operation{
			item.Get, item.Post, item.Put, item.Patch,
			item.Delete, item.Head, item.Options, item.Trace,
		} {
			if op == nil {
				continue
			}
			// Fix response schema refs
			for _, resp := range op.Responses {
				if resp.Value == nil {
					continue
				}
				for _, mt := range resp.Value.Content {
					fixSchemaRef(mt.Schema, renames)
				}
			}
			// Fix request body schema refs
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				for _, mt := range op.RequestBody.Value.Content {
					fixSchemaRef(mt.Schema, renames)
				}
			}
			// Fix parameter schema refs
			for _, param := range op.Parameters {
				if param.Value != nil && param.Value.Schema != nil {
					fixSchemaRef(param.Value.Schema, renames)
				}
			}
		}
	}
}

// fixSchemaRef rewrites a $ref if it points to a renamed schema.
func fixSchemaRef(ref *openapi3.SchemaRef, renames map[string]string) {
	if ref == nil {
		return
	}
	if ref.Ref != "" {
		for oldName, newName := range renames {
			oldRef := "#/components/schemas/" + oldName
			if ref.Ref == oldRef {
				ref.Ref = "#/components/schemas/" + newName
				break
			}
		}
	}
	// Recurse into inline schemas
	if ref.Value != nil {
		if ref.Value.Items != nil {
			fixSchemaRef(ref.Value.Items, renames)
		}
		for _, prop := range ref.Value.Properties {
			fixSchemaRef(prop, renames)
		}
		if ref.Value.AdditionalProperties.Schema != nil {
			fixSchemaRef(ref.Value.AdditionalProperties.Schema, renames)
		}
	}
}
