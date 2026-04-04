// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package mirror

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"forgejo.org/models/db"
	forge_model "forgejo.org/models/f3/forge"
	"forgejo.org/modules/keying"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

type Mirror struct {
	ID                int64              `xorm:"pk autoincr"`
	ForgeID           int64              `xorm:"INDEX UNIQUE(f3_mirror_index) REFERENCES(f3_forge, id)"`
	Forge             *forge_model.Forge `xorm:"-"`
	RemotePath        string             `xorm:"INDEX UNIQUE(f3_mirror_index)"`
	RemoteToken       []byte             `xorm:"BLOB"` // encrypted data
	LocalPath         string
	LocalToken        []byte `xorm:"BLOB"` // encrypted data
	LocalUserID       int64
	Since             timeutil.TimeStamp
	Interval          time.Duration
	SendNotifications bool
	UpdatedUnix       timeutil.TimeStamp `xorm:"INDEX"`
	NextUpdateUnix    timeutil.TimeStamp `xorm:"INDEX"`
}

func NewMirror() *Mirror {
	return &Mirror{}
}

func (o *Mirror) String() string {
	tokenRemoved := *o
	tokenRemoved.LocalToken = nil
	tokenRemoved.RemoteToken = nil
	return fmt.Sprintf("%+v", tokenRemoved)
}

func (o *Mirror) SetID(id int64) {
	o.ID = id
}

func (o Mirror) GetID() int64 {
	return o.ID
}

func (o *Mirror) SetLocalToken(token string) {
	o.LocalToken = []byte(token)
}

func (o Mirror) GetLocalToken() string {
	return string(o.LocalToken)
}

func (o *Mirror) SetLocalUserID(localUserID int64) {
	o.LocalUserID = localUserID
}

func (o Mirror) GetLocalUserID() int64 {
	return o.LocalUserID
}

func (o *Mirror) SetRemoteToken(token string) {
	o.RemoteToken = []byte(token)
}

func (o Mirror) GetRemoteToken() string {
	return string(o.RemoteToken)
}

func decryptToken(token *[]byte, column string, id int64) error {
	decryptedToken, err := keying.F3Mirror.Decrypt(*token, keying.ColumnAndID(column, id))
	if err != nil {
		return fmt.Errorf("decrypt token %s %d: %w", column, id, err)
	}
	*token = decryptedToken
	return nil
}

func (o *Mirror) decryptTokens() error {
	if err := decryptToken(&o.LocalToken, "local_token", o.ID); err != nil {
		return fmt.Errorf("mirror Find decryptToken(LocalToken, %v): %w", o.ID, err)
	}
	if err := decryptToken(&o.RemoteToken, "remote_token", o.ID); err != nil {
		return fmt.Errorf("mirror Find decryptToken(RemoteToken, %v): %w", o.ID, err)
	}
	return nil
}

func encryptToken(token *[]byte, column string, id int64) {
	encryptedToken := keying.F3Mirror.Encrypt(*token, keying.ColumnAndID(column, id))
	*token = encryptedToken
}

func (o *Mirror) encryptTokens() {
	encryptToken(&o.LocalToken, "local_token", o.ID)
	encryptToken(&o.RemoteToken, "remote_token", o.ID)
}

func (o *Mirror) SetForgeID(forgeID int64) {
	o.ForgeID = forgeID
}

func (o Mirror) GetForgeID() int64 {
	return o.ForgeID
}

func (o *Mirror) SetRemotePath(remotePath string) {
	o.RemotePath = remotePath
}

func (o *Mirror) GetRemotePath() string {
	return o.RemotePath
}

func (o *Mirror) SetLocalPath(localPath string) {
	o.LocalPath = localPath
}

func (o *Mirror) GetLocalPath() string {
	return o.LocalPath
}

func (o *Mirror) SetSince(since timeutil.TimeStamp) {
	o.Since = since
}

func (o *Mirror) GetSince() timeutil.TimeStamp {
	return o.Since
}

func (o *Mirror) SetInterval(interval time.Duration) {
	o.Interval = interval
}

func (o *Mirror) GetInterval() time.Duration {
	return o.Interval
}

func (o *Mirror) SetSendNotifications(sendNotifications bool) {
	o.SendNotifications = sendNotifications
}

func (o *Mirror) GetSendNotifications() bool {
	return o.SendNotifications
}

