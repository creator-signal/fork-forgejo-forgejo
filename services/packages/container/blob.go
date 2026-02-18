// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"forgejo.org/models/db"
	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/modules/util"
	api_ctx "forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"

	digest "github.com/opencontainers/go-digest"
)

var uploadVersionMutex sync.Mutex

func GetLocalManifest(ctx *api_ctx.Context, ownerID int64, imageName, reference string) (*packages_model.PackageFileDescriptor, error) {
	opts, err := GetManifestSearchOptions(
		ownerID,
		imageName,
		reference,
	)
	if err != nil {
		return nil, err
	}
	return WorkaroundGetContainerBlob(ctx, opts)
}

// GetLocalBlob finds a local blob if it exists, returns ErrContainerBlobNotExist otherwise
func GetLocalBlob(ctx *api_ctx.Context, ownerID int64, dig, imageName string, remote ...bool) (*packages_model.PackageFileDescriptor, error) {
	if digest.Digest(dig).Validate() != nil {
		return nil, container_model.ErrContainerBlobNotExist
	}

	opts := &container_model.BlobSearchOptions{
		OwnerID: ownerID,
		Image:   imageName,
		Digest:  dig,
	}

	// Get blob or err
	log.Debug("Trying to find blob %s locally", dig)
	blobDescriptor, err := WorkaroundGetContainerBlob(ctx, opts)
	if err != nil {
		return nil, err
	}

	if len(remote) > 0 {
		// Update cache time (if it exists), as we are using this blob again
		pf, err := packages_model.GetFileForVersionByName(
			ctx,
			blobDescriptor.File.VersionID,
			blobDescriptor.File.LowerName,
			packages_model.EmptyFileKey)
		if err != nil {
			log.Error("Could not find file for blob %s: %v", dig, err)
			return nil, err
		}
		err = packages_model.UpdateProperty(ctx,
			&packages_model.PackageProperty{
				RefType: packages_model.PropertyTypeFile,
				RefID:   pf.ID,
				Name:    container_module.PropertyCacheTime,
				Value:   fmt.Sprintf("%d", time.Now().Unix()),
			})
		if err != nil {
			log.Warn("Failed to set/update blob property %s for remote blob: %v", container_module.PropertyCacheTime, err)
		}
	}

	return blobDescriptor, nil
}

// saveAsPackageBlob creates a package blob from an upload
// The uploaded blob gets stored in a special upload version to link them to the package/image
func SaveAsPackageBlob(ctx context.Context, hsr packages_module.HashedSizeReader, pci *packages_service.PackageCreationInfo) (*packages_model.PackageBlob, *packages_model.PackageFile, error) {
	pb := packages_service.NewPackageBlob(hsr)
	pf := &packages_model.PackageFile{}

	exists := false

	contentStore := packages_module.NewContentStore()

	uploadVersion, err := GetOrCreateUploadVersion(ctx, &pci.PackageInfo)
	if err != nil {
		return nil, nil, err
	}

	err = db.WithTx(ctx, func(ctx context.Context) error {
		if err := packages_service.CheckSizeQuotaExceeded(ctx, pci.Creator, pci.Owner, packages_model.TypeContainer, hsr.Size()); err != nil {
			return err
		}

		pb, exists, err = packages_model.GetOrInsertBlob(ctx, pb)
		if err != nil {
			log.Error("Error inserting package blob: %v", err)
			return err
		}
		// FIXME: Workaround to be removed in v1.20
		// https://github.com/go-gitea/gitea/issues/19586
		if exists {
			err = contentStore.Has(packages_module.BlobHash256Key(pb.HashSHA256))
			if err != nil && (errors.Is(err, util.ErrNotExist) || errors.Is(err, os.ErrNotExist)) {
				log.Debug("Package registry inconsistent: blob %s does not exist on file system", pb.HashSHA256)
				exists = false
			}
		}
		if !exists {
			if err := contentStore.Save(packages_module.BlobHash256Key(pb.HashSHA256), hsr, hsr.Size()); err != nil {
				log.Error("Error saving package blob in content store: %v", err)
				return err
			}
		}

		pf, err = CreateFileForBlob(ctx, uploadVersion, pb)
		return err
	})
	if err != nil {
		if !exists {
			if err := contentStore.Delete(packages_module.BlobHash256Key(pb.HashSHA256)); err != nil {
				log.Error("Error deleting package blob from content store: %v", err)
			}
		}
		return nil, nil, err
	}

	return pb, pf, nil
}

