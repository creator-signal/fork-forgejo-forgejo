// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package markup

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FilePreviewPathParser(t *testing.T) {
	type testPair struct {
		item    string
		want    FilePreviewPath
		wantErr error
	}

	tests := map[string]testPair{
		"file path no lines": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md",
			want: FilePreviewPath{
				Org:        "test-org",
				Repo:       "test-repo",
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantErr: nil,
		},
		"file path w/ single line": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1",
			want: FilePreviewPath{
				Org:        "test-org",
				Repo:       "test-repo",
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantErr: nil,
		},
		"file path w/ multiple lines": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1-L3",
			want: FilePreviewPath{
				Org:        "test-org",
				Repo:       "test-repo",
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tURL, err := Pointer(url.URL{}).Parse(tt.item)
			if err != nil {
				t.Fatalf("error parsing URL: %v", err)
			}
			got, err := FilePreviewPathParser.ParseString("", tURL.Path)
			if err != nil {
				t.Fatalf("error parsing FilePreviewPath: %v", err)
			}

			assert.Equal(t, tt.want, *got, "error parsing, want: %v, got: %v", tt.want, *got)
		})
	}
}

func Test_LineNumbersParser(t *testing.T) {
	type testPair struct {
		item    string
		want    LineNumbers
		wantErr error
	}

	tests := map[string]testPair{
		"file path w/ single line": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1",
			want: LineNumbers{
				Begin: 1,
			},
			wantErr: nil,
		},
		"file path w/ multiple lines": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1-L3",
			want: LineNumbers{
				Begin: 1,
				End:   Pointer(int(3)),
			},
			wantErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tURL, err := Pointer(url.URL{}).Parse(tt.item)
			if err != nil {
				t.Fatalf("error parsing URL: %v", err)
			}

			if len(tURL.Fragment) > 0 {
				got, err := LineNumbersParser.ParseString("", tURL.Fragment)
				if err != nil {
					t.Fatalf("error parsing LineNumber(s): %v", err)
				}

				assert.Equal(t, tt.want, *got, "error parsing, want: %v, got: %v", tt.want, *got)
			}
		})
	}
}
