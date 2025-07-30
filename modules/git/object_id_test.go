// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidSHAPattern(t *testing.T) {
	h := Sha1ObjectFormat
	assert.True(t, h.IsValid("fee1"))
	assert.True(t, h.IsValid("abc000"))
	assert.True(t, h.IsValid("9023902390239023902390239023902390239023"))
	assert.False(t, h.IsValid("90239023902390239023902390239023902390239023"))
	assert.False(t, h.IsValid("abc"))
	assert.False(t, h.IsValid("123g"))
	assert.False(t, h.IsValid("some random text"))

	assert.Equal(t, "79ee38a6416c1ede423ec7ee0a8639ceea4aad22", ComputeBlobHash(Sha1ObjectFormat, []byte("some random blob")).String())
	assert.Equal(t, "d5c6407415d85df49592672aa421aed39b9db5e3", ComputeBlobHash(Sha1ObjectFormat, []byte("same length blob")).String())
	assert.Equal(t, "df0b5174ed06ae65aea40d43316bcbc21d82c9e3158ce2661df2ad28d7931dd6", ComputeBlobHash(Sha256ObjectFormat, []byte("some random blob")).String())
}

func TestIsEmptyCommitID(t *testing.T) {
	assert.True(t, IsEmptyCommitID("", nil))
	assert.True(t, IsEmptyCommitID("", Sha1ObjectFormat))
	assert.True(t, IsEmptyCommitID("", Sha256ObjectFormat))

	assert.False(t, IsEmptyCommitID("79ee38a6416c1ede423ec7ee0a8639ceea4aad20", Sha1ObjectFormat))
	assert.True(t, IsEmptyCommitID("0000000000000000000000000000000000000000", nil))
	assert.True(t, IsEmptyCommitID("0000000000000000000000000000000000000000", Sha1ObjectFormat))
	assert.False(t, IsEmptyCommitID("0000000000000000000000000000000000000000", Sha256ObjectFormat))

	assert.False(t, IsEmptyCommitID("00000000000000000000000000000000000000000", nil))

	assert.False(t, IsEmptyCommitID("0f0b5174ed06ae65aea40d43316bcbc21d82c9e3158ce2661df2ad28d7931dd6", nil))
	assert.True(t, IsEmptyCommitID("0000000000000000000000000000000000000000000000000000000000000000", nil))
	assert.False(t, IsEmptyCommitID("0000000000000000000000000000000000000000000000000000000000000000", Sha1ObjectFormat))
	assert.True(t, IsEmptyCommitID("0000000000000000000000000000000000000000000000000000000000000000", Sha256ObjectFormat))

	assert.False(t, IsEmptyCommitID("1", nil))
	assert.False(t, IsEmptyCommitID("0", nil))

	assert.False(t, IsEmptyCommitID("010", nil))
	assert.False(t, IsEmptyCommitID("0 0", nil))
}

func TestCommitIDInMap(t *testing.T) {
	commitIDMap := make(map[ObjectIDKey]struct{})
	commitIDsha1 := MustIDFromString("79ee38a6416c1ede423ec7ee0a8639ceea4aad22")
	commitIDsha256 := MustIDFromString("df0b5174ed06ae65aea40d43316bcbc21d82c9e3158ce2661df2ad28d7931dd6")

	commitIDsha1Lookup := MustIDFromString("79ee38a6416c1ede423ec7ee0a8639ceea4aad22")
	commitIDsha256Lookup := MustIDFromString("df0b5174ed06ae65aea40d43316bcbc21d82c9e3158ce2661df2ad28d7931dd6")

	commitIDsha1NoLookup := MustIDFromString("79ee38a6416c1ede423ec7ee0a8639ceea4aad20")
	commitIDsha256NoLookup := MustIDFromString("0f0b5174ed06ae65aea40d43316bcbc21d82c9e3158ce2661df2ad28d7931dd6")

	commitIDMap[commitIDsha1.Key()] = struct{}{}
	commitIDMap[commitIDsha256.Key()] = struct{}{}

	assert.Len(t, commitIDMap, 2)

	_, exists := commitIDMap[commitIDsha1.Key()]
	assert.True(t, exists)

	_, exists = commitIDMap[commitIDsha256.Key()]
	assert.True(t, exists)

	_, exists = commitIDMap[commitIDsha1Lookup.Key()]
	assert.True(t, exists)

	_, exists = commitIDMap[commitIDsha256Lookup.Key()]
	assert.True(t, exists)

	_, exists = commitIDMap[commitIDsha1NoLookup.Key()]
	assert.False(t, exists)

	_, exists = commitIDMap[commitIDsha256NoLookup.Key()]
	assert.False(t, exists)

	assert.Len(t, commitIDMap, 2)
}
