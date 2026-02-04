// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strconv"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestRepoStatuses(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeWriteRepository)

	// check test data in models/fixtures/commit_status.yml
	// the index values map to the CommitStatus id
	cases := []struct {
		name          string
		query         string
		expectedIDs   []int64
		expectedTotal int8
	}{{
		name:          "All statuses without limit",
		query:         "/user2/repo1/statuses/1234123412341234123412341234123412341234",
		expectedIDs:   []int64{1, 2, 3, 4, 5, 6},
		expectedTotal: 6,
	},
		{
			name:          "With limit but without page",
			query:         "/user2/repo1/statuses/1234123412341234123412341234123412341234?limit=3",
			expectedIDs:   []int64{1, 2, 3},
			expectedTotal: 6,
		},
		{
			name:          "With limit and page",
			query:         "/user2/repo1/statuses/1234123412341234123412341234123412341234?limit=2&page=3",
			expectedIDs:   []int64{5, 6},
			expectedTotal: 6,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			req := NewRequestf(
				t,
				"GET",
				"/api/v1/repos/%s",
				tt.query,
			).AddTokenAuth(token)

			res := MakeRequest(t, req, http.StatusOK)

			statuses := make([]*api.CommitStatus, 0, len(tt.expectedIDs))

			DecodeJSON(t, res, &statuses)

			resultIDs := make([]int64, len(statuses))
			for i, status := range statuses {
				resultIDs[i] = status.ID
			}

			assert.ElementsMatch(t, tt.expectedIDs, resultIDs)

			if len(tt.expectedIDs) != int(tt.expectedTotal) {
				assert.NotEmpty(t, res.Header().Get("Link"))
			}
			assert.NotEmpty(t, res.Header().Get("X-Total-Count"))
			assert.Equal(t, strconv.Itoa(int(tt.expectedTotal)), res.Header().Get("X-Total-Count"))
		})
	}

}
