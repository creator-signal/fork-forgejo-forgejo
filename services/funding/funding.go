// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"context"
	"fmt"
	"io"
	"reflect"
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

func IsErrFundingValidationError(err error) bool {
	return IsErrInvalidFundingProvider(err) || IsErrTooManyOfFundingProvider(err) || IsErrInvalidYamlType(err)
}

// ErrInvalidFundingProvider represents an "UnknownFundingProvider" kind of error.
type ErrInvalidFundingProvider struct {
	Name string
}

// IsErrInvalidFundingProvider checks if an error is a ErrInvalidFundingProvider.
func IsErrInvalidFundingProvider(err error) bool {
	_, ok := err.(ErrInvalidFundingProvider)
	return ok
}

func (err ErrInvalidFundingProvider) Error() string {
	// TODO: make this better
	return fmt.Sprintf("funding provider %s is unknown", err.Name)
}

// ErrTooManyOfFundingProvider represents a "TooManyOfFundingProvider" kind of error.
type ErrTooManyOfFundingProvider struct {
	Name string
	Limit uint
}

func IsErrTooManyOfFundingProvider(err error) bool {
	_, ok := err.(ErrTooManyOfFundingProvider)
	return ok
}

func (err ErrTooManyOfFundingProvider) Error() string {
	if err.Limit == 0 {
		return fmt.Sprintf("Expected exactly 0 of funding provider %s", err.Name)
	} else {
		return fmt.Sprintf("Expected up to %d of funding provider %s", err.Limit, err.Name)
	}
}

// ErrInvalidYamlType represents a "InvalidYamlType" kind of error.
type ErrInvalidYamlType struct {
	Name string
}

// IsErrInvalidYamlType checks if an error is a ErrInvalidYamlType.
func IsErrInvalidYamlType(err error) bool {
	_, ok := err.(ErrInvalidYamlType)
	return ok
}

func (err ErrInvalidYamlType) Error() string {
	// TODO: make this better
	return fmt.Sprintf("%s has a invalid type. Expected string or string array", err.Name)
}

func getFundingEntry(provider *api.FundingProvider, text string) *api.RepoFundingEntry {
	entry := new(api.RepoFundingEntry)
	entry.ProviderName = provider.Name
	entry.Text = fmt.Sprintf(provider.Text, text)
	entry.URL = fmt.Sprintf(provider.URL, text)

	if provider.Icon != "" {
		entry.Icon = setting.AppSubURL + "/assets/" + provider.Icon
	}

	return entry
}

type RepoFunding struct {
	// Funding options for the repository
	Entries    []*api.RepoFundingEntry

	// The navigable path to the repository's funding config file
	ConfigPath string

	// Parsing issues, if any, from parsing the repository's funding config
	Errors     []error
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
		return nil, nil
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
	sort.Strings(fundingKeys) // TODO: This works for now, but consider a stricter order based on config later on

	entryList := make([]*api.RepoFundingEntry, 0)
	var errs []error
	for _, providerName := range fundingKeys {
		fundingData := fundingMap[providerName]
		provider := setting.GetFundingProviderByName(providerName)
		if provider == nil {
			errs = append(errs, ErrInvalidFundingProvider{Name: providerName})
			continue
		}

		dataType := reflect.TypeOf(fundingData)
		switch dataType.Kind() {
		case reflect.String:
			if provider.Limit == 0 {
				// 1 is too many! this provider is disabled.
				errs = append(errs, ErrTooManyOfFundingProvider{Name: providerName, Limit: provider.Limit})
				continue
			}
			entryList = append(entryList, getFundingEntry(provider, fundingData.(string)))
		case reflect.Slice:
			// no need to sort these, they'll come in the same order as they were given
			stringSlice := reflect.ValueOf(fundingData)
			for i := 0; i < stringSlice.Len(); i++ {
				if uint(i) >= provider.Limit {
					errs = append(errs, ErrTooManyOfFundingProvider{Name: providerName, Limit: provider.Limit})
					break // stop here for this provider, we've got enough
				}
				str, ok := stringSlice.Index(i).Interface().(string)
				if !ok {
					errs = append(errs, ErrInvalidYamlType{Name: providerName})
					continue // keep searching this provider, there may be more we want
				}
				entryList = append(entryList, getFundingEntry(provider, str))
			}
		default:
			errs = append(errs, ErrInvalidYamlType{Name: providerName})
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

	return nil, nil
}

// GetFundingFromDefaultBranch returns the funding for this repo.
func GetFundingFromDefaultBranch(ctx context.Context, r *repo_model.Repository) (*RepoFunding, error) {
	if r.IsEmpty {
		return nil, nil
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

// IsFundingConfig returns if the given path is a funding config.
func IsFundingConfig(path string) bool {
	for _, name := range fundingCandidates {
		if strings.EqualFold(path, name) {
			return true
		}
	}
	return false
}
