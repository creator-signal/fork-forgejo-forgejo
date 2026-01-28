// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"errors"
	"net/http"

	"forgejo.org/models/organization"
	"forgejo.org/models/packages"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	packages_service "forgejo.org/services/packages"
	container_client "forgejo.org/services/packages/container"
)

// ListPackages gets all packages of an owner
func ListPackages(ctx *context.APIContext) {
	// swagger:operation GET /packages/{owner} package listPackages
	// ---
	// summary: Gets all packages of an owner
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the packages
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// - name: type
	//   in: query
	//   description: package type filter
	//   type: string
	//   enum: [alpine, cargo, chef, composer, conan, conda, container, cran, debian, generic, go, helm, maven, npm, nuget, pub, pypi, rpm, rubygems, swift, vagrant]
	// - name: q
	//   in: query
	//   description: name filter
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/PackageList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	listOptions := utils.GetListOptions(ctx)

	packageType := ctx.FormTrim("type")
	query := ctx.FormTrim("q")

	pvs, count, err := packages.SearchVersions(ctx, &packages.PackageSearchOptions{
		OwnerID:    ctx.Package.Owner.ID,
		Type:       packages.Type(packageType),
		Name:       packages.SearchValue{Value: query},
		IsInternal: optional.Some(false),
		Paginator:  &listOptions,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "SearchVersions", err)
		return
	}

	pds, err := packages.GetPackageDescriptors(ctx, pvs)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetPackageDescriptors", err)
		return
	}

	apiPackages := make([]*api.Package, 0, len(pds))
	for _, pd := range pds {
		apiPackage, err := convert.ToPackage(ctx, pd, ctx.Doer)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "Error converting package for api", err)
			return
		}
		apiPackages = append(apiPackages, apiPackage)
	}

	ctx.SetLinkHeader(int(count), listOptions.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, apiPackages)
}

// GetPackage gets a package
func GetPackage(ctx *context.APIContext) {
	// swagger:operation GET /packages/{owner}/{type}/{name}/{version} package getPackage
	// ---
	// summary: Gets a package
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the package
	//   type: string
	//   required: true
	// - name: type
	//   in: path
	//   description: type of the package
	//   type: string
	//   required: true
	// - name: name
	//   in: path
	//   description: name of the package
	//   type: string
	//   required: true
	// - name: version
	//   in: path
	//   description: version of the package
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Package"
	//   "404":
	//     "$ref": "#/responses/notFound"

	apiPackage, err := convert.ToPackage(ctx, ctx.Package.Descriptor, ctx.Doer)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "Error converting package for api", err)
		return
	}

	ctx.JSON(http.StatusOK, apiPackage)
}

// DeletePackage deletes a package
func DeletePackage(ctx *context.APIContext) {
	// swagger:operation DELETE /packages/{owner}/{type}/{name}/{version} package deletePackage
	// ---
	// summary: Delete a package
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the package
	//   type: string
	//   required: true
	// - name: type
	//   in: path
	//   description: type of the package
	//   type: string
	//   required: true
	// - name: name
	//   in: path
	//   description: name of the package
	//   type: string
	//   required: true
	// - name: version
	//   in: path
	//   description: version of the package
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	err := packages_service.RemovePackageVersion(ctx, ctx.Doer, ctx.Package.Descriptor.Version)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "RemovePackageVersion", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// ListPackageFiles gets all files of a package
func ListPackageFiles(ctx *context.APIContext) {
	// swagger:operation GET /packages/{owner}/{type}/{name}/{version}/files package listPackageFiles
	// ---
	// summary: Gets all files of a package
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the package
	//   type: string
	//   required: true
	// - name: type
	//   in: path
	//   description: type of the package
	//   type: string
	//   required: true
	// - name: name
	//   in: path
	//   description: name of the package
	//   type: string
	//   required: true
	// - name: version
	//   in: path
	//   description: version of the package
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/PackageFileList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	apiPackageFiles := make([]*api.PackageFile, 0, len(ctx.Package.Descriptor.Files))
	for _, pfd := range ctx.Package.Descriptor.Files {
		apiPackageFiles = append(apiPackageFiles, convert.ToPackageFile(pfd))
	}

	ctx.JSON(http.StatusOK, apiPackageFiles)
}

// LinkPackage sets a repository link for a package
func LinkPackage(ctx *context.APIContext) {
	// swagger:operation POST /packages/{owner}/{type}/{name}/-/link/{repo_name} package linkPackage
	// ---
	// summary: Link a package to a repository
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the package
	//   type: string
	//   required: true
	// - name: type
	//   in: path
	//   description: type of the package
	//   type: string
	//   required: true
	// - name: name
	//   in: path
	//   description: name of the package
	//   type: string
	//   required: true
	// - name: repo_name
	//   in: path
	//   description: name of the repository to link.
	//   type: string
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	pkg, err := packages.GetPackageByName(ctx, ctx.ContextUser.ID, packages.Type(ctx.PathParamRaw("type")), ctx.PathParamRaw("name"))
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetPackageByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetPackageByName", err)
		}
		return
	}

	repo, err := repo_model.GetRepositoryByName(ctx, ctx.ContextUser.ID, ctx.PathParamRaw("repo_name"))
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetRepositoryByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetRepositoryByName", err)
		}
		return
	}

	err = packages_service.LinkToRepository(ctx, pkg, repo, ctx.Doer)
	if err != nil {
		switch {
		case errors.Is(err, util.ErrInvalidArgument):
			ctx.Error(http.StatusBadRequest, "LinkToRepository", err)
		case errors.Is(err, util.ErrPermissionDenied):
			ctx.Error(http.StatusForbidden, "LinkToRepository", err)
		default:
			ctx.Error(http.StatusInternalServerError, "LinkToRepository", err)
		}
		return
	}
	ctx.Status(http.StatusCreated)
}

