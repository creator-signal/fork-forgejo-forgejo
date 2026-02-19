// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
	container_service "forgejo.org/services/packages/container"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types/descriptor"
)

const (
	blobServeBufferSize = 64 * 1024 // 64KB buffer
)

// serveBlobFromBuffer serves a blob from an existing buffer to a client
func serveBlobFromBuffer(ctx *context.Context, buf *packages_module.HashedBuffer, mediaType, digest string) error {
	// Reset buffer to beginning
	log.Debug("Serving blob %s from buffer", digest)
	if _, err := buf.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek buffer for serving: %w", err)
	}

	// Set response headers
	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: digest,
		ContentType:   mediaType,
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

func serveManifestFromBuffer(ctx *context.Context, regMan manifest.Manifest) error {
	log.Debug("Serving manifest from buffer")
	body, err := regMan.RawBody()
	if err != nil {
		return fmt.Errorf("failed to serve index manifest from buffer: %w", err)
	}
	headers, err := regMan.RawHeaders()
	if err != nil {
		return fmt.Errorf("failed to serve index manifest from buffer: %w", err)
	}
	ctx.Resp.Header().Set("Content-Length", headers.Get("Content-Length"))
	ctx.Resp.Header().Set("Content-Type", headers.Get("Content-Type"))
	ctx.Resp.Header().Set("Docker-Distribution-Api-Version", headers.Get("Docker-Distribution-Api-Version"))
	ctx.Resp.Header().Set("ETag", headers.Get("ETag"))
	ctx.Resp.Header().Set("Docker-Content-Digest", headers.Get("Docker-Content-Digest"))
	ctx.Resp.Write(body)
	return nil
}

func GetRemoteTagList(ctx *context.Context) {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	} else if remoteCtx.GetLocalImageName() == "" {
		apiErrorDefined(ctx, container_service.ErrBlobUnknown)
		return
	}
	last := ctx.FormTrim("last")
	n := -1
	if ctx.FormTrim("n") != "" {
		n = ctx.FormInt("n")
	}

	tagList, vals, err := container_service.GetLocalTagList(ctx,
		ctx.Package.Owner.LowerName,
		remoteCtx.GetLocalImageName(),
		last,
		n,
		ctx.Package.Owner.ID)

	if errors.Is(err, packages_model.ErrPackageNotExist) {
		client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
		if err != nil {
			log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer client.Close(ctx)

		tagList, vals, err = container_service.GetRemoteTagList(ctx, &client, ctx.Package.Owner.LowerName, remoteCtx.ImageName, n)
		if err != nil {
			log.Error("Failed to get tag list for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
	} else if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	if len(tagList.Tags) > 0 {
		ctx.Resp.Header().Set("Link", fmt.Sprintf(`</v2/%s/%s/tags/list?%s>; rel="next"`, ctx.Package.Owner.LowerName, remoteCtx.ImageName, vals.Encode()))
	}

	jsonResponse(ctx, http.StatusOK, tagList)
}

func RemoteHeadBlob(ctx *context.Context) {
	var respDigest string
	var size int64

	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	} else if remoteCtx.Reference == "" || remoteCtx.ImageName == "" {
		apiErrorDefined(ctx, container_service.ErrBlobUnknown)
		return
	}

	blob, err := container_service.GetLocalBlob(ctx, ctx.ContextUser.ID, remoteCtx.Reference, remoteCtx.GetLocalImageName(), true)
	if err != nil {
		if errors.Is(err, container_model.ErrContainerBlobNotExist) {
			log.Debug("Did not find blob with digest %s locally, getting from remote %v", remoteCtx.Reference)

			client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
			if err != nil {
				log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			defer client.Close(ctx)

			regDigest := digest.Digest(remoteCtx.Reference)
			regLayer := descriptor.Descriptor{Digest: regDigest}
			buf, err := container_service.GetRemoteBlob(ctx, &client, &regLayer)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			defer buf.Close()

			// save to package TODO this could happen in a go routine
			err = container_service.SaveBlobToPackage(ctx, buf, remoteCtx, remoteCtx.Reference, ctx.ContextUser, ctx.Doer)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}

			respDigest = remoteCtx.Reference
			size = buf.Size()
		} else {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
	} else {
		respDigest = blob.Properties.GetByName(container_module.PropertyDigest)
		size = blob.Blob.Size
	}

	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: respDigest,
		ContentLength: size,
		Status:        http.StatusOK,
	})
}

