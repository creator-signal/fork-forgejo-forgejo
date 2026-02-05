// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"errors"
	"regexp"

	"forgejo.org/models/user"
	container_module "forgejo.org/modules/packages/container"

	digest "github.com/opencontainers/go-digest"
)

var ReferencePattern = regexp.MustCompile(`\A[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}\z`)

// ManifestCreationInfo describes a manifest to create
type ManifestCreationInfo struct {
	MediaType  string
	Owner      *user.User
	Creator    *user.User
	Image      string
	Reference  string
	IsTagged   bool
	Properties map[string]string
}

func NewManifestCreationInfo(owner, creator *user.User, mediaType, image, reference string) (*ManifestCreationInfo, error) {
	isTagged := digest.Digest(reference).Validate() != nil

	mci := &ManifestCreationInfo{
		MediaType: mediaType,
		Owner:     owner,
		Creator:   creator,
		Image:     image,
		Reference: reference,
		IsTagged:  isTagged,
	}

	if mci.IsTagged && !ReferencePattern.MatchString(reference) {
		return &ManifestCreationInfo{}, errors.New("Tag is invalid")
	}

	return mci, nil
}
