// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"fmt"
	"reflect"
	"slices"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"

	"go.yaml.in/yaml/v3"
)

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
	configLoop: for _, entry := range config {
		if len(entryList) >= setting.MAX_FUNDING_ENTRIES_PER_CONFIG {
			errs = append(errs, &ErrTooManyFundingProviders{TotalLimit: setting.MAX_FUNDING_ENTRIES_PER_CONFIG})
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
				if len(entryList) >= setting.MAX_FUNDING_ENTRIES_PER_CONFIG {
					errs = append(errs, &ErrTooManyFundingProviders{TotalLimit: setting.MAX_FUNDING_ENTRIES_PER_CONFIG})
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
