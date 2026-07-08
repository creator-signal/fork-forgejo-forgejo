// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"reflect"
	"slices"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"

	"go.yaml.in/yaml/v3"
)

// Parses the given file data for funding entries. Fails if the data could not
// be understood as a funding config documenet.
func getFundingFromBlob(content []byte) (entries []*api.RepoFundingEntry, lineErrors []error, parseErr error) {
	config := make(rawRepoFundingConfig, 0)
	if parseErr := yaml.Unmarshal(content, &config); parseErr != nil {
		return nil, nil, parseErr
	}

	entryList := make([]*api.RepoFundingEntry, 0)
	var errs []error

	// no need to sort these, they'll come in the order they were given
configLoop:
	for _, entry := range config {
		if len(entryList) >= setting.MaxFundingEntriesPerConfig {
			errs = append(errs, &ErrTooManyFundingProviders{TotalLimit: setting.MaxFundingEntriesPerConfig})
			break configLoop // we've reached our limit; no point checking further, even for parse errors (there may be many!)
		}

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
				if len(entryList) >= setting.MaxFundingEntriesPerConfig {
					errs = append(errs, &ErrTooManyFundingProviders{TotalLimit: setting.MaxFundingEntriesPerConfig})
					break configLoop // we've reached our limit; no point checking further, even for parse errors (there may be many!)
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
					return e.Value == newEntry.Value
				}) {
					errs = append(errs, &ErrDuplicateFundingEntry{Name: providerName, Value: newEntry.Value})
					continue
				}
				entryList = append(entryList, newEntry)
			}
		default:
			errs = append(errs, &ErrInvalidYamlType{Name: providerName})
			continue
		}
	}

	return entryList, errs, nil
}
