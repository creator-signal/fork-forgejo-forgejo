// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/http"

	"forgejo.org/models"
	"forgejo.org/models/organization"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

// MembersAction response for operation to a member of organization
func MembersAction(ctx *context.Context) {
	var redirect string
	uid := ctx.FormInt64("uid")
	if uid == 0 {
		ctx.Redirect(string(tplSettingsOrganization))
		return
	}

	org := ctx.Org.Organization
	var err error
	switch ctx.Params(":action") {
	case "private":
		if ctx.Doer.ID != uid && !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			return
		}
		err = organization.ChangeOrgUserStatus(ctx, org.ID, uid, false)
	case "public":
		if ctx.Doer.ID != uid && !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			return
		}
		err = organization.ChangeOrgUserStatus(ctx, org.ID, uid, true)
	case "remove":
		if !ctx.Org.IsOwner {
			ctx.Error(http.StatusNotFound)
			return
		}
		err = models.RemoveOrgUser(ctx, org.ID, uid)
		if organization.IsErrLastOrgOwner(err) {
			ctx.Flash.Error(ctx.Tr("form.last_org_owner"))
			ctx.JSONRedirect(ctx.Org.OrgLink + "/members")
			return
		}
	case "leave":
		err = models.RemoveOrgUser(ctx, org.ID, ctx.Doer.ID)
		if err == nil {
			ctx.Flash.Success(ctx.Tr("form.organization_leave_success", org.DisplayName()))
			ctx.JSON(http.StatusOK, map[string]any{
				"redirect": "", // keep the user stay on current page, in case they want to do other operations.
			})
		} else if organization.IsErrLastOrgOwner(err) {
			ctx.Flash.Error(ctx.Tr("form.last_org_owner"))
			ctx.JSONRedirect(ctx.Org.OrgLink + "/members")
		} else {
			log.Error("RemoveOrgUser(%d,%d): %v", org.ID, ctx.Doer.ID, err)
		}
		return
	}

	if err != nil {
		log.Error("Action(%s): %v", ctx.Params(":action"), err)
		ctx.JSON(http.StatusOK, map[string]any{
			"ok":  false,
			"err": err.Error(),
		})
		return
	}

	if ctx.Doer.ID == uid {
		redirect = string(tplSettingsOrganization)
	} else {
		redirect = ctx.Org.OrgLink + "/members"
	}

	if ctx.Params(":action") == "leave" {
		redirect = setting.AppSubURL + "/"
	}

	ctx.JSONRedirect(redirect)
}
