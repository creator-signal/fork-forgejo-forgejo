// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/timeutil"
	issue_service "forgejo.org/services/issue"
	notify_service "forgejo.org/services/notify"
	pull_service "forgejo.org/services/pull"

	"code.forgejo.org/f3/gof3/v3/f3"
	"code.forgejo.org/f3/gof3/v3/f3/markdown"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &pullRequest{}

type pullRequest struct {
	common

	forgejoPullRequest *issues_model.Issue
	headRepository     f3.Reference
	baseRepository     f3.Reference
	fetchFunc          f3.PullRequestFetchFunc
}

func (o *pullRequest) SetNative(pullRequest any) {
	o.forgejoPullRequest = pullRequest.(*issues_model.Issue)
}

func (o *pullRequest) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoPullRequest.Index)
}

func (o *pullRequest) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *pullRequest) repositoryToReference(ctx context.Context, repository *repo_model.Repository) f3.Reference {
	if repository == nil {
		panic("unexpected nil repository")
	}
	forge := o.getTree().GetRoot().GetChild(f3_id.NewNodeID(f3_kind.KindForge)).GetDriver().(*forge)
	owners := forge.getOwnersPath(ctx, fmt.Sprintf("%d", repository.OwnerID))
	return f3_tree.NewRepositoryReference(owners.String(), f3_util.ToString(repository.OwnerID), f3_util.ToString(repository.ID), f3.RepositoryNameDefault)
}

func (o *pullRequest) referenceToRepository(reference f3.Reference) int64 {
	var project int64
	if reference.Get() == "../../repositories/vcs" {
		project = f3_tree.GetProjectID(o.GetNode())
	} else {
		p := generic.PathAbsolute(o.GetNode().GetCurrentPath().String(), reference.Get())
		o.Trace("%v %v", o.GetNode().GetCurrentPath().String(), p)
		_, project = p.OwnerAndProjectID()
	}
	return project
}

func (o *pullRequest) relativeRepositoryReference(reference f3.Reference) f3.Reference {
	s := reference.Get()
	if !strings.HasPrefix(s, ".") {
		project := f3_tree.GetProject(o.GetNode())
		projectPath := project.GetCurrentPath().String()
		if strings.HasPrefix(s, projectPath) {
			s = "../../repositories/vcs"
		}
	}
	return f3.NewRepositoryReference(s)
}

func makePullRequestBranch(_ context.Context, repo *repo_model.Repository, refName string) f3.PullRequestBranch {
	r, err := git.OpenRepository(context.Background(), repo.RepoPath())
	if err != nil {
		panic(err)
	}
	defer r.Close()

	ref := git.RefName(refName)
	if ref.IsPull() {
		ref = git.RefName(f3_tree.PullRequestIDToF3Ref(ref.PullName()))
	} else {
		ref = git.RefNameFromBranch(refName)
	}

	sha, err := r.GetRefCommitID(ref.String())
	if err != nil {
		panic(fmt.Errorf("%s: %v", repo.RepoPath(), err))
	}

	return f3.PullRequestBranch{
		Ref: refName,
		SHA: sha,
	}
}

func makeHeadPullRequestBranch(ctx context.Context, pr *issues_model.PullRequest, refName string) f3.PullRequestBranch {
	if git.RefName(refName).IsPull() {
		// that happens, for instance, when the branch of the head repository is deleted
		return makePullRequestBranch(ctx, pr.BaseRepo, refName)
	}
	return makePullRequestBranch(ctx, pr.HeadRepo, refName)
}

