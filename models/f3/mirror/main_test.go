// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package mirror

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models"
	_ "forgejo.org/models/f3/resource"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
