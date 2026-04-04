// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package resource

import (
	"testing"

	f3_forge_model "forgejo.org/models/f3/forge"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3Model_ResourceAccessors(t *testing.T) {
	resource := NewResource(0, 0, 0)

	id := int64(1324)
	resource.SetID(id)
	assert.Equal(t, id, resource.GetID())

	forgeID := int64(888)
	resource.SetForgeID(forgeID)
	assert.Equal(t, forgeID, resource.GetForgeID())

	resourceID := int64(888)
	resource.SetResourceID(resourceID)
	assert.Equal(t, resourceID, resource.GetResourceID())

	kind := KindOwner
	resource.SetKind(kind)
	assert.Equal(t, kind, resource.GetKind())
}

func TestF3Model_ResourceDatabase(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)
	forgeID := forge.ID
	resourceID := int64(2)
	unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: resourceID})
	kind := KindOwner

	resource := NewResource(forgeID, resourceID, kind)

	inserted, err := Upsert(t.Context(), resource)
	require.NoError(t, err)
	assert.NotZero(t, inserted.GetID())
	assert.Equal(t, forgeID, resource.GetForgeID())
	assert.Equal(t, resourceID, resource.GetResourceID())
	assert.Equal(t, kind, resource.GetKind())

	same, err := Upsert(t.Context(), inserted)
	require.NoError(t, err)
	assert.Equal(t, inserted.ID, same.ID)
	Equal(inserted, same)

	{
		resources, err := Find(t.Context(), FindOptions{ForgeID: &forgeID})
		require.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.True(t, Equal(inserted, resources[0]))
	}
}
