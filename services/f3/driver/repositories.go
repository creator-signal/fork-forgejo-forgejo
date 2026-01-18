// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_tree_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
)

type repositories struct {
	container
}

func (o *repositories) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	children := f3_tree_generic.NewChildrenList(0)
	if page > 1 {
		return children
	}

	createRepository := func(id string) *f3.Repository {
		r := (&f3.Repository{}).Init()
		r.SetID(id)
		return r
	}
	repositories := []*f3.Repository{createRepository(f3.RepositoryNameDefault)}
	project := f3_tree.GetProject(node).ToFormat().(*f3.Project)
	if project.HasWiki {
		repositories = append(repositories, createRepository(RepositoryNameWiki))
	}

	return f3_tree.ConvertListed(ctx, node, f3_tree.ConvertToAny(repositories...)...)
}

func (o *repositories) LookupMappedID(ctx context.Context, id f3_id.NodeID, f f3.Interface) f3_id.NodeID {
	return id
}

func newRepositories() f3_tree_generic.NodeDriverInterface {
	return &repositories{}
}
