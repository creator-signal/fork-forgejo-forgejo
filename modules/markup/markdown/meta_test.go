// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
IssueTemplate is a legacy to keep the unit tests working.
Copied from structs.IssueTemplate, the original type has been changed a lot to support yaml template.
*/
type IssueTemplate struct {
	Name   string   `json:"name" yaml:"name" toml:"name"`
	Title  string   `json:"title" yaml:"title" toml:"title"`
	About  string   `json:"about" yaml:"about" toml:"about"`
	Labels []string `json:"labels" yaml:"labels" toml:"labels"`
	Ref    string   `json:"ref" yaml:"ref" toml:"ref"`
}

/*
Used specifically for TOML fallback.
*/
type TomlFallback struct {
	Issues []IssueTemplate `toml:"issues"`
}

func (it *IssueTemplate) Valid() bool {
	return strings.TrimSpace(it.Name) != "" && strings.TrimSpace(it.About) != ""
}

func TestExtractMetadata(t *testing.T) {
	t.Run("NoFrontmatter", func(t *testing.T) {
		var meta IssueTemplate
		body, err := ExtractMetadata(bodyTest, &meta)
		require.Error(t, err)
		assert.Equal(t, bodyTest, body)
	})

	t.Run("OopsAllOpens", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadata(fmt.Sprintf("{\n%s\n{\n%s", frontendTestJSONBody, bodyTest), &meta)
		require.Error(t, err)
	})

	t.Run("OopsAllCloses", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadata(fmt.Sprintf("}\n%s\n}\n%s", frontendTestJSONBody, bodyTest), &meta)
		require.Error(t, err)
	})

	t.Run("BackwardsJSON", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadata(fmt.Sprintf("}\n%s\n{\n%s", frontendTestJSONBody, bodyTest), &meta)
		require.Error(t, err)
	})

	t.Run("BackwardsJSON", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadata(fmt.Sprintf("}\n%s\n{\n%s", frontendTestJSONBody, bodyTest), &meta)
		require.Error(t, err)
	})
	t.Run("BraceYourselves", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadata(fmt.Sprintf("{{\n%s\n}}\n%s", frontendTestJSONBody, bodyTest), &meta)
		require.Error(t, err)
	})

	t.Run("PlainJSONAndBody", func(t *testing.T) {
		var meta IssueTemplate
		body, err := ExtractMetadata(fmt.Sprintf("%s\n%s", frontTestJSON, bodyTest), &meta)
		require.NoError(t, err)
		assert.Equal(t, bodyTest, body)
		assert.Equal(t, metaTest, meta)
		assert.True(t, meta.Valid())
	})

	t.Run("PlainJSONOnly", func(t *testing.T) {
		var meta IssueTemplate
		body, err := ExtractMetadata(frontTestJSON, &meta)
		require.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, metaTest, meta)
		assert.True(t, meta.Valid())
	})

	t.Run("ValidFrontAndBody", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				body, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", sep, front, sep, bodyTest), &meta)
				require.NoError(t, err)
				assert.Equal(t, bodyTest, body)
				assert.Equal(t, metaTest, meta)
				assert.True(t, meta.Valid())
			}
		}
	})

	t.Run("ShortSeparators", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range badSepTests {
			for _, front := range frontTests {
				_, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", sep, front, sep, bodyTest), &meta)
				require.Error(t, err)
			}
		}
	})

	t.Run("MismatchedSeparators", func(t *testing.T) {
		var meta IssueTemplate
		for _, startSep := range sepTests {
			for _, endSep := range sepTests {
				if startSep == endSep {
					continue
				}
				for _, front := range frontTests {
					_, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", startSep, front, endSep, bodyTest), &meta)
					require.Error(t, err)
				}
			}
		}
	})

	t.Run("NoFirstSeparator", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				// JSON's brace becomes the separator in this case
				if front == frontTestJSON {
					continue
				}
				_, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s", front, sep, bodyTest), &meta)
				require.Error(t, err)
			}
		}
	})

	t.Run("NoLastSeparator", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				_, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s", sep, front, bodyTest), &meta)
				require.Error(t, err)
			}
		}
	})

	t.Run("NoBody", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				body, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s", sep, front, sep), &meta)
				require.NoError(t, err)
				assert.Empty(t, body)
				assert.Equal(t, metaTest, meta)
				assert.True(t, meta.Valid())
			}
		}
	})

	t.Run("FallbackYAML", func(t *testing.T) {
		var meta []string
		body, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", minuses, fallbackYAML, minuses, bodyTest), &meta)
		require.NoError(t, err)
		assert.Equal(t, bodyTest, body)
		assert.Equal(t, []string{"test"}, meta)
		_, err2 := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", pluses, fallbackYAML, pluses, bodyTest), &meta)
		require.Error(t, err2)
	})

	t.Run("FallbackTOML", func(t *testing.T) {
		var meta TomlFallback
		body, err := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", pluses, fallbackTOML, pluses, bodyTest), &meta)
		require.NoError(t, err)
		assert.Equal(t, bodyTest, body)
		assert.Equal(t, []IssueTemplate{{}}, meta.Issues)
		_, err2 := ExtractMetadata(fmt.Sprintf("%s\n%s\n%s\n%s", minuses, fallbackTOML, minuses, bodyTest), &meta)
		require.Error(t, err2)
	})
}

