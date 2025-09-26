// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package markup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FilePreviewPathParser(t *testing.T) {
	type testPair struct {
		item	string
		want	FilePreviewPath
		wantErr error
	}

	tests := map[string]testPair{
		"file path no lines": {
			item: "/test-org/test-repo/src/commit/a0b1c2/test-file.md",
			want: FilePreviewPath{
				Org: "test-org",
				Repo: "test-repo",
				CommitHash: "a0b1c2" ,
				FilePath: "test-file.md",
			},
			wantErr: nil,
		},
		"file path w/ single line": {
			item: "/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1",
			want: FilePreviewPath{
				Org: "test-org",
				Repo: "test-repo",
				CommitHash: "a0b1c2" ,
				FilePath: "test-file.md",
				LineNumber: LineNumbers { Begin: "L1" },
			},
			wantErr: nil,
		},
		"file path w/ multiple lines": {
			item: "/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1-L3",
			want: FilePreviewPath{
				Org: "test-org",
				Repo: "test-repo",
				CommitHash: "a0b1c2" ,
				FilePath: "test-file.md",
				LineNumber: LineNumbers { Begin: "L1", End: "L3" },
			},
			wantErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := FilePreviewPathParser.ParseString("", tt.item)
			if err != nil {
				t.Fatalf("error parsing FilePreviewPathParser: %v", err)
			}

			assert.Equal(t, *got, tt.want, "error parsing, got: %v, want: %v", *got, tt.want)
		})
	}
}
