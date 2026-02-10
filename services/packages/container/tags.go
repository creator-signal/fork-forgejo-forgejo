// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"context"
	"strings"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
)

type TagList struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func NewTagList(ctx context.Context, ownerLower, image, last string, n int, ownerID int64) (*TagList, error) {
	_, err := packages_model.GetPackageByName(ctx, ownerID, packages_model.TypeContainer, image)
	if err != nil {
		return &TagList{}, err
	}

	tags, err := container_model.GetImageTags(ctx, ownerID, image, n, last)
	if err != nil {
		return &TagList{}, err
	}

	return &TagList{
		Name: strings.ToLower(ownerLower + "/" + image),
		Tags: tags,
	}, nil

}
