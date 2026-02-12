// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	rr_module "forgejo.org/modules/packages/remote_registry"
	"forgejo.org/services/context"
	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/manifest"
)

func GetRemoteManifest(ctx *context.Context, remoteCtx *rr_module.RemoteRegistryContext, client *RegistryClient) (manifest.Manifest, error) {
	// Get manifest metadata from remote registry
	remoteManifest, err := client.GetManifest(ctx)
	if err != nil {
		log.Error("Failed to GET manifest %s from remote registry %s: %v",
			remoteCtx.ImageName, remoteCtx.RemoteRegistry.Name, err)
		return nil, err
	}
	return remoteManifest, err
}

func GetAllBlobsFromRemote(ctx *context.Context, remoteCtx *rr_module.RemoteRegistryContext, client *RegistryClient, man manifest.Manifest) error {
	img := client.NewImager(man)

	layers, err := img.GetLayers()
	if err != nil {
		return err
	}

	for _, layer := range layers {
		// get from remote
		buf, err := GetBlobFromRemote(ctx, client, &layer)
		if err != nil {
			return err
		}
		defer buf.Close()

		// check digest
		if layer.Digest.String() != DigestFromHashSummer(buf) {
			return ErrDigestInvalid
		}

		// save to package
		err = SaveBlobToPackage(ctx, buf, remoteCtx, layer.Digest.String(), ctx.ContextUser, ctx.Doer)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetBlobFromRemote(ctx *context.Context, client *RegistryClient, layer *descriptor.Descriptor) (*packages_module.HashedBuffer, error) {
	log.Debug("Getting blob %s locally, getting from remote %v", layer.Digest, client.Reference.Registry)
	br, err := client.GetBlob(ctx, *layer)
	if err != nil {
		return nil, err
	}
	buf, err := packages_module.CreateHashedBufferFromReader(br)
	if err != nil {
		return nil, err
	}
	defer br.Close()
	return buf, nil
}
