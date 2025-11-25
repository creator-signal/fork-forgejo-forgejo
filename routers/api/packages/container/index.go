// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

type indexImage struct {
	Tags         []string
	Digest       string
	MediaType    string
	OS           string
	Architecture string
	Annotations  map[string]string
	Labels       map[string]string
}

type indexRepository struct {
	Name   string
	Images []indexImage
}

type indexResponse struct {
	Registry string
	Results  []indexRepository
}

// Implements the check for the label: param
func checkLabelParam(labels map[string]string, labelParam string) (bool, error) {
	if labelParam == "" {
		return true, nil
	}

	if strings.HasSuffix(labelParam, ":exists=1") {
		labelName := strings.TrimSuffix(labelParam, ":exists=1")
		_, ok := labels[labelName]
		return ok, nil
	}

	labelParamParts := strings.Split(labelParam, "=")
	if len(labelParamParts) != 2 {
		return false, fmt.Errorf("invalid label parameter")
	}

	content, ok := labels[labelParamParts[0]]
	if !ok {
		return false, nil
	}

	return labelParamParts[1] == content, nil
}

// Returns all properties needed by index
func getAllPackageVersionProperties(ctx *context.Context, packageVersion *packages_model.PackageVersion) ([]*packages_model.PackageProperty, error) {
	// FIrst we need the container.repository property which is part of the package
	packageProperties, err := packages_model.GetProperties(ctx, packages_model.PropertyTypePackage, packageVersion.PackageID)
	if err != nil {
		return nil, err
	}

	// Now we need the container.digest and container.mediatype properties
	// Both are properties of the manifest.json file
	// First we need to get the package file
	packageFile, err := packages_model.GetFileForVersionByName(ctx, packageVersion.ID, container_model.ManifestFilename, "")
	if err != nil {
		return nil, err
	}

	// Now we can use the file id to query the properties
	fileProperties, err := packages_model.GetProperties(ctx, packages_model.PropertyTypeFile, packageFile.ID)
	if err != nil {
		return nil, err
	}

	return slices.Concat(packageProperties, fileProperties), nil
}

// Returns the label param which is separated by a : rather than a =
func getLabelParam(rawQuery string) string {
	unescapedQuery, err := url.QueryUnescape(rawQuery)
	if err != nil {
		return ""
	}

	for _, currentPart := range strings.Split(unescapedQuery, "&") {
		if strings.HasPrefix(currentPart, "label:") {
			return strings.TrimPrefix(currentPart, "label:")
		}
	}

	return ""
}

// Implements https://github.com/flatpak/flatpak-oci-specs/blob/32a803aaa58f8406b49c5f9c81fc3f6ca761c06d/registry-index.md
func Index(ctx *context.Context) {
	architectureParam := ctx.FormString("architecture")
	labelParam := getLabelParam(ctx.Req.URL.RawQuery)
	tagParam := ctx.FormString("tag")
	osParam := ctx.FormString("os")

	searchOptions := &packages_model.PackageSearchOptions{
		OwnerID: ctx.ContextUser.ID,
		Type:    packages_model.TypeContainer,
	}

	if tagParam != "" {
		searchOptions.Version = packages_model.SearchValue{Value: tagParam, ExactMatch: true}
	}

	pvs, _, err := packages_model.SearchVersions(ctx, searchOptions)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	repositories := make([]indexRepository, 0)
	for _, packageVersion := range pvs {
		var metadata container_module.Metadata
		err = json.Unmarshal([]byte(packageVersion.MetadataJSON), &metadata)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		platformParts := strings.Split(metadata.Platform, "/")
		if len(platformParts) != 2 {
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("Invalid platform: %s", metadata.Platform))
			return
		}

		os := platformParts[0]
		architecture := platformParts[1]

		if osParam != "" && osParam != os {
			continue
		}

		if architectureParam != "" && architectureParam != architecture {
			continue
		}

		labelAllowed, err := checkLabelParam(metadata.Labels, labelParam)
		if err != nil {
			apiError(ctx, http.StatusBadRequest, err)
			return
		}
		if !labelAllowed {
			continue
		}

		properties, err := getAllPackageVersionProperties(ctx, packageVersion)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		image := indexImage{
			Tags:         []string{packageVersion.Version},
			OS:           os,
			Architecture: architecture,
			Annotations:  make(map[string]string),
			Labels:       metadata.Labels,
		}

		repoName := ""
		for _, property := range properties {
			switch property.Name {
			case container_module.PropertyRepository:
				repoName = property.Value
			case container_module.PropertyDigest:
				image.Digest = property.Value
			case container_module.PropertyMediaType:
				image.MediaType = property.Value
			}
		}

		repository := indexRepository{Name: repoName, Images: []indexImage{image}}
		repositories = append(repositories, repository)
	}

	ctx.JSON(http.StatusOK, indexResponse{
		Registry: setting.AppURL,
		Results:  repositories,
	})
}
