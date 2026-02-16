// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// ProjectTemplate describes a project template and its columns
type ProjectTemplate struct {
	// Name is the template key (e.g. "basic_kanban", "bug_triage")
	Key string `json:"key"`
	// Columns lists the columns created by this template, in order.
	// The first column is the default column for new issues.
	Columns []ProjectTemplateColumn `json:"columns"`
}

// ProjectTemplateColumn describes a column within a project template
type ProjectTemplateColumn struct {
	// Title of the column
	Title string `json:"title"`
	// Default indicates this column receives new issues by default
	Default bool `json:"default"`
}
