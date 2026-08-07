// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package method

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.org/services/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedOutputMethod is an auth.Method that always returns the same output.
type fixedOutputMethod struct {
	output auth.MethodOutput
}

func (m *fixedOutputMethod) Name() string { return "fixedOutput" }

func (m *fixedOutputMethod) Verify(_ *http.Request, _ http.ResponseWriter, _ auth.SessionStore) auth.MethodOutput {
	return m.output
}

func TestGroupVerifyReportsCancellation(t *testing.T) {
	// The error shape a method produces when the database lookup is interrupted, as seen in
	// https://codeberg.org/forgejo/forgejo/issues/13782
	canceled := fmt.Errorf("access token GetAccessTokenBySHA: %w", context.Canceled)

	t.Run("canceled error is reported as cancellation", func(t *testing.T) {
		group := NewGroup(&fixedOutputMethod{output: &auth.AuthenticationError{Error: canceled}})

		output := group.Verify(httptest.NewRequest("GET", "/v2/owner/image/blobs/sha256:cafe", nil), httptest.NewRecorder(), nil)

		result, ok := output.(*auth.AuthenticationCancelled)
		require.True(t, ok, "expected AuthenticationCancelled, got %T", output)
		assert.ErrorIs(t, result.Error, context.Canceled)
	})

	t.Run("deadline exceeded is reported as cancellation", func(t *testing.T) {
		deadline := fmt.Errorf("access token GetAccessTokenBySHA: %w", context.DeadlineExceeded)
		group := NewGroup(&fixedOutputMethod{output: &auth.AuthenticationError{Error: deadline}})

		output := group.Verify(httptest.NewRequest("GET", "/v2/owner/image/blobs/sha256:cafe", nil), httptest.NewRecorder(), nil)

		assert.IsType(t, &auth.AuthenticationCancelled{}, output)
	})

	t.Run("genuine error stays an error", func(t *testing.T) {
		group := NewGroup(&fixedOutputMethod{output: &auth.AuthenticationError{Error: errors.New("database is closed")}})

		output := group.Verify(httptest.NewRequest("GET", "/v2/owner/image/blobs/sha256:cafe", nil), httptest.NewRecorder(), nil)

		assert.IsType(t, &auth.AuthenticationError{}, output)
	})
}
