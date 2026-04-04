// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package mirror

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/keying"
	"forgejo.org/modules/timeutil"
	permissions_errors "forgejo.org/services/permissions/errors"

	"xorm.io/builder"
)

type Mirror struct {
	ID                int64  `xorm:"pk autoincr"`
	Token             []byte `xorm:"BLOB"` // encrypted data
	ForgeID           int64  `xorm:"INDEX UNIQUE(f3_mirror_index) REFERENCES(f3_forge, id)"`
	FromPath          string `xorm:"INDEX UNIQUE(f3_mirror_index)"`
	ToPath            string
	Since             timeutil.TimeStamp
	Interval          time.Duration
	SendNotifications bool
	UpdatedUnix       timeutil.TimeStamp `xorm:"INDEX"`
	NextUpdateUnix    timeutil.TimeStamp `xorm:"INDEX"`
	Err               string
	ErrMessage        string
}

func NewMirror() *Mirror {
	return &Mirror{}
}

func (o *Mirror) SetID(id int64) {
	o.ID = id
}

func (o Mirror) GetID() int64 {
	return o.ID
}

func (o *Mirror) SetTokenFromAuthorizationHeader(authorizationHeader string) error {
	auths := strings.SplitN(authorizationHeader, " ", 2)
	if len(auths) < 1 || !slices.Contains([]string{"bearer", "token"}, strings.ToLower(auths[0])) {
		return fmt.Errorf("Authorization header empty or not starting with 'bearer' or 'token'")
	}
	o.SetToken(auths[1])
	return nil
}

func (o *Mirror) SetToken(token string) {
	o.Token = []byte(token)
}

func (o Mirror) GetToken() string {
	return string(o.Token)
}

func (o *Mirror) decryptToken() ([]byte, error) {
	token, err := keying.F3Mirror.Decrypt(o.Token, keying.ColumnAndID("token", o.ID))
	if err != nil {
		return nil, fmt.Errorf("decrypt token %d: %w", o.ID, err)
	}
	return token, nil
}

func (o *Mirror) encryptToken() []byte {
	return keying.F3Mirror.Encrypt(o.Token, keying.ColumnAndID("token", o.ID))
}

func (o *Mirror) SetForgeID(forgeID int64) {
	o.ForgeID = forgeID
}

func (o Mirror) GetForgeID() int64 {
	return o.ForgeID
}

func (o *Mirror) SetFromPath(fromPath string) {
	o.FromPath = fromPath
}

func (o *Mirror) GetFromPath() string {
	return o.FromPath
}

func (o *Mirror) SetToPath(toPath string) {
	o.ToPath = toPath
}

func (o *Mirror) GetToPath() string {
	return o.ToPath
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

func (o *Mirror) TableName() string {
	return "f3_mirror"
}

func init() {
	db.RegisterModel(new(Mirror))
}

func Equal(a, b *Mirror) bool {
	return string(a.Token) == string(b.Token) &&
		a.ForgeID == b.ForgeID &&
		a.FromPath == b.FromPath &&
		a.ToPath == b.ToPath &&
		a.Since == b.Since &&
		a.Interval == b.Interval &&
		a.NextUpdateUnix == b.NextUpdateUnix
}

func Upsert(ctx context.Context, mirror *Mirror) (*Mirror, error) {
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		found, redundants, err := findFromPathEquivalents(ctx, mirror.ForgeID, mirror.FromPath)
		if err != nil {
			return err
		}
		if found == nil {
			if err := Insert(ctx, mirror); err != nil {
				return err
			}
		} else {
			if found.FromPath == mirror.FromPath {
				mirror.SetID(found.GetID())
				if !Equal(mirror, found) {
					if err := Update(ctx, mirror); err != nil {
						return err
					}
				}
			} else {
				mirror = nil
			}
		}
		return removeRedundants(ctx, redundants)
	}); err != nil {
		return nil, err
	}
	return mirror, nil
}

func Insert(ctx context.Context, mirror *Mirror) error {
	if err := db.Insert(ctx, mirror); err != nil {
		return fmt.Errorf("insert %v %v: %w", mirror.ForgeID, mirror.FromPath, err)
	}
	mirrorCopy := *mirror
	mirrorCopy.Token = mirrorCopy.encryptToken()
	_, err := db.GetEngine(ctx).ID(mirrorCopy.ID).Cols("token").Update(mirrorCopy)
	return err
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
		return nil, err
	}
	for _, mirror := range mirrors {
		token, err := mirror.decryptToken()
		if err != nil {
			return nil, err
		}
		mirror.SetToken(string(token))
	}
	return mirrors, nil
}

