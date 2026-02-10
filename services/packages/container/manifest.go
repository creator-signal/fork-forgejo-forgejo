// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"errors"
	"regexp"
	"time"

	"forgejo.org/models/user"

	digest "github.com/opencontainers/go-digest"
)

var ReferencePattern = regexp.MustCompile(`\A[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}\z`)

// ManifestCreationInfo describes a manifest to create
type ManifestCreationInfo struct {
	MediaType          string
	Owner              *user.User
	Creator            *user.User
	Image              string
	Reference          string
	IsTagged           bool
	RemoteRegistryHost string
	CacheTimeUnix      int64
	Properties         map[string]string
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

func NewRemoteManifestCreationInfo(owner, creator *user.User, mediaType, image, reference, host string) (*ManifestCreationInfo, error) {
	isTagged := digest.Digest(reference).Validate() != nil
	time := time.Now().Unix()

	mci := &ManifestCreationInfo{
		MediaType:          mediaType,
		Owner:              owner,
		Creator:            creator,
		Image:              image,
		Reference:          reference,
		IsTagged:           isTagged,
		RemoteRegistryHost: host,
		CacheTimeUnix:      time,
	}

	if mci.IsTagged && !ReferencePattern.MatchString(reference) {
		return &ManifestCreationInfo{}, errors.New("Tag is invalid")
	}

	return mci, nil
}