func TestExtractMetadataBytes(t *testing.T) {
	t.Run("NoFrontmatter", func(t *testing.T) {
		var meta IssueTemplate
		body, err := ExtractMetadataBytes([]byte(bodyTest), &meta)
		require.Error(t, err)
		assert.Equal(t, bodyTest, string(body))
	})

	t.Run("OopsAllOpens", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("{\n%s\n{\n%s", frontendTestJSONBody, bodyTest)), &meta)
		require.Error(t, err)
	})

	t.Run("OopsAllCloses", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("}\n%s\n}\n%s", frontendTestJSONBody, bodyTest)), &meta)
		require.Error(t, err)
	})

	t.Run("BackwardsJSON", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("}\n%s\n{\n%s", frontendTestJSONBody, bodyTest)), &meta)
		require.Error(t, err)
	})

	t.Run("BraceYourselves", func(t *testing.T) {
		var meta IssueTemplate
		_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("{{\n%s\n}}\n%s", frontendTestJSONBody, bodyTest)), &meta)
		require.Error(t, err)
	})

	t.Run("PlainJSONAndBody", func(t *testing.T) {
		var meta IssueTemplate
		body, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s", frontTestJSON, bodyTest)), &meta)
		require.NoError(t, err)
		assert.Equal(t, bodyTest, string(body))
		assert.Equal(t, metaTest, meta)
		assert.True(t, meta.Valid())
	})

	t.Run("PlainJSONOnly", func(t *testing.T) {
		var meta IssueTemplate
		body, err := ExtractMetadataBytes([]byte(frontTestJSON), &meta)
		require.NoError(t, err)
		assert.Empty(t, string(body))
		assert.Equal(t, metaTest, meta)
		assert.True(t, meta.Valid())
	})

	t.Run("ValidFrontAndBody", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				body, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", sep, front, sep, bodyTest)), &meta)
				require.NoError(t, err)
				assert.Equal(t, bodyTest, string(body))
				assert.Equal(t, metaTest, meta)
				assert.True(t, meta.Valid())
			}
		}
	})

	t.Run("ShortSeparators", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range badSepTests {
			for _, front := range frontTests {
				_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", sep, front, sep, bodyTest)), &meta)
				require.Error(t, err)
			}
		}
	})

	t.Run("MismatchedSeparators", func(t *testing.T) {
		var meta IssueTemplate
		for _, startSep := range sepTests {
			for _, endSep := range sepTests {
				if startSep == endSep {
					continue
				}
				for _, front := range frontTests {
					_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", startSep, front, endSep, bodyTest)), &meta)
					require.Error(t, err)
				}
			}
		}
	})

	t.Run("NoFirstSeparator", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				// JSON's brace becomes the separator in this case
				if front == frontTestJSON {
					continue
				}
				_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s", front, sep, bodyTest)), &meta)
				require.Error(t, err)
			}
		}
	})

	t.Run("NoLastSeparator", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				_, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s", sep, front, bodyTest)), &meta)
				require.Error(t, err)
			}
		}
	})

	t.Run("NoBody", func(t *testing.T) {
		var meta IssueTemplate
		for _, sep := range sepTests {
			for _, front := range frontTests {
				body, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s", sep, front, sep)), &meta)
				require.NoError(t, err)
				assert.Empty(t, string(body))
				assert.Equal(t, metaTest, meta)
				assert.True(t, meta.Valid())
			}
		}
	})

	t.Run("FallbackYAML", func(t *testing.T) {
		var meta []string
		body, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", minuses, fallbackYAML, minuses, bodyTest)), &meta)
		require.NoError(t, err)
		assert.Equal(t, bodyTest, string(body))
		assert.Equal(t, []string{"test"}, meta)
		_, err2 := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", pluses, fallbackYAML, pluses, bodyTest)), &meta)
		require.Error(t, err2)
	})

	t.Run("FallbackTOML", func(t *testing.T) {
		var meta TomlFallback
		body, err := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", pluses, fallbackTOML, pluses, bodyTest)), &meta)
		require.NoError(t, err)
		assert.Equal(t, bodyTest, string(body))
		assert.Equal(t, []IssueTemplate{{}}, meta.Issues)
		_, err2 := ExtractMetadataBytes([]byte(fmt.Sprintf("%s\n%s\n%s\n%s", minuses, fallbackTOML, minuses, bodyTest)), &meta)
		require.Error(t, err2)
	})
}

var (
	minuses       = "-----"
	pluses        = "++++"
	sepTests      = []string{minuses, pluses}
	badMinuses    = "--"
	badPluses     = "++"
	badSepTests   = []string{badMinuses, badPluses}
	frontTestYAML = `name: Test
about: "A Test"
title: "Test Title"
labels:
  - bug
  - "test label"`
	frontTestTOML = `name = "Test"
about = "A Test"
title = "Test Title"
labels = ["bug", "test label"]`
	frontendTestJSONBody = `"name": "Test",
"about": "A Test",
"title": "Test Title",
"labels": ["bug", "test label"]`
	frontTestJSON = fmt.Sprintf("{\n%s\n}", frontendTestJSONBody)
	frontTests    = []string{frontTestYAML, frontTestTOML, frontTestJSON}
	bodyTest      = "\nThis is the head\nAnd this is the body\nAnd this is the foot\n"
	metaTest      = IssueTemplate{
		Name:   "Test",
		About:  "A Test",
		Title:  "Test Title",
		Labels: []string{"bug", "test label"},
	}
	fallbackYAML = `- test`
	fallbackTOML = `[[issues]]`
)
