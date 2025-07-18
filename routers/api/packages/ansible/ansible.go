// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package ansible

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	packages_model "forgejo.org/models/packages"
	packages_module "forgejo.org/modules/packages"
	ansible_module "forgejo.org/modules/packages/ansible"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
	"forgejo.org/routers/api/packages/helper"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"

	gouuid "github.com/google/uuid"
)

func apiError(ctx *context.Context, status int, obj any) {
	helper.LogAndProcessError(ctx, status, obj, func(message string) {
		ctx.JSON(status, map[string]string{
			"error": message,
		})
	})
}

// Identity just replies with HTTP 200 to show we are actually here
func Identity(ctx *context.Context) {
	ctx.Status(http.StatusOK)
}

// AvailableApis returns the supported API versions
// We currently only support the v3 API for collections
func AvailableApis(ctx *context.Context) {
	type APIResult struct {
		AvailableVersions map[string]string `json:"available_versions"`
	}
	ctx.JSON(http.StatusOK, APIResult{
		AvailableVersions: map[string]string{
			"v3": "v3/",
		},
	})
}

// UploadCollection receives a collection via HTTP POST
// The collection is passed via the "file" multipart form element
func UploadCollection(ctx *context.Context) {
	file, fileHeader, err := ctx.Req.FormFile("file")
	if err != nil {
		apiError(ctx, http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	var receivedData io.Reader
	// If the package is uploaded through galaxy client the data in the form is encoded as base64
	if fileHeader.Header.Get("Content-Transfer-Encoding") == "base64" {
		receivedData = base64.NewDecoder(base64.StdEncoding, file)
	} else {
		receivedData = file
	}

	buffer, err := packages_module.CreateHashedBufferFromReader(receivedData)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer buffer.Close()

	pck, err := ansible_module.BuildCollectionFromArchive(buffer)
	if err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			apiError(ctx, http.StatusBadRequest, err)
		} else {
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	// "Rewind" the buffer after initial processing
	if _, err := buffer.Seek(0, io.SeekStart); err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	_, _, err = packages_service.CreatePackageAndAddFile(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       ctx.Package.Owner,
				PackageType: packages_model.TypeAnsible,
				Name:        pck.CollectionInfo.Namespace + "." + pck.CollectionInfo.Name,
				Version:     pck.CollectionInfo.Version,
			},
			Creator:          ctx.Doer,
			SemverCompatible: true,
			Metadata:         pck,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: strings.ToLower(pck.CollectionInfo.Namespace + "-" + pck.CollectionInfo.Name + "-" + pck.CollectionInfo.Version + ".tar.gz"),
			},
			Creator: ctx.Doer,
			Data:    buffer,
			IsLead:  true,
		},
	)
	if err != nil {
		switch err {
		case packages_model.ErrDuplicatePackageVersion:
			apiError(ctx, http.StatusConflict, err)
		case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
			apiError(ctx, http.StatusForbidden, err)
		default:
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return
	}

	// We just return a random UUID here. The galaxy client assumes an asyncronous parsing process.
	// Since we already did everything before, we just fake the task handling here.
	ctx.JSON(http.StatusCreated, map[string]string{
		"task": "/api/v3/imports/collections/" + gouuid.NewString() + "/",
	})
}

// This method just mocks the success behavior of Ansible Galaxy, to be compatible to the client
// We don't post-process the packages so in case of problems the upload fails directly
func ImportResult(ctx *context.Context) {
	type ImportResultData struct {
		ID       string    `json:"id"`
		State    string    `json:"state"`
		Error    *string   `json:"error"`
		Messages []string  `json:"messages"`
		Created  time.Time `json:"created_at"`
		Updated  time.Time `json:"updated_at"`
		Started  time.Time `json:"started_at"`
		Finished time.Time `json:"finished_at"`
	}
	ctx.JSON(http.StatusOK, ImportResultData{
		ID:       ctx.Params("uuid"),
		State:    "completed",
		Created:  time.Now(),
		Updated:  time.Now(),
		Started:  time.Now(),
		Finished: time.Now(),
		Messages: []string{},
	})
}

