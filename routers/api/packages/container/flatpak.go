// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"fmt"
	"net/http"
	"strings"

	packages_model "forgejo.org/models/packages"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	container_module "forgejo.org/modules/packages/container"
	flatpak_module "forgejo.org/modules/packages/container/flatpak"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

func FlatpakRepo(ctx *context.Context) {
	var repoBuilder strings.Builder
	fmt.Fprintf(&repoBuilder, "[Flatpak Repo]\n")
	fmt.Fprintf(&repoBuilder, "Title=%s-%s\n", setting.AppName, ctx.ContextUser.Name)
	fmt.Fprintf(&repoBuilder, "Url=oci+%sapi/packages/%s/container\n", setting.AppURL, ctx.ContextUser.Name)
	fmt.Fprintf(&repoBuilder, "Homepage=%s\n", ctx.ContextUser.HTMLURL())
	fmt.Fprintf(&repoBuilder, "Comment=Flatpak repo of %s/%s\n", setting.AppName, ctx.ContextUser.Name)
	fmt.Fprintf(&repoBuilder, "Description=Flatpak repo of %s/%s\n", setting.AppName, ctx.ContextUser.Name)
	fmt.Fprintf(&repoBuilder, "Icon=%s\n", ctx.ContextUser.AvatarLink(ctx))

	ctx.Resp.Header().Set("Content-Type", "application/vnd.flatpak.repo")
	ctx.Resp.WriteHeader(http.StatusOK)
	_, err := ctx.Resp.Write([]byte(repoBuilder.String()))
	if err != nil {
		log.Error("write to resp err: %v", err)
	}
}

func getFlatpakPackageVersion(ctx *context.Context) (bool, *packages_model.PackageVersion, *container_module.Metadata) {
	packageName := ctx.Params(":package")
	version := ctx.Params(":version")

	packageVersion, err := packages_model.GetVersionByNameAndVersion(ctx, ctx.ContextUser.ID, packages_model.TypeContainer, packageName, version)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return false, nil, nil
	}

	metadata := new(container_module.Metadata)
	err = json.Unmarshal([]byte(packageVersion.MetadataJSON), metadata)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return false, nil, nil
	}

	if metadata.Type != container_module.TypeFlatpak || metadata.Flatpak == nil {
		apiError(ctx, http.StatusBadGateway, fmt.Errorf("Not an Flatpak"))
		return false, nil, nil
	}

	return true, packageVersion, metadata
}

func FlatpakRef(ctx *context.Context) {
	ok, _, metadata := getFlatpakPackageVersion(ctx)
	if !ok {
		return
	}

	var refBuilder strings.Builder
	fmt.Fprintf(&refBuilder, "[Flatpak Ref]\n")
	fmt.Fprintf(&refBuilder, "Name=%s\n", metadata.Flatpak.ID)
	fmt.Fprintf(&refBuilder, "Title=%s from %s\n", metadata.Flatpak.ID, setting.AppName)
	fmt.Fprintf(&refBuilder, "SuggestRemoteName=%s\n", flatpak_module.GetRepoName(ctx.ContextUser.LowerName))
	fmt.Fprintf(&refBuilder, "Url=%s\n", flatpak_module.GetRepoURL(ctx.ContextUser.LowerName))
	fmt.Fprintf(&refBuilder, "Branch=%s\n", metadata.Flatpak.Branch)

	if metadata.Flatpak.RuntimeRepo == "" {
		fmt.Fprintf(&refBuilder, "RuntimeRepo=%s\n", flatpak_module.GetRepoInfoURL(ctx.ContextUser.LowerName))
	} else {
		fmt.Fprintf(&refBuilder, "RuntimeRepo=%s\n", metadata.Flatpak.RuntimeRepo)
	}

	if metadata.Flatpak.Kind == flatpak_module.FlatpakKindRuntime {
		fmt.Fprintf(&refBuilder, "IsRuntime=true\n")
	} else {
		fmt.Fprintf(&refBuilder, "IsRuntime=false\n")
	}

	ctx.Resp.Header().Set("Content-Type", "application/vnd.flatpak.ref")
	ctx.Resp.WriteHeader(http.StatusOK)
	_, err := ctx.Resp.Write([]byte(refBuilder.String()))
	if err != nil {
		log.Error("write to resp err: %v", err)
	}
}
