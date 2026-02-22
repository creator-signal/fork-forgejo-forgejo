# Ansible Package Registry

This document explains the implementation of the Ansible Package Registry.
The main focus is the documentation of the API behaviour as well as the assumptions/simplifications that have been taken.

## General Implementation

The Pacakge Registry only implements the API v3, since that is the default behaviour
of Galaxy-NG and is compatible with all current ansible versions.
The Ansible-NG API documentation at (https://docs.ansible.com/projects/galaxy-ng/en/latest/community/api_v3.html)
documents the Pulp-based API of -NG, but doesn't match the URLs that the actual Galaxy client queries.
Instead the client queries some "simpler" URLs and is then redirected in the NG implementation to the documented endpoints.
This implementation implements the requested endpoints directly, without any redirections.

## Client behaviour and API documentation

*Naming/Path assumption for these examples:*

* *The forge is running at `https://forge.example.org`*
* *The `ansible-galaxy` client is run with the parameter `-s https://forge.example.org/api/packages/exampleOrg/ansible`*
* *The requested collection is using the name `exampleNs.exampleName`*

### 1. Initial connection

**Client Behaviour:** Whenever the `ansible-galaxy` CLI performs a request against the registry, it initially connects to the "plain" registry url,
so the first request goes directly to `https://forge.example.org/api/packages/exampleOrg/ansible`.

**Observed Response:** Galaxy responds with HTTP 200 at `https://galaxy.ansible.com`, but it is not accessed.
Client source code shows that it expects a API discovery JSON here.

**Implementation:** We respond with the JSON body `{"available_versions":{"v3":"v3/"}}`. (see func `AvailableApis`)

### 2. Collection metadata

**Client Behaviour:** When requesting a collection from the registry the CLI requests collection metadata from the endpoint `/v3/collections/exampleNs/exampleName/`.

**Observed Response:** Galaxy responds with JSON, containing metadata, including references to all available versions and the highest version.

    {
        "href": "/api/v3/plugin/ansible/content/published/collections/index/exampleNs/exampleName",
        "namespace": "exampleNs",
        "name": "exampleName",
        "deprecated": false,
        "versions_url": "/api/v3/plugin/ansible/content/published/collections/index/exampleNs/exampleName/versions/",
        "highest_version": {
            "href": "/api/v3/plugin/ansible/content/published/collections/index/exampleName/versions/1.1.1/",
            "version": "1.1.1"
        },
        "created_at": "2025-11-03T10:04:55.839006Z",
        "updated_at": "2026-01-28T10:18:51.998800Z",
        "download_count": 1439
    }

**Implementation:** We respond with a reduced JSON, containing the versioning information, but no timestamps or download count. (see func `CollectionMetadata`)

    {
        "href": "/api/packages/exampleOrg/ansible/v3/collections/exampleNs/exampleName",
        "namespace": "exampleNs",
        "name": "exampleName",
        "deprecated": false,
        "versions_url": "/api/packages/exampleOrg/ansible/v3/collections/exampleNs/exampleName/versions/",
        "highest_version": {
            "href": "/api/packages/exampleOrg/ansible/v3/collections/exampleNs/exampleName/versions/1.1.1/",
            "version": "1.1.1"
        },
    }

### 3. Collection version listing

**Client Behaviour:** The client requests the `versions_url` of the collection for a listing of known versions.
It requests the given URL with the parameter `?limit=100`.

**Observed Response:** Galaxy replies with JSON containing the number of versions as well as metadata for each version.

    {
        "meta": {
            "count": 2
        },
        "links": {
            "first": "/api/v3/plugin/ansible/content/published/collections/index/exampleNs/exampleName/versions/?limit=10&offset=0",
            "previous": null,
            "next": null,
            "last": "/api/v3/plugin/ansible/content/published/collections/index/exampleNs/exampleName/versions/?limit=10&offset=0"
        },
        "data": [
            {
                "version": "1.1.0",
                "href": "/api/v3/plugin/ansible/content/published/collections/index/exampleNs/exampleName/versions/1.1.0/",
                "created_at": "2026-01-08T16:04:58.634253Z",
                "updated_at": "2026-01-08T16:04:58.669356Z",
                "requires_ansible": ">=2.15.0",
                "marks": []
            },
            {
                "version": "1.0.0",
                "href": "/api/v3/plugin/ansible/content/published/collections/index/exampleNs/exampleName/versions/1.0.0/",
                "created_at": "2025-11-03T10:04:55.871205Z",
                "updated_at": "2025-11-03T10:04:55.983712Z",
                "requires_ansible": ">=2.15.0",
                "marks": []
            }
        ]
    }

**Implementation:** We replicate the response from Galaxy, with simplified pagination links. (see func `ListVersions`)

    {
        "meta": {
            "count": 2
        },
        "links": {
            "next": null
        },
        "data": [
            {
                "version": "1.1.0",
                "href": "/api/packages/exampleOrg/ansible/v3/collections/exampleNs/exampleName/versions/1.1.0/",
                "created_at": "2026-01-08T16:04:58.634253Z",
                "updated_at": "2026-01-08T16:04:58.669356Z",
                "requires_ansible": ">=2.15.0",
                "marks": []
            },
            {
                "version": "1.0.0",
                "href": "/api/packages/exampleOrg/ansible/v3/collections/exampleNs/exampleName/versions/1.0.0/",
                "created_at": "2025-11-03T10:04:55.871205Z",
                "updated_at": "2025-11-03T10:04:55.983712Z",
                "requires_ansible": ">=2.15.0",
                "marks": []
            }
        ]
    }

### 4. Specific version data

**Client Behaviour:** The client queries the `href` of the desired version. After receiving this data it uses the data for the actual file transfer.

**Observed Response:** Galaxy responds with the full metadata of the collection, including the full manifest and a complete file listing of the files contained in the collection.

**Implementation:** We construct a reduced response, containing links to the namespace and collection, as well as links to the downloadable artifact (see func `ServeCollection`).

**Limitation:** The Galaxy response contains hashes for the artifacts as well as parts of the metadata. We don't provide these hashes and they don't seem to be checked by the CLI.

### 5. Upload collection

**Client Behaviour:** The client POSTs the collection to the endpoint `/v3/artifacts/collections`.

**Observed Response:** Galaxy receives the collection and responds with a URL to an "import task".
This is used to process the collection asynchronously and then notify the client when the process is finished.

**Implementation:** We directly handle the collection upon upload and store the metadata. (see func `UploadCollection`)
We respond with a "dummy" task URI, that can be queried by the client (see func `ImportResult`).

## Test scenarios

### 1. Uploading a collection

**Requirements:** 
* `ansible-galaxy` installed to command line
* Source code of the valid collection (`exampleNs.exampleName`in this example)
* Personal access token for write access to `exampleOrg` organization in Forgejo.

**Process:**
Execute these commands in the directory of the collection:
* `ansible-galaxy collection build` - This compiles the collection and creates the archive `exampleNs-exampleName-version.tar.gz`.
* `ansible-galaxy collection publish -s https://forge.example.org/api/packages/exampleOrg/ansible --token <Personal-Access-Token> exampleNs-exampleName-version.tar.gz` - The archive is uploaded to the registy.

**Expected result:**
The collection is uploaded to the organization and available for download.

**Debugging options:**
* If the second command is amended with `-vvvvv`, the debug output can be analyzed for errors.

### 2. Install a collection from command line

**Requirements:** 
* `ansible-galaxy` installed to command line
* Collection `exampleNs.exampleName` uploaded to the package registry

**Process:**
Execute the command `ansible-galaxy collection install -s https://forge.example.org/api/packages/exampleOrg/ansible exampleNs.exampleName`

**Expected result:**
The collection is downloaded from the registry and installed to the configured directory (default: `~/.ansible/collections`)

**Notes:**
* The client CLI caches a lot of requests. This cache data is stored in folders below `~/.ansible`.

### 3. Install a collection from requirements file

**Requirements:** 
* `ansible-galaxy` installed to command line
* `requirements.yml` file with the following content

```
---
collections:
- name: 'exampleNs.exampleName'
    version: '1.0.0'
    source: 'https://forge.example.org/api/packages/exampleOrg/ansible'
```

**Process:**
Execute the command `ansible-galaxy collection install -r requirements.yml`

**Expected result:**
The collection is downloaded from the registry and installed to the configured directory (default: `~/.ansible/collections`)