func RemoteGetBlob(ctx *context.Context) {
	// Serve from cache
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	} else if remoteCtx.Reference == "" || remoteCtx.ImageName == "" {
		apiErrorDefined(ctx, container_service.ErrBlobUnknown)
		return
	}

	blob, err := container_service.GetLocalBlob(ctx, ctx.ContextUser.ID, remoteCtx.Reference, remoteCtx.GetLocalImageName(), true)
	if err == container_model.ErrContainerBlobNotExist {
		log.Debug("Did not find blob with digest %s locally, getting from remote %v", remoteCtx.Reference, remoteCtx.Reference)

		client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
		if err != nil {
			log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer client.Close(ctx)

		regDigest := digest.Digest(remoteCtx.Reference)
		regLayer := descriptor.Descriptor{
			Digest: regDigest,
		}

		// get from remote
		buf, err := container_service.GetRemoteBlob(ctx, &client, &regLayer)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer buf.Close()

		// check digest
		digest := regLayer.Digest
		if digest.String() != container_service.DigestFromHashSummer(buf) {
			apiErrorDefined(ctx, container_service.ErrDigestInvalid)
			return
		}

		// save to package TODO this could happen in a go routine
		err = container_service.SaveBlobToPackage(ctx, buf, remoteCtx, remoteCtx.Reference, ctx.ContextUser, ctx.Doer)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		// serve from buffer
		err = serveBlobFromBuffer(ctx, buf, "application/octet-stream", regDigest.String())
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

func RemoteHeadManifest(ctx *context.Context) {
	var contentDigest string
	var contentType string
	var contentLength int64

	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	} else if remoteCtx.Reference == "" {
		apiErrorDefined(ctx, container_service.ErrManifestUnknown)
		return
	}

	// Do we have the manifest cached locally?
	manifest, err := container_service.GetLocalManifest(ctx, ctx.ContextUser.ID, remoteCtx.GetLocalImageName(), remoteCtx.Reference)
	if errors.Is(err, container_model.ErrContainerBlobNotExist) {
		client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
		if err != nil {
			log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer client.Close(ctx)

		// Get manifest metadata from remote registry
		manifestResp, err := client.HeadManifest(ctx)
		if err != nil {
			log.Error("Failed to HEAD manifest %s:%s from remote registry %s: %v",
				remoteCtx.ImageName, remoteCtx.Reference, remoteCtx.RemoteRegistry.Name, err)
			if strings.Contains(err.Error(), "404") {
				apiErrorDefined(ctx, container_service.ErrManifestUnknown)
			} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
				apiErrorDefined(ctx, container_service.ErrUnauthorized)
			} else {
				apiError(ctx, http.StatusBadGateway, err)
			}
			return
		}
		contentDigest = manifestResp.GetRef().Digest
		contentType = manifestResp.GetDescriptor().MediaType
		contentLength = manifestResp.GetDescriptor().Size
	} else {
		contentDigest = manifest.Properties.GetByName(container_module.PropertyDigest)
		contentType = manifest.Properties.GetByName(container_module.PropertyMediaType)
		contentLength = manifest.Blob.Size
	}

	log.Trace("Serving manifest")
	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: contentDigest,
		ContentType:   contentType,
		ContentLength: contentLength,
		Status:        http.StatusOK,
	})
}