type equivalent struct {
	ID             int64
	FromPath       string
	FromPathLength int
}

func findFromPathEquivalents(ctx context.Context, forgeID int64, fromPath string) (*Mirror, []int64, error) {
	if !strings.HasPrefix(fromPath, "/") {
		return nil, nil, nil //nolint:nilnil
	}
	path := fromPath
	pathCond := builder.NewCond()
	pathCond = pathCond.Or(builder.Like{"`from_path`", path + "/%"})
	for path != "/" {
		pathCond = pathCond.Or(builder.Eq{"`from_path`": path})
		path = filepath.Dir(path)
	}
	pathCond = pathCond.Or(builder.Eq{"`from_path`": ""})
	cond := builder.NewCond()
	cond.And(builder.Eq{"`forge_id`": forgeID})
	cond.And(pathCond)

	equivalents := make([]*equivalent, 0, 10)

	if err := db.GetEngine(ctx).Table("f3_mirror").Select("`id`, `from_path`, LENGTH(`from_path`) as `from_path_length`").Where(pathCond).OrderBy("`from_path_length`").Find(&equivalents); err != nil {
		return nil, nil, err
	}
	if len(equivalents) == 0 {
		return nil, nil, nil //nolint:nilnil
	}
	var found *Mirror
	var shortestFromPath string
	if equivalents[0].FromPathLength > len(fromPath) {
		shortestFromPath = fromPath
	} else {
		mirror, err := Get(ctx, FindOptions{ID: &equivalents[0].ID})
		if err != nil {
			return nil, nil, err
		}
		found = mirror
		shortestFromPath = found.FromPath
	}
	var redundants []int64
	for _, equivalent := range equivalents {
		if len(equivalent.FromPath) > len(shortestFromPath) {
			redundants = append(redundants, equivalent.ID)
		}
	}
	return found, redundants, nil
}

func removeRedundants(ctx context.Context, redundants []int64) error {
	_, err := db.GetEngine(ctx).Table("f3_mirror").Where(builder.In("`id`", redundants)).Delete()
	return err
}

func Get(ctx context.Context, opts FindOptions) (*Mirror, error) {
	mirrors, err := db.Find[Mirror](ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(mirrors) == 0 {
		return nil, nil //nolint:nilnil
	}
	if len(mirrors) != 1 {
		return nil, fmt.Errorf("expected to find one mirror but found %d instead", len(mirrors))
	}
	mirror := mirrors[0]
	token, err := mirror.decryptToken()
	if err != nil {
		return nil, err
	}
	mirror.SetToken(string(token))
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

const (
	OtherError = "other"
	NoError    = ""
)

func (o *Mirror) SetError(err error) {
	if err == nil {
		o.Err = NoError
		o.ErrMessage = ""
		return
	}
	if errors.Is(err, permissions_errors.NotFound) {
		o.Err = string(permissions_errors.NotFound)
	} else if errors.Is(err, permissions_errors.Server) {
		o.Err = string(permissions_errors.Server)
	} else if errors.Is(err, permissions_errors.Forbidden) {
		o.Err = string(permissions_errors.Forbidden)
	} else {
		o.Err = OtherError
	}
	o.ErrMessage = strings.TrimPrefix(err.Error(), o.Err+": ")
}

func (o *Mirror) GetError() error {
	if o.Err == "" {
		return nil
	}
	switch o.Err {
	case NoError:
		return nil
	case OtherError:
		return errors.New(o.ErrMessage)
	case string(permissions_errors.NotFound):
		return permissions_errors.NewNotFound(o.ErrMessage)
	case string(permissions_errors.Server):
		return permissions_errors.NewServer(o.ErrMessage)
	case string(permissions_errors.Forbidden):
		return permissions_errors.NewForbidden(o.ErrMessage)
	default:
		return fmt.Errorf("unexpected error type %s: %s", o.Err, o.ErrMessage)
	}
}

func Update(ctx context.Context, mirror *Mirror) error {
	mirror.UpdatedUnix = timeutil.TimeStampNow()
	mirrorCopy := *mirror
	mirrorCopy.Token = mirrorCopy.encryptToken()
	if _, err := db.GetEngine(ctx).ID(mirrorCopy.ID).AllCols().Update(mirrorCopy); err != nil {
		return fmt.Errorf("Update(%v, %v): %w", mirror.ForgeID, mirror.FromPath, err)
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
