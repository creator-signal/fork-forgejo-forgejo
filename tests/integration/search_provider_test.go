// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/services/federation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebfingerProvider(t *testing.T) {
	defer test.MockVariableValue(&setting.IsProd, false)()

	mock := test.NewFederationServerMock()
	server := mock.DistantServer(t)
	url, _ := url.Parse(server.URL)

	defer server.Close()

	ctx := t.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// User is set to sleep for five seconds before completing
	provider := federation.WebfingerSearch{}
	_, err := provider.Search(timeoutCtx, fmt.Sprintf("@sloth@%s", url.Host))

	require.Error(t, err)

	// Explicitly checking errors.is as this is the code used in the user search
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
}
