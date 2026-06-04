// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"fmt"
	"regexp"

	repo_model "forgejo.org/models/repo"
)

// ErrFundingNotExist occurs when a repo has no funding config.
type ErrFundingNotExist struct {
	Repo *repo_model.Repository
}

func (err ErrFundingNotExist) Error() string {
	return fmt.Sprintf("No funding config found in repo %s/%s", err.Repo.OwnerName, err.Repo.Name)
}

// IsErrFundingNotExist returns `true` if the error is an `ErrFundingNotExist`.
func IsErrFundingNotExist(err error) bool {
	_, ok := err.(ErrFundingNotExist)
	return ok
}

// ErrUnknownFundingProvider occurs when a funding config contains an unknown
// funding provider name.
type ErrUnknownFundingProvider struct {
	Name string
}

func (err ErrUnknownFundingProvider) Error() string {
	return fmt.Sprintf("Unknown funding provider: %s", err.Name)
}

// ErrTooManyOfFundingProvider occurs when a funding config contains more
// values for a funding provider than expected.
type ErrTooManyOfFundingProvider struct {
	Name  string
	Limit uint
}

func (err ErrTooManyOfFundingProvider) Error() string {
	if err.Limit == 0 {
		return fmt.Sprintf("Funding provider %s is not allowed", err.Name)
	}
	return fmt.Sprintf("Expected up to %d of funding provider %s", err.Limit, err.Name)
}

// ErrDuplicateFundingEntry occurs when a funding config contains a provider
// with duplicate entries.
type ErrDuplicateFundingEntry struct {
	Name string
	URL  string
}

func (err ErrDuplicateFundingEntry) Error() string {
	return fmt.Sprintf("Duplicate entry for key '%s': %s", err.Name, err.URL)
}

// ErrBadInput represents a failure to match the input string against the regex
// pattern.
type ErrBadInput struct {
	Name    string
	Pattern *regexp.Regexp
}

func (err ErrBadInput) Error() string {
	return fmt.Sprintf("Value for key '%s' does not match pattern /%s/", err.Name, err.Pattern.String())
}

// ErrCannotParseURL represents a failure to parse a funding entry URL.
type ErrCannotParseURL struct {
	Name string
	Err  error
}

func (err ErrCannotParseURL) Error() string {
	return fmt.Sprintf("Invalid URL value for key '%s': %v", err.Name, err.Err.Error())
}

// ErrInvalidYamlType occurs when a funding config is misshaped.
type ErrInvalidYamlType struct {
	Name string
}

func (err ErrInvalidYamlType) Error() string {
	return fmt.Sprintf("Invalid type for key '%s', expected a string or string array", err.Name)
}
