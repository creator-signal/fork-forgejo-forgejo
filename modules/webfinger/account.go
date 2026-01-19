// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webfinger

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/oleiade/gomme"
)

// joinStrings is a helper function to join an array of strings with no delimiter.
func joinStrings(s []string) (string, error) {
	return strings.Join(s, ""), nil
}

// runeToString is a helper function to convert a rune to a string.
func runeToString(r rune) (string, error) {
	return string(r), nil
}

// ParseWebfingerAccount parses a WebFinger `resource` component using the `acct` format for ActivityPub accounts.
func parseWebfingerAccount() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Preceded(
			gomme.Optional(gomme.Token[string]("@")),
			gomme.Map(
				gomme.Many1(gomme.Alternative(
					uriUnreserved(),
					uriPercentEncoded(),
					uriSubDelims(),
				)),
				joinStrings,
			),
		)(input)
	}
}

// URIUnreserved returns a gomme parser for
// [URI unreserved](https://datatracker.ietf.org/doc/rfc3986/) characters.
func uriUnreserved() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Alternative(
			gomme.Alphanumeric1[string](),
			gomme.Map(
				gomme.OneOf[string]('-', '.', '_', '~', ':'),
				runeToString,
			),
		)(input)
	}
}

// URIPercentEncoded returns a gomme parser for
// [URI pct-encoded](https://datatracker.ietf.org/doc/rfc3986/) characters.
func uriPercentEncoded() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.Preceded(
				gomme.Token[string]("%"),
				gomme.Count(gomme.HexDigit0[string](), 2),
			),
			func(hex []string) (string, error) {
				return fmt.Sprintf("%%%s%s", hex[0], hex[1]), nil
			},
		)(input)
	}
}

// URISubDelims returns a gomme parser for
// [URI sub-delims](https://datatracker.ietf.org/doc/rfc3986/) characters.
func uriSubDelims() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.OneOf[string]('!', '$', '&', '\'', '(', ')', '*', '+', ',', ';', '='),
			runeToString,
		)(input)
	}
}

// uriHost returns a gomme parser for the host part of a URI
// https://datatracker.ietf.org/doc/rfc3986/
func uriHost() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.Alternative(
				ipLiteral(),
				ipv4Address(),
				regName(),
			),
			url.PathUnescape,
		)(input)
	}
}

// uriPort returns the port of the host part of a URI
// https://datatracker.ietf.org/doc/rfc3986
func uriPort() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Preceded(
			gomme.Token[string](":"),
			gomme.Digit1[string](),
		)(input)
	}
}

// ipLiteral returns a literal in square brackets. This is technically not spec
// conform as it accept everything which is ":" seperated in square brackets
// https://datatracker.ietf.org/doc/rfc3986
func ipLiteral() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.Sequence(
				gomme.Token[string]("["),
				ipLiteralPart(),
				gomme.Token[string]("]"),
			),
			joinStrings,
		)(input)
	}
}

// ipLiteralPart returns a literal consisting of hex digits and ":"
// https://datatracker.ietf.org/doc/rfc3986
func ipLiteralPart() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.Many1(
				gomme.Alternative(
					gomme.HexDigit1[string](),
					gomme.Token[string](":"),
				),
			),
			joinStrings,
		)(input)
	}
}

// ipv4Address returns an IPv4 address
// https://datatracker.ietf.org/doc/rfc3986
func ipv4Address() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.Count(
				gomme.Map(
					gomme.Sequence(
						gomme.Digit1[string](),
						gomme.Optional(gomme.Token[string](".")),
					),
					joinStrings,
				),
				4),
			joinStrings,
		)(input)
	}
}

// regName returns an URL encoded hostname
// https://datatracker.ietf.org/doc/rfc3986
func regName() gomme.Parser[string, string] {
	return func(input string) gomme.Result[string, string] {
		return gomme.Map(
			gomme.Many0(
				gomme.Alternative(
					uriUnreserved(),
					uriPercentEncoded(),
					uriSubDelims(),
				),
			),
			joinStrings,
		)(input)
	}
}
