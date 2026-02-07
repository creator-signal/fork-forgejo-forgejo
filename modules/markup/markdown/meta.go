// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package markdown

import (
	"bytes"
	"errors"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
	"github.com/BurntSushi/toml"
	"encoding/json"
)

const (
	Json = iota
	Yaml
	Toml
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

	// dashes/pluses require 3 characters
	if (sep == '-' || sep == '+') && sepCount < 3 {
		return 0
	}

	// open/close brace require 1 character
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
		return contents, errors.New("frontmatter must start with a separator line")
	}

	var frontMatterStart int
	if startSep == '{' {
		frontMatterStart = start
	} else {
		frontMatterStart = end + 1
	}

	var endSep byte
	for start = frontMatterStart; start < len(contents); start = end + 1 {
		end = len(contents)
		idx := bytes.IndexByte(contents[start:], '\n')
		if idx >= 0 {
			end = start + idx
		}
		line := contents[start:end]
		endSep = frontmatterSeparator(line, false)
		if endSep == startSep || (startSep == '{' && endSep == '}') {
			if endSep == '}' {
				front = contents[frontMatterStart:end]
			} else {
				front = contents[frontMatterStart:start]
			}
			if end+1 < len(contents) {
				body = contents[end+1:]
			}
			break
		}
	}

	if len(front) == 0 {
		return contents, errors.New("could not determine metadata")
	}

	var format int

	// assume JSON if delimited by {}
	if startSep == '{' || endSep == '}' {
		format = Json
	} else {
		// disambiguate based upon first appearance of {, :, or =
		firstBrace := bytes.IndexByte(front, '{') // JSON
		firstColon := bytes.IndexByte(front, ':') // YAML
		firstEqual := bytes.IndexByte(front, '=') // TOML

		if firstBrace < 0 {
			firstBrace = 0xFFFFFFFF
		}
		if firstColon < 0 {
			firstColon = 0xFFFFFFFF
		}
		if firstEqual < 0 {
			firstEqual = 0xFFFFFFFF
		}

		if firstBrace <= firstColon && firstBrace <= firstEqual {
			format = Json
		} else if firstColon <= firstBrace && firstColon <= firstEqual {
			format = Yaml
		} else if firstEqual <= firstBrace && firstEqual <= firstColon {
			format = Toml
		} else if startSep == '-' {
			format = Yaml
		} else if startSep == '+' {
			format = Json
		}
	}

	switch format {
		case Yaml:
			if err := yaml.Unmarshal(front, out); err != nil {
				return contents, err
			}
		case Toml:
			if err := toml.Unmarshal(front, out); err != nil {
				return contents, err
			}
		case Json:
			if err := json.Unmarshal(front, out); err != nil {
				return contents, err
			}
	}

	return body, nil
}
