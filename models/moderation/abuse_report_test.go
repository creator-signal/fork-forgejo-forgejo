// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package moderation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbuseCategoryTypeString(t *testing.T) {
	category := AbuseCategoryTypeOther
	assert.Equal(t, "Other", category.String())

	category = AbuseCategoryTypeSpam
	assert.Equal(t, "Spam", category.String())

	category = AbuseCategoryTypeMalware
	assert.Equal(t, "Malware", category.String())

	category = AbuseCategoryTypeIllegalContent
	assert.Equal(t, "Illegal content", category.String())

	// Keep this as the latest in the enum + 1
	category = AbuseCategoryTypeIllegalContent + 1
	assert.Equal(t, "Unknown category", category.String())
}
