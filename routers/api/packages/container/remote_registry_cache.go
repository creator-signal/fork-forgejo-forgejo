// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"forgejo.org/services/context"

	packages_model "forgejo.org/models/packages"
)

func getCachedRemoteManifest(ctx *context.Context) (*packages_model.PackageFileDescriptor, error) {
	// TODO Later we need a distinction between manifests from remote and local
	return getManifestFromContext(ctx)
}
