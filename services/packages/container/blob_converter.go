// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/user"
	"github.com/regclient/regclient/types/manifest"
)

func ConvertToPackageFileDescriptor(doer *user.User, contextUser *user.User, regClientMan manifest.Manifest) (*packages_model.PackageFileDescriptor, error) {

	tagged := true
	tag := regClientMan.GetRef().Tag
	if tag == "" {
		tagged = false
		tag = string(regClientMan.GetDescriptor().Digest)
	}

	forgejoManifest := ManifestCreationInfo{
		MediaType: regClientMan.GetDescriptor().MediaType,
		Owner:     contextUser,
		Creator:   doer,
		Image:     regClientMan.GetRef().Reference,
		Reference: tag,
		IsTagged:  tagged,
	}

	packageFile := packages_model.PackageFile{
		VersionID: int64(0),
		BlobID:    int64(0),
		Name:      regClientMan.GetRef().Reference,
		LowerName: regClientMan.GetRef().Reference,
		IsLead:    true,
	}

	println(forgejoManifest.Creator)
	println(packageFile.BlobID)
	return &packages_model.PackageFileDescriptor{
		File: &packageFile,
		Blob: &packages_model.PackageBlob{},
	}, nil
}
