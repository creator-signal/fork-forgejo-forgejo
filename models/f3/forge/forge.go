// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/keying"

	"xorm.io/builder"
)

type Forge struct { //revive:disable-line:exported
	ID    int64  `xorm:"pk autoincr"`
	URL   string `xorm:"INDEX UNIQUE"`
	Type  string
	Token []byte `xorm:"BLOB"` // encrypted data
}

func NewForge() *Forge {
	return &Forge{}
}

func (o *Forge) SetID(id int64) {
	o.ID = id
}

func (o Forge) GetID() int64 {
	return o.ID
}

func (o *Forge) SetURL(url string) {
	o.URL = url
}

func (o Forge) GetURL() string {
	return o.URL
}

func (o *Forge) SetType(t string) {
	o.Type = t
}

func (o Forge) GetType() string {
	return o.Type
}

func (o *Forge) SetToken(token string) {
	o.Token = []byte(token)
}

func (o Forge) GetToken() string {
	return string(o.Token)
}

func (o *Forge) TableName() string {
	return "f3_forge"
}

func (o *Forge) decryptToken() ([]byte, error) {
	token, err := keying.F3Forge.Decrypt(o.Token, keying.ColumnAndID("token", o.ID))
	if err != nil {
		return nil, fmt.Errorf("decrypt token %d: %w", o.ID, err)
	}
	return token, nil
}

func (o *Forge) encryptToken() []byte {
	return keying.F3Forge.Encrypt(o.Token, keying.ColumnAndID("token", o.ID))
}

func init() {
	db.RegisterModel(new(Forge))
}

func Equal(a, b *Forge) bool {
	return a.URL == b.URL &&
		a.Type == b.Type &&
		string(a.Token) == string(b.Token)
}

func Upsert(ctx context.Context, forge *Forge) (*Forge, error) {
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		found, err := Get(ctx, FindOptions{URL: &forge.URL})
		if err != nil {
			return err
		}
		if found == nil {
			return Insert(ctx, forge)
		}
		forge.SetID(found.GetID())
		if !Equal(forge, found) {
			if err := Update(ctx, found, forge); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return forge, nil
}

func Update(ctx context.Context, old, new *Forge) error {
	if new.GetToken() == "" {
		// if the token is empty there is no need to update anything as it is the only thing
		// that can be updated
		new.Token = old.Token
		return nil
	}
	forgeCopy := *new
	forgeCopy.Token = forgeCopy.encryptToken()
	_, err := db.GetEngine(ctx).ID(forgeCopy.ID).Update(&forgeCopy)
	return err
}

func Insert(ctx context.Context, forge *Forge) error {
	if err := db.Insert(ctx, forge); err != nil {
		return fmt.Errorf("insert %s: %w", forge.URL, err)
	}
	forgeCopy := *forge
	forgeCopy.Token = forge.encryptToken()
	_, err := db.GetEngine(ctx).ID(forgeCopy.ID).Cols("token").Update(forgeCopy)
	return err
}

type FindOptions struct {
	db.ListOptions
	ID  *int64
	URL *string
}

func (opts FindOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.ID != nil {
		cond = cond.And(builder.Eq{"id": opts.ID})
	}
	if opts.URL != nil {
		cond = cond.And(builder.Eq{"url": opts.URL})
	}
	return cond
}

type ForgeList []*Forge //revive:disable-line:exported

// using AfterLoad() to decrypt the token would be simpler but
// it is assumed to return nothing / no error handling as of xorm@v1.3.9-forgejo.4
func Find(ctx context.Context, opts FindOptions) (ForgeList, error) {
	forges, err := db.Find[Forge](ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, forge := range forges {
		token, err := forge.decryptToken()
		if err != nil {
			return nil, err
		}
		forge.SetToken(string(token))
	}
	return forges, nil
}

func Get(ctx context.Context, opts FindOptions) (*Forge, error) {
	forges, err := db.Find[Forge](ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(forges) == 0 {
		return nil, nil //nolint:nilnil
	}
	if len(forges) != 1 {
		return nil, fmt.Errorf("expected to find one forge but found %d instead", len(forges))
	}
	forge := forges[0]
	token, err := forge.decryptToken()
	if err != nil {
		return nil, err
	}
	forge.SetToken(string(token))
	return forge, nil
}
