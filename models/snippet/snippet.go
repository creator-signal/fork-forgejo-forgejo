// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"

	"github.com/google/uuid"
)

type SnippetVisibility int8 //revive:disable-line:exported

const (
	SnippetVisibilityPublic  SnippetVisibility = iota + 1 // 1， GistVisibilityPublic the snippet can be seen be anyone
	SnippetVisibilityHidden                               // 2， SnippetVisibilityHidden  the snippet can be seen by anyone but don't appear in the search
	SnippetVisibilityPrivate                              // 3， SnippetVisibilityPrivate the snippet can only been seen by the owner
)

func (visibility SnippetVisibility) String() string {
	switch visibility {
	case SnippetVisibilityPublic:
		return "public"
	case SnippetVisibilityHidden:
		return "hidden"
	case SnippetVisibilityPrivate:
		return "private"
	default:
		return "unknown"
	}
}

func SnippetVisibilityFromName(name string) (SnippetVisibility, error) { //revive:disable-line:exported
	switch strings.ToLower(name) {
	case "public":
		return SnippetVisibilityPublic, nil
	case "hidden":
		return SnippetVisibilityHidden, nil
	case "private":
		return SnippetVisibilityPrivate, nil
	default:
		return 0, fmt.Errorf("%s is not a valid snippet visibility name", name)
	}
}

// ErrSnippetNotExist represents a "SnippetNotExist" kind of error.
type ErrSnippetNotExist struct {
	UUID string
}

// IsErrSnippetNotExist checks if an error is a ErrSnippetNotExist.
func IsErrSnippetNotExist(err error) bool {
	_, ok := err.(ErrSnippetNotExist)
	return ok
}

func (err ErrSnippetNotExist) Error() string {
	return fmt.Sprintf("snippet does not exists [uuid: %s]", err.UUID)
}

type Snippet struct {
	ID          int64            `xorm:"pk autoincr"`
	OwnerID     int64            `xorm:"INDEX REFERENCES(user, id)"`
	Owner       *user_model.User `xorm:"-"`
	UUID        string           `xorm:"UNIQUE"`
	Name        string
	Description string `xorm:"TEXT"`
	Visibility  SnippetVisibility
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

func init() {
	db.RegisterModel(new(Snippet))
}

// generateUUID generates a random UUID for a Snippet
func generateUUID() string {
	uuidParts := strings.Split(uuid.New().String(), "-")
	return strings.ToLower(uuidParts[0])
}

// Create creates a new Snippet
func Create(ctx context.Context, snippet *Snippet) error {
	snippet.UUID = generateUUID()
	_, err := db.GetEngine(ctx).Insert(snippet)
	return err
}

// GetSnippetByUUID finds the Snippet with the given UUID
func GetSnippetByUUID(ctx context.Context, uuid string) (*Snippet, error) {
	snippet := new(Snippet)
	has, err := db.GetEngine(ctx).Where("uuid = ?", strings.ToLower(uuid)).Get(snippet)
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrSnippetNotExist{UUID: uuid}
	}

	return snippet, nil
}

// CountOwnerSnippets returns how many Snippets a User Owns
// Note: This function does not check if the caller has permission to view the Snippets
func CountOwnerSnippets(ctx context.Context, ownerID int64) (int64, error) {
	return db.GetEngine(ctx).Where("owner_id = ?", ownerID).Count(new(Snippet))
}

// GetRepoPath returns the Path to the Snippet Repo
func (snippet *Snippet) GetRepoPath() string {
	return filepath.Join(setting.Snippet.RootPath, fmt.Sprintf("%s.git", snippet.UUID))
}

// Link returns the Link to the Repo
func (snippet *Snippet) Link() string {
	return fmt.Sprintf("/snippets/%s", url.PathEscape(snippet.UUID))
}

// HTMLURL returns the snippet HTML URL
func (snippet *Snippet) HTMLURL() string {
	return fmt.Sprintf("%ssnippets/%s", setting.AppURL, url.PathEscape(snippet.UUID))
}

// LoadOwner loads the owner field
func (snippet *Snippet) LoadOwner(ctx context.Context) error {
	owner, err := user_model.GetUserByID(ctx, snippet.OwnerID)
	if err != nil {
		return err
	}

	snippet.Owner = owner

	return nil
}

// Update cols updates the given columns
func (snippet *Snippet) UpdateCols(ctx context.Context, cols ...string) error {
	_, err := db.GetEngine(ctx).ID(snippet.ID).Cols(cols...).Update(snippet)
	return err
}

// IsOwner checks if the given User is the Owner of the Repo
func (snippet *Snippet) IsOwner(user *user_model.User) bool {
	if user == nil {
		return false
	}

	return snippet.OwnerID == user.ID
}

// HasAccess checks if the given User has access to the Snippet
func (snippet *Snippet) HasAccess(user *user_model.User) bool {
	if snippet.Visibility != SnippetVisibilityPrivate {
		return true
	}

	return snippet.IsOwner(user)
}
