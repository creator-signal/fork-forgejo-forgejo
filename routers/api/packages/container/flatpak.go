// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	packages_model "forgejo.org/models/packages"
	container_module "forgejo.org/modules/packages/container"
	flatpak_module "forgejo.org/modules/packages/container/flatpak"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

func FlatpakRepo(ctx *context.Context) {
	var repoBuilder strings.Builder
	repoBuilder.WriteString("[Flatpak Repo]\n")
	repoBuilder.WriteString(fmt.Sprintf("Title=%s-%s\n", setting.AppName, ctx.ContextUser.Name))
	repoBuilder.WriteString(fmt.Sprintf("Url=oci+%sapi/packages/%s/container\n", setting.AppURL, ctx.ContextUser.Name))
	repoBuilder.WriteString(fmt.Sprintf("Homepage=%s\n", ctx.ContextUser.HTMLURL()))
	repoBuilder.WriteString(fmt.Sprintf("Comment=Flatpak repo of %s/%s\n", setting.AppName, ctx.ContextUser.Name))
	repoBuilder.WriteString(fmt.Sprintf("Description=Flatpak repo of %s/%s\n", setting.AppName, ctx.ContextUser.Name))
	repoBuilder.WriteString(fmt.Sprintf("Icon=%s\n", ctx.ContextUser.AvatarLink(ctx)))

	ctx.Resp.Header().Set("Content-Type", "application/vnd.flatpak.repo")
	ctx.Resp.WriteHeader(http.StatusOK)
	ctx.Resp.Write([]byte(repoBuilder.String()))
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

func SetFlatpakRuntimeRepo(ctx *context.Context) {
	ok, packageVersion, metadata := getFlatpakPackageVersion(ctx)
	if !ok {
		return
	}

	body, err := io.ReadAll(ctx.Req.Body)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	metadata.Flatpak.RuntimeRepo = string(body)
	fmt.Println(metadata.Flatpak)

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	packageVersion.MetadataJSON = string(metadataJSON)

	err = packages_model.UpdateVersion(ctx, packageVersion)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func FlatpakRef(ctx *context.Context) {
	ok, _, metadata := getFlatpakPackageVersion(ctx)
	if !ok {
		return
	}

	var refBuilder strings.Builder
	refBuilder.WriteString("[Flatpak Ref]\n")
	refBuilder.WriteString(fmt.Sprintf("Name=%s\n", metadata.Flatpak.ID))
	refBuilder.WriteString(fmt.Sprintf("Title=%s from %s\n", metadata.Flatpak.ID, setting.AppName))
	refBuilder.WriteString(fmt.Sprintf("SuggestRemoteName=%s\n", flatpak_module.GetRepoName(ctx.ContextUser.LowerName)))
	refBuilder.WriteString(fmt.Sprintf("Url=%s\n", flatpak_module.GetRepoURL(ctx.ContextUser.LowerName)))
	refBuilder.WriteString(fmt.Sprintf("Branch=%s\n", metadata.Flatpak.Branch))

	if metadata.Flatpak.RuntimeRepo == "" {
		refBuilder.WriteString(fmt.Sprintf("RuntimeRepo=%s\n", flatpak_module.GetRepoInfoURL(ctx.ContextUser.LowerName)))
	} else {
		refBuilder.WriteString(fmt.Sprintf("RuntimeRepo=%s\n", metadata.Flatpak.RuntimeRepo))
	}

	if metadata.Flatpak.Kind == flatpak_module.FlatpakKindRuntime {
		refBuilder.WriteString("IsRuntime=true\n")
	} else {
		refBuilder.WriteString("IsRuntime=false\n")
	}

	ctx.Resp.Header().Set("Content-Type", "application/vnd.flatpak.ref")
	ctx.Resp.WriteHeader(http.StatusOK)
	ctx.Resp.Write([]byte(refBuilder.String()))
}