// mountBlob mounts the specific blob to a different package
func MountBlob(ctx context.Context, pi *packages_service.PackageInfo, pb *packages_model.PackageBlob) error {
	uploadVersion, err := GetOrCreateUploadVersion(ctx, pi)
	if err != nil {
		return err
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		_, err := CreateFileForBlob(ctx, uploadVersion, pb)
		return err
	})
}

func GetOrCreateUploadVersion(ctx context.Context, pi *packages_service.PackageInfo) (*packages_model.PackageVersion, error) {
	var uploadVersion *packages_model.PackageVersion

	// FIXME: Replace usage of mutex with database transaction
	// https://github.com/go-gitea/gitea/pull/21862
	uploadVersionMutex.Lock()
	err := db.WithTx(ctx, func(ctx context.Context) error {
		created := true
		p := &packages_model.Package{
			OwnerID:   pi.Owner.ID,
			Type:      packages_model.TypeContainer,
			Name:      strings.ToLower(pi.Name),
			LowerName: strings.ToLower(pi.Name),
		}
		var err error
		if p, err = packages_model.TryInsertPackage(ctx, p); err != nil {
			if err == packages_model.ErrDuplicatePackage {
				created = false
			} else {
				log.Error("Error inserting package: %v", err)
				return err
			}
		}

		if created {
			if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypePackage, p.ID, container_module.PropertyRepository, strings.ToLower(pi.Owner.LowerName+"/"+pi.Name)); err != nil {
				log.Error("Error setting package property: %v", err)
				return err
			}
		}

		pv := &packages_model.PackageVersion{
			PackageID:    p.ID,
			CreatorID:    pi.Owner.ID,
			Version:      container_model.UploadVersion,
			LowerVersion: container_model.UploadVersion,
			IsInternal:   true,
			MetadataJSON: "null",
		}
		if pv, err = packages_model.GetOrInsertVersion(ctx, pv); err != nil {
			if err != packages_model.ErrDuplicatePackageVersion {
				log.Error("Error inserting package: %v", err)
				return err
			}
		}

		uploadVersion = pv

		return nil
	})
	uploadVersionMutex.Unlock()

	return uploadVersion, err
}

func CreateFileForBlob(ctx context.Context, pv *packages_model.PackageVersion, pb *packages_model.PackageBlob) (*packages_model.PackageFile, error) {
	filename := strings.ToLower(fmt.Sprintf("sha256_%s", pb.HashSHA256))

	pf := &packages_model.PackageFile{
		VersionID:    pv.ID,
		BlobID:       pb.ID,
		Name:         filename,
		LowerName:    filename,
		CompositeKey: packages_model.EmptyFileKey,
	}
	var err error
	if pf, err = packages_model.TryInsertFile(ctx, pf); err != nil {
		if err == packages_model.ErrDuplicatePackageFile {
			return pf, nil
		}
		log.Error("Error inserting package file: %v", err)
		return &packages_model.PackageFile{}, err
	}

	if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypeFile, pf.ID, container_module.PropertyDigest, DigestFromPackageBlob(pb)); err != nil {
		log.Error("Error setting package file property: %v", err)
		return &packages_model.PackageFile{}, err
	}

	return pf, nil
}

func DeleteBlob(ctx context.Context, ownerID int64, image, digest string) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		pfds, err := container_model.GetContainerBlobs(ctx, &container_model.BlobSearchOptions{
			OwnerID: ownerID,
			Image:   image,
			Digest:  digest,
		})
		if err != nil {
			return err
		}

		for _, file := range pfds {
			if err := packages_service.DeletePackageFile(ctx, file.File); err != nil {
				return err
			}
		}
		return nil
	})
}

func DigestFromHashSummer(h packages_module.HashSummer) string {
	_, _, hashSHA256, _, _ := h.Sums()
	return "sha256:" + hex.EncodeToString(hashSHA256)
}

func DigestFromPackageBlob(pb *packages_model.PackageBlob) string {
	return "sha256:" + pb.HashSHA256
}
