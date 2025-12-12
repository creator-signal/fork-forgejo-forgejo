// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remoteregistry

import (
	"testing"

	"forgejo.org/models/packages"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"
)

func Test_RemoteRegistryValidation(t *testing.T) {
	sut := RemoteRegistry{
		ID:          int64(1),
		Name:        "rr",
		OwnerType:   RRUser,
		OwnerID:     int64(10),
		RemoteURL:   "https://codeberg.org",
		RemoteType:  packages.Type("container"),
		CreatedUnix: timeutil.TimeStampNow(),
		UpdatedUnix: timeutil.TimeStampNow(),
	}

	if ok, err := validation.IsValid(sut); !ok {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}
}
