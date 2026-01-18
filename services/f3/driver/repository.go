// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	repo_model "forgejo.org/models/repo"

	"code.forgejo.org/f3/gof3/v3/f3"
	helpers_repository "code.forgejo.org/f3/gof3/v3/forges/helpers/repository"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
)

var _ f3_tree.ForgeDriverInterface = &repository{}

type repository struct {
	common

	h helpers_repository.Interface
	f *f3.Repository
}

func (o *repository) SetNative(repository any) {
	o.f = repository.(*f3.Repository).Clone().(*f3.Repository)
}

func (o *repository) GetNativeID() string {
	return o.f.GetID()
}

func (o *repository) NewFormat() f3.Interface {
	return (&f3.Repository{}).Init()
}

func (o *repository) ToFormat() f3.Interface {
	return (&f3.Repository{
		Common:    f3.NewCommon(o.GetNativeID()),
		FetchFunc: o.f.FetchFunc,
	}).Init()
}

func (o *repository) FromFormat(content f3.Interface) {
	o.f = content.Clone().(*f3.Repository)
}

func (o *repository) Get(ctx context.Context) bool {
	return o.h.Get(ctx)
}

func (o *repository) setIsEmpty(ctx context.Context) {
	repoID := f3_tree.GetProjectID(o.GetNode())
	repo, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		panic(fmt.Errorf("GetRepositoryByID(%v): %w", repoID, err))
	}
	if repo.IsEmpty {
		repo.IsEmpty = false
		if err = repo_model.UpdateRepositoryCols(ctx, repo, "is_empty"); err != nil {
			panic(fmt.Errorf("UpdateRepositoryCols(%s): IsEmpty = false: %w", repo.Name, err))
		}
	}
}

func (o *repository) Put(ctx context.Context) f3_id.NodeID {
	id := o.upsert(ctx)
	o.setIsEmpty(ctx)
	return id
}

func (o *repository) Patch(ctx context.Context) {
	o.upsert(ctx)
	o.setIsEmpty(ctx)
}

func (o *repository) upsert(ctx context.Context) f3_id.NodeID {
	o.h.Upsert(ctx, o.f)
	o.Trace("repository created %s", o.f.GetID())
	return f3_id.NewNodeID(o.f.GetID())
}

func (o *repository) SetFetchFunc(fetchFunc func(ctx context.Context, destination, internalRef string)) {
	o.f.FetchFunc = fetchFunc
}

const RepositoryNameWiki = "vcs.wiki"

func (o *repository) getURL() string {
	owner := f3_tree.GetOwnerName(o.GetNode())
	repoName := f3_tree.GetProjectName(o.GetNode())
	if o.f.GetID() == RepositoryNameWiki {
		repoName += ".wiki"
	}
	return repo_model.RepoPath(owner, repoName)
}

func (o *repository) GetRepositoryURL() string {
	return o.getURL()
}

func (o *repository) Delete(ctx context.Context) {
	o.Trace("ignore attempt to delete repository")
}

func (o *repository) GetRepositoryPushURL() string {
	return o.getURL()
}

func (o *repository) GetRepositoryInternalRef() string {
	return "refs/pull/*"
}

func (o *repository) GetPullRequestBranch(pr *f3.PullRequestBranch) *f3.PullRequestBranch {
	panic("")
}
func (o *repository) CreatePullRequestBranch(pr *f3.PullRequestBranch) {}
func (o *repository) DeletePullRequestBranch(pr *f3.PullRequestBranch) {}

func newRepository(_ context.Context) generic.NodeDriverInterface {
	r := &repository{
		f: (&f3.Repository{}).Init(),
	}
	r.h = helpers_repository.NewHelper(r)
	return r
}
