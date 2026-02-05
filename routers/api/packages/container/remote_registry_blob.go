// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	container_model "forgejo.org/models/packages/container"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/services/context"
	container_service "forgejo.org/services/packages/container"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types/descriptor"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
)

const (
	blobServeBufferSize = 64 * 1024 // 64KB buffer
)

func GetRemoteTagList(ctx *context.Context) {
}

func RemoteHeadBlob(ctx *context.Context) {
	// Do we have the blob cached locally?
	blob, err := getBlobFromContext(ctx)
	if err != nil {
		if err == container_model.ErrContainerBlobNotExist {

			dig := ctx.Params("digest")
			if dig == "" {
				apiErrorDefined(ctx, errBlobUnknown)
				return
			}

			regDigest := digest.Digest(dig)
			regLayer := descriptor.Descriptor{
				Digest: regDigest,
			}

			remoteCtx, err := GetRemoteRegistryContext(ctx)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}

			client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry)
			if err != nil {
				log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}

			ref, err := client.NewRef(remoteCtx.ImageName)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			defer client.Close(ctx, ref)

			buf, err := getBlobFromRemote(ctx, &client, regLayer, ref)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			defer buf.Close()

			// save to package TODO this could happen in a go routine
			err = saveBlobToPackage(ctx, buf, remoteCtx, dig, ctx.ContextUser, ctx.Doer)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}

		} else {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	// try again
	blob, err = getBlobFromContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: blob.Properties.GetByName(container_module.PropertyDigest),
		ContentLength: blob.Blob.Size,
		Status:        http.StatusOK,
	})
}

func RemoteGetBlob(ctx *context.Context) {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	dig := ctx.Params("digest")
	if dig == "" {
		apiErrorDefined(ctx, errBlobUnknown)
		return
	}

	client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry)
	if err != nil {
		log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Blob Ref
	ref, err := client.NewRef(remoteCtx.ImageName)
	if err != nil {
		log.Error("Failed to create reference for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer client.Close(ctx, ref)

	// Serve from cache
	blob, err := getLocalBlob(ctx, remoteCtx, dig)
	if err == container_model.ErrContainerBlobNotExist {
		log.Debug("Did not find blob with digest %s locally, getting from remote %v", dig)
		regDigest := digest.Digest(dig)
		regLayer := descriptor.Descriptor{
			Digest: regDigest,
		}

		// get from remote
		buf, err := getBlobFromRemote(ctx, &client, regLayer, ref)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer buf.Close()

		// check digest
		digest := regLayer.Digest
		if digest.String() != digestFromHashSummer(buf) {
			apiErrorDefined(ctx, errDigestInvalid)
			return
		}

		// save to package TODO this could happen in a go routine
		err = saveBlobToPackage(ctx, buf, remoteCtx, dig, ctx.ContextUser, ctx.Doer)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		// serve from buffer
		err = serveBlobFromBuffer(ctx, buf, regDigest.String())
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		return
	} else if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	serveBlob(ctx, blob)
}

func getBlobFromRemote(ctx *context.Context, client *container_service.RegistryClient, layer descriptor.Descriptor, regRef ref.Ref) (*packages_module.HashedBuffer, error) {
	log.Debug("Getting blob %s locally, getting from remote %v", layer.Digest, regRef.Registry)
	br, err := client.GetBlob(ctx, regRef, layer)
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

func getAllBlobsFromRemote(ctx *context.Context, client *container_service.RegistryClient, man manifest.Manifest) error {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return err
	}

	// get ref
	ref, err := client.NewRef(remoteCtx.ImageName)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return err
	}

	img := client.NewImager(man)

	layers, err := img.GetLayers()
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return err
	}

	for _, layer := range layers {
		// get from remote
		buf, err := getBlobFromRemote(ctx, client, layer, ref)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return err
		}
		defer buf.Close()

		// check digest
		if layer.Digest.String() != digestFromHashSummer(buf) {
			apiErrorDefined(ctx, errDigestInvalid)
			return err
		}

		// save to package
		err = saveBlobToPackage(ctx, buf, remoteCtx, layer.Digest.String(), ctx.ContextUser, ctx.Doer)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return err
		}
	}
	return nil
}

