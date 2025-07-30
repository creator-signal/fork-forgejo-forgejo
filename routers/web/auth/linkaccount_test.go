package auth

import (
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkAccountPageRenderWithOrWithoutCaptcha(t *testing.T) {
	require.NoError(t, unittest.LoadFixtures())
	ctx, resp := contexttest.MockContext(t, "/link_account")
	LinkAccount(ctx)
	contexttest.LoadUser(t, ctx, 30)
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 30})
	ctx.Session.Set("linkAccountGothUser", user)
	assert.Equal(t, http.StatusOK, resp.Code)

	context.SetCaptchaData(ctx)
	LinkAccount(ctx)
	assert.Equal(t, http.StatusOK, resp.Code)
}
