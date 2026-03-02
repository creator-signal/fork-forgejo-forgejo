// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWechatworkPayload(t *testing.T) {
	wc := wechatworkConvertor{}

	t.Run("WorkflowJob", func(t *testing.T) {
		p := workflowJobTestPayload()

		pl, err := wc.WorkflowJob(p)
		require.NoError(t, err)

		assert.Equal(t, "Workflow Job queued: test-job(#1)[2020558fe2]:waiting by user1", pl.Markdown.Content)
	})
}
