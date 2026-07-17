// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/forgefed"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
