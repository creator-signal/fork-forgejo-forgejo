package source

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveMappedMemberships tests resolveMappedMemberships
func TestResolveMappedMemberships(t *testing.T) {
	type test struct {
		srcGroups  container.Set[string]
		mappings   map[string]map[string][]string
		removal    bool
		wantAdd    map[string][]string
		wantRemove map[string][]string
	}

	// get from test db:
	// test user with id 2 with memberships:
	// - "org3":  {"owners", "team1", "teamcreaterepo"},
	// - "org17": {"test_team"},
	require.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	ctx := db.DefaultContext
	for _, test := range []test{
		// add, static, no match
		{
			srcGroups: container.SetOf("does-not-matter"),
			mappings: map[string]map[string][]string{
				"test-static": {"static-org": {"static-team"}},
			},
			removal:    false,
			wantAdd:    map[string][]string{},
			wantRemove: map[string][]string{},
		},
		// add, static, match
		{
			srcGroups: container.SetOf("test-static"),
			mappings: map[string]map[string][]string{
				"test-static": {"static-org": {"static-team"}},
			},
			removal: false,
			wantAdd: map[string][]string{
				"static-org": {"static-team"},
			},
			wantRemove: map[string][]string{},
		},
		// add, dynamic, no match
		{
			srcGroups: container.SetOf("test-notmatching"),
			mappings: map[string]map[string][]string{
				"test-{org}-{team}": {"org-{org}": {"team-{team}"}},
			},
			removal:    false,
			wantAdd:    map[string][]string{},
			wantRemove: map[string][]string{},
		},
		// add, dynamic, match org
		{
			srcGroups: container.SetOf("test2-someorg-with-dashes-myteam"),
			mappings: map[string]map[string][]string{
				"test2-{org}-myteam": {"org-{org}": {"myteam"}},
			},
			removal: false,
			wantAdd: map[string][]string{
				"org-someorg-with-dashes": {"myteam"},
			},
			wantRemove: map[string][]string{},
		},
		// add, dynamic, match team
		{
			srcGroups: container.SetOf("test3-myorg-team-with-dashes"),
			mappings: map[string]map[string][]string{
				"test3-myorg-{team}": {"org-myorg": {"myteam-{team}"}},
			},
			removal: false,
			wantAdd: map[string][]string{
				"org-myorg": {"myteam-team-with-dashes"},
			},
			wantRemove: map[string][]string{},
		},
		// add, dynamic, match org and team
		{
			srcGroups: container.SetOf("test-myorg1-myteam1"),
			mappings: map[string]map[string][]string{
				"test-{org}-{team}": {"org-{org}": {"team-{team}"}},
			},
			removal: false,
			wantAdd: map[string][]string{
				"org-myorg1": {"team-myteam1"},
			},
			wantRemove: map[string][]string{},
		},
		// add, dynamic, multiple matches
		{
			srcGroups: container.SetOf(
				"does-not-matter",
				"test-myorg1-myteam1",
				"test-org2-team2",
				"whatevs",
				"test-static",
				"test2-someorg-with-dashes-myteam",
				"test3-myorg-team-with-dashes",
			),
			mappings: map[string]map[string][]string{
				"test-{org}-{team}":  {"org-{org}": {"team-{team}"}},
				"test2-{org}-myteam": {"org-{org}": {"myteam"}},
				"test3-myorg-{team}": {"org-myorg": {"myteam-{team}"}},
				"test-static":        {"static-org": {"static-team"}},
			},
			removal: false,
			wantAdd: map[string][]string{
				"org-myorg1":              {"team-myteam1"},
				"org-org2":                {"team-team2"},
				"static-org":              {"static-team"},
				"org-someorg-with-dashes": {"myteam"},
				"org-myorg":               {"myteam-team-with-dashes"},
			},
			wantRemove: map[string][]string{},
		},
		// remove, static, no match
		{
			srcGroups: container.SetOf("does-not-matter"),
			mappings: map[string]map[string][]string{
				"test-static": {"static-org": {"static-team"}},
			},
			removal: true,
			wantAdd: map[string][]string{},
			wantRemove: map[string][]string{
				"org17": {"test_team"},
				"org3":  {"owners", "team1", "teamcreaterepo"},
			},
		},
		// remove, static, match all
		{
			srcGroups: container.SetOf("org3", "org17"),
			mappings: map[string]map[string][]string{
				"org17": {"org17": {"test_team"}},
				"org3":  {"org3": {"owners", "team1", "teamcreaterepo"}},
			},
			removal: true,
			wantAdd: map[string][]string{
				"org17": {"test_team"},
				"org3":  {"owners", "team1", "teamcreaterepo"},
			},
			wantRemove: map[string][]string{},
		},
		// remove, static, match some
		{
			srcGroups: container.SetOf("org3", "org17"),
			mappings: map[string]map[string][]string{
				"org17": {"org17": {"test_team"}},
				"org3":  {"org3": {"owners", "teamcreaterepo"}},
			},
			removal: true,
			wantAdd: map[string][]string{
				"org17": {"test_team"},
				"org3":  {"owners", "teamcreaterepo"},
			},
			wantRemove: map[string][]string{
				"org3": {"team1"},
			},
		},
		// remove, dynamic, no match
		{
			srcGroups: container.SetOf("test-notmatching"),
			mappings: map[string]map[string][]string{
				"test-{org}-{team}": {"org-{org}": {"team-{team}"}},
			},
			removal: true,
			wantAdd: map[string][]string{},
			wantRemove: map[string][]string{
				"org17": {"test_team"},
				"org3":  {"owners", "team1", "teamcreaterepo"},
			},
		},
		// remove, dynamic, match all (org)
		{
			srcGroups: container.SetOf("test17-org17", "test3-org3"),
			mappings: map[string]map[string][]string{
				"test17-{org}": {"{org}": {"test_team"}},
				"test3-{org}":  {"{org}": {"owners", "team1", "teamcreaterepo"}},
			},
			removal: true,
			wantAdd: map[string][]string{
				"org17": {"test_team"},
				"org3":  {"owners", "team1", "teamcreaterepo"},
			},
			wantRemove: map[string][]string{},
		},
		// remove, dynamic, match some
		{
			srcGroups: container.SetOf("test17-org17", "test3-org3"),
			mappings: map[string]map[string][]string{
				"test17-{org}": {"{org}": {"test_team"}},
				"test3-{org}":  {"{org}": {"owners", "teamcreaterepo"}},
			},
			removal: true,
			wantAdd: map[string][]string{
				"org17": {"test_team"},
				"org3":  {"owners", "teamcreaterepo"},
			},
			wantRemove: map[string][]string{
				"org3": {"team1"},
			},
		},
	} {
		gotAdd, gotRemove := resolveMappedMemberships(ctx, user,
			test.srcGroups, test.mappings, test.removal)

		assert.Equal(t, gotAdd, test.wantAdd)
		assert.Equal(t, gotRemove, test.wantRemove)
	}
}
