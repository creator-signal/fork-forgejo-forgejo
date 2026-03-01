// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package ansible

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	packageName          = "server"
	packageNamespace     = "forgejo"
	packageVersion       = "3.1.4"
	packageAuthor        = "eNBeWe <author@example.com>"
	packageDescription   = "Dummy package metadata for testing"
	packageRepositoryURL = "https://code.forgejo.org/forgejo/forgejo"
)

func TestBuildCollectionFromArchive(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testData := generateTestArchive(generateValidMetadata(), generateValidRuntimeData())
		collection, err := BuildCollectionFromArchive(&testData)
		require.NoError(t, err)
		assert.NotNil(t, collection)
		assert.Equal(t, packageNamespace, collection.Namespace)
		assert.Equal(t, packageName, collection.Name)
		assert.Equal(t, packageVersion, collection.Version)
		assert.Equal(t, packageAuthor, collection.Authors[0])
		assert.Equal(t, packageDescription, collection.Description)
		assert.Equal(t, packageRepositoryURL, collection.Repository)
		assert.Equal(t, ">=2.18.0", collection.RequiresAnsible)
	})
	t.Run("Missing Runtime", func(t *testing.T) {
		testData := generateTestArchive(generateValidMetadata(), "")
		collection, err := BuildCollectionFromArchive(&testData)
		require.ErrorIs(t, err, ErrMissingRuntimeFile)
		assert.Nil(t, collection)
	})
	t.Run("Missing Manifest", func(t *testing.T) {
		testData := generateTestArchive("", generateValidRuntimeData())
		collection, err := BuildCollectionFromArchive(&testData)
		require.ErrorIs(t, err, ErrMissingManifestFile)
		assert.Nil(t, collection)
	})
	t.Run("Missing everything", func(t *testing.T) {
		testData := generateTestArchive("", "")
		collection, err := BuildCollectionFromArchive(&testData)
		require.ErrorIs(t, err, ErrMissingManifestFile)
		assert.Nil(t, collection)
	})
}

func TestVerifyManifestData(t *testing.T) {
	t.Run("Valid minimal data", func(t *testing.T) {
		testData := generateValidManifest()
		err := verifyManifestData(&testData)
		require.NoError(t, err)
	})
	t.Run("Valid full data", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Tags = []string{"valid", "tag", "with_underscores"}
		testData.CollectionInfo.Description = packageDescription
		testData.CollectionInfo.License = []string{"GPLv3"}
		testData.CollectionInfo.Dependencies = map[string]string{"ansible.posix": ">=2.0.0"}
		testData.CollectionInfo.Repository = packageRepositoryURL
		testData.CollectionInfo.Documentation = packageRepositoryURL
		testData.CollectionInfo.Homepage = packageRepositoryURL
		testData.CollectionInfo.Issues = packageRepositoryURL
		err := verifyManifestData(&testData)
		require.NoError(t, err)
	})
	t.Run("Missing Namespace", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Namespace = ""
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid Namespace", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Namespace = "_leading_underscore"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Missing Name", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Name = ""
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid Name", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Name = "Uppercase_Name"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Missing Version", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Version = ""
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid Version", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Namespace = "1.2"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Missing Readme", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Readme = ""
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid Tags", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Tags = []string{
			"Invalid",
		}
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid Tags", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Tags = []string{
			"Invalid",
		}
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid repository URL", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Repository = "invalid"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid documentation URL", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Documentation = "invalid"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid homepage URL", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Homepage = "invalid"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
	t.Run("Invalid issues URL", func(t *testing.T) {
		testData := generateValidManifest()
		testData.CollectionInfo.Issues = "invalid"
		err := verifyManifestData(&testData)
		require.Error(t, err)
	})
}

func TestVerifyRuntimeData(t *testing.T) {
	t.Run("Valid single version", func(t *testing.T) {
		testData := RawRuntimeData{
			RequiresAnsible: "1.2.3",
		}
		err := verifyRuntimeData(&testData)
		require.NoError(t, err)
	})
	t.Run("Valid minimum version", func(t *testing.T) {
		testData := RawRuntimeData{
			RequiresAnsible: ">=1.2.3",
		}
		err := verifyRuntimeData(&testData)
		require.NoError(t, err)
	})
	t.Run("Multiple versions", func(t *testing.T) {
		testData := RawRuntimeData{
			RequiresAnsible: "1.2.3, 2.3.4, 3.4.5",
		}
		err := verifyRuntimeData(&testData)
		require.NoError(t, err)
	})
	t.Run("Various ranges", func(t *testing.T) {
		testData := RawRuntimeData{
			RequiresAnsible: "<1.2.3, <=1.2.3, ==1.2.3, >=1.2.3, >1.2.3",
		}
		err := verifyRuntimeData(&testData)
		require.NoError(t, err)
	})
	t.Run("Invalid version", func(t *testing.T) {
		testData := RawRuntimeData{
			RequiresAnsible: "abc.2.3",
		}
		err := verifyRuntimeData(&testData)
		require.Error(t, err)
	})
}

func generateTestArchive(metadata, runtime string) bytes.Buffer {
	var testData bytes.Buffer
	testZipWriter := gzip.NewWriter(&testData)
	testTarWriter := tar.NewWriter(testZipWriter)

	if metadata != "" {
		metadataHeader := &tar.Header{
			Name: "MANIFEST.json",
			Mode: 0o600,
			Size: int64(len(metadata)),
		}
		testTarWriter.WriteHeader(metadataHeader)
		testTarWriter.Write([]byte(metadata))
	}

	if runtime != "" {
		runtimeHeader := &tar.Header{
			Name: "meta/runtime.yml",
			Mode: 0o600,
			Size: int64(len(runtime)),
		}
		testTarWriter.WriteHeader(runtimeHeader)
		testTarWriter.Write([]byte(runtime))
	}
	testTarWriter.Close()
	testZipWriter.Close()
	return testData
}

func generateValidRuntimeData() string {
	return `---
requires_ansible: '>=2.18.0'`
}

func generateValidMetadata() string {
	return `{
	"collection_info": {
		"namespace": "` + packageNamespace + `",
		"name": "` + packageName + `",
		"version": "` + packageVersion + `",
		"authors": [
			"` + packageAuthor + `"
		],
		"readme": "README.md",
		"tags": [
			"testing"
		],
		"description": "` + packageDescription + `",
		"license": [
			"MIT"
		],
		"license_file": null,
		"dependencies": {
			"ansible.posix": ">=2.0.0"
		},
		"repository": "` + packageRepositoryURL + `",
		"documentation": null,
		"homepage": null,
		"issues": null
	},
	"file_manifest_file": {
		"name": "FILES.json",
		"ftype": "file",
		"chksum_type": "sha256",
		"chksum_sha256": "fcccbf4b0f68264bc75872d3d9f74beb663a7c4f0420acfe90cc983391e53e7a",
		"format": 1
	},
	"format": 1
}`
}

// Generates a manifest file with all required fields filled with valid data
func generateValidManifest() RawManifest {
	return RawManifest{
		CollectionInfo: RawManifestCollectionInfo{
			Namespace:     packageNamespace,
			Name:          packageName,
			Version:       packageVersion,
			Authors:       []string{packageAuthor},
			Readme:        "README.md",
			Tags:          []string{},
			Description:   packageDescription,
			License:       []string{},
			LicenseFile:   "",
			Dependencies:  map[string]string{},
			Repository:    "",
			Documentation: "",
			Homepage:      "",
			Issues:        "",
		},
	}
}
