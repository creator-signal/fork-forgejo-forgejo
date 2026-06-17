// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package audit_test

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/modules/testimport"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
