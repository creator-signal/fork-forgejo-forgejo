// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	packages_model "forgejo.org/models/packages"
	"github.com/regclient/regclient/types/manifest"
)

func ConvertToPackageFileDescriptor(regClientMan manifest.Manifest) (*packages_model.PackageFileDescriptor, error) {

	forgejoManifest := ManifestCreationInfo{
		MediaType: regClientMan.GetDescriptor().MediaType,
		Image:     regClientMan.GetRef().Reference,
		Reference: regClientMan.GetDescriptor(),
	}

	println(forgejoManifest)
	return nil
}
