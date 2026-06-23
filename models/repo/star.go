// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	"xorm.io/builder"
)

// Star represents a starred repo by an user.
type Star struct {
	ID          int64              `xorm:"pk autoincr"`
	UID         int64              `xorm:"UNIQUE(s)"`
	RepoID      int64              `xorm:"UNIQUE(s)"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(Star))
}

// StarRepo or unstar repository.
func StarRepo(ctx context.Context, userID, repoID int64, star bool) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()
	staring := IsStaring(ctx, userID, repoID)

	if star {
		if staring {
			return nil
		}

		if err := db.Insert(ctx, &Star{UID: userID, RepoID: repoID}); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = num_stars + 1 WHERE id = ?", repoID); err != nil {
			return err
		}
	} else {
		if !staring {
			return nil
		}
		if _, err := db.DeleteByBean(ctx, &Star{UID: userID, RepoID: repoID}); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = num_stars - 1 WHERE id = ?", repoID); err != nil {
			return err
		}
	}

	return committer.Commit()
}

// IsStaring checks if user has starred given repository.
func IsStaring(ctx context.Context, userID, repoID int64) bool {
	has, _ := db.GetEngine(ctx).Get(&Star{UID: userID, RepoID: repoID})
	return has
}

// GetStargazers returns the users that starred the repo.
func GetStargazers(ctx context.Context, repo *Repository, opts db.ListOptions) ([]*user_model.User, error) {
	sess := db.GetEngine(ctx).Where("star.repo_id = ?", repo.ID).
		Join("LEFT", "star", "`user`.id = star.uid")
	if opts.Page > 0 {
		sess = db.SetSessionPagination(sess, &opts)

		users := make([]*user_model.User, 0, opts.PageSize)
		return users, sess.Find(&users)
	}

	users := make([]*user_model.User, 0, 8)
	return users, sess.Find(&users)
}
func GetVisibleStarCount(ctx context.Context, profileUser, doer *user_model.User, orgName string) (int, error) {
	e := db.GetEngine(ctx)

	// viewer is the profile owner — sees ALL their starred repos
	if doer != nil && doer.ID == profileUser.ID {
		count, err := e.
			Where("uid = ?", profileUser.ID).
			Count(new(Star))
		return int(count), err
	}

	// anonymous visitor — only public repos + public owners
	if doer == nil {
		count, err := e.
			Table("star").
			Join("INNER", "repository", "star.repo_id = repository.id").
			Join("LEFT", "`user`", "repository.owner_id = `user`.id").
			Where("star.uid = ?", profileUser.ID).
			And("repository.is_private = ?", false).
			And("`user`.visibility = ?", structs.VisibleTypePublic).
			Count(new(Star))
		return int(count), err
	}

	// doer is a signed-in user from here on.
	// public repo whose owner is public or limited
	isPublicRepo := builder.And(
		builder.Eq{"repository.is_private": false},
		builder.Lte{"`user`.visibility": structs.VisibleTypeLimited},
	)

	// doer is a direct collaborator on this repo
	isCollabRepo := builder.Exists(
		builder.Select("1").
			From("collaboration").
			Where(builder.And(
				builder.Eq{"collaboration.repo_id": builder.Expr("repository.id")},
				builder.Eq{"collaboration.user_id": doer.ID},
			)),
	)

	//  private repo belonging to an org the doer is a member of
	doerOrgIDs := builder.Select("team_user.org_id").
		From("team_user").
		Where(builder.Eq{"team_user.uid": doer.ID})

	isOrgPrivateRepo := builder.And(
		builder.Eq{"repository.is_private": true},
		builder.In("repository.owner_id", doerOrgIDs),
	)

	count, err := e.
		Table("star").
		Join("INNER", "repository", "star.repo_id = repository.id").
		Join("LEFT", "`user`", "repository.owner_id = `user`.id").
		Where("star.uid = ?", profileUser.ID).
		And(builder.Or(isPublicRepo, isCollabRepo, isOrgPrivateRepo)).
		Count(new(Star))
	return int(count), err
}

// ClearRepoStars clears all stars for a repository and from the user that starred it.
// Used when a repository is set to private.
func ClearRepoStars(ctx context.Context, repoID int64) error {
	if _, err := db.Exec(ctx, "UPDATE `repository` SET num_stars = 0 WHERE id = ?", repoID); err != nil {
		return err
	}

	return db.DeleteBeans(ctx, Star{RepoID: repoID})
}
