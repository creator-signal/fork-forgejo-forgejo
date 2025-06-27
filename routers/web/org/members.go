// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors.
// SPDX-License-Identifier: MIT

package org

import (
	"net/http"

	"forgejo.org/models/organization"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	shared_user "forgejo.org/routers/web/shared/user"
	"forgejo.org/services/context"
)

const (
	// tplMembers template for organization members page
	tplMembers base.TplName = "org/member/members"
)

// Members render organization users page
func Members(ctx *context.Context) {
	org := ctx.Org.Organization
	ctx.Data["Title"] = org.FullName
	ctx.Data["PageIsOrgMembers"] = true

	page := ctx.FormInt("page")
	if page <= 1 {
		page = 1
	}

	opts := &organization.FindOrgMembersOpts{
		Doer:  ctx.Doer,
		OrgID: org.ID,
	}

	if ctx.Doer != nil {
		isMember, err := ctx.Org.Organization.IsOrgMember(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "IsOrgMember")
			return
		}
		opts.IsDoerMember = isMember
	}
	ctx.Data["PublicOnly"] = opts.PublicOnly()

	total, err := organization.CountOrgMembers(ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CountOrgMembers")
		return
	}

	err = shared_user.LoadHeaderCount(ctx)
	if err != nil {
		ctx.ServerError("LoadHeaderCount", err)
		return
	}

	pager := context.NewPagination(int(total), setting.UI.MembersPagingNum, page, 5)
	opts.Page = page
	opts.PageSize = setting.UI.MembersPagingNum
	members, membersIsPublic, err := organization.FindOrgMembers(ctx, opts)
	if err != nil {
		ctx.ServerError("GetMembers", err)
		return
	}
	ctx.Data["Page"] = pager
	ctx.Data["Members"] = members
	ctx.Data["MembersIsPublicMember"] = membersIsPublic
	ctx.Data["MembersIsUserOrgOwner"] = organization.IsUserOrgOwner(ctx, members, org.ID)
	ctx.Data["MembersTwoFaStatus"] = members.GetTwoFaStatus(ctx)

	ctx.HTML(http.StatusOK, tplMembers)
}