// serveBlobFromBuffer serves a blob from an existing buffer to a client
func serveBlobFromBuffer(ctx *context.Context, buf *packages_module.HashedBuffer, digest string) error {
	// Reset buffer to beginning
	log.Debug("Serving blob %s from buffer", digest)
	if _, err := buf.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek buffer for serving: %w", err)
	}

	// Set response headers
	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: digest,
		ContentType:   "application/octet-stream",
		ContentLength: buf.Size(),
		Status:        http.StatusOK,
	})

	// Copy buffer content to client
	sbuf := make([]byte, blobServeBufferSize)
	_, err := io.CopyBuffer(ctx.Resp, buf, sbuf)
	if err != nil {
		return fmt.Errorf("failed to serve blob from buffer: %w", err)
	}

	log.Debug("Served blob %s from buffer", digest)
	return nil
}

func RemoteHeadManifest(ctx *context.Context) {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	reference := ctx.Params("reference")
	if reference == "" {
		apiErrorDefined(ctx, errManifestUnknown)
		return
	}

	// Do we have the manifest cached locally?
	manifest, err := getCachedRemoteManifest(ctx)
	if manifest != nil && err == nil {
		setResponseHeaders(ctx.Resp, &containerHeaders{
			ContentDigest: manifest.Properties.GetByName(container_module.PropertyDigest),
			ContentType:   manifest.Properties.GetByName(container_module.PropertyMediaType),
			ContentLength: manifest.Blob.Size,
			Status:        http.StatusOK,
		})
		log.Trace("Remote manifest with file ID: %s existed", manifest.File.ID)
		return
	}

	// Not cached, fetch from remote registry
	client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry)
	if err != nil {
		log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Manifest Ref
	r, err := client.NewRef(remoteCtx.ImageName)
	if err != nil {
		log.Error("Failed to create reference for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer client.Close(ctx, r)

	// Get manifest metadata from remote registry
	manifestResp, err := client.HeadManifest(ctx, r)
	if err != nil {
		log.Error("Failed to HEAD manifest %s:%s from remote registry %s: %v",
			remoteCtx.ImageName, reference, remoteCtx.RemoteRegistry.Name, err)

		if strings.Contains(err.Error(), "404") {
			apiErrorDefined(ctx, errManifestUnknown)
		} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			apiErrorDefined(ctx, errUnauthorized)
		} else {
			apiError(ctx, http.StatusBadGateway, err)
		}
		return
	}

	// Set response headers from remote
	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: manifestResp.GetRef().Digest,
		ContentType:   manifestResp.GetDescriptor().MediaType,
		ContentLength: manifestResp.GetDescriptor().Size,
		Status:        http.StatusOK,
	})
}

func RemoteGetManifest(ctx *context.Context) {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Do we have the manifest cached locally?
	man, err := getCachedRemoteManifest(ctx)
	if man != nil && err == nil {
		serveBlob(ctx, man)
		log.Trace("Remote manifest with file ID: %s existed", man.File.ID)
		return
	}

	// Not cached, fetch from remote registry
	client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry)
	if err != nil {
		log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Manifest Ref
	r, err := client.NewRef(remoteCtx.ImageName)
	if err != nil {
		log.Error("Failed to create reference for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer client.Close(ctx, r)

	// Get manifest metadata from remote registry
	regManifest, err := client.GetManifest(ctx, r)
	if err != nil {
		log.Error("Failed to HEAD manifest %s from remote registry %s: %v",
			remoteCtx.ImageName, remoteCtx.RemoteRegistry.Name, err)

		if strings.Contains(err.Error(), "404") {
			apiErrorDefined(ctx, errManifestUnknown)
		} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			apiErrorDefined(ctx, errUnauthorized)
		} else {
			apiError(ctx, http.StatusBadGateway, err)
		}
		return
	}

	err = getAllBlobsFromRemote(ctx, &client, regManifest)
	if err != nil {
		log.Error("Failed to get blobs for manifest: %v", err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// TODO save manifest config as its own blob as this will later be referenced by processManifest

	err = saveManifest(ctx, regManifest)
	if err != nil {
		log.Error("Failed to save manifest: %v", err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Serve the manifest content
	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: regManifest.GetRef().Digest,
		ContentType:   regManifest.GetDescriptor().MediaType,
		ContentLength: regManifest.GetDescriptor().Size,
		Status:        http.StatusOK,
	})

	manifestBody, err := regManifest.RawBody()
	if err != nil {
		log.Error("Failed to get manifest body for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	_, err = ctx.Resp.Write(manifestBody)
	if err != nil {
		log.Error("Failed to write response for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
}