func (o *pullRequest) ToFormat() f3.Interface {
	if o.forgejoPullRequest == nil {
		return o.NewFormat()
	}

	milestone := f3.NewMilestoneReference("")
	if o.forgejoPullRequest.Milestone != nil && o.forgejoPullRequest.Milestone.ID != 0 {
		milestone = f3_tree.NewIssueMilestoneReference(f3_util.ToString(o.forgejoPullRequest.Milestone.ID))
	}

	var mergedTime *time.Time
	if o.forgejoPullRequest.PullRequest.HasMerged {
		mergedTime = o.forgejoPullRequest.PullRequest.MergedUnix.AsTimePtr()
	}

	var closedTime *time.Time
	if o.forgejoPullRequest.IsClosed {
		closedTime = o.forgejoPullRequest.ClosedUnix.AsTimePtr()
	}

	if err := o.forgejoPullRequest.PullRequest.LoadHeadRepo(db.DefaultContext); err != nil {
		panic(err)
	}

	if err := o.forgejoPullRequest.PullRequest.LoadBaseRepo(db.DefaultContext); err != nil {
		panic(err)
	}
	base := makePullRequestBranch(context.Background(), o.forgejoPullRequest.PullRequest.BaseRepo, o.forgejoPullRequest.PullRequest.BaseBranch)
	base.Repository = o.relativeRepositoryReference(o.baseRepository)

	head := makeHeadPullRequestBranch(context.Background(), o.forgejoPullRequest.PullRequest, o.forgejoPullRequest.PullRequest.HeadBranch)
	head.Repository = o.relativeRepositoryReference(o.headRepository)

	return (&f3.PullRequest{
		Common:         f3.NewCommon(o.GetNativeID()),
		PosterID:       f3_tree.NewUserReference(f3_util.ToString(o.forgejoPullRequest.Poster.ID)),
		Title:          o.forgejoPullRequest.Title,
		Content:        markdown.NewContent().Set(o.forgejoPullRequest.Content),
		Milestone:      milestone,
		State:          string(o.forgejoPullRequest.State()),
		IsLocked:       o.forgejoPullRequest.IsLocked,
		Created:        o.forgejoPullRequest.CreatedUnix.AsTime(),
		Updated:        o.forgejoPullRequest.UpdatedUnix.AsTime(),
		Closed:         closedTime,
		Merged:         o.forgejoPullRequest.PullRequest.HasMerged,
		MergedTime:     mergedTime,
		MergeCommitSHA: o.forgejoPullRequest.PullRequest.MergedCommitID,
		Head:           head,
		Base:           base,
		FetchFunc:      o.fetchFunc,
	}).Init()
}

func (o *pullRequest) FromFormat(content f3.Interface) {
	pullRequest := content.(*f3.PullRequest)
	var milestone *issues_model.Milestone
	if pullRequest.Milestone != nil {
		milestone = &issues_model.Milestone{
			ID: pullRequest.Milestone.GetIDAsInt(),
		}
	}

	o.headRepository = f3.NewRepositoryReference(pullRequest.Head.Repository.Get())
	o.baseRepository = f3.NewRepositoryReference(pullRequest.Base.Repository.Get())
	pr := issues_model.PullRequest{
		HeadBranch:   pullRequest.Head.Ref,
		HeadRepoID:   o.referenceToRepository(o.headRepository),
		HeadCommitID: pullRequest.Head.SHA,
		BaseBranch:   pullRequest.Base.Ref,
		BaseRepoID:   o.referenceToRepository(o.baseRepository),

		MergeBase: pullRequest.Base.SHA,
		Index:     f3_util.ParseInt(pullRequest.GetID()),
		HasMerged: pullRequest.Merged,
	}

	o.forgejoPullRequest = &issues_model.Issue{
		Index:    f3_util.ParseInt(pullRequest.GetID()),
		PosterID: pullRequest.PosterID.GetIDAsInt(),
		Poster: &user_model.User{
			ID: pullRequest.PosterID.GetIDAsInt(),
		},
		Title:       pullRequest.Title,
		Content:     pullRequest.Content.Get(),
		Milestone:   milestone,
		IsClosed:    pullRequest.State == f3.PullRequestStateClosed,
		CreatedUnix: timeutil.TimeStamp(pullRequest.Created.Unix()),
		UpdatedUnix: timeutil.TimeStamp(pullRequest.Updated.Unix()),
		IsLocked:    pullRequest.IsLocked,
		PullRequest: &pr,
		IsPull:      true,
	}

	if pullRequest.Closed != nil {
		o.forgejoPullRequest.ClosedUnix = timeutil.TimeStamp(pullRequest.Closed.Unix())
	}
}

func (o *pullRequest) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())
	id := node.GetID().Int64()

	issue, err := issues_model.GetIssueByIndex(ctx, project, id)
	if issues_model.IsErrIssueNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("issue %v %w", id, err))
	}

	o.forgejoPullRequest = issue
	o.loadAttributes(ctx)

	o.Trace("ID = %v", o.forgejoPullRequest.ID)
	return true
}

