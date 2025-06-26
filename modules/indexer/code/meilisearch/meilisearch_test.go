// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package meilisearch

import (
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "test", "test"},
		{"empty string", "", ""},
		{"non-string", 123, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertResult(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		total, hits, langs, err := convertResult(nil, 10)
		require.Error(t, err)
		assert.Equal(t, int64(0), total)
		assert.Nil(t, hits)
		assert.Nil(t, langs)
	})

	t.Run("valid result", func(t *testing.T) {
		searchResult := &meilisearch.SearchResponse{
			Hits: []any{
				map[string]any{
					"repo_id":    float64(1),
					"content":    "test content",
					"filename":   "test.txt",
					"commit_id":  "abc123",
					"language":   "Go",
					"updated_at": float64(1234567890),
					"_matchesPosition": map[string]any{
						"content": []any{
							map[string]any{
								"start":  float64(5),
								"length": float64(4),
							},
						},
					},
				},
			},
			FacetDistribution: map[string]any{
				"language": map[string]any{
					"Go": float64(1),
				},
			},
		}

		total, hits, langs, err := convertResult(searchResult, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, hits, 1)
		assert.Len(t, langs, 1)

		assert.Equal(t, int64(1), hits[0].RepoID)
		assert.Equal(t, "test.txt", hits[0].Filename)
		assert.Equal(t, "abc123", hits[0].CommitID)
		assert.Equal(t, "test content", hits[0].Content)
		assert.Equal(t, "Go", hits[0].Language)
		assert.Len(t, hits[0].Matches, 1)
		assert.Equal(t, 5, hits[0].Matches[0].Start)
		assert.Equal(t, 9, hits[0].Matches[0].End)

		assert.Equal(t, "Go", langs[0].Language)
		assert.Equal(t, 1, langs[0].Count)
	})
}

func TestExtractAggs(t *testing.T) {
	t.Run("nil facet distribution", func(t *testing.T) {
		result := extractAggs(&meilisearch.SearchResponse{})
		assert.Empty(t, result)
	})

	t.Run("valid facet distribution", func(t *testing.T) {
		searchResult := &meilisearch.SearchResponse{
			FacetDistribution: map[string]any{
				"language": map[string]any{
					"Go":   float64(5),
					"Rust": float64(3),
				},
			},
		}

		result := extractAggs(searchResult)
		assert.Len(t, result, 2)

		assert.Equal(t, "Go", result[0].Language)
		assert.Equal(t, 5, result[0].Count)
		assert.Equal(t, "Rust", result[1].Language)
		assert.Equal(t, 3, result[1].Count)
	})
}
