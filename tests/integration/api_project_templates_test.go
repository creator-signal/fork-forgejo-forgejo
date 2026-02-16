// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIListProjectTemplates(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/project/templates")
	resp := MakeRequest(t, req, http.StatusOK)

	var names []string
	DecodeJSON(t, resp, &names)

	// Should return template names without "none"
	assert.Equal(t, []string{"basic_kanban", "bug_triage"}, names)
}

func TestAPIGetProjectTemplate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("basic_kanban", func(t *testing.T) {
		req := NewRequest(t, "GET", "/api/v1/project/templates/basic_kanban")
		resp := MakeRequest(t, req, http.StatusOK)

		var tmpl api.ProjectTemplate
		DecodeJSON(t, resp, &tmpl)

		assert.Equal(t, "basic_kanban", tmpl.Key)
		require.Len(t, tmpl.Columns, 4) // Backlog + To Do, In Progress, Done
		assert.Equal(t, "Backlog", tmpl.Columns[0].Title)
		assert.True(t, tmpl.Columns[0].Default)
		assert.Equal(t, "To Do", tmpl.Columns[1].Title)
		assert.False(t, tmpl.Columns[1].Default)
		assert.Equal(t, "In Progress", tmpl.Columns[2].Title)
		assert.Equal(t, "Done", tmpl.Columns[3].Title)
	})

	t.Run("bug_triage", func(t *testing.T) {
		req := NewRequest(t, "GET", "/api/v1/project/templates/bug_triage")
		resp := MakeRequest(t, req, http.StatusOK)

		var tmpl api.ProjectTemplate
		DecodeJSON(t, resp, &tmpl)

		assert.Equal(t, "bug_triage", tmpl.Key)
		require.Len(t, tmpl.Columns, 5) // Backlog + Needs Triage, High Priority, Low Priority, Closed
		assert.Equal(t, "Backlog", tmpl.Columns[0].Title)
		assert.True(t, tmpl.Columns[0].Default)
		assert.Equal(t, "Needs Triage", tmpl.Columns[1].Title)
	})

	t.Run("nonexistent", func(t *testing.T) {
		req := NewRequest(t, "GET", "/api/v1/project/templates/nonexistent")
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("none_is_not_found", func(t *testing.T) {
		req := NewRequest(t, "GET", "/api/v1/project/templates/none")
		MakeRequest(t, req, http.StatusNotFound)
	})
}
