// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"
	"strconv"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	"forgejo.org/models/unittest"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToggleOrgMemberVisibility(t *testing.T) {
	// Test cases for toggling organization member visibility from user settings
	testCases := []struct {
		name           string
		userID         int64  // The logged-in user
		orgID          int64  // The organization
		initialPublic  bool   // Initial visibility state
		action         string // "public" or "private"
		expectedPublic bool   // Expected visibility after action
	}{
		{
			name:           "Make member visible",
			userID:         4,
			orgID:          3,
			initialPublic:  false,
			action:         "public",
			expectedPublic: true,
		},
		{
			name:           "Make member hidden",
			userID:         2,
			orgID:          3,
			initialPublic:  true,
			action:         "private",
			expectedPublic: false,
		},
		{
			name:           "Make already visible member visible again",
			userID:         2,
			orgID:          3,
			initialPublic:  true,
			action:         "public",
			expectedPublic: true,
		},
		{
			name:           "Make already hidden member hidden again",
			userID:         4,
			orgID:          3,
			initialPublic:  false,
			action:         "private",
			expectedPublic: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unittest.PrepareTestEnv(t)

			// Set up initial visibility state
			err := organization.ChangeOrgUserStatus(db.DefaultContext, tc.orgID, tc.userID, tc.initialPublic)
			require.NoError(t, err)

			// Verify initial state
			orgUser := unittest.AssertExistsAndLoadBean(t, &organization.OrgUser{OrgID: tc.orgID, UID: tc.userID})
			assert.Equal(t, tc.initialPublic, orgUser.IsPublic)

			// Create mock context for the user
			ctx, _ := contexttest.MockContext(t, "/org/org3/members/action/"+tc.action)
			contexttest.LoadUser(t, ctx, tc.userID)

			// Set up the organization context
			org, err := organization.GetOrgByID(db.DefaultContext, tc.orgID)
			require.NoError(t, err)
			ctx.Org = &context.Organization{
				Organization: org,
				IsOwner:      tc.userID == 2, // user2 is owner of org3
			}

			// Set up the request parameters
			ctx.Req.Form.Set("uid", strconv.FormatInt(tc.userID, 10))
			ctx.SetParams(":action", tc.action)

			// Call the handler function that toggles visibility
			MembersAction(ctx)

			// Verify the response status
			assert.Equal(t, http.StatusSeeOther, ctx.Resp.Status())

			// Verify the database was updated correctly
			orgUser = unittest.AssertExistsAndLoadBean(t, &organization.OrgUser{OrgID: tc.orgID, UID: tc.userID})
			assert.Equal(t, tc.expectedPublic, orgUser.IsPublic)
		})
	}
}
