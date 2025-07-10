package ansible

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"strings"

	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"

	"gopkg.in/yaml.v3"
)

var (
	ErrMissingManifestFile = util.NewInvalidArgumentErrorf("MANIFEST.json file is missing")
	ErrMissingRuntimeFile  = util.NewInvalidArgumentErrorf("meta/runtime.yml file is missing")
)

type CollectionManifest struct {
	CollectionInfo   CollectionInfo   `json:"collection_info"`
	FileManifestFile FileManifestFile `json:"file_manifest_file"`
	Format           int              `json:"format"`
}

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
	RequiresAnsible string            `json:"requires_ansible" yaml:"requires_ansible"`
}

type FileManifestFile struct {
	Name         string `json:"name"`
	Ftype        string `json:"ftype"`
	ChecksumType string `json:"chksum_type"`
	Checksum     string `json:"chksum_sha256"`
	Format       int    `json:"format"`
}

// ParseChartArchive parses the metadata of a Helm archive
func BuildCollectionFromArchive(r io.Reader) (*CollectionManifest, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		// Skip directories
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Gather the metadata from manifest file. This is pretty much everything, except ansible version.
		if strings.HasSuffix(header.FileInfo().Name(), "MANIFEST.json") {
			manifestData, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
		}
		// Grab the required ansible version from the meta/runtime.yml file
		if strings.HasSuffix(header.Name, "meta/runtime.yml") {
			runtimeData, err = io.ReadAll(tarReader)
			if err != nil {
				return nil, err
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

	log.Info("Building collection from gathered metadata")
	var metadata *CollectionManifest
	err = json.Unmarshal(manifestData, &metadata)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(runtimeData, &(metadata.CollectionInfo))
	if err != nil {
		return nil, err
	}
	return metadata, nil
}
