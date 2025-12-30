// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"encoding/hex"
	"strings"
	"time"

	"forgejo.org/models/avatars"
	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/httpcache"
	"forgejo.org/services/context"
)

func cacheableRedirect(ctx *context.Context, location string) {
	// here we should not use `setting.StaticCacheTime`, it is pretty long (default: 6 hours)
	// we must make sure the redirection cache time is short enough, otherwise a user won't see the updated avatar in 6 hours
	// it's OK to make the cache time short, it is only a redirection, and doesn't cost much to make a new request
	httpcache.SetCacheControlInHeader(ctx.Resp.Header(), 5*time.Minute)
	ctx.Redirect(location)
}

// AvatarByUserName redirect browser to user avatar of requested size
func AvatarByUserName(ctx *context.Context) {
	userName := ctx.Params(":username")
	size := int(ctx.ParamsInt64(":size"))

	var user *user_model.User
	if strings.ToLower(userName) != user_model.GhostUserLowerName {
		var err error
		if user, err = user_model.GetUserByName(ctx, userName); err != nil {
			if user_model.IsErrUserNotExist(err) {
				ctx.NotFound("GetUserByName", err)
				return
			}
			ctx.ServerError("Invalid user: "+userName, err)
			return
		}
	} else {
		user = user_model.NewGhostUser()
	}

	cacheableRedirect(ctx, user.AvatarLinkWithSize(ctx, size))
}

// AvatarByEmailHash redirects the browser to the email avatar link
func AvatarByEmailHash(ctx *context.Context) {
	hash := ctx.Params(":hash")

	email, err := avatars.GetEmailForHash(ctx, hash)
	if err != nil {
		ctx.ServerError("invalid avatar hash: "+hash, err)
		return
	}
	size := ctx.FormInt("size")
	cacheableRedirect(ctx, avatars.GenerateEmailAvatarFinalLink(ctx, email, size))
}

// ConvertSvgAvatar adds vital boilerplate to generated identicons
func ConvertSvgAvatar(raw string, isCustomAvatar bool) string {
	if isCustomAvatar {
		return raw
	}
	return `<svg viewBox="0 0 72 72" xmlns="http://www.w3.org/2000/svg">` + raw + `</svg>`
}

// FindSvgAvatarByHash looks for svg avatar in the database
func FindSvgAvatarByHash(ctx *context.Context, hash []byte) (string, error) {
	user := new(user_model.User)
	found, err := db.GetEngine(ctx).Where("avatar_svg_hash=?", hash).Get(user)
	if err != nil {
		return "", err
	}

	if found && user.AvatarSVG != "" {
		return ConvertSvgAvatar(user.AvatarSVG, user.UseCustomAvatar), nil
	}
	return "", nil
}

// GetSvgAvatarByHash does not make this code not compile
func GetSvgAvatarByHash(ctx *context.Context) {
	hash := ctx.Params(":hash")

	hashBytes, err := hex.DecodeString(hash)
	if err != nil {
		ctx.ServerError("Invalid hex of avatar hash: "+hash, err)
		return
	}

	avatarXML, err := FindSvgAvatarByHash(ctx, hashBytes)
	if err != nil {
		ctx.ServerError("Invalid avatar hash: "+hash, err)
		return
	}

	if avatarXML != "" {
		ctx.Resp.Header().Add("Content-Type", "image/svg+xml")
		_, err := ctx.Resp.Write([]byte(avatarXML))
		if err != nil {
			ctx.ServerError("Couldn't write response (how is client supposed to see this then)", err)
		}
		return
	}
	ctx.NotFound("Avatar was not found", nil)
}
