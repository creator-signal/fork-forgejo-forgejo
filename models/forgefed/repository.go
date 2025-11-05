// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

// RepositoryActivity is used to store federated inbox/outbox activities.
type RepositoryActivity struct {
	ID         int64  `xorm:"pk"`
	RepoID     int64  `xorm:"repo_id"`
	ActivityID string `xorm:"TEXT 'activity_id'"`
	Activity   string `xorm:"TEXT 'activity'"`
}
