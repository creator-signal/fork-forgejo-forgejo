// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"net/http"
	"slices"
	"strings"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	"forgejo.org/modules/json"
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

// Implements https://github.com/flatpak/flatpak-oci-specs/blob/32a803aaa58f8406b49c5f9c81fc3f6ca761c06d/registry-index.md
func Index(ctx *context.Context) {
	architectureParam := ctx.FormString("architecture")
	tagParam := ctx.FormString("tag")
	osParam := ctx.FormString("os")

	searchOptions := &packages_model.PackageSearchOptions{
		OwnerID:    ctx.ContextUser.ID,
		Type:       packages_model.TypeContainer,
		Properties: make(map[string]string),
	}

	if tagParam != "" {
		searchOptions.Version = packages_model.SearchValue{Value: tagParam, ExactMatch: true}
	}

	if osParam != "" {
		searchOptions.Properties[container_module.PropertyOperatingSystem] = osParam
	}

	if architectureParam != "" {
		searchOptions.Properties[container_module.PropertyArchitecture] = architectureParam
	}

	// Search for the label
	for key, values := range ctx.Req.URL.Query() {
		if len(values) != 1 {
			continue
		}

		if !strings.HasPrefix(key, "label:") {
			continue
		}

		label, _ := strings.CutPrefix(key, "label:")

		if strings.HasSuffix(label, ":exists") {
			name, _ := strings.CutSuffix(label, ":exists")
			searchOptions.ExistingProperties = append(searchOptions.ExistingProperties, container_module.GetLabelPropertyKey(name))
		} else {
			searchOptions.Properties[label] = values[0]
		}
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

		properties, err := getAllPackageVersionProperties(ctx, packageVersion)
		if err != nil {
			apiError(ctx, http.StatusInternalServerError, err)
			return
		}

		image := indexImage{
			Tags:         []string{packageVersion.Version},
			OS:           metadata.OperatingSystem,
			Architecture: metadata.Architecture,
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
