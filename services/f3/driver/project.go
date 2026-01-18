// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	f3_resource_model "forgejo.org/models/f3/resource"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	notify_service "forgejo.org/services/notify"
	repo_service "forgejo.org/services/repository"

	"code.forgejo.org/f3/gof3/v3/f3"
	"code.forgejo.org/f3/gof3/v3/f3/markdown"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
	f3_http "code.forgejo.org/f3/gof3/v3/util/http"
)

var _ f3_tree.ForgeDriverInterface = &project{}

type project struct {
	common

	forgejoProject *repo_model.Repository
	forked         f3.Reference
	avatar         string
}

func (o *project) SetNative(project any) {
	o.forgejoProject = project.(*repo_model.Repository)
}

func (o *project) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoProject.ID)
}

func (o *project) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *project) setForkedReference(ctx context.Context) {
	if !o.forgejoProject.IsFork {
		return
	}

	if err := o.forgejoProject.GetBaseRepo(ctx); err != nil {
		panic(fmt.Errorf("GetBaseRepo %v %w", o.forgejoProject, err))
	}
	forkParent := o.forgejoProject.BaseRepo
	if err := forkParent.LoadOwner(ctx); err != nil {
		panic(fmt.Errorf("LoadOwner %v %w", forkParent, err))
	}
	owners := "users"
	if forkParent.Owner.IsOrganization() {
		owners = "organizations"
	}

	o.forked = f3_tree.NewForkReference(owners, fmt.Sprintf("%d", forkParent.Owner.ID), fmt.Sprintf("%d", forkParent.ID))
}

func (o *project) ToFormat() f3.Interface {
	if o.forgejoProject == nil {
		return o.NewFormat()
	}
	return (&f3.Project{
		Common:        f3.NewCommon(fmt.Sprintf("%d", o.forgejoProject.ID)),
		Name:          o.forgejoProject.Name,
		IsPrivate:     o.forgejoProject.IsPrivate,
		IsMirror:      o.forgejoProject.IsMirror,
		Description:   markdown.NewContent().Set(o.forgejoProject.Description),
		DefaultBranch: o.forgejoProject.DefaultBranch,
		Avatar:        o.avatar,
		Forked:        o.forked,
	}).Init()
}

func (o *project) FromFormat(content f3.Interface) {
	project := content.(*f3.Project)
	o.forgejoProject = &repo_model.Repository{
		ID:            f3_util.ParseInt(project.GetID()),
		Name:          project.Name,
		IsPrivate:     project.IsPrivate,
		IsMirror:      project.IsMirror,
		Description:   project.Description.Get(),
		DefaultBranch: project.DefaultBranch,
	}
	if project.Forked != nil {
		o.forgejoProject.IsFork = true
		o.forgejoProject.ForkID = project.Forked.GetIDAsInt()
	}
	o.forked = project.Forked
	o.avatar = project.Avatar
}

func (o *project) getProjectAvatar(ctx context.Context) string {
	url := o.forgejoProject.AvatarLink(ctx)
	if url != "" {
		return f3.AvatarEncode(f3_http.GetBody(&http.Client{}, url))
	}
	return ""
}

func (o *project) setProjectAvatar(ctx context.Context) {
	if o.avatar == "" {
		if err := repo_service.DeleteAvatar(ctx, o.forgejoProject); err != nil {
			panic(fmt.Errorf("repo_service.DeleteAvatar(%+v): %w", o.forgejoProject, err))
		}
		return
	}
	avatar := f3.AvatarDecode(o.avatar)
	if o.forgejoProject.Avatar != "" && o.avatarIsUpToDate(ctx, o.forgejoProject.Avatar, o.forgejoProject.ID, avatar) {
		return
	}
	if err := repo_service.UploadAvatar(ctx, o.forgejoProject, avatar); err != nil {
		panic(fmt.Errorf("repo_service.UploadAvatar(%+v): %w", o.forgejoProject, err))
	}
}

func (o *project) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())
	id := node.GetID().Int64()
	p, err := repo_model.GetRepositoryByID(ctx, id)
	if repo_model.IsErrRepoNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("project %v %w", id, err))
	}
	o.forgejoProject = p
	o.setForkedReference(ctx)
	o.avatar = o.getProjectAvatar(ctx)
	return true
}

func (o *project) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoProject.ID)
	o.forgejoProject.LowerName = strings.ToLower(o.forgejoProject.Name)
	if err := repo_model.UpdateRepositoryCols(ctx, o.forgejoProject,
		"description",
		"name",
		"lower_name",
	); err != nil {
		panic(fmt.Errorf("UpdateRepositoryCols: %v %v", o.forgejoProject, err))
	}
	o.setProjectAvatar(ctx)
}

func (o *project) Put(ctx context.Context) f3_id.NodeID {
	ownerID := f3_tree.GetOwnerID(o.GetNode())
	owner, err := user_model.GetUserByID(ctx, ownerID)
	if err != nil {
		panic(fmt.Errorf("GetUserByID %v %w", ownerID, err))
	}
	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}

	if o.forked.IsNil() {
		repo, err := repo_service.CreateRepositoryDirectly(ctx, doer, owner, repo_service.CreateRepoOptions{
			Name:          o.forgejoProject.Name,
			Description:   o.forgejoProject.Description,
			IsPrivate:     o.forgejoProject.IsPrivate,
			DefaultBranch: o.forgejoProject.DefaultBranch,
		})
		if err != nil {
			panic(err)
		}
		o.forgejoProject = repo
		o.Trace("project created %s/%s %d", owner.Name, o.forgejoProject.Name, o.forgejoProject.ID)
	} else {
		if err = o.forgejoProject.GetBaseRepo(ctx); err != nil {
			panic(fmt.Errorf("GetBaseRepo %v %w", o.forgejoProject, err))
		}
		if err = o.forgejoProject.BaseRepo.LoadOwner(ctx); err != nil {
			panic(fmt.Errorf("LoadOwner %v %w", o.forgejoProject.BaseRepo, err))
		}

		repo, err := repo_service.ForkRepositoryIfNotExists(ctx, doer, owner, repo_service.ForkRepoOptions{
			BaseRepo:    o.forgejoProject.BaseRepo,
			Name:        o.forgejoProject.Name,
			Description: o.forgejoProject.Description,
		})
		if err != nil {
			panic(err)
		}
		o.forgejoProject = repo
		o.Trace("project created (fork) %s/%s %d", owner.Name, o.forgejoProject.Name, o.forgejoProject.ID)
	}
	_, err = f3_resource_model.Upsert(ctx, f3_resource_model.NewResource(o.getForgejoForgeID(ctx), o.forgejoProject.ID, f3_resource_model.KindRepository))
	if err != nil {
		panic(err)
	}
	o.setProjectAvatar(ctx)
	if o.sendNotifications(ctx) {
		notify_service.CreateRepository(ctx, doer, owner, o.forgejoProject)
	}
	return f3_id.NewNodeID(o.forgejoProject.ID)
}

func (o *project) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	doer, err := user_model.GetAdminUser(ctx)
	if err != nil {
		panic(fmt.Errorf("GetAdminUser %w", err))
	}

	if err := repo_service.DeleteRepository(ctx, doer, o.forgejoProject, true); err != nil {
		panic(err)
	}
}

func newProject() generic.NodeDriverInterface {
	return &project{
		forked: f3.NewForkReference(""),
	}
}
