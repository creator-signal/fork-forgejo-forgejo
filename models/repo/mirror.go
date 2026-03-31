// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"
	"errors"
	"net/url"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/keying"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// ErrMirrorNotExist mirror does not exist error
var ErrMirrorNotExist = util.NewNotExistErrorf("Mirror does not exist")

// Mirror represents mirror information of a repository.
type Mirror struct {
	ID          int64       `xorm:"pk autoincr"`
	RepoID      int64       `xorm:"INDEX"`
	Repo        *Repository `xorm:"-"`
	Interval    time.Duration
	EnablePrune bool `xorm:"NOT NULL DEFAULT true"`

	UpdatedUnix    timeutil.TimeStamp `xorm:"INDEX"`
	NextUpdateUnix timeutil.TimeStamp `xorm:"INDEX"`

	LFS         bool   `xorm:"lfs_enabled NOT NULL DEFAULT false"`
	LFSEndpoint string `xorm:"lfs_endpoint TEXT"`

	RemoteAddress     string `xorm:"VARCHAR(2048)"` // sanitized and stripped of credentials
	RemoteAddressAuth []byte `xorm:"BLOB NULL"`     // encrypted remote address w/ credentials; often NULL for older mirrors
}

func init() {
	db.RegisterModel(new(Mirror))
}

// BeforeInsert will be invoked by XORM before inserting a record
func (m *Mirror) BeforeInsert() {
	if m != nil {
		m.UpdatedUnix = timeutil.TimeStampNow()
		m.NextUpdateUnix = timeutil.TimeStampNow()
	}
}

// GetRepository returns the repository.
func (m *Mirror) GetRepository(ctx context.Context) *Repository {
	if m.Repo != nil {
		return m.Repo
	}
	var err error
	m.Repo, err = GetRepositoryByID(ctx, m.RepoID)
	if err != nil {
		log.Error("getRepositoryByID[%d]: %v", m.ID, err)
	}
	return m.Repo
}

// GetRemoteName returns the name of the remote.
func (m *Mirror) GetRemoteName() string {
	return "origin"
}

// ScheduleNextUpdate calculates and sets next update time.
func (m *Mirror) ScheduleNextUpdate() {
	if m.Interval != 0 {
		m.NextUpdateUnix = timeutil.TimeStampNow().AddDuration(m.Interval)
	} else {
		m.NextUpdateUnix = 0
	}
}

// InsertMirror inserts a mirror to database. RemoteAddress must be provided so that it can be encrypted and stored
// during the insert process.
func (m *Mirror) InsertWithAddress(ctx context.Context, addr string) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		if _, err := db.GetEngine(ctx).Insert(m); err != nil {
			return err
		}
		return m.UpdateRemoteAddress(ctx, addr)
	})
}

// Stores a credential-free version of the address in `RemoteAddress`, encrypts the original into `RemoteAddressAuth`,
// and stores both in the database. The ID of the mirror must be known, so this must be done after the mirror is
// inserted.
func (m *Mirror) UpdateRemoteAddress(ctx context.Context, addr string) error {
	if m.ID == 0 {
		return errors.New("must persist mirror to database before using UpdateRemoteAddress")
	}

	parsedURL, err := url.Parse(addr)
	if err != nil {
		return err
	}
	// For `RemoteAddress`, remove the password if present.  Retain the username for consistency with
	// `AddAuthCredentialHelperForRemote` which retains the username for the `git clone` command line, which ends up as
	// the remote URL in the mirror's git config.
	if parsedURL.User != nil {
		parsedURL.User = url.User(parsedURL.User.Username())
	}
	sanitized := parsedURL.String()

	m.RemoteAddress = sanitized
	m.RemoteAddressAuth = keying.PullMirror.Encrypt(
		[]byte(addr),
		keying.ColumnAndID("remote_address_auth", m.ID),
	)

	_, err = db.GetEngine(ctx).ID(m.ID).Cols("remote_address", "remote_address_auth").Update(m)
	return err
}

// Retrieves the encrypted RemoteAddressAuth and decrypts it. Note that this field is expected to be absent for mirrors
// created before the introduction of RemoteAddressAuth, in which case credentials are not known to Forgejo (but may be
// on-disk in the repository's config file) and None will be returned.
func (m *Mirror) DecryptRemoteAddressAuth() (optional.Option[string], error) {
	if m.RemoteAddressAuth == nil {
		return optional.None[string](), nil
	}

	contents, err := keying.PullMirror.Decrypt(m.RemoteAddressAuth, keying.ColumnAndID("remote_address_auth", m.ID))
	if err != nil {
		return optional.None[string](), err
	}
	return optional.Some(string(contents)), nil
}

// GetMirrorByRepoID returns mirror information of a repository.
func GetMirrorByRepoID(ctx context.Context, repoID int64) (*Mirror, error) {
	m := &Mirror{RepoID: repoID}
	has, err := db.GetEngine(ctx).Get(m)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrMirrorNotExist
	}
	return m, nil
}

// UpdateMirror updates the mirror
func UpdateMirror(ctx context.Context, m *Mirror) error {
	_, err := db.GetEngine(ctx).ID(m.ID).AllCols().Update(m)
	return err
}

// TouchMirror updates the mirror updatedUnix
func TouchMirror(ctx context.Context, m *Mirror) error {
	m.UpdatedUnix = timeutil.TimeStampNow()
	_, err := db.GetEngine(ctx).ID(m.ID).Cols("updated_unix").Update(m)
	return err
}

// DeleteMirrorByRepoID deletes a mirror by repoID
func DeleteMirrorByRepoID(ctx context.Context, repoID int64) error {
	_, err := db.GetEngine(ctx).Delete(&Mirror{RepoID: repoID})
	return err
}

// MirrorsIterate iterates all mirror repositories.
func MirrorsIterate(ctx context.Context, limit int, f func(idx int, bean any) error) error {
	sess := db.GetEngine(ctx).
		Where("next_update_unix<=?", time.Now().Unix()).
		And("next_update_unix!=0").
		OrderBy("updated_unix ASC")
	if limit > 0 {
		sess = sess.Limit(limit)
	}
	return sess.Iterate(new(Mirror), f)
}
