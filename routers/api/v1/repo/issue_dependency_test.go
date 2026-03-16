package repo

import (
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
)

// TestGetIssueDependenciesTotalCount tests total count in GetIssueDependencies.
func TestGetIssueDependenciesTotalCount(t *testing.T) {
	unittest.PrepareTestEnv(t)

	// request
	ctx, resp := contexttest.MockAPIContext(t, "user2/repo256/issues/1/dependencies")
	contexttest.LoadRepo(t, ctx, 66)
	ctx.SetParams(":index", "1")
	GetIssueDependencies(ctx)

	// check
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	count := resp.Header().Get("X-Total-Count")
	assert.NotEmpty(t, count, "Total count header should be set")
	assert.Equal(t, "0", count)
}

// TestGetIssueBlocksTotalCount tests total count in GetIssueBlocks.
func TestGetIssueBlocksTotalCount(t *testing.T) {
	unittest.PrepareTestEnv(t)

	// request
	ctx, resp := contexttest.MockAPIContext(t, "user2/repo1/issues/1/blocks")
	contexttest.LoadRepo(t, ctx, 1)
	ctx.SetParams(":index", "1")
	GetIssueBlocks(ctx)

	// check
	assert.Equal(t, http.StatusOK, ctx.Resp.Status())
	count := resp.Header().Get("X-Total-Count")
	assert.NotEmpty(t, count, "Total count header should be set")
	assert.Equal(t, "0", count)
}
