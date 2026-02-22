// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"unicode"
	"unicode/utf8"

	"forgejo.org/modules/json"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"
)

const (
	JSON = iota
	YAML
	TOML
)

func frontmatterSeparator(line []byte, isStart bool) byte {
	idx := 0
	for ; idx < len(line); idx++ {
		if line[idx] >= utf8.RuneSelf {
			r, sz := utf8.DecodeRune(line[idx:])
			if !unicode.IsSpace(r) {
				return 0
			}
			idx += sz
			continue
		}
		if line[idx] != ' ' {
			break
		}
	}
	if len(line) == 0 {
		return 0
	}

	sep := line[idx]

	// disallow } to start and { to end
	if (isStart && sep == '}') || (!isStart && sep == '{') {
		return 0
	}

	if sep != '-' && sep != '+' && sep != '{' && sep != '}' {
		return 0
	}
	sepCount := 0
	for ; idx < len(line); idx++ {
		if line[idx] != sep {
			break
		}
		sepCount++
	}

	// dashes/pluses require at laest 3 characters
	if (sep == '-' || sep == '+') && sepCount < 3 {
		return 0
	}

	// open/close brace require exactly 1 character
	if (sep == '{' || sep == '}') && sepCount != 1 {
		return 0
	}

	for ; idx < len(line); idx++ {
		if line[idx] >= utf8.RuneSelf {
			r, sz := utf8.DecodeRune(line[idx:])
			if !unicode.IsSpace(r) {
				return 0
			}
			idx += sz
			continue
		}
		if line[idx] != ' ' {
			return 0
		}
	}
	return sep
}

// ExtractMetadata consumes a markdown file, parses YAML frontmatter,
// and returns the frontmatter metadata separated from the markdown content
func ExtractMetadata(contents string, out any) (string, error) {
	body, err := ExtractMetadataBytes([]byte(contents), out)
	return string(body), err
}

// error returned by ExtractMetadataBytes
func FailedParseFrontmatter(format int, heuristicUsed string, err error) error {
	var formatString string
	switch format {
	case JSON:
		formatString = "JSON"
	case YAML:
		formatString = "YAML"
	case TOML:
		formatString = "TOML"
	}
	return errors.Join(
		fmt.Errorf("failed to parse frontmatter (%s assumed; %s)", formatString, heuristicUsed),
		err,
	)
}

// ExtractMetadata consumes a markdown file, parses frontmatter,
// and returns the frontmatter metadata separated from the markdown content
func ExtractMetadataBytes(contents []byte, out any) ([]byte, error) {
	var front, body []byte

	start, end := 0, len(contents)
	idx := bytes.IndexByte(contents[start:], '\n')
	if idx >= 0 {
		end = start + idx
	}
	line := contents[start:end]

	startSep := frontmatterSeparator(line, true)

	if startSep == 0 {
		return contents, errors.New("no frontmatter detected")
	}

	var frontMatterStart int

	// the braces are part of the JSON frontmatter,
	// but the other separators aren't part of frontmatter
	if startSep == '{' {
		frontMatterStart = start
	} else {
		frontMatterStart = end + 1
	}

	foundFrontmatter := false
	for start = frontMatterStart; start < len(contents); start = end + 1 {
		end = len(contents)
		idx := bytes.IndexByte(contents[start:], '\n')
		if idx >= 0 {
			end = start + idx
		}
		line := contents[start:end]
		endSep := frontmatterSeparator(line, false)
		if endSep != 0 && (endSep == startSep || (startSep == '{' && endSep == '}')) {
			// the braces are part of the JSON frontmatter,
			// but the other separators aren't part of frontmatter
			if endSep == '}' {
				front = contents[frontMatterStart:end]
			} else {
				front = contents[frontMatterStart:start]
			}
			foundFrontmatter = true
			if end+1 < len(contents) {
				body = contents[end+1:]
			}
			break
		}
	}

	if !foundFrontmatter {
		return contents, errors.New("no frontmatter detected")
	}

	// since ---- counts as valid markdown, ignore empty frontmatter with these
	if startSep == '-' {
		nonSpace := bytes.IndexFunc(front, func(r rune) bool {
			return !unicode.IsSpace(r)
		})
		if nonSpace < 0 {
			return contents, errors.New("no frontmatter detected")
		}
	}

	var format int
	var heuristicUsed string

	// assume JSON if delimited by {}
	if startSep == '{' {
		format = JSON
		heuristicUsed = "bare {} frontmatter"
	} else {
		// disambiguate based upon first appearance of {, :, or =
		firstBrace := bytes.IndexByte(front, '{') // JSON
		firstColon := bytes.IndexByte(front, ':') // YAML
		firstEqual := bytes.IndexByte(front, '=') // TOML

		if firstBrace < 0 {
			firstBrace = math.MaxInt
		}
		if firstColon < 0 {
			firstColon = math.MaxInt
		}
		if firstEqual < 0 {
			firstEqual = math.MaxInt
		}

		if firstBrace < firstColon && firstBrace < firstEqual {
			format = JSON
			heuristicUsed = "{ appeared before : or ="
		} else if firstColon < firstBrace && firstColon < firstEqual {
			format = YAML
			heuristicUsed = ": appeared before { or ="
		} else if firstEqual < firstBrace && firstEqual < firstColon {
			format = TOML
			heuristicUsed = "= appeared before { or :"
		} else if startSep == '-' {
			format = YAML
			heuristicUsed = "minus separators fall back to YAML"
		} else if startSep == '+' {
			format = TOML
			heuristicUsed = "plus separators fall back to TOML"
		}
	}

	switch format {
	case YAML:
		if err := yaml.Unmarshal(front, out); err != nil {
			return contents, FailedParseFrontmatter(format, heuristicUsed, err)
		}
	case TOML:
		if err := toml.Unmarshal(front, out); err != nil {
			return contents, FailedParseFrontmatter(format, heuristicUsed, err)
		}
	case JSON:
		if err := json.Unmarshal(front, out); err != nil {
			return contents, FailedParseFrontmatter(format, heuristicUsed, err)
		}
	}

	return body, nil
}
