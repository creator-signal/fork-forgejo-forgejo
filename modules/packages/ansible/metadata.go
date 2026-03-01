// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package ansible

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/url"
	"regexp"
	"strings"

	"forgejo.org/modules/json"
	"forgejo.org/modules/util"

	pep440 "github.com/aquasecurity/go-pep440-version"
	"github.com/hashicorp/go-version"
	"go.yaml.in/yaml/v3"
)

var (
	ErrMissingManifestFile = util.NewInvalidArgumentErrorf("MANIFEST.json file is missing")
	ErrMissingRuntimeFile  = util.NewInvalidArgumentErrorf("meta/runtime.yml file is missing")

	reName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

type CollectionInfo struct {
	Namespace       string            `json:"namespace"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Authors         []string          `json:"authors"`
	Readme          string            `json:"readme"`
	Tags            []string          `json:"tags"`
	Description     string            `json:"description"`
	License         []string          `json:"license"`
	LicenseFile     string            `json:"license_file"`
	Dependencies    map[string]string `json:"dependencies"`
	Repository      string            `json:"repository"`
	Documentation   string            `json:"documentation"`
	Homepage        string            `json:"homepage"`
	Issues          string            `json:"issues"`
	RequiresAnsible string            `json:"requires_ansible"`
}

type RawManifest struct {
	CollectionInfo   RawManifestCollectionInfo   `json:"collection_info"`
	FileManifestFile RawManifestFileManifestFile `json:"file_manifest_file"`
	Format           int                         `json:"format"`
}

type RawManifestCollectionInfo struct {
	Namespace     string            `json:"namespace"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Authors       []string          `json:"authors"`
	Readme        string            `json:"readme"`
	Tags          []string          `json:"tags"`
	Description   string            `json:"description"`
	License       []string          `json:"license"`
	LicenseFile   string            `json:"license_file"`
	Dependencies  map[string]string `json:"dependencies"`
	Repository    string            `json:"repository"`
	Documentation string            `json:"documentation"`
	Homepage      string            `json:"homepage"`
	Issues        string            `json:"issues"`
}

type RawManifestFileManifestFile struct {
	Name         string `json:"name"`
	Ftype        string `json:"ftype"`
	ChecksumType string `json:"chksum_type"`
	Checksum     string `json:"chksum_sha256"`
	Format       int    `json:"format"`
}

type RawRuntimeData struct {
	RequiresAnsible string `yaml:"requires_ansible"`
}

// ParseChartArchive parses the metadata of a Helm archive
func BuildCollectionFromArchive(r io.Reader) (*CollectionInfo, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, util.NewSilentWrapErrorf(err, "Error creating Gzip reader for collection data")
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)

	var manifestData []byte
	var runtimeData []byte

	for {
		// Get the next file from tar archive
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, util.NewSilentWrapErrorf(err, "Problem reading collection tar archive")
		}
		// Skip directories
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Gather the metadata from manifest file. This is pretty much everything, except ansible version.
		if strings.HasSuffix(header.FileInfo().Name(), "MANIFEST.json") {
			manifestData, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, util.NewSilentWrapErrorf(err, "Problem reading MANIFEST.json")
			}
		}
		// Grab the required ansible version from the meta/runtime.yml file
		if strings.HasSuffix(header.Name, "meta/runtime.yml") {
			runtimeData, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, util.NewSilentWrapErrorf(err, "Problem reading meta/runtime.yml")
			}
		}

		if manifestData != nil && runtimeData != nil {
			break
		}
	}

	if manifestData == nil {
		return nil, ErrMissingManifestFile
	}
	if runtimeData == nil {
		return nil, ErrMissingRuntimeFile
	}

	var rawManifest *RawManifest
	err = json.Unmarshal(manifestData, &rawManifest)
	if err != nil {
		return nil, util.NewSilentWrapErrorf(err, "Problem unmarshalling MANIFEST.json")
	}
	err = verifyManifestData(rawManifest)
	if err != nil {
		return nil, util.NewSilentWrapErrorf(err, "Parsing data from MANIFEST.json failed")
	}

	var rawRuntimeData *RawRuntimeData
	err = yaml.Unmarshal(runtimeData, &rawRuntimeData)
	if err != nil {
		return nil, util.NewSilentWrapErrorf(err, "Problem unmarshalling runtime.yml")
	}
	err = verifyRuntimeData(rawRuntimeData)
	if err != nil {
		return nil, util.NewSilentWrapErrorf(err, "Parsing data from meta/runtime.yml failed")
	}

	var metadata *CollectionInfo = assembleCollectionManifest(rawManifest, rawRuntimeData)
	return metadata, nil
}

