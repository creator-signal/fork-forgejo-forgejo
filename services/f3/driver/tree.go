// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	f3_context "forgejo.org/services/f3/context"
	forgejo_options "forgejo.org/services/f3/driver/options"
	permissions_context "forgejo.org/services/permissions/context"

	f3_forgejo "code.forgejo.org/f3/gof3/v3/forges/forgejo"
	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

type treeDriver struct {
	generic.NullTreeDriver

	options           *forgejo_options.Options
	converterURLAndF3 f3_forgejo.ConverterURLAndF3
}

func (o *treeDriver) F3PathToURLPath(ctx context.Context, f3PathString string) string {
	return o.converterURLAndF3.F3PathToURLPath(ctx, f3PathString)
}

func (o *treeDriver) URLPathToF3Path(ctx context.Context, urlPath string) string {
	return o.converterURLAndF3.URLPathToF3Path(ctx, urlPath)
}

func (o *treeDriver) Init() {
	o.NullTreeDriver.Init()
}

func (o *treeDriver) Factory(ctx context.Context, kind f3_kind.Kind) generic.NodeDriverInterface {
	switch kind {
	case f3_kind.KindForge:
		return newForge()
	case f3_kind.KindOrganizations:
		return newOrganizations()
	case f3_kind.KindOrganization:
		return newOrganization()
	case f3_kind.KindUsers:
		return newUsers()
	case f3_kind.KindUser:
		return newUser()
	case f3_kind.KindProjects:
		return newProjects()
	case f3_kind.KindProject:
		return newProject()
	case f3_kind.KindIssues:
		return newIssues()
	case f3_kind.KindIssue:
		return newIssue()
	case f3_kind.KindComments:
		return newComments()
	case f3_kind.KindComment:
		return newComment()
	case f3_kind.KindAttachments:
		return newAttachments()
	case f3_kind.KindAttachment:
		return newAttachment()
	case f3_kind.KindLabels:
		return newLabels()
	case f3_kind.KindLabel:
		return newLabel()
	case f3_kind.KindReactions:
		return newReactions()
	case f3_kind.KindReaction:
		return newReaction()
	case f3_kind.KindReviews:
		return newReviews()
	case f3_kind.KindReview:
		return newReview()
	case f3_kind.KindReviewComments:
		return newReviewComments()
	case f3_kind.KindReviewComment:
		return newReviewComment()
	case f3_kind.KindMilestones:
		return newMilestones()
	case f3_kind.KindMilestone:
		return newMilestone()
	case f3_kind.KindPullRequests:
		return newPullRequests()
	case f3_kind.KindPullRequest:
		return newPullRequest()
	case f3_kind.KindReleases:
		return newReleases()
	case f3_kind.KindRelease:
		return newRelease()
	case f3_kind.KindTeams:
		return newTeams()
	case f3_kind.KindTeam:
		return newTeam()
	case f3_kind.KindTeamMembers:
		return newTeamMembers()
	case f3_kind.KindTeamMember:
		return newTeamMember()
	case f3_kind.KindTeamProjects:
		return newTeamProjects()
	case f3_kind.KindTeamProject:
		return newTeamProject()
	case f3_kind.KindTopics:
		return newTopics()
	case f3_kind.KindTopic:
		return newTopic()
	case f3_kind.KindRepositories:
		return newRepositories()
	case f3_kind.KindRepository:
		return newRepository(ctx)
	case f3_kind.KindRoot:
		return newRoot(o.GetTree().(f3_tree.TreeInterface).NewFormat(kind))
	default:
		panic(fmt.Errorf("unexpected kind %s", kind))
	}
}

func newTreeDriver(ctx context.Context, tree generic.TreeInterface, anyOptions any) generic.TreeDriverInterface {
	options := anyOptions.(*forgejo_options.Options)
	converterURLAndF3 := f3_forgejo.NewConverterURLAndF3()
	converterURLAndF3.SetTree(tree)

	converterUUIDAndAttachment := NewConverterUUIDAndAttachment()
	converterURLAndF3.SetConverterUUIDAndAttachment(converterUUIDAndAttachment)
	converterURLAndF3.SetBuildOwnerPath(buildOwnerPath)
	converterURLAndF3.SetBuildProjectPath(buildProjectPath)
	converterURLAndF3.SetGetReviewIDByReviewCommentID(getReviewIDByReviewCommentID)

	f3Ctx := f3_context.Get(ctx)
	if err := permissions_context.SetToken(f3Ctx, options.GetToken()); err != nil {
		panic(fmt.Errorf("permissions_context.SetToken: %v", err))
	}

	driver := &treeDriver{
		options:           options,
		converterURLAndF3: converterURLAndF3,
	}
	driver.Init()
	return driver
}
