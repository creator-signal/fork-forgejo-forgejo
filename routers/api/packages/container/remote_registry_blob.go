// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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

func GetRemoteTagList(ctx *context.Context) {
	image := ctx.Params("image")
	last := ctx.FormTrim("last")
	n := -1
	if ctx.FormTrim("n") != "" {
		n = ctx.FormInt("n")
	}

	tagList, err := container_service.NewTagList(ctx,
		ctx.Package.Owner.LowerName,
		image,
		last,
		n,
		ctx.Package.Owner.ID)

	if errors.Is(err, packages_model.ErrPackageNotExist) {
		remoteCtx, err := GetRemoteRegistryContext(ctx)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
		if err != nil {
			log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}
		defer client.Close(ctx)

		tagList, err = client.ListTags(ctx, ctx.Package.Owner.LowerName, image)
		if err != nil {
			log.Error("Failed to get tag list for %s: %v", remoteCtx.RemoteRegistry.Name, err)
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		if len(tagList.Tags) > 0 {
			v := url.Values{}
			if n > 0 {
				v.Add("n", strconv.Itoa(n))
			}
			v.Add("last", tagList.Tags[len(tagList.Tags)-1])

			ctx.Resp.Header().Set("Link", fmt.Sprintf(`</v2/%s/%s/tags/list?%s>; rel="next"`, ctx.Package.Owner.LowerName, image, v.Encode()))
		}

		jsonResponse(ctx, http.StatusOK, tagList)
	} else if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	if len(tagList.Tags) > 0 {
		v := url.Values{}
		if n > 0 {
			v.Add("n", strconv.Itoa(n))
		}
		v.Add("last", tagList.Tags[len(tagList.Tags)-1])

		ctx.Resp.Header().Set("Link", fmt.Sprintf(`</v2/%s/%s/tags/list?%s>; rel="next"`, ctx.Package.Owner.LowerName, image, v.Encode()))
	}

	jsonResponse(ctx, http.StatusOK, tagList)
}

func RemoteHeadBlob(ctx *context.Context) {
	// Do we have the blob cached locally?
	_, err := getBlobFromContext(ctx)
	if err != nil {
		if err == container_model.ErrContainerBlobNotExist {
			dig := ctx.Params("digest")
			if dig == "" {
				apiErrorDefined(ctx, container_service.ErrBlobUnknown)
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

			client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
			if err != nil {
				log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			defer client.Close(ctx)

			buf, err := container_service.GetBlobFromRemote(ctx, &client, &regLayer)
			if err != nil {
				apiError(ctx, http.StatusInternalServerError, err)
				return
			}
			defer buf.Close()

			// save to package TODO this could happen in a go routine
			err = container_service.SaveBlobToPackage(ctx, buf, remoteCtx, dig, ctx.ContextUser, ctx.Doer)
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
	blob, err := getBlobFromContext(ctx)
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
		apiErrorDefined(ctx, container_service.ErrBlobUnknown)
		return
	}

	client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry, remoteCtx.ImageName)
	if err != nil {
		log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer client.Close(ctx)

	// Serve from cache
	blob, err := container_service.GetLocalBlob(ctx, remoteCtx, dig)
	if err == container_model.ErrContainerBlobNotExist {
		log.Debug("Did not find blob with digest %s locally, getting from remote %v", dig)
		regDigest := digest.Digest(dig)
		regLayer := descriptor.Descriptor{
			Digest: regDigest,
		}

		// get from remote
		buf, err := container_service.GetBlobFromRemote(ctx, &client, &regLayer)
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
		err = container_service.SaveBlobToPackage(ctx, buf, remoteCtx, dig, ctx.ContextUser, ctx.Doer)
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
		apiErrorDefined(ctx, container_service.ErrManifestUnknown)
		return
	}

	// Do we have the manifest cached locally?
	manifest, err := getManifestFromContext(ctx)
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
			remoteCtx.ImageName, reference, remoteCtx.RemoteRegistry.Name, err)

		if strings.Contains(err.Error(), "404") {
			apiErrorDefined(ctx, container_service.ErrManifestUnknown)
		} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			apiErrorDefined(ctx, container_service.ErrUnauthorized)
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
	man, err := getManifestFromContext(ctx)
	if man != nil && err == nil {
		serveBlob(ctx, man)
		log.Trace("Remote manifest with file ID: %s existed", man.File.ID)
		return
	}

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
	}

	err = container_service.GetAllBlobsFromRemote(ctx, remoteCtx, &client, regManifest)
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

	cfgbuf, err := container_service.GetBlobFromRemote(ctx, &client, cfg)
	if err != nil {
		log.Error("Failed to save configBlob: %v", err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer cfgbuf.Close()

	err = container_service.SaveBlobToPackage(ctx, cfgbuf, remoteCtx, cfg.Digest.String(), ctx.ContextUser, ctx.Doer)
	if err != nil {
		log.Error("Failed to save config: %v", err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	err = container_service.SaveManifest(ctx, regManifest)
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
