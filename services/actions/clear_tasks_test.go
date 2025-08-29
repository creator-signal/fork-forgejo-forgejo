// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/unittest"
	notify_service "forgejo.org/services/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelAbandonedJobs(t *testing.T) {
	defer unittest.OverrideFixtures("services/actions/TestCancelAbandonedJobs")()
	require.NoError(t, unittest.PrepareTestDatabase())

	notifier := &mockNotifier{}
	notify_service.RegisterNotifier(notifier)
	defer notify_service.UnregisterNotifier(notifier)

	require.NoError(t, CancelAbandonedJobs(t.Context()))

	// status waiting, too long, ready to be abandoned
	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 600})
	assert.Equal(t, actions_model.StatusCancelled, job.Status)

	// status blocked, too long, ready to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 601})
	assert.Equal(t, actions_model.StatusCancelled, job.Status)

	// status blocked, *not* too long, not to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 602})
	assert.Equal(t, actions_model.StatusBlocked, job.Status)

	// status running, not to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 603})
	assert.Equal(t, actions_model.StatusRunning, job.Status)

	// related run needs approval, not to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 604})
	assert.Equal(t, actions_model.StatusWaiting, job.Status)

	// WorkflowJobStatusUpdate is called once for each abandoned job (600 and 601)
	require.Len(t, notifier.workflowJobStatusCalls, 2)
	cancelledJobIDs := []int64{notifier.workflowJobStatusCalls[0].job.ID, notifier.workflowJobStatusCalls[1].job.ID}
	assert.ElementsMatch(t, []int64{600, 601}, cancelledJobIDs)
	assert.Nil(t, notifier.workflowJobStatusCalls[0].task)
	assert.Nil(t, notifier.workflowJobStatusCalls[1].task)
}
