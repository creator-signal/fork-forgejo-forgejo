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
	"strings"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"

	"go.yaml.in/yaml/v3"
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

type RawRepoFundingConfigEntry struct {
	Key   string
	Value any
}

// A funding config consists of unique key-value pairs, considered in order.
// Each key corresponds to at least one value, which should be either a bare
// string or a list of strings, but the parser doesn't have to worry about the
// actual type; that's the validator's job.
type RawRepoFundingConfig []RawRepoFundingConfigEntry

// called by `yaml.Unmarshal` when decoding file data
func (c *RawRepoFundingConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("Expected YAML mapping, got %v", value.Kind)
	}

	// in a mapping, Content contains pairs of nodes, so we iterate two at a time
	for i := 0; i < len(value.Content); i += 2 {
		// peeking ahead at the value, store it if it's there
		var entryData any
		if err := value.Content[i+1].Decode(&entryData); err != nil {
			return err // not sure what we hit that won't fit into `any`, but it wasn't good :/
		}

		// since we have a value, grab the key
		key := value.Content[i].Value
		for _, alreadyEntry := range *c {
			if alreadyEntry.Key == key {
				return fmt.Errorf("Duplicate YAML key: %s", key)
			}
		}

		// record the pair
		*c = append(*c, RawRepoFundingConfigEntry{Key: key, Value: entryData})
	}

	return nil
}

type EntriesAndLineErrors struct {
	EntryList []*api.RepoFundingEntry
	Errs      []error
}

// Parses the given file data for funding entries. Fails if the data could not
// be understood for some reason.
func GetFundingFromBlob(content []byte) (*EntriesAndLineErrors, error) {
	config := make(RawRepoFundingConfig, 0)
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	entryList := make([]*api.RepoFundingEntry, 0)
	var errs []error

	// no need to sort these, they'll come in the order they were given
	for _, entry := range config {
		providerName := entry.Key
		entryData := entry.Value
		provider := setting.GetFundingProviderByName(providerName)
		if provider == nil {
			errs = append(errs, &ErrUnknownFundingProvider{Name: providerName})
			continue
		}

		if entryData == nil {
			errs = append(errs, &ErrInvalidYamlType{Name: providerName})
			continue
		}

		dataType := reflect.TypeOf(entryData)
		switch dataType.Kind() {
		case reflect.String:
			if provider.Limit == 0 {
				// 1 is too many! this provider is disabled.
				errs = append(errs, &ErrTooManyOfFundingProvider{Name: providerName, Limit: provider.Limit})
				continue
			}
			newEntry, err := getFundingEntry(provider, entryData.(string))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			entryList = append(entryList, newEntry)

		case reflect.Slice:
			// no need to sort these either, they'll come in the order they were given
			stringSlice := reflect.ValueOf(entryData)
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

	return &EntriesAndLineErrors{entryList, errs}, nil
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

	data, err := GetFundingFromBlob(configContent)
	if err != nil {
		return nil, err
	}

	funding := &RepoFunding{data.EntryList, configPath, data.Errs}
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
