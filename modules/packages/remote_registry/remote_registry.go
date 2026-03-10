// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remote_registry

import (
	remote_registry_model "forgejo.org/models/remote_registry"
)

// RemoteRegistryContext represents remote registry information in the request context
type RemoteRegistryContext struct {
	RemoteRegistry *remote_registry_model.RemoteRegistry
	OwnerName      string
	ImageName      string
	Reference      string
}

func (rrc *RemoteRegistryContext) GetLocalImageName() string {
	res := ""
	if rrc.ImageName != "" {
		res = rrc.RemoteRegistry.Name + "/" + rrc.ImageName
	}
	return res
}