func (o *Mirror) SetForge(forge *forge_model.Forge) {
	o.Forge = forge
}

func (o *Mirror) GetForge() *forge_model.Forge {
	return o.Forge
}

func (o *Mirror) LoadForge(ctx context.Context) error {
	if o.Forge != nil {
		return nil
	}
	forge, err := forge_model.Get(ctx, forge_model.FindOptions{
		ID: &o.ForgeID,
	})
	if err != nil {
		return fmt.Errorf("mirror LoadForge Get(%v): %w", o.ID, err)
	}
	o.SetForge(forge)
	return nil
}

func (o *Mirror) TableName() string {
	return "f3_mirror"
}

func init() {
	db.RegisterModel(new(Mirror))
}

func Equal(a, b *Mirror) bool {
	return string(a.RemoteToken) == string(b.RemoteToken) &&
		string(a.LocalToken) == string(b.LocalToken) &&
		a.LocalUserID == b.LocalUserID &&
		a.ForgeID == b.ForgeID &&
		a.RemotePath == b.RemotePath &&
		a.LocalPath == b.LocalPath &&
		a.Since == b.Since &&
		a.Interval == b.Interval &&
		a.NextUpdateUnix == b.NextUpdateUnix
}

func Upsert(ctx context.Context, mirror *Mirror) (*Mirror, error) {
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		found, redundants, err := findRemotePathEquivalents(ctx, mirror.ForgeID, mirror.RemotePath)
		if err != nil {
			return fmt.Errorf("mirror Upsert findRemotePathEquivalents(%v, %v): %w", mirror.ForgeID, mirror.RemotePath, err)
		}
		if found == nil {
			if err := Insert(ctx, mirror); err != nil {
				return fmt.Errorf("mirror Upsert Insert(%v, %v, %v): %w", mirror.ForgeID, mirror.RemotePath, mirror.LocalPath, err)
			}
		} else {
			if found.RemotePath == mirror.RemotePath {
				mirror.SetID(found.GetID())
				if !Equal(mirror, found) {
					if err := Update(ctx, mirror); err != nil {
						return fmt.Errorf("mirror Upsert Update(%v): %w", mirror.ID, err)
					}
				}
			} else {
				mirror = found
			}
		}
		return removeRedundants(ctx, redundants)
	}); err != nil {
		return nil, fmt.Errorf("mirror Upsert WithTx: %w", err)
	}
	log.Debug("mirror = %s", mirror)
	return mirror, nil
}

func Insert(ctx context.Context, mirror *Mirror) error {
	if err := db.Insert(ctx, mirror); err != nil {
		return fmt.Errorf("mirror Insert %v %v: %w", mirror.ForgeID, mirror.RemotePath, err)
	}
	mirrorCopy := *mirror
	mirrorCopy.encryptTokens()
	if _, err := db.GetEngine(ctx).ID(mirrorCopy.ID).Cols("local_token", "remote_token").Update(mirrorCopy); err != nil {
		return fmt.Errorf("mirror Insert Update(%v): %w", mirrorCopy.ID, err)
	}
	return nil
}

type FindOptions struct {
	db.ListOptions
	ID      *int64
	ForgeID *int64
}

func (opts FindOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.ID != nil {
		cond = cond.And(builder.Eq{"`id`": opts.ID})
	}
	if opts.ForgeID != nil {
		cond = cond.And(builder.Eq{"`forge_id`": opts.ForgeID})
	}
	return cond
}

type MirrorList []*Mirror //revive:disable-line:exported

func Find(ctx context.Context, opts FindOptions) (MirrorList, error) {
	mirrors, err := db.Find[Mirror](ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mirror Find Find(%v): %w", opts, err)
	}
	for _, mirror := range mirrors {
		if err := mirror.decryptTokens(); err != nil {
			return nil, fmt.Errorf("mirror Find decryptTokens(%v): %w", mirror.ID, err)
		}
	}
	return mirrors, nil
}

type equivalent struct {
	ID               int64
	RemotePath       string
	RemotePathLength int
}

