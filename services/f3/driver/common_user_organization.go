// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	avatar_module "forgejo.org/modules/avatar"
	"forgejo.org/modules/storage"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_http "code.forgejo.org/f3/gof3/v3/util/http"
)

func (o *common) avatarIsUpToDate(_ context.Context, avatarHash string, id int64, avatar []byte) bool {
	return avatarHash == avatar_module.HashAvatar(id, avatar)
}

func (o *common) getUserAvatar(ctx context.Context, user *user_model.User) string {
	url := user.AvatarLink(ctx)
	return f3.AvatarEncode(f3_http.GetBody(&http.Client{}, url))
}

func (o *common) setUserAvatar(ctx context.Context, user *user_model.User, avatarEncoded string) {
	avatarData := f3.AvatarDecode(avatarEncoded)
	if user.UseCustomAvatar && o.avatarIsUpToDate(ctx, user.Avatar, user.ID, avatarData) {
		return
	}

	// do not use user_service.UploadAvatar because it could transform the avatar and
	// different from the one found in the source forge
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		panic(fmt.Errorf("db.TxContext: %w", err))
	}
	defer committer.Close()

	user.UseCustomAvatar = true
	user.Avatar = avatar_module.HashAvatar(user.ID, avatarData)
	if err = user_model.UpdateUserCols(ctx, user, "use_custom_avatar", "avatar"); err != nil {
		panic(fmt.Errorf("UpdateUserCols(%+v): %w", user, err))
	}

	if err := storage.SaveFrom(storage.Avatars, user.CustomAvatarRelativePath(), func(w io.Writer) error {
		_, err := w.Write(avatarData)
		return err
	}); err != nil {
		panic(fmt.Errorf("storage.SaveFrom(%s): %w", user.CustomAvatarRelativePath(), err))
	}

	if err := committer.Commit(); err != nil {
		panic(fmt.Errorf("committer.Commit: %w", err))
	}
}
