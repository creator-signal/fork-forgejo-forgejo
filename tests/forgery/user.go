// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgery

import (
	"math/rand/v2"
	"regexp"
	"strconv"
	"testing"

	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/require"
)

var nameCleaner = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func newEntityName(prefix, testName string) string {
	return prefix + nameCleaner.ReplaceAllLiteralString(testName, "_") + "-" + strconv.FormatUint(uint64(rand.Uint32()), 16)
}

type CreateUserOptions struct{}

func CreateUser(t testing.TB, opts *CreateUserOptions) *user_model.User {
	u := &user_model.User{}

	name := newEntityName("user-", t.Name())

	u.Name = name
	u.Email = name + "@test.forgejo.org"

	err := user_model.CreateUser(t.Context(), u)
	require.NoError(t, err)
	return u
}
