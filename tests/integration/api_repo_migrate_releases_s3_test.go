// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"io"
	"strings"
	"testing"

	"forgejo.org/modules/storage"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isS3Storage() bool {
	attachmentStorage := storage.Attachments
	if attachmentStorage == nil {
		return false
	}
	return true
}

func TestValidateAndBufferAsset(t *testing.T) {
	if !isS3Storage() {
		t.Skip("Test skipped for non-Minio-storage.")
		return
	}

	defer tests.PrepareTestEnv(t)()

	testCases := []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"small", "test asset content"},
		{"medium", strings.Repeat("medium content\n", 100)},
		{"large", strings.Repeat("large content for S3 buffering\n", 1000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalRC := io.NopCloser(strings.NewReader(tc.content))

			bufferedContent, err := io.ReadAll(originalRC)
			require.NoError(t, err)
			originalRC.Close()

			bufferedRC := io.NopCloser(strings.NewReader(string(bufferedContent)))
			defer bufferedRC.Close()

			result, err := io.ReadAll(bufferedRC)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(result))
		})
	}
}
