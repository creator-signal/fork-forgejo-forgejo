// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"context"
	"fmt"
	"sort"
	"io"
	"reflect"
	"strings"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"

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

// ErrInvalidFundingProvider represents a "InvalidFundingProvider" kind of error.
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
	entry.Text = fmt.Sprintf(provider.Text, text)
	entry.URL = fmt.Sprintf(provider.URL, text)

	if provider.Icon != "" {
		entry.Icon = setting.AppSubURL + "/assets/" + provider.Icon
	}

	return entry
}

// GetFundingFromPath the given funding file.
// It never returns a nil config.
func GetFundingFromPath(r *repo_model.Repository, path string, commit *git.Commit) ([]*api.RepoFundingEntry, []error) {
	var errs []error

	treeEntry, err := commit.GetTreeEntryByFoldedPath(path)
	if err != nil {
		return nil, []error{err}
	}

	reader, err := treeEntry.Blob().DataAsync()
	if err != nil {
		log.Error("DataAsync: failed to read blob for funding config due to error: %v", err)
		return nil, []error{}
	}

	defer reader.Close()

	configContent, err := io.ReadAll(reader)
	if err != nil {
		return nil, []error{err}
	}

	fundingMap := make(map[string]any)
	if err := yaml.Unmarshal(configContent, &fundingMap); err != nil {
		return nil, []error{err}
	}

	// Sort keys so we return a consistent order
	fundingKeys := make([]string, 0, len(fundingMap))
	for key := range fundingMap {
		fundingKeys = append(fundingKeys, key)
	}
	sort.Strings(fundingKeys) // TODO: This works for now, but consider a stricter order based on config later on

	entryList := make([]*api.RepoFundingEntry, 0)
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
			entryList = append(entryList, getFundingEntry(provider, fundingData.(string)))
		case reflect.Slice:
			// no need to sort these, they'll come in the same order as they were given
			// FIXME: only custom should support a list, and then only up to 4
			stringSlice := reflect.ValueOf(fundingData)
			for i := 0; i < stringSlice.Len(); i++ {
				str, ok := stringSlice.Index(i).Interface().(string)
				if !ok {
					errs = append(errs, ErrInvalidYamlType{Name: providerName})
					continue
				}
				entryList = append(entryList, getFundingEntry(provider, str))
			}
		default:
			errs = append(errs, ErrInvalidYamlType{Name: providerName})
			continue
		}
	}

	return entryList, errs
}

func GetFundingFromCommit(r *repo_model.Repository, commit *git.Commit) ([]*api.RepoFundingEntry, []error) {
	for _, configName := range fundingCandidates {
		if _, err := commit.GetTreeEntryByFoldedPath(configName); err == nil {
			return GetFundingFromPath(r, configName, commit)
		}
	}

	return nil, []error{}
}

// GetFundingFromDefaultBranch returns the funding for this repo.
// It never returns a nil config.
func GetFundingFromDefaultBranch(ctx context.Context, r *repo_model.Repository) ([]*api.RepoFundingEntry, []error) {
	if r.IsEmpty {
		return nil, []error{}
	}

	gitRepo, err := git.OpenRepository(ctx, r.RepoPath())
	if err != nil {
		return nil, []error{err}
	}
	defer gitRepo.Close()

	commit, err := gitRepo.GetBranchCommit(r.DefaultBranch)
	if err != nil {
		return nil, []error{err}
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
