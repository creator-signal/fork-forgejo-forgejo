package moderation

import (
	"testing"

	"forgejo.org/models/unittest"

	_ "forgejo.org/models/forgefed"
	_ "forgejo.org/models/moderation"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}
