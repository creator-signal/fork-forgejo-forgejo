package v2

import (
	"testing"

	"forgejo.org/models/user"

	"github.com/elimity-com/scim"
	"github.com/stretchr/testify/assert"
)

func TestExtractPrimaryEmail_PrimaryFlagSet(t *testing.T) {
	attrs := scim.ResourceAttributes{
		"emails": []any{
			map[string]any{"value": "a@example.com", "primary": false},
			map[string]any{"value": "b@example.com", "primary": true},
		},
	}
	assert.Equal(t, "b@example.com", extractPrimaryEmail(attrs))
}

func TestExtractPrimaryEmail_FallbackToFirstWhenNoPrimary(t *testing.T) {
	attrs := scim.ResourceAttributes{
		"emails": []any{
			map[string]any{"value": "first@example.com"},
			map[string]any{"value": "second@example.com"},
		},
	}
	assert.Equal(t, "first@example.com", extractPrimaryEmail(attrs))
}

func TestExtractPrimaryEmail_NoEmailsKey(t *testing.T) {
	assert.Empty(t, extractPrimaryEmail(scim.ResourceAttributes{}))
}

func TestExtractPrimaryEmail_EmptyArray(t *testing.T) {
	attrs := scim.ResourceAttributes{
		"emails": []any{},
	}
	assert.Empty(t, extractPrimaryEmail(attrs))
}

func TestExtractPrimaryEmail_WrongType(t *testing.T) {
	attrs := scim.ResourceAttributes{
		"emails": "something else",
	}
	assert.Empty(t, extractPrimaryEmail(attrs))
}

func TestToResource_FullUser(t *testing.T) {
	u := &user.User{
		ID:          42,
		Name:        "jdoe",
		Email:       "jdoe@example.com",
		FullName:    "John Doe",
		IsActive:    true,
		LoginSource: 5,
		LoginName:   "ext-id-123",
	}
	res := toResource(u)

	assert.Equal(t, "42", res.ID)
	assert.Equal(t, "jdoe", res.Attributes["userName"])
	assert.Equal(t, "John Doe", res.Attributes["displayName"])
	assert.Equal(t, true, res.Attributes["active"])
	assert.Equal(t, "ext-id-123", res.ExternalID.Value())
}

func TestToResource_NoFullName(t *testing.T) {
	u := &user.User{
		ID:          7,
		Name:        "noname",
		Email:       "noname@example.com",
		IsActive:    false,
		LoginSource: 1,
		LoginName:   "ext-7",
	}
	res := toResource(u)

	_, hasDisplayName := res.Attributes["displayName"]
	assert.False(t, hasDisplayName)
	assert.Equal(t, false, res.Attributes["active"])
}

func TestToResource_NoLoginSource(t *testing.T) {
	u := &user.User{
		ID:          1,
		Name:        "localuser",
		Email:       "local@example.com",
		IsActive:    true,
		LoginSource: 0,
	}
	res := toResource(u)

	assert.False(t, res.ExternalID.Present())
}