func RemoteGetManifest(ctx *context.Context) {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	} else if remoteCtx.Reference == "" || remoteCtx.ImageName == "" {
		apiErrorDefined(ctx, container_service.ErrBlobUnknown)
		return
	}

	man, err := container_service.GetLocalManifest(ctx, ctx.ContextUser.ID, remoteCtx.GetLocalImageName(), remoteCtx.Reference)

	if errors.Is(err, container_model.ErrContainerBlobNotExist) {
		client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
		if err != nil {
			log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer client.Close(ctx)

		// Not cached, fetch from remote registry
		regManifest, err := container_service.GetRemoteManifest(ctx, remoteCtx, &client)
		if err != nil {
			log.Error("Failed to get blobs for manifest: %v", err)
			if strings.Contains(err.Error(), "404") {
				apiErrorDefined(ctx, container_service.ErrManifestUnknown)
			} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
				apiErrorDefined(ctx, container_service.ErrUnauthorized)
			} else {
				apiError(ctx, http.StatusInternalServerError, err)
			}
			return
		}

		if regManifest.IsList() {
			// Got an index manifest, serve it directly as we will get a request for the correct manifest in return
			mediaType := regManifest.GetDescriptor().MediaType
			digest := regManifest.GetDescriptor().Digest
			log.Debug("Serving index manifest for %s with content-type: %s and digest: %s",
				remoteCtx.RemoteRegistry.Name,
				mediaType,
				digest)
			serveManifestFromBuffer(ctx, regManifest)
			return
		}

		err = container_service.GetAllRemoteBlobs(ctx, remoteCtx, &client, regManifest)
		if err != nil {
			log.Error("Failed to get blobs for manifest: %v", err)
			apiError(ctx, http.StatusInternalServerError, err)
			if strings.Contains(err.Error(), "404") {
				apiErrorDefined(ctx, container_service.ErrManifestUnknown)
			} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
				apiErrorDefined(ctx, container_service.ErrUnauthorized)
			} else {
				apiError(ctx, http.StatusInternalServerError, err)
			}
			return
		}

		cfg, err := container_service.GetConfigDescriptor(&client, regManifest)
		if err != nil {
			log.Error("Failed to get config: %v", err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		cfgbuf, err := container_service.GetRemoteBlob(ctx, &client, cfg)
		if err != nil {
			log.Error("Failed to save configBlob: %v", err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer cfgbuf.Close()

		err = container_service.SaveBlobToPackage(
			ctx,
			cfgbuf,
			remoteCtx,
			cfg.Digest.String(),
			ctx.ContextUser,
			ctx.Doer)
		if err != nil {
			log.Error("Failed to save config: %v", err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		err = container_service.SaveManifest(
			ctx,
			ctx.ContextUser,
			ctx.Doer,
			*remoteCtx,
			regManifest)
		if err != nil {
			log.Error("Failed to save manifest: %v", err)
			if errors.Is(err, container_service.ErrTagInvalid) {
				apiErrorDefined(ctx, container_service.ErrManifestInvalid.
					WithMessage(err.Error()))
			} else if errors.Is(err, container_service.ErrManifestTooLarge) {
				apiErrorDefined(ctx, container_service.ErrManifestInvalid.
					WithMessage(err.Error()).
					WithStatusCode(http.StatusRequestEntityTooLarge))
			} else if errors.Is(err, container_model.ErrContainerBlobNotExist) {
				apiErrorDefined(ctx, container_service.ErrBlobUnknown)
			} else {
				var namedError *container_service.NamedError
				if errors.As(err, &namedError) {
					apiErrorDefined(ctx, namedError)
				} else {
					switch err {
					case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
						apiError(ctx, http.StatusForbidden, err)
					default:
						apiError(ctx, http.StatusInternalServerError, err)
					}
				}
			}
			return
		}
		err = serveManifestFromBuffer(ctx, regManifest)
		if err != nil {
			log.Error("Failed to serve manifest %v", err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		return
	}
	log.Trace("Remote manifest with file ID: %s existed", man.File.ID)
	serveBlob(ctx, man)
}
