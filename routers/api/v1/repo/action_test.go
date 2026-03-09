// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestActions(t *testing.T) {
	unittest.PrepareTestEnv(t)

	t.Run("ListActionRuns", func(t *testing.T) {
		var runResp api.ListActionRunResponse
		ctx, resp := contexttest.MockAPIContext(t, "user5/repo4/actions/runs")
		contexttest.LoadRepo(t, ctx, 4)

		ctx.SetFormString("status", "some-invalid-value")
		ListActionRuns(ctx)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		resp.Body.Reset()
		resp.Code = 200
		ctx.Req.Form.Del("status")

		ctx.SetFormString("status", "failure")
		ctx.SetFormString("limit", "1")
		ctx.SetFormString("page", "1")

		ListActionRuns(ctx)
		assert.Equal(t, http.StatusOK, resp.Code)
		contexttest.DecodeJSON(t, resp, &runResp)
		assert.Len(t, runResp.Entries, 1)
		assert.Equal(t, int64(2), runResp.TotalCount)

		ctx.SetFormString("workflow_id", "some.yml")
		resp.Body.Reset()
		resp.Code = 200
		ListActionRuns(ctx)
		assert.Equal(t, http.StatusOK, resp.Code)
		contexttest.DecodeJSON(t, resp, &runResp)
		assert.Empty(t, runResp.Entries)
		assert.Equal(t, int64(0), runResp.TotalCount)
	})

	t.Run("ListActionTasks", func(t *testing.T) {
		var taskResp api.ActionTaskResponse
		ctx, resp := contexttest.MockAPIContext(t, "user5/repo4/actions/tasks")
		contexttest.LoadRepo(t, ctx, 4)

		ctx.SetFormString("status", "some-invalid-value")
		ListActionTasks(ctx)
		assert.Equal(t, http.StatusBadRequest, resp.Code)
		resp.Body.Reset()
		resp.Code = 200

		ctx.SetFormString("status", "failure")
		ctx.SetFormString("limit", "1")
		ctx.SetFormString("page", "1")

		ListActionTasks(ctx)
		assert.Equal(t, http.StatusOK, resp.Code)
		contexttest.DecodeJSON(t, resp, &taskResp)
		assert.Len(t, taskResp.Entries, 1)
		assert.Equal(t, int64(2), taskResp.TotalCount)
	})
}
