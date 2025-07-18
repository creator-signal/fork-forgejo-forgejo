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
		assert.Equal(t, packageNamespace, collection.CollectionInfo.Namespace)
		assert.Equal(t, packageName, collection.CollectionInfo.Name)
		assert.Equal(t, packageVersion, collection.CollectionInfo.Version)
		assert.Equal(t, packageAuthor, collection.CollectionInfo.Authors[0])
		assert.Equal(t, packageDescription, collection.CollectionInfo.Description)
		assert.Equal(t, packageRepositoryURL, collection.CollectionInfo.Repository)
		assert.Equal(t, ">=2.18.0", collection.CollectionInfo.RequiresAnsible)
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
