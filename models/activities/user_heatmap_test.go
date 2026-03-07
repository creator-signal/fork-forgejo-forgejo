// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activities_test

import (
	"testing"
	"time"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserHeatmapDataByUser(t *testing.T) {
	testCases := []struct {
		desc        string
		userID      int64
		doerID      int64
		CountResult int
		JSONResult  string
	}{
		{
			"self looks at action in private repo",
			2, 2, 1, `[{"timestamp":1603227600,"contributions":1}]`,
		},
		{
			"admin looks at action in private repo",
			2, 1, 1, `[{"timestamp":1603227600,"contributions":1}]`,
		},
		{
			"other user looks at action in private repo",
			2, 3, 0, `[]`,
		},
		{
			"nobody looks at action in private repo",
			2, 0, 0, `[]`,
		},
		{
			"collaborator looks at action in private repo",
			16, 15, 1, `[{"timestamp":1603267200,"contributions":1}]`,
		},
		{
			"no action action not performed by target user",
			3, 3, 0, `[]`,
		},
		{
			"multiple actions performed with two grouped together",
			10, 10, 3, `[{"timestamp":1603009800,"contributions":1},{"timestamp":1603010700,"contributions":2}]`,
		},
		{
			"test cutoff within",
			40, 40, 1, `[{"timestamp":1577404800,"contributions":1}]`,
		},
	}
	// Prepare
	require.NoError(t, unittest.PrepareTestDatabase())

	// Mock time
	timeutil.MockSet(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
	defer timeutil.MockUnset()

	for _, tc := range testCases {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: tc.userID})

		doer := &user_model.User{ID: tc.doerID}
		_, err := unittest.LoadBeanIfExists(doer)
		require.NoError(t, err)
		if tc.doerID == 0 {
			doer = nil
		}

		// get the action for comparison
		actions, count, err := activities_model.GetFeeds(db.DefaultContext, activities_model.GetFeedsOptions{
			RequestedUser:   user,
			Actor:           doer,
			IncludePrivate:  true,
			OnlyPerformedBy: true,
		})
		require.NoError(t, err)

		// Get the heatmap and compare
		heatmap, err := activities_model.GetUserHeatmapDataByUser(db.DefaultContext, user, doer, 0)
		var contributions int
		for _, hm := range heatmap {
			contributions += int(hm.Contributions)
		}
		require.NoError(t, err)
		assert.Len(t, actions, contributions, "invalid action count: did the test data became too old?")
		assert.Equal(t, count, int64(contributions))
		assert.Equal(t, tc.CountResult, contributions, tc.desc)

		// Test JSON rendering
		jsonData, err := json.Marshal(heatmap)
		require.NoError(t, err)
		assert.JSONEq(t, tc.JSONResult, string(jsonData))
	}
}

func TestGetUserHeatmapDataYear(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 34})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// Add actions in different years
	action2024 := &activities_model.Action{
		UserID:    user.ID,
		ActUserID: user.ID,
		RepoID:    repo.ID,
		IsPrivate: false,
		OpType:    activities_model.ActionCreateRepo,
	}
	require.NoError(t, db.Insert(db.DefaultContext, action2024))
	_, err := db.GetEngine(db.DefaultContext).Exec("UPDATE action SET created_unix = ? WHERE id = ?", time.Date(2024, 6, 1, 12, 0, 0, 0, setting.DefaultUILocation).Unix(), action2024.ID)
	require.NoError(t, err)

	action2025 := &activities_model.Action{
		UserID:    user.ID,
		ActUserID: user.ID,
		RepoID:    repo.ID,
		IsPrivate: false,
		OpType:    activities_model.ActionCreateRepo,
	}
	require.NoError(t, db.Insert(db.DefaultContext, action2025))
	_, err = db.GetEngine(db.DefaultContext).Exec("UPDATE action SET created_unix = ? WHERE id = ?", time.Date(2025, 6, 1, 12, 0, 0, 0, setting.DefaultUILocation).Unix(), action2025.ID)
	require.NoError(t, err)

	// rounded expected timestamps
	ts2024 := time.Date(2024, 6, 1, 12, 0, 0, 0, setting.DefaultUILocation).Unix() / 900 * 900
	ts2025 := time.Date(2025, 6, 1, 12, 0, 0, 0, setting.DefaultUILocation).Unix() / 900 * 900

	// Test heatmap for 2024
	heatmap, err := activities_model.GetUserHeatmapDataByUser(db.DefaultContext, user, admin, 2024)
	require.NoError(t, err)
	assert.Len(t, heatmap, 1)
	assert.EqualValues(t, ts2024, heatmap[0].Timestamp)
	assert.EqualValues(t, 1, heatmap[0].Contributions)

	// Test heatmap for 2025
	heatmap, err = activities_model.GetUserHeatmapDataByUser(db.DefaultContext, user, admin, 2025)
	require.NoError(t, err)
	assert.Len(t, heatmap, 1)
	assert.EqualValues(t, ts2025, heatmap[0].Timestamp)
	assert.EqualValues(t, 1, heatmap[0].Contributions)

	// Test heatmap for 2022 (empty)
	heatmap, err = activities_model.GetUserHeatmapDataByUser(db.DefaultContext, user, admin, 2022)
	require.NoError(t, err)
	assert.Empty(t, heatmap)
}

