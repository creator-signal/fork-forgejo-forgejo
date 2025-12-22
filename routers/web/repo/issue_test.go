package repo

import (
	"net/http/httptest"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/templates"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

func TestRenderIssueView(t *testing.T) {
	unittest.PrepareTestEnv(t)

	pr, _ := issues_model.GetPullRequestByID(db.DefaultContext, 1)
	_ = pr.LoadIssue(db.DefaultContext)
	_ = pr.Issue.LoadPoster(db.DefaultContext)
	_ = pr.Issue.LoadRepo(db.DefaultContext)

	run := func(name string, cb func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder)) {
		t.Run(name, func(t *testing.T) {
			ctx, resp := contexttest.MockContext(t, "/")
			ctx.Render = templates.HTMLRenderer()
			contexttest.LoadUser(t, ctx, pr.Issue.PosterID)
			contexttest.LoadRepo(t, ctx, pr.BaseRepoID)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()
			cb(t, ctx, resp)
		})
	}

	run("pending reviews don't count as participants", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.SetParams(":index", "1")
		ctx.SetParams(":type", "issues")
		ctx.Data["Link"] = "1"
		ViewIssue(ctx)
		participantNames := make([]string, 0, 3)
		for _, v := range ctx.Data["Participants"].([]*user_model.User) {
			participantNames = append(participantNames, v.Name)
		}
		assert.Equal(t, 3, ctx.Data["NumParticipants"])
		assert.Equal(t, []string{"user1", "org3", "user5"}, participantNames)
	})
}