// ListVersions returns a JSON response with a list of the available collection versions
// as well as set of general metadata of the collection
func ListVersions(ctx *context.Context) {
	packageNamespace := ctx.Params("namespace")
	packageName := ctx.Params("name")

	pvs, err := packages_model.GetVersionsByPackageName(ctx, ctx.Package.Owner.ID, packages_model.TypeAnsible, packageNamespace+"."+packageName)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	if len(pvs) == 0 {
		apiError(ctx, http.StatusNotFound, "Requested collection has no artifact attached to it")
		return
	}

	pds, err := packages_model.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	sort.Slice(pds, func(i, j int) bool {
		return pds[i].SemVer.GreaterThan(pds[j].SemVer)
	})

	type AnsibleVersionsResponseMeta struct {
		Count int `json:"count"`
	}

	type AnsibleVersionsResponseData struct {
		Version         string    `json:"version"`
		Href            string    `json:"href"`
		Created         time.Time `json:"created_at"`
		Updated         time.Time `json:"updated_at"`
		RequiredAnsible string    `json:"requires_ansible"`
		Marks           []string  `json:"marks"`
	}

	type AnsibleVersionsResponse struct {
		Meta AnsibleVersionsResponseMeta   `json:"meta"`
		Data []AnsibleVersionsResponseData `json:"data"`
	}

	responseCoreData := make([]AnsibleVersionsResponseData, 0, len(pds))
	for _, pd := range pds {
		responseCoreData = append(responseCoreData, AnsibleVersionsResponseData{
			Version:         pd.SemVer.String(),
			Href:            fmt.Sprintf("/api/v3/collections/%v/%v/versions/%v/", packageNamespace, packageName, pd.SemVer),
			Created:         pd.Version.CreatedUnix.AsTime(),
			Updated:         pd.Version.CreatedUnix.AsTime(),
			RequiredAnsible: pd.Metadata.(*ansible_module.CollectionManifest).CollectionInfo.RequiresAnsible,
			Marks:           []string{},
		})
	}

	ctx.JSON(http.StatusOK, AnsibleVersionsResponse{
		Meta: AnsibleVersionsResponseMeta{
			Count: len(pds),
		},
		Data: responseCoreData,
	})
}

// ServeCollection returns a JSON object with the data of a single version of a collection.
func ServeCollection(ctx *context.Context) {
	packageNamespace := ctx.Params("namespace")
	packageName := ctx.Params("name")
	packageVersion := ctx.Params("version")

	pv, err := packages_model.GetVersionByNameAndVersion(ctx, ctx.Package.Owner.ID, packages_model.TypeAnsible, packageNamespace+"."+packageName, packageVersion)
	if err != nil {
		if errors.Is(err, packages_model.ErrPackageNotExist) {
			apiError(ctx, http.StatusNotFound, err)
			return
		}
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	pd, err := packages_model.GetPackageDescriptor(ctx, pv)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	if len(pd.Files) == 0 {
		apiError(ctx, http.StatusInternalServerError, "The package has no files attached to it. This seems to be an internal error.")
		return
	}
	fileDescriptor := pd.Files[0].File

	type AnsibleSpecificVersionResponseNamespace struct {
		Name string  `json:"name"`
		Hash *string `json:"metadata_sha256"`
	}
	type AnsibleSpecificVersionResponseCollection struct {
		Name string `json:"name"`
		Href string `json:"href"`
	}
	type AnsibleSpecificVersionResponseArtifact struct {
		Filename string  `json:"filename"`
		Hash     *string `json:"sha256"`
	}
	type AnsibleSpecificVersionResponseMetadata struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	type AnsibleSpecificVersionResponse struct {
		Version    string                                   `json:"version"`
		Href       string                                   `json:"href"`
		URL        string                                   `json:"download_url"`
		Namespace  AnsibleSpecificVersionResponseNamespace  `json:"namespace"`
		Collection AnsibleSpecificVersionResponseCollection `json:"collection"`
		Artifact   AnsibleSpecificVersionResponseArtifact   `json:"artifact"`
		Metadata   AnsibleSpecificVersionResponseMetadata   `json:"metadata"`
	}

	ctx.JSON(http.StatusOK, AnsibleSpecificVersionResponse{
		Version: pd.SemVer.String(),
		Href:    fmt.Sprintf("/api/v3/collections/%v/%v/versions/%v/", packageNamespace, packageName, pd.SemVer),
		URL:     util.URLJoin(setting.AppURL, pd.VersionWebLink(), fmt.Sprintf("/files/%v/", fileDescriptor.ID)),
		Namespace: AnsibleSpecificVersionResponseNamespace{
			Name: packageNamespace,
		},
		Collection: AnsibleSpecificVersionResponseCollection{
			Name: packageName,
			Href: fmt.Sprintf("/api/v3/collections/%v/%v/", packageNamespace, packageName),
		},
		Artifact: AnsibleSpecificVersionResponseArtifact{
			Filename: fileDescriptor.LowerName,
		},
		Metadata: AnsibleSpecificVersionResponseMetadata{
			Dependencies: pd.Metadata.(*ansible_module.CollectionManifest).CollectionInfo.Dependencies,
		},
	})
}
