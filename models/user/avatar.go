// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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

// HashSvgAvatar returns a half of 256 bit blake2b-hash of avatar code
// Requiremenents:
// - not too many characters when encoded as string
// - not expensive to compute
func HashSvgAvatar(avatarXml string) []byte {
	hasher, _ := blake2b.New256(nil)
	hasher.Write([]byte(avatarXml))
	sum := hasher.Sum(nil)
	return sum[:16]
}

// GenerateRandomAvatar generates a random avatar for user.
func GenerateRandomAvatar(ctx context.Context, u *User) error {
	seed := u.Email
	if len(seed) == 0 {
		seed = u.Name
	}

	identicon, err := avatar.RandomImage([]byte(seed))
	if err != nil {
		return fmt.Errorf("RandomImage: %w", err)
	}

	u.Avatar = avatars.HashEmail(seed)
	u.AvatarSVG = identicon.Vector
	u.AvatarSVGHash = HashSvgAvatar(identicon.Vector)

	debug_hex := hex.EncodeToString(u.AvatarSVGHash)
	println("New hash: " + debug_hex)

	_, err = storage.Avatars.Stat(u.CustomAvatarRelativePath())
	if err != nil {
		// If unable to Stat the avatar file (usually it means non-existing), then try to save a new one
		// Don't share the images so that we can delete them easily
		if err := storage.SaveFrom(storage.Avatars, u.CustomAvatarRelativePath(), func(w io.Writer) error {
			if err := png.Encode(w, &identicon.Raster); err != nil {
				log.Error("Encode: %v", err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("failed to save avatar %s: %w", u.CustomAvatarRelativePath(), err)
		}
	}

	if _, err := db.GetEngine(ctx).ID(u.ID).Cols("avatar", "avatar_svg", "avatar_svg_hash").Update(u); err != nil {
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

type AvatarDisplayProperties struct {
	RasterLink string
	RasterSize int
	SvgContent string
}

// todo dsc
func (u *User) SolveAvatar(ctx context.Context, size int) AvatarDisplayProperties {
	properties := AvatarDisplayProperties{
		RasterLink: "",
		RasterSize: 0,
		SvgContent: "",
	}

	useLocalAvatar, autoGenerateAvatar := u.ChooseAvatarToUse(ctx)

	if !useLocalAvatar {
		properties.RasterLink = avatars.GenerateEmailAvatarFastLink(ctx, u.AvatarEmail, size)
		return properties
	}

	// Attempt to generate avatar if needed
	if u.Avatar == "" && autoGenerateAvatar {
		if err := GenerateRandomAvatar(ctx, u); err != nil {
			log.Error("GenerateRandomAvatar: %v", err)
		}
	}

	// If even after that there's still no avatar available
	if u.Avatar == "" {
		properties.RasterLink = avatars.DefaultAvatarLink()
		return properties
	}

	// Some raster avatar is available. If there's also a vector version available
	// (not "") alongside of it, this is an identicon and we'll use both versions
	properties.SvgContent = u.AvatarSVG
	properties.RasterLink = avatars.GenerateUserAvatarImageLink(u.Avatar)
	return properties
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
