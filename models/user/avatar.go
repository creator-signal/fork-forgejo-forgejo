// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"crypto/md5"
	"fmt"
	"image/png"
	"io"
	"strings"

	"forgejo.org/models/avatars"
	"forgejo.org/models/db"
	"forgejo.org/modules/avatar"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"golang.org/x/crypto/blake2b"
)

// CustomAvatarRelativePath returns user custom avatar relative path.
func (u *User) CustomAvatarRelativePath() string {
	return u.Avatar
}

type AvatarVector struct {
	ID     int64 `xorm:"pk autoincr"`
	UserID int64 `xorm:"NOT NULL REFERENCES(user, id)"`
	// Hash of SVG avatar
	SvgHash []byte `xorm:"VARBINARY(16)"`
	// Raw SVG avatar as text
	Svg string `xorm:"TEXT"`
}

// HashSvgAvatar returns a half of 256 bit blake2b-hash of avatar code
// Requiremenents:
// - not too many characters when encoded as string
// - not expensive to compute
func HashSvgAvatar(avatarXML string) []byte {
	hasher, _ := blake2b.New256(nil)
	hasher.Write([]byte(avatarXML))
	sum := hasher.Sum(nil)
	return sum[:16]
}

func GetSvgAvatarHash(ctx context.Context, userID int64) ([]byte, error) {
	var foundAvatar AvatarVector
	found, err := db.GetEngine(ctx).Table(&AvatarVector{}).Where("user_id=?", userID).Get(&foundAvatar)
	if !found {
		return nil, fmt.Errorf("GetSvgAvatarHash: vector avatar not found for user %d", userID)
	}
	return foundAvatar.SvgHash, err
}

// GenerateRandomAvatar generates a random avatar for user.
func GenerateRandomAvatar(ctx context.Context, u *User) error {
	seed := u.Email
	if len(seed) == 0 {
		seed = u.Name
	}

	identicon, err := avatar.RandomImage([]byte(seed))
	if err != nil {
		return fmt.Errorf("Failed to generate identicon: %w", err)
	}
	vectorHash := HashSvgAvatar(identicon.Vector)

	if _, err = storage.Avatars.Stat(u.CustomAvatarRelativePath()); err != nil {
		// If unable to Stat the avatar file (usually it means non-existing), then try to save a new one
		// Don't share the images so that we can delete them easily
		if err := storage.SaveFrom(storage.Avatars, u.CustomAvatarRelativePath(), func(w io.Writer) error {
			if err := png.Encode(w, &identicon.Raster); err != nil {
				log.Error("Encode: %v", err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("Failed to save avatar %s: %w", u.CustomAvatarRelativePath(), err)
		}
	}

	// Save info about the new avatar into the database
	err = db.WithTx(ctx, func(ctx context.Context) error {
		if err = db.Insert(ctx, &AvatarVector{
			UserID:  u.ID,
			SvgHash: vectorHash,
			Svg:     identicon.Vector,
		}); err != nil {
			return err
		}

		u.Avatar = avatars.HashEmail(seed)

		if _, err := db.GetEngine(ctx).ID(u.ID).Cols("avatar").Update(u); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Info("New random avatar created: %d", u.ID)

	return nil
}

// todo separate commit
func (u *User) ChooseAvatarToUse(ctx context.Context) (bool, bool) {
	useLocalAvatar := false
	autoGenerateAvatar := false

	disableGravatar := setting.Config().Picture.DisableGravatar.Value(ctx)

	switch {
	case u.UseCustomAvatar:
		useLocalAvatar = true
	case disableGravatar, setting.OfflineMode:
		useLocalAvatar = true
		autoGenerateAvatar = true
	}
	return useLocalAvatar, autoGenerateAvatar
}

// AvatarLinkWithSize returns a link to the user's avatar. Size is only used for
// GenerateEmailAvatarFastLink, for external email-based avatar services
func (u *User) AvatarLinkWithSize(ctx context.Context, size int) string {
	if u.IsGhost() || u.ID <= 0 {
		return avatars.DefaultAvatarLink()
	}

	useLocalAvatar, autoGenerateAvatar := u.ChooseAvatarToUse(ctx)

	if useLocalAvatar {
		if u.Avatar == "" && autoGenerateAvatar {
			if err := GenerateRandomAvatar(ctx, u); err != nil {
				log.Error("GenerateRandomAvatar: %v", err)
			}
		}
		if u.Avatar == "" {
			return avatars.DefaultAvatarLink()
		}
		return avatars.GenerateUserAvatarImageLink(u.Avatar)
	}
	return avatars.GenerateEmailAvatarFastLink(ctx, u.AvatarEmail, size)
}

// AvatarLink returns the full avatar link with http host
func (u *User) AvatarLink(ctx context.Context) string {
	link := u.AvatarLinkWithSize(ctx, 0)
	if !strings.HasPrefix(link, "//") && !strings.Contains(link, "://") {
		return setting.AppURL + strings.TrimPrefix(link, setting.AppSubURL+"/")
	}
	return link
}

// IsUploadAvatarChanged returns true if the current user's avatar would be changed with the provided data
func (u *User) IsUploadAvatarChanged(data []byte) bool {
	if !u.UseCustomAvatar || len(u.Avatar) == 0 {
		return true
	}
	avatarID := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d-%x", u.ID, md5.Sum(data)))))
	return u.Avatar != avatarID
}

// ExistsWithAvatarAtStoragePath returns true if there is a user with this Avatar
func ExistsWithAvatarAtStoragePath(ctx context.Context, storagePath string) (bool, error) {
	// See func (u *User) CustomAvatarRelativePath()
	// u.Avatar is used directly as the storage path - therefore we can check for existence directly using the path
	return db.GetEngine(ctx).Where("`avatar`=?", storagePath).Exist(new(User))
}
