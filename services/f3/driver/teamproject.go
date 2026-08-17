// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	repo_service "forgejo.org/services/repository"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &teamProject{}

type forgejoTeamRepo struct {
	ID     int64
	Name   string
	OrgID  int64
	RepoID int64
}

func getForgejoTeamRepoByID(ctx context.Context, id int64) *forgejoTeamRepo {
	var repo forgejoTeamRepo
	has, err := db.GetEngine(ctx).
		Select("`team_repo`.`id` as `id`, `repository`.`name` as `name`, `repository`.`owner_id` as `org_id`, `repository`.`id` as `repo_id`").
		Table("`team_repo`").
		Join("INNER", "`repository`", "`repository`.`id` = `team_repo`.`repo_id`").
		Where("`team_repo`.`id` = ?", id).
		Get(&repo)
	if err != nil {
		panic(fmt.Errorf("looking for repo %v: %w", id, err))
	}
	if !has {
		return nil
	}
	return &repo
}

func getForgejoTeamRepoByTeamIDAndName(ctx context.Context, teamID int64, name string) *forgejoTeamRepo {
	var repo forgejoTeamRepo
	has, err := db.GetEngine(ctx).
		Select("`team_repo`.`id` as `id`, `repository`.`name` as `name`, `repository`.`owner_id` as `org_id`, `repository`.`id` as `repo_id`").
		Table("`repository`").
		Join("INNER", "`team_repo`", "`team_repo`.`repo_id` = `repository`.`id`").
		Where("`repository`.`name` = ?", name).
		And("`team_repo`.`team_id` = ?", teamID).
		Get(&repo)
	if err != nil {
		panic(fmt.Errorf("looking for repo %v: %w", name, err))
	}
	if !has {
		return nil
	}
	return &repo
}

func getForgejoTeamReposByTeamID(ctx context.Context, teamID int64, pageSize, page int) []*forgejoTeamRepo {
	var repos []*forgejoTeamRepo
	sess := db.GetEngine(ctx)
	if pageSize > 0 && page > 0 {
		sess = sess.Limit(pageSize, (page-1)*pageSize)
	}
	if err := sess.Select("`team_repo`.`id` as `id`, `repository`.`name` as `name`, `repository`.`owner_id` as `org_id`, `repository`.`id` as `repo_id`").
		Table("`team_repo`").
		Join("INNER", "`team`", "`team`.`id` = `team_repo`.`team_id`").
		Join("INNER", "`repository`", "`repository`.`id` = `team_repo`.`repo_id`").
		Where("`team`.`id` = ?", teamID).
		Find(&repos); err != nil {
		panic(fmt.Errorf("teamID=%v pageSize=%v page=%v", teamID, pageSize, page))
	}
	return repos
}

type teamProject struct {
	common

	repo *forgejoTeamRepo
}

func (o *teamProject) SetNative(repo any) {
	o.repo = repo.(*forgejoTeamRepo)
}

func (o *teamProject) GetNativeID() string {
	return fmt.Sprintf("%d", o.repo.ID)
}

func (o *teamProject) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *teamProject) ToFormat() f3.Interface {
	if o.repo == nil {
		return o.NewFormat()
	}

	teamProject := (&f3.TeamProject{
		Common:  f3.NewCommon(o.GetNativeID()),
		Name:    o.repo.Name,
		Project: f3_tree.NewProjectReference(f3.ResourceOrganizations, f3_util.ToString(o.repo.OrgID), f3_util.ToString(o.repo.RepoID)),
	}).Init()
	o.Trace("teamProject ToFormat %+v => %v", o.repo, teamProject)
	return teamProject
}

func (o *teamProject) FromFormat(content f3.Interface) {
	teamProject := content.(*f3.TeamProject)
	o.Trace("teamProject FromFormat ID %v Project %v", teamProject.GetID(), teamProject.Project.Get())
	path := generic.NewPathFromString(teamProject.Project.Get())
	ownersPath := path.Root().Forge().Organizations()
	owner := ownersPath.First().GetID().Int64()
	projectsPath := ownersPath.RemoveFirst().Projects()
	project := projectsPath.First().GetID().Int64()
	o.repo = &forgejoTeamRepo{
		ID:     f3_util.ParseInt(teamProject.GetID()),
		Name:   teamProject.Name,
		OrgID:  owner,
		RepoID: project,
	}
}

func (o *teamProject) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	repo := getForgejoTeamRepoByID(ctx, id)
	if repo == nil {
		return false
	}
	o.repo = repo
	return true
}

func (o *teamProject) Patch(ctx context.Context) {
	panic("not implemented")
}

func getRepositoryByID(ctx context.Context, repoID int64) *repo_model.Repository {
	repository, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		panic(fmt.Errorf("GetRepositoryByID(%v): %w", repoID, err))
	}
	return repository
}

func (o *teamProject) Put(ctx context.Context) f3_id.NodeID {
	repository := getRepositoryByID(ctx, o.repo.RepoID)
	teamRepo, err := models.InsertTeamRepository(ctx, o.getTeam(ctx), repository)
	if err != nil {
		panic(fmt.Errorf("AddAndReturnTeamRepository(%+v): %w", o.repo, err))
	}

	repo := getForgejoTeamRepoByID(ctx, teamRepo.ID)
	if repo == nil {
		panic(fmt.Errorf("unable to retrieve %+v", teamRepo))
	}
	o.repo = repo
	o.Trace("teamRepo created %+v", o.repo)
	return f3_id.NewNodeID(o.repo.ID)
}

func (o *teamProject) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := repo_service.RemoveRepositoryFromTeam(ctx, o.getTeam(ctx), o.repo.RepoID); err != nil {
		panic(fmt.Errorf("RemoveRepositoryFromTeam(%v): %w", o.repo.RepoID, err))
	}
}

func newTeamProject() generic.NodeDriverInterface {
	return &teamProject{}
}
