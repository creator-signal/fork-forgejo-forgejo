package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add field BlockOnCodeownerReviews to the protected branch struct",
		Upgrade:     AddBlockOnCodeownerReviews,
	})
}

// AddBlockOnCodeownerReviews adds block on codeowner reviews branch protection
func AddBlockOnCodeownerReviews(x *xorm.Engine) error {
	type ProtectedBranch struct {
		BlockOnCodeownerReviews bool `xorm:"NOT NULL DEFAULT false"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreConstrains:  true,
		IgnoreDropIndices: true,
	}, new(ProtectedBranch))
	return err
}
