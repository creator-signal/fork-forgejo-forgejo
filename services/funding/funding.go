// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"

	"go.yaml.in/yaml/v3"
)

var fundingCandidates = []string{
	".forgejo/FUNDING.yaml",
	".forgejo/FUNDING.yml",
	".github/FUNDING.yaml",
	".github/FUNDING.yml",
	"FUNDING.yaml",
	"FUNDING.yml",
}

// ErrFundingNotExist occurs when a repo has no funding config.
type ErrFundingNotExist struct {
	Repo *repo_model.Repository
}

func (err ErrFundingNotExist) Error() string {
	return fmt.Sprintf("No funding config found in repo %s/%s", err.Repo.OwnerName, err.Repo.Name)
}

// IsErrFundingNotExist checks if an error is a ErrFundingNotExist.
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

// ErrCannotParseURL represents a failure to parse an entry URL.
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

// Constructs a funding entry from the known funding provider config and the
// user-provided `text`.
func getFundingEntry(provider *setting.FundingProviderConfig, input string) (*api.RepoFundingEntry, error) {
	input = strings.TrimSpace(input)

	if !provider.InputPattern.Match([]byte(input)) {
		return nil, &ErrBadInput{Name: provider.Name, Pattern: provider.InputPattern}
	}

	urlText := fmt.Sprintf(provider.URL, input)
	urlValue, err := url.Parse(urlText) // value should parse as a URL, just in case interpolation got us something invalid
	if err != nil {
		return nil, &ErrCannotParseURL{Name: provider.Name, Err: err}
	}
	if urlValue.Scheme == "" {
		// assume HTTP
		urlText = "http://" + urlValue.String()
	} else {
		urlText = urlValue.String()
	}

	entry := new(api.RepoFundingEntry)
	entry.ProviderName = provider.Name
	entry.Text = fmt.Sprintf(provider.Text, input)
	entry.URL = urlText
	entry.Icon = setting.IconForProvider(provider)
	entry.IconDark = setting.DarkIconForProvider(provider)

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

// GetFundingFromPath the given funding file.
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

	fundingMap := make(map[string]any)
	if err := yaml.Unmarshal(configContent, &fundingMap); err != nil {
		return nil, err
	}

	// Sort keys so we return a consistent order
	fundingKeys := make([]string, 0, len(fundingMap))
	for key := range fundingMap {
		fundingKeys = append(fundingKeys, key)
	}
	sort.Strings(fundingKeys) // TODO: This works for now, but consider a stricter order based on the funding config later on

	entryList := make([]*api.RepoFundingEntry, 0)
	var errs []error
	for _, providerName := range fundingKeys {
		fundingData := fundingMap[providerName]
		provider := setting.GetFundingProviderByName(providerName)
		if provider == nil {
			errs = append(errs, &ErrUnknownFundingProvider{Name: providerName})
			continue
		}

		dataType := reflect.TypeOf(fundingData)
		switch dataType.Kind() {
		case reflect.String:
			if provider.Limit == 0 {
				// 1 is too many! this provider is disabled.
				errs = append(errs, &ErrTooManyOfFundingProvider{Name: providerName, Limit: provider.Limit})
				continue
			}
			newEntry, err := getFundingEntry(provider, fundingData.(string))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			entryList = append(entryList, newEntry)
		case reflect.Slice:
			// no need to sort these, they'll come in the same order as they were given
			stringSlice := reflect.ValueOf(fundingData)
			for i := 0; i < stringSlice.Len(); i++ {
				if uint(i) >= provider.Limit {
					errs = append(errs, &ErrTooManyOfFundingProvider{Name: providerName, Limit: provider.Limit})
					break // stop here for this provider, we've got enough
				}
				str, ok := stringSlice.Index(i).Interface().(string)
				if !ok {
					errs = append(errs, &ErrInvalidYamlType{Name: providerName})
					continue // keep searching this provider, there may be more we want
				}
				newEntry, err := getFundingEntry(provider, str)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				if slices.ContainsFunc(entryList, func(e *api.RepoFundingEntry) bool {
					return e.URL == newEntry.URL
				}) {
					errs = append(errs, &ErrDuplicateFundingEntry{Name: providerName, URL: newEntry.URL})
					continue
				}
				entryList = append(entryList, newEntry)
			}
		default:
			errs = append(errs, &ErrInvalidYamlType{Name: providerName})
			continue
		}
	}

	configPath = fmt.Sprintf("/%s/src/branch/%s/%s", util.PathEscapeSegments(r.FullName()), util.PathEscapeSegments(r.DefaultBranch), configPath)

	funding := &RepoFunding{Entries: entryList, ConfigPath: configPath, Errors: errs}
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

// IsFundingConfig returns true if the given path is a funding config.
func IsFundingConfig(path string) bool {
	for _, name := range fundingCandidates {
		if strings.EqualFold(path, name) {
			return true
		}
	}
	return false
}