func (o *pullRequest) loadAttributes(ctx context.Context) {
	if err := o.forgejoPullRequest.LoadAttributes(ctx); err != nil {
		panic(err)
	}
	if err := o.forgejoPullRequest.PullRequest.LoadHeadRepo(ctx); err != nil {
		panic(err)
	}
	o.headRepository = o.repositoryToReference(ctx, o.forgejoPullRequest.PullRequest.HeadRepo)
	if err := o.forgejoPullRequest.PullRequest.LoadBaseRepo(ctx); err != nil {
		panic(err)
	}
	o.baseRepository = o.repositoryToReference(ctx, o.forgejoPullRequest.PullRequest.BaseRepo)
}

func (o *pullRequest) Patch(ctx context.Context) {
	node := o.GetNode()
	project := f3_tree.GetProjectID(o.GetNode())
	id := node.GetID().Int64()
	o.Trace("repo_id = %d, index = %d", project, id)
	if _, err := db.GetEngine(ctx).Where("`repo_id` = ? AND `index` = ?", project, id).Cols("name", "content", "updated").NoAutoTime().Update(o.forgejoPullRequest); err != nil {
		panic(fmt.Errorf("%v %v", o.forgejoPullRequest, err))
	}
}

func (o *pullRequest) GetPullRequestHead(ctx context.Context) string {
	return o.forgejoPullRequest.PullRequest.HeadBranch
}

func (o *pullRequest) GetPullRequestRef() string {
	return fmt.Sprintf("%s/%s/head", git.PullPrefix, o.GetNativeID())
}

var PullRequestAddToQueue = func(ctx context.Context, pr *issues_model.PullRequest) {
	pull_service.AddToTaskQueue(ctx, pr)
}

func (o *pullRequest) Put(ctx context.Context) f3_id.NodeID {
	o.forgejoPullRequest.RepoID = f3_tree.GetProjectID(o.GetNode())

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		panic(err)
	}
	defer committer.Close()

	idx, err := db.GetNextResourceIndex(ctx, "issue_index", o.forgejoPullRequest.RepoID)
	if err != nil {
		panic(fmt.Errorf("generate issue index failed: %w", err))
	}
	o.forgejoPullRequest.Index = idx

	sess := db.GetEngine(ctx)

	if _, err = sess.NoAutoTime().Insert(o.forgejoPullRequest); err != nil {
		panic(err)
	}

	pr := o.forgejoPullRequest.PullRequest
	pr.Index = o.forgejoPullRequest.Index
	pr.IssueID = o.forgejoPullRequest.ID
	pr.HeadRepoID = o.referenceToRepository(o.headRepository)
	if pr.HeadRepoID == 0 {
		panic(fmt.Errorf("HeadRepoID == 0 in %v", pr))
	}
	pr.BaseRepoID = o.referenceToRepository(o.baseRepository)
	if pr.BaseRepoID == 0 {
		panic(fmt.Errorf("BaseRepoID == 0 in %v", pr))
	}

	if _, err = sess.NoAutoTime().Insert(pr); err != nil {
		panic(err)
	}

	if err = committer.Commit(); err != nil {
		panic(fmt.Errorf("Commit: %w", err))
	}

	if err := pr.LoadBaseRepo(ctx); err != nil {
		panic(err)
	}
	if err := pr.LoadHeadRepo(ctx); err != nil {
		panic(err)
	}

	if git.IsBranchExist(ctx, pr.HeadRepo.RepoPath(), pr.HeadBranch) {
		if err := pull_service.PushToBaseRepo(ctx, pr); err != nil {
			panic(err)
		}
	} else {
		if err := pull_service.UpdateRef(ctx, pr); err != nil {
			panic(err)
		}
	}

	PullRequestAddToQueue(ctx, pr)

	o.Trace("pullRequest created %d/%d", o.forgejoPullRequest.ID, o.forgejoPullRequest.Index)
	if o.sendNotifications(ctx) {
		notify_service.NewPullRequest(ctx, o.forgejoPullRequest.PullRequest, nil)
	}
	return f3_id.NewNodeID(o.forgejoPullRequest.Index)
}

func (o *pullRequest) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	owner := f3_tree.GetOwnerName(o.GetNode())
	project := f3_tree.GetProjectName(o.GetNode())
	repoPath := repo_model.RepoPath(owner, project)
	gitRepo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		panic(err)
	}
	defer gitRepo.Close()

	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}

	if err := issue_service.DeleteIssue(ctx, doer, gitRepo, o.forgejoPullRequest); err != nil {
		panic(err)
	}
}

func newPullRequest() generic.NodeDriverInterface {
	return &pullRequest{}
}
