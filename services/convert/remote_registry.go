// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	rr_model "forgejo.org/models/remote_registry"
	api "forgejo.org/modules/structs"
)

func ToRemoteRegistry(rr *rr_model.RemoteRegistry) *api.RemoteRegistry {
	result := &api.RemoteRegistry{
		ID:         rr.ID,
		Name:       rr.Name,
		OwnerType:  rr.OwnerType.Name(),
		OwnerID:    rr.OwnerID,
		RemoteURL:  rr.RemoteURL,
		RemoteHost: rr.RemoteHost,
		RemotePort: rr.RemotePort,
		RemoteUser: rr.RemoteUser,
		RemoteType: rr.RemoteType.Name(),
	}

	return result
}
