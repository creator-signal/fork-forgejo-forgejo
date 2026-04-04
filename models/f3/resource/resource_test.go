// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package resource

import (
	"testing"

	f3_forge_model "forgejo.org/models/f3/forge"
	f3_mirror_model "forgejo.org/models/f3/mirror"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3Model_ResourceDatabase(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)
	mirror := f3_mirror_model.NewMirror()
	mirror.SetForgeID(forge.ID)
	mirror, err = f3_mirror_model.Upsert(t.Context(), mirror)
	require.NoError(t, err)
	mirrorID := mirror.ID
	resourceID := int64(2)
	unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: resourceID})
	kind := KindOwner

	var inserted *Resource

	t.Run("Upsert", func(t *testing.T) {
		resource := NewResource(mirrorID, resourceID, kind)

		var err error
		inserted, err = Upsert(t.Context(), resource)
		require.NoError(t, err)
		assert.NotZero(t, inserted.GetID())
		assert.Equal(t, mirrorID, resource.GetMirrorID())
		assert.Equal(t, resourceID, resource.GetResourceID())
		assert.Equal(t, kind, resource.GetKind())

		same, err := Upsert(t.Context(), inserted)
		require.NoError(t, err)
		assert.Equal(t, inserted.ID, same.ID)
		Equal(inserted, same)
	})

	t.Run("Find", func(t *testing.T) {
		resources, err := Find(t.Context(), FindOptions{MirrorID: &mirrorID})
		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.True(t, Equal(inserted, resources[0]))
	})
}
