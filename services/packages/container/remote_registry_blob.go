// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"bytes"
	"io"
	"net/url"
	"time"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	rr_module "forgejo.org/modules/packages/remote_registry"
	"forgejo.org/services/context"

	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/manifest"
)

func GetRemoteTagList(ctx *context.Context, client *RegistryClient, userLowerName, image string, n int) (*TagList, *url.Values, error) {
	tagList, err := client.ListTags(ctx, userLowerName, image)
	if err != nil {
		return nil, nil, err
	}
	v := setLinkHeaderValues(tagList, n)
	return tagList, v, nil
}

func GetRemoteManifest(ctx *context.Context, remoteCtx *rr_module.RemoteRegistryContext, client *RegistryClient) (manifest.Manifest, error) {
	remoteManifest, err := client.GetManifest(ctx)
	if err != nil {
		log.Error("Failed to GET manifest %s from remote registry %s: %v",
			remoteCtx.ImageName, remoteCtx.RemoteRegistry.Name, err)
		return nil, err
	}
	return remoteManifest, err
}

func GetConfigDescriptor(client *RegistryClient, man manifest.Manifest) (*descriptor.Descriptor, error) {
	img := client.NewImager(man)
	cfg, err := img.GetConfig()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func GetAllRemoteBlobs(ctx *context.Context, remoteCtx *rr_module.RemoteRegistryContext, client *RegistryClient, man manifest.Manifest) error {
	img := client.NewImager(man)

	layers, err := img.GetLayers()
	if err != nil {
		return err
	}

	for _, layer := range layers {
		buf, err := GetRemoteBlob(ctx, client, &layer)
		if err != nil {
			return err
		}
		defer buf.Close()

		if layer.Digest.String() != DigestFromHashSummer(buf) {
			return ErrDigestInvalid
		}

		err = SaveBlobToPackage(ctx, buf, remoteCtx, layer.Digest.String(), ctx.ContextUser, ctx.Doer)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetRemoteBlob(ctx *context.Context, client *RegistryClient, layer *descriptor.Descriptor) (*packages_module.HashedBuffer, error) {
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

func SaveManifest(ctx *context.Context, owner, creator *user_model.User, remoteCtx rr_module.RemoteRegistryContext, man manifest.Manifest) error {
	mci, err := NewManifestCreationInfo(
		owner,
		creator,
		man.GetDescriptor().MediaType,
		remoteCtx.GetLocalImageName(),
		remoteCtx.Reference,
	)
	if err != nil {
		return err
	}
	mci.RemoteRegistryHost = man.GetRef().Registry
	mci.CacheTimeUnix = time.Now().Unix()

	buf, err := CreateManifestBuffer(man)
	if err != nil {
		return err
	}

	_, err = ProcessManifest(ctx, *mci, buf)
	if err != nil {
		return err
	}
	return nil
}

func CreateManifestBuffer(man manifest.Manifest) (*packages_module.HashedBuffer, error) {
	maxSize := MaxManifestSize + 1
	b, err := man.RawBody()
	if err != nil {
		return nil, err
	}

	reader := &io.LimitedReader{R: bytes.NewReader(b), N: int64(maxSize)}
	buf, err := packages_module.CreateHashedBufferFromReaderWithSize(reader, maxSize)
	if err != nil {
		return nil, err
	}
	defer buf.Close()

	if buf.Size() > MaxManifestSize {
		return nil, ErrManifestTooLarge
	}

	return buf, nil
}
