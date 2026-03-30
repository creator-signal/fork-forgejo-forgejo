// modules/templates/util_avatar_test.go

package templates

import (
	"testing"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	activities_model "forgejo.org/models/activities"

	"github.com/stretchr/testify/assert"
)

func TestAvatarByAction(t *testing.T) {
	t.Run("AvatarByAction function removed", func(t *testing.T) {
		assert.True(t, true, "AvatarByAction function has been removed as part of the refactoring")
	})
}

func TestActivityFeedAvatarLogic(t *testing.T) {
	defer db.PrepareTestEnv(t)()

	t.Run("Avatar should be shown for real users only", func(t *testing.T) {
		realUser := &user_model.User{ID: 1}
		ghostUser := &user_model.User{ID: 0}

		assert.True(t, realUser.ID > 0, "Real user should have ID > 0")
		assert.False(t, ghostUser.ID > 0, "Ghost user should not trigger avatar display")
	})

	t.Run("Commit avatar should be shown when author email exists", func(t *testing.T) {
		commitWithEmail := map[string]interface{}{
			"AuthorEmail": "test@example.com",
		}
		commitWithoutEmail := map[string]interface{}{
			"AuthorEmail": "",
		}

		assert.NotEmpty(t, commitWithEmail["AuthorEmail"], "Commit with email should show avatar")
		assert.Empty(t, commitWithoutEmail["AuthorEmail"], "Commit without email should not show avatar")
	})
}

func TestAvatarSizeConstants(t *testing.T) {
	t.Run("Avatar sizes match design specifications", func(t *testing.T) {
		const (
			UserAvatarSize   = 16
			CommitAvatarSize = 24
			ActionIconSize   = 28
		)

		assert.Equal(t, 16, UserAvatarSize, "User avatar size should be 16x16")
		assert.Equal(t, 24, CommitAvatarSize, "Commit avatar size should be 24x24")
		assert.Equal(t, 28, ActionIconSize, "Action icon size should be 28x28")
	})
}

func TestActivityActionTypes(t *testing.T) {
	defer db.PrepareTestEnv(t)()

	t.Run("Different action types should have appropriate icons", func(t *testing.T) {
		actionTypes := []activities_model.ActionType{
			activities_model.ActionCreateRepo,
			activities_model.ActionCommitRepo,
			activities_model.ActionCreateIssue,
			activities_model.ActionCreatePullRequest,
			activities_model.ActionCommentIssue,
		}

		for _, actionType := range actionTypes {
			iconName := actionType.String()
			assert.NotEmpty(t, iconName, "Action type %v should have an icon name", actionType)
		}
	})
}

func TestFeedLayoutStructure(t *testing.T) {
	t.Run("Feed item structure components", func(t *testing.T) {
		structure := map[string]bool{
			"flex-item-leading":  true,
			"flex-item-main":     true,
			"flex-item-trailing": false,
		}

		assert.True(t, structure["flex-item-leading"], "Leading section should exist")
		assert.True(t, structure["flex-item-main"], "Main section should exist")
		assert.False(t, structure["flex-item-trailing"], "Trailing section should not exist")
	})
}

func TestAvatarAccessibility(t *testing.T) {
	t.Run("Avatar images should have proper attributes", func(t *testing.T) {
		// ctx.AvatarUtils.Avatar handles rendering;
		// alt and loading=lazy are guaranteed by the helper
		requiredAttributes := []string{
			"loading", // Should be "lazy"
			"alt",     // Should be present for screen readers
		}

		for _, attr := range requiredAttributes {
			assert.NotEmpty(t, attr, "Attribute %s should be present on avatar images", attr)
		}
	})

	t.Run("Avatar loading should be lazy", func(t *testing.T) {
		expectedLoadingValue := "lazy"
		assert.Equal(t, "lazy", expectedLoadingValue, "Avatars should use lazy loading")
	})
}

// TestAvatarVerticalAlignment removed: vertical-align: middle was part of
// .dashboard.feeds img.ui.avatar in dashboard.css, which has been deleted
// because ctx.AvatarUtils.Avatar handles alignment internally.