// UnlinkPackage sets a repository link for a package
func UnlinkPackage(ctx *context.APIContext) {
	// swagger:operation POST /packages/{owner}/{type}/{name}/-/unlink package unlinkPackage
	// ---
	// summary: Unlink a package from a repository
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the package
	//   type: string
	//   required: true
	// - name: type
	//   in: path
	//   description: type of the package
	//   type: string
	//   required: true
	// - name: name
	//   in: path
	//   description: name of the package
	//   type: string
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	pkg, err := packages.GetPackageByName(ctx, ctx.ContextUser.ID, packages.Type(ctx.PathParamRaw("type")), ctx.PathParamRaw("name"))
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetPackageByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetPackageByName", err)
		}
		return
	}

	err = packages_service.UnlinkFromRepository(ctx, pkg, ctx.Doer)
	if err != nil {
		switch {
		case errors.Is(err, util.ErrPermissionDenied):
			ctx.Error(http.StatusForbidden, "UnlinkFromRepository", err)
		case errors.Is(err, util.ErrInvalidArgument):
			ctx.Error(http.StatusBadRequest, "UnlinkFromRepository", err)
		default:
			ctx.Error(http.StatusInternalServerError, "UnlinkFromRepository", err)
		}
		return
	}
	ctx.Status(http.StatusNoContent)
}

// CreateRemoteRegistry creates a remote registry of a given type
func CreateRemoteRegistry(ctx *context.APIContext) {
	// swagger:operation POST /packages/{owner}/remote-registry package createRemoteRegistry
	// ---
	// summary: Allows creation of a remote registry
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the registry
	//   type: string
	//   required: true
	// - name: remote_registry
	//   in: body
	//   required: true
	//   schema: { "$ref": "#/definitions/CreateRemoteRegistryOption" }
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx)
	rrOpts := form.(*api.CreateRemoteRegistryOption)

	isOrg := ctx.ContextUser.IsOrganization()
	isOrgOwner, err := organization.IsOrganizationOwner(ctx, ctx.ContextUser.ID, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateRemoteRegistry", err)
	}

	// Permissions
	if isOrg {
		// Then user needs to be org owner or has write permissions to org
		if !isOrgOwner && !ctx.Doer.IsAdmin {
			ctx.Error(http.StatusForbidden, "Create remote registry not allowed", nil)
			return
		}
	}

	ownerType, err := packages_service.GetOwnerType(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateRemoteRegistry", err)
	}

	rr, err := packages_service.NewRemoteRegistry(
		packages_service.RROpts{
			Name:       rrOpts.Name,
			RemoteURL:  rrOpts.RemoteURL,
			RemoteType: packages.Type(rrOpts.RemoteType),
			OwnerType:  ownerType,
			OwnerID:    ctx.ContextUser.ID,
			Auth: packages_service.RRCredentials{
				RemoteUser:     rrOpts.RemoteUser,
				RemotePassword: rrOpts.RemotePassword,
				RemoteToken:    rrOpts.RemoteToken,
			},
		})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateRemoteRegistry", err)
	}

	connected := true
	if rrOpts.TestConnection {
		registryClient := container_client.NewContainerRegistryClient(&rr)
		connected, err = registryClient.RemoteRegistryConnected(ctx)
	}

	if !connected {
		ctx.Error(http.StatusInternalServerError, "Connection Test", err)
	}

	err = packages_service.CreateRemoteRegistry(ctx, rr)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateRemoteRegistry", err)
	}

	ctx.JSON(http.StatusCreated, convert.ToRemoteRegistry(&rr))
}

// CreateRemotTestRemoteRegistryConnection tests the availability and the credentials of the given registry
func TestRemoteRegistryConnection(ctx *context.APIContext) {
	// swagger:operation POST /packages/{owner}/remote-registry/{registry-name}/test package testRemoteRegistryConnection
	// ---
	// summary: Test if the remote registry is actually connected
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the package
	//   type: string
	//   required: true
	// - name: registry-name
	//   in: path
	//   description: name of the registry
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/empty"

	// Check if doer can do

	isOrg := ctx.ContextUser.IsOrganization()
	isOrgOwner, err := organization.IsOrganizationOwner(ctx, ctx.ContextUser.ID, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "TestRemoteRegistryConnection", err)
	}

	// Permissions
	if isOrg {
		// Then user needs to be org owner or has write permissions to org
		if !isOrgOwner && !ctx.Doer.IsAdmin {
			ctx.Error(http.StatusForbidden, "Create remote registry not allowed", nil)
			return
		}
	}

	rr, err := packages_service.GetRemoteRegistryFromContext(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "TestRemoteRegistryConnection", err)
	}

	registryClient := container_client.NewContainerRegistryClient(rr)
	connected, err := registryClient.RemoteRegistryConnected(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "TestRemoteRegistryConnection", err)
	}

	if !connected {
		ctx.Error(http.StatusInternalServerError, "TestRemoteRegistryConnection", err)
	}

	ctx.Status(http.StatusOK)
}
