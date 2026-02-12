// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remote_registry

import (
	rr_model "forgejo.org/models/remote_registry"
)

// RemoteRegistryContext represents remote registry information in the request context
type RemoteRegistryContext struct {
	RemoteRegistry *rr_model.RemoteRegistry
	OwnerName      string
	ImageName      string
}
