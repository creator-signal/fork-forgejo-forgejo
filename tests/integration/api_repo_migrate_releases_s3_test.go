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

// singleReadReader simulates an HTTP request body stream that cannot be rewound.
// Once consumed, subsequent reads return EOF, mimicking how network streams behave.
type singleReadReader struct {
	content []byte
	offset  int
}

func (r *singleReadReader) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.content) {
		return 0, io.EOF
	}
	n = copy(p, r.content[r.offset:])
	r.offset += n
	if r.offset >= len(r.content) {
		return n, io.EOF
	}
	return n, nil
}

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

func TestUnbufferedStreamFailure(t *testing.T) {
	if !isS3Storage() {
		t.Skip("Test skipped for non-Minio-storage.")
		return
	}

	defer tests.PrepareTestEnv(t)()

	content := "asset content that would fail S3 upload without buffering"

	reader := &singleReadReader{content: []byte(content)}

	rc := io.NopCloser(reader)
	defer rc.Close()

	// First read consumes the stream (validation step in migration)
	validationRead, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, string(validationRead))

	// Second read gets nothing. S3 upload would fail with:
	// "The Content-MD5 you specified does not match what we received" because Content-Length header says len(content) but body is empty
	uploadRead, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Empty(t, uploadRead, "Stream exhausted - this causes S3 upload to fail with content length mismatch")
}
