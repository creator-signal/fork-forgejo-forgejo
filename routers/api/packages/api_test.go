// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	stdctx "context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRequestCanceled(t *testing.T) {
	// A canceled or deadline-exceeded request context (the client went away,
	// e.g. docker push aborting some of its concurrent blob HEAD requests) must
	// be recognized so it is reported as 499 rather than a 500 server error.
	assert.True(t, isRequestCanceled(stdctx.Canceled))
	assert.True(t, isRequestCanceled(stdctx.DeadlineExceeded))

	// It must also see through wrapping, since the cancellation surfaces from a
	// deeper call (e.g. "access token GetAccessTokenBySHA: context canceled").
	assert.True(t, isRequestCanceled(fmt.Errorf("access token GetAccessTokenBySHA: %w", stdctx.Canceled)))
	assert.True(t, isRequestCanceled(fmt.Errorf("determineAccessMode: %w", stdctx.DeadlineExceeded)))

	// Genuine errors and nil must not be treated as cancellations.
	assert.False(t, isRequestCanceled(errors.New("boom")))
	assert.False(t, isRequestCanceled(nil))
}
