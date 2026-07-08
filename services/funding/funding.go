// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/idna"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
)

// Funding config files are considered in this order. When a file is found
// matching one of these (case-insensitive) paths, it is treated as the config
// and others are ignored.
//
// If that config is invalid, the other candidates are still ignored. This is
// because a funding file in one of the more specific .forgejo or .github
// directories is more likely to have intentional meaning than one at the
// directory root, so users would probably expect this degree of strictness.
var fundingCandidates = []string{
	".forgejo/FUNDING.yaml",
	".forgejo/FUNDING.yml",
	".github/FUNDING.yaml",
	".github/FUNDING.yml",
	"FUNDING.yaml",
	"FUNDING.yml",
}

// Transforms unicode in the given URL's domain name into its punycode
// representation.
//
// See also https://en.wikipedia.org/wiki/Internationalized_domain_name
func toASCII(url *url.URL) (*url.URL, error) {
	port := url.Port()

	// Punycode!
	hostname, err := idna.ToASCII(url.Hostname())
	if err != nil {
		return nil, err
	}

	// Domain names are always lowercase
	hostname = strings.ToLower(hostname)

	// url.Hostname() removes brackets, so we replace them if the host had them before
	if strings.HasPrefix(url.Host, "[") {
		url.Host = "[" + hostname + "]"
	} else {
		url.Host = hostname
	}
	if port != "" {
		url.Host += ":" + port
	}

	return url, nil
}

// Constructs a funding entry from the known funding provider config and the
// user-provided `text`.
func getFundingEntry(provider *setting.FundingProviderConfig, input string) (*api.RepoFundingEntry, error) {
	input = strings.TrimSpace(input)

	if !provider.InputPattern.Match([]byte(input)) {
		return nil, &ErrBadInput{Name: provider.Name, Pattern: provider.InputPattern}
	}

	urlText := fmt.Sprintf(provider.URL, input)
	if !strings.Contains(urlText, "://") {
		// assume HTTP before parsing (otherwise, url.Parse may think the *hostname* is the scheme!)
		urlText = "http://" + urlText
	}

	urlValue, err := url.Parse(urlText) // value should parse as a URL, just in case interpolation got us something invalid
	if err != nil {
		return nil, &ErrCannotParseURL{Name: provider.Name, Err: err}
	}

	// TODO: Look into whether this should also respect setting.Service.ValidSiteURLSchemes
	validSchemes := []string{"http", "https"}
	if !slices.Contains(validSchemes, urlValue.Scheme) {
		return nil, &ErrCannotParseURL{Name: provider.Name, Err: &ErrBadURLScheme{
			ValidSchemes: validSchemes,
			GivenScheme: urlValue.Scheme,
		}}
	}

	urlValue, err = toASCII(urlValue)
	if err != nil {
		return nil, &ErrCannotParseURL{Name: provider.Name, Err: err}
	}

	entry := new(api.RepoFundingEntry)
	entry.ProviderName = provider.Name
	entry.Text = fmt.Sprintf(provider.Text, input)
	entry.URL = urlValue.String()

	return entry, nil
}

type RepoFunding struct {
	// Funding options for the repository
	Entries []*api.RepoFundingEntry

	// The navigable path to the repository's funding config file
	ConfigPath string

	// Parsing issues, if any, from parsing the repository's funding config
	Errors []error
}

// GetFundingFromPath parses a funding config from the file at the given `path`
// in the given commit on the repository.
func GetFundingFromPath(r *repo_model.Repository, path string, commit *git.Commit) (*RepoFunding, error) {
	var err error

	treeEntry, err := commit.GetTreeEntryByFoldedPath(path)
	if err != nil {
		return nil, err
	}

	configPath, err := treeEntry.Path()
	if err != nil {
		return nil, err
	}

	reader, err := treeEntry.Blob().DataAsync()
	if err != nil {
		log.Error("DataAsync: failed to read blob for funding config due to error: %v", err)
		return nil, err
	}
	defer reader.Close()

	configContent, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	configPath = fmt.Sprintf("/%s/src/branch/%s/%s", util.PathEscapeSegments(r.FullName()), util.PathEscapeSegments(r.DefaultBranch), configPath)

	data, lineErrors, err := getFundingFromBlob(configContent)
	if err != nil {
		return nil, err
	}

	funding := &RepoFunding{
		Entries: data,
		ConfigPath: configPath,
		Errors: lineErrors,
	}
	return funding, nil
}

func GetFundingFromCommit(r *repo_model.Repository, commit *git.Commit) (*RepoFunding, error) {
	for _, configName := range fundingCandidates {
		if _, err := commit.GetTreeEntryByFoldedPath(configName); err == nil {
			return GetFundingFromPath(r, configName, commit)
		}
	}

	return nil, ErrFundingNotExist{Repo: r}
}

// GetFundingFromDefaultBranch returns the funding for this repo.
func GetFundingFromDefaultBranch(ctx context.Context, r *repo_model.Repository) (*RepoFunding, error) {
	if r.IsEmpty {
		return nil, ErrFundingNotExist{Repo: r}
	}

	gitRepo, err := git.OpenRepository(ctx, r.RepoPath())
	if err != nil {
		return nil, err
	}
	defer gitRepo.Close()

	commit, err := gitRepo.GetBranchCommit(r.DefaultBranch)
	if err != nil {
		return nil, err
	}

	return GetFundingFromCommit(r, commit)
}
