// ABOUTME: Test setup and initialization for container package router tests
// ABOUTME: Provides database setup and cleanup for integration testing
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/packages"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
