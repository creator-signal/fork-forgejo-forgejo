// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	driver_options "forgejo.org/services/f3/driver/options"

	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	"code.forgejo.org/f3/gof3/v3/options"
	tests_forge "code.forgejo.org/f3/gof3/v3/tree/tests/f3/forge"
)

type forgeTest struct {
	tests_forge.Base
}

func (o *forgeTest) NewOptions(t *testing.T) options.Interface {
	return newTestOptions(t)
}

func (o *forgeTest) GetExceptions() []f3_kind.Kind {
	return []f3_kind.Kind{}
}

func (o *forgeTest) GetNonTestUsers() []string {
	return []string{
		"user1",
	}
}

func (o *forgeTest) GetF3PathFixtures() []tests_forge.F3PathFixture {
	return []tests_forge.F3PathFixture{
		{Path: "/forge/organizations"},
		{Path: "/forge/organizations/3330001/teams"},
		{Path: "/forge/users"},
		{Path: "/forge/users/10111/projects/74823/issues/1234567/attachments/939393"},
		{Path: "/forge/organizations/3330001"},
		{Path: "/forge/users/10111"},
		{
			Path: "/forge/users/10111/projects/74823/issues/1234567",
			URLAliases: func(url string) []string {
				return []string{
					url + "#issue-1234567",
					url + "#issue-1234567:",
				}
			},
		},
		{Path: "/forge/users/10111/projects/74823/pull_requests/2222"},
		{Path: "/forge/users/10111/projects/74823/issues/1234567/comments/1111999"},
		{Path: "/forge/users/10111/projects/74823/pull_requests/2222/reviews/4593/reviewcomments/9876543"},
	}
}

func newForgeTest() tests_forge.Interface {
	t := &forgeTest{}
	t.SetName(driver_options.Name)
	return t
}