func findRemotePathEquivalents(ctx context.Context, forgeID int64, remotePath string) (*Mirror, []int64, error) {
	var notFound *Mirror
	if !strings.HasPrefix(remotePath, "/") {
		return notFound, nil, nil
	}
	path := remotePath
	pathCond := builder.NewCond()
	pathCond = pathCond.Or(builder.Like{"`remote_path`", path + "/%"})
	for path != "/" {
		pathCond = pathCond.Or(builder.Eq{"`remote_path`": path})
		path = filepath.Dir(path)
	}
	pathCond = pathCond.Or(builder.Eq{"`remote_path`": ""})
	cond := builder.NewCond()
	cond.And(builder.Eq{"`forge_id`": forgeID})
	cond.And(pathCond)

	equivalents := make([]*equivalent, 0, 10)

	if err := db.GetEngine(ctx).Table("f3_mirror").Select("`id`, `remote_path`, LENGTH(`remote_path`) as `remote_path_length`").Where(pathCond).OrderBy("`remote_path_length`").Find(&equivalents); err != nil {
		return nil, nil, fmt.Errorf("mirror findRemotePathEquivalents Find(%v): %w", cond, err)
	}
	if len(equivalents) == 0 {
		return notFound, nil, nil
	}
	var found *Mirror
	var shortestRemotePath string
	if equivalents[0].RemotePathLength > len(remotePath) {
		shortestRemotePath = remotePath
	} else {
		mirror, err := Get(ctx, FindOptions{ID: &equivalents[0].ID})
		if err != nil && !errors.Is(err, util.ErrNotExist) {
			return nil, nil, fmt.Errorf("mirror findRemotePathEquivalents Get(%v): %w", &equivalents[0].ID, err)
		}
		found = mirror
		shortestRemotePath = found.RemotePath
	}
	var redundants []int64
	for _, equivalent := range equivalents {
		if len(equivalent.RemotePath) > len(shortestRemotePath) {
			redundants = append(redundants, equivalent.ID)
		}
	}
	return found, redundants, nil
}

func removeRedundants(ctx context.Context, redundants []int64) error {
	if _, err := db.GetEngine(ctx).Table("f3_resource").Where(builder.In("`mirror_id`", redundants)).Delete(); err != nil {
		return fmt.Errorf("mirror removeRedundants Delete f3_resource (%v): %w", redundants, err)
	}
	if _, err := db.GetEngine(ctx).Table("f3_mirror").Where(builder.In("`id`", redundants)).Delete(); err != nil {
		return fmt.Errorf("mirror removeRedundants Delete f3_resource (%v): %w", redundants, err)
	}
	return nil
}

func Get(ctx context.Context, opts FindOptions) (*Mirror, error) {
	mirrors, err := db.Find[Mirror](ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mirror Get Find(%v): %w", opts, err)
	}
	if len(mirrors) == 0 {
		return nil, util.ErrNotExist
	}
	if len(mirrors) != 1 {
		return nil, fmt.Errorf("mirror Get(%v) expected to find one mirror but found %d instead", opts, len(mirrors))
	}
	mirror := mirrors[0]
	if err := mirror.decryptTokens(); err != nil {
		return nil, fmt.Errorf("mirror Get decryptTokens(%v): %w", mirror.ID, err)
	}
	return mirror, nil
}

// BeforeInsert will be invoked by XORM
func (o *Mirror) BeforeInsert() {
	if o != nil {
		now := timeutil.TimeStampNow()
		o.UpdatedUnix = now
		o.NextUpdateUnix = now
	}
}

func (o *Mirror) ScheduleNextUpdate() {
	if o.Interval != 0 {
		o.NextUpdateUnix = timeutil.TimeStampNow().AddDuration(o.Interval)
	} else {
		o.NextUpdateUnix = 0
	}
}

func Update(ctx context.Context, mirror *Mirror) error {
	mirror.UpdatedUnix = timeutil.TimeStampNow()
	mirrorCopy := *mirror
	mirrorCopy.encryptTokens()
	if _, err := db.GetEngine(ctx).ID(mirrorCopy.ID).AllCols().Update(mirrorCopy); err != nil {
		return fmt.Errorf("mirror Update Update(%v): %w", mirrorCopy.ID, err)
	}
	return nil
}

func Iterate(ctx context.Context, f func(idx int, bean any) error) error {
	sess := db.GetEngine(ctx).
		Where("next_update_unix<=?", time.Now().Unix()).
		And("next_update_unix!=0").
		OrderBy("updated_unix ASC")
	return sess.Iterate(new(Mirror), f)
}
