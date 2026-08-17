// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_tree_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
	"xorm.io/builder"
)

type projects struct {
	container
}

func (o *projects) LookupMappedID(ctx context.Context, id f3_id.NodeID, f f3.Interface) f3_id.NodeID {
	project := f.(*f3.Project)
	return o.GetIDFromName(ctx, project.Name)
}

func (o *projects) GetIDFromName(ctx context.Context, name string) f3_id.NodeID {
	owner := f3_tree.GetOwnerName(o.GetNode())
	forgejoProject, err := repo_model.GetRepositoryByOwnerAndName(ctx, owner, name)
	if repo_model.IsErrRepoNotExist(err) {
		return f3_id.NilID
	}

	if err != nil {
		panic(fmt.Errorf("error GetRepositoryByOwnerAndName(%s, %s): %v", owner, name, err))
	}

	return f3_id.NewNodeID(forgejoProject.ID)
}

func getProjects(ctx context.Context, ownerID int64, page, pageSize int) ([]*repo_model.Repository, error) {
	var repositories []*repo_model.Repository
	sess := db.GetEngine(ctx)
	if pageSize > 0 && page > 0 {
		sess = sess.Limit(pageSize, (page-1)*pageSize)
	}
	if err := sess.Where(builder.Eq{"owner_id": ownerID}).Find(&repositories); err != nil {
		return nil, err
	}
	return repositories, nil
}

func (o *projects) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	pageSize := o.getPageSize()

	owner := f3_tree.GetOwner(node)

	forgejoProjects, err := getProjects(ctx, owner.GetID().Int64(), page, pageSize)
	if err != nil {
		panic(fmt.Errorf("error while listing %s projects: %v", owner, err))
	}

	for _, forgejoProject := range forgejoProjects {
		if err := forgejoProject.LoadAttributes(ctx); err != nil {
			panic(fmt.Errorf("LoadAttributes(%+v): %w", forgejoProject, err))
		}
	}
	return f3_tree.ConvertListed(ctx, node, f3_tree.ConvertToAny(forgejoProjects...)...)
}

func newProjects() f3_tree_generic.NodeDriverInterface {
	return &projects{}
}
