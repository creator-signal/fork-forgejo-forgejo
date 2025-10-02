// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package markup

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_FilePreviewPath(t *testing.T) {
	type testPair struct {
		item         string
		want         FilePreviewPath
		wantExternal *string
		wantOrg      string
		wantRepo     string
		wantErr      error
	}

	tests := map[string]testPair{
		"file path no lines": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md",
			want: FilePreviewPath{
				OrgRepo:    OrgRepo{Parts: []string{"test-org", "test-repo"}},
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantExternal: nil,
			wantOrg:      "test-org",
			wantRepo:     "test-repo",
			wantErr:      nil,
		},
		"file path w/ single line": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1",
			want: FilePreviewPath{
				OrgRepo:    OrgRepo{Parts: []string{"test-org", "test-repo"}},
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantExternal: nil,
			wantOrg:      "test-org",
			wantRepo:     "test-repo",
			wantErr:      nil,
		},
		"file path w/ multiple lines": {
			item: "https://example.repo/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1-L3",
			want: FilePreviewPath{
				OrgRepo:    OrgRepo{Parts: []string{"test-org", "test-repo"}},
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantExternal: nil,
			wantOrg:      "test-org",
			wantRepo:     "test-repo",
			wantErr:      nil,
		},
		"file path external w/ single sub-directory": {
			item: "https://example.repo/test-sub/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1-L3",
			want: FilePreviewPath{
				OrgRepo:    OrgRepo{Parts: []string{"test-sub", "test-org", "test-repo"}},
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantExternal: Pointer(string("test-sub")),
			wantOrg:      "test-org",
			wantRepo:     "test-repo",
			wantErr:      nil,
		},
		"file path external w/ multiple sub-directory": {
			item: "https://example.repo/test-sub/sub2/test-org/test-repo/src/commit/a0b1c2/test-file.md#L1-L3",
			want: FilePreviewPath{
				OrgRepo:    OrgRepo{Parts: []string{"test-sub", "sub2", "test-org", "test-repo"}},
				CommitHash: "a0b1c2",
				FilePath:   []string{"test-file.md"},
			},
			wantExternal: Pointer(string("test-sub/sub2")),
			wantOrg:      "test-org",
			wantRepo:     "test-repo",
			wantErr:      nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tURL, err := url.Parse(tt.item)
			if err != nil {
				t.Fatalf("error parsing URL: %v", err)
			}
			got, err := FilePreviewPathParser.ParseString("", tURL.Path)
			if err != nil {
				t.Fatalf("error parsing FilePreviewPath: %v", err)
			}

			assert.Equal(t, tt.want, *got, "error parsing, want: %v, got: %v", tt.want, *got)
			assert.Equal(t, tt.wantOrg, got.OrgRepo.Org(), "error getting org, want: %v, got: %v", tt.wantOrg, got.OrgRepo.Org())
			assert.Equal(t, tt.wantRepo, got.OrgRepo.Repo(), "error getting org, want: %v, got: %v", tt.wantRepo, got.OrgRepo.Repo())
			if tt.wantExternal != nil {
				assert.Equal(t, tt.wantExternal, got.OrgRepo.External(), "error getting external, want: %v, got: %v", tt.wantExternal, got.OrgRepo.External())
			}
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
			tURL, err := url.Parse(tt.item)
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