// Verify the data of the collection according to rules given in
// https://docs.ansible.com/projects/ansible/latest/dev_guide/collections_galaxy_meta.html
func verifyManifestData(data *RawManifest) error {
	if !reName.MatchString(data.CollectionInfo.Name) {
		return util.NewInvalidArgumentErrorf("Invalid collection name")
	}
	if !reName.MatchString(data.CollectionInfo.Namespace) {
		return util.NewInvalidArgumentErrorf("Invalid collection namespace")
	}
	_, err := version.NewVersion(data.CollectionInfo.Version)
	if err != nil {
		return err
	}
	for _, tag := range data.CollectionInfo.Tags {
		if !reName.MatchString(tag) {
			return util.NewInvalidArgumentErrorf("Invalid tag name")
		}
	}
	if len(data.CollectionInfo.Authors) == 0 {
		return util.NewInvalidArgumentErrorf("Missing author information")
	}
	for _, author := range data.CollectionInfo.Authors {
		if len(author) == 0 {
			return util.NewInvalidArgumentErrorf("Empty author name")
		}
	}
	if len(data.CollectionInfo.Readme) == 0 {
		return util.NewInvalidArgumentErrorf("Missing readme path")
	}
	if len(data.CollectionInfo.Repository) > 0 {
		_, err := url.ParseRequestURI(data.CollectionInfo.Repository)
		if err != nil {
			return err
		}
	}
	if len(data.CollectionInfo.Documentation) > 0 {
		_, err := url.ParseRequestURI(data.CollectionInfo.Documentation)
		if err != nil {
			return err
		}
	}
	if len(data.CollectionInfo.Homepage) > 0 {
		_, err := url.ParseRequestURI(data.CollectionInfo.Homepage)
		if err != nil {
			return err
		}
	}
	if len(data.CollectionInfo.Issues) > 0 {
		_, err := url.ParseRequestURI(data.CollectionInfo.Issues)
		if err != nil {
			return err
		}
	}
	return nil
}

// Verifies the used data in the meta/runtime.yml file
func verifyRuntimeData(data *RawRuntimeData) error {
	_, err := pep440.NewSpecifiers(data.RequiresAnsible)
	if err != nil {
		return err
	}
	return nil
}

func assembleCollectionManifest(manifest *RawManifest, runtime *RawRuntimeData) *CollectionInfo {
	return &CollectionInfo{
		Namespace:       manifest.CollectionInfo.Namespace,
		Name:            manifest.CollectionInfo.Name,
		Version:         manifest.CollectionInfo.Version,
		Authors:         manifest.CollectionInfo.Authors,
		Readme:          manifest.CollectionInfo.Readme,
		Tags:            manifest.CollectionInfo.Tags,
		Description:     manifest.CollectionInfo.Description,
		License:         manifest.CollectionInfo.License,
		LicenseFile:     manifest.CollectionInfo.LicenseFile,
		Dependencies:    manifest.CollectionInfo.Dependencies,
		Repository:      manifest.CollectionInfo.Repository,
		Documentation:   manifest.CollectionInfo.Documentation,
		Homepage:        manifest.CollectionInfo.Homepage,
		Issues:          manifest.CollectionInfo.Issues,
		RequiresAnsible: runtime.RequiresAnsible,
	}
}
