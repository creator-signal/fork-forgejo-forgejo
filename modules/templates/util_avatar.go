// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"context"
	"encoding/hex"
	"fmt"
	"html"
	"html/template"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/avatars"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	gitea_html "forgejo.org/modules/html"
	"forgejo.org/modules/setting"
)

type AvatarUtils struct {
	ctx context.Context
}

func NewAvatarUtils(ctx context.Context) *AvatarUtils {
	return &AvatarUtils{ctx: ctx}
}

// AvatarHTML creates the HTML for an avatar
func AvatarHTML(src string, size int, class, name string) template.HTML {
	sizeStr := fmt.Sprintf(`%d`, size)

	if name == "" {
		name = "avatar"
	}

	return template.HTML(`<img loading="lazy" alt="" class="` + class + `" src="` + src + `" title="` + html.EscapeString(name) + `" width="` + sizeStr + `" height="` + sizeStr + `"/>`)
}

// AvatarHTMLSVG creates the HTML for an SVG avatar
func AvatarHTMLSVG(size int, svgHash, class, name string) template.HTML {
	sizeStr := fmt.Sprintf(`%d`, size)

	if name == "" {
		name = "avatar"
	}

	return template.HTML(`<img loading="lazy" alt="" class="svg identicon ` + class + `" src="` + setting.AppSubURL + `/svg-avatars/` + svgHash + `.svg" title="` + html.EscapeString(name) + `" width="` + sizeStr + `" height="` + sizeStr + `"/>`)
}

// Avatar renders user avatars. args: user, size (int), class (string)
func (au *AvatarUtils) Avatar(item any, others ...any) template.HTML {
	size, class := gitea_html.ParseSizeAndClass(avatars.DefaultAvatarPixelSize, avatars.DefaultAvatarClass, others...)

	var vectorHash []byte
	var rasterUrl string
	var displayName string

	switch t := item.(type) {
	case *user_model.User:
		vectorHash = t.AvatarSVGHash
		rasterUrl = t.AvatarLinkWithSize(au.ctx, size*setting.Avatar.RenderedSizeFactor)
		displayName = t.DisplayName()
	case *repo_model.Collaborator:
		vectorHash = t.AvatarSVGHash
		rasterUrl = t.AvatarLinkWithSize(au.ctx, size*setting.Avatar.RenderedSizeFactor)
		displayName = t.DisplayName()
	case *organization.Organization:
		vectorHash = t.AsUser().AvatarSVGHash
		rasterUrl = t.AsUser().AvatarLinkWithSize(au.ctx, size*setting.Avatar.RenderedSizeFactor)
		displayName = t.DisplayName()
	}

	// Try vector avatar first - if it is unwanted it wouldn't be present in the database
	if vectorHash != nil {
		return AvatarHTMLSVG(size, hex.EncodeToString(vectorHash), class, displayName)
	}

	// Fall back to raster avatar if present
	if rasterUrl != "" {
		return AvatarHTML(rasterUrl, size, class, displayName)
	}

	// Fall back to default avatar URL
	return AvatarHTML(avatars.DefaultAvatarLink(), size, class, "")
}

// AvatarByAction renders user avatars from action. args: action, size (int), class (string)
func (au *AvatarUtils) AvatarByAction(action *activities_model.Action, others ...any) template.HTML {
	action.LoadActUser(au.ctx)
	return au.Avatar(action.ActUser, others...)
}

// AvatarByEmail renders avatars by email address. args: email, name, size (int), class (string)
func (au *AvatarUtils) AvatarByEmail(email, name string, others ...any) template.HTML {
	size, class := gitea_html.ParseSizeAndClass(avatars.DefaultAvatarPixelSize, avatars.DefaultAvatarClass, others...)
	src := avatars.GenerateEmailAvatarFastLink(au.ctx, email, size*setting.Avatar.RenderedSizeFactor)

	if src != "" {
		return AvatarHTML(src, size, class, name)
	}

	return ""
}
