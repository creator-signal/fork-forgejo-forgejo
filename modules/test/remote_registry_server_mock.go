package test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
)

var (
	indexManifestDigest      = "sha256:a45c5523b043174617cac9bda33134956c14e9add96fcacca1da36278aadbaba"
	indexManifestContent     = `{"manifests":[{"annotations":{"com.docker.official-images.bashbrew.arch":"amd64","org.opencontainers.image.base.digest":"sha256:346fa035ca82052ce8ec3ddb9df460b255507acdeb1dc880a8b6930e778a553c","org.opencontainers.image.base.name":"debian:trixie-slim","org.opencontainers.image.created":"2026-02-04T23:52:22Z","org.opencontainers.image.revision":"ffe72978e08c5b0dacecd604e528f6d0741a9ae5","org.opencontainers.image.source":"https:\/\/github.com\/nginx\/docker-nginx.git#ffe72978e08c5b0dacecd604e528f6d0741a9ae5:mainline\/debian","org.opencontainers.image.url":"https:\/\/hub.docker.com\/_\/nginx","org.opencontainers.image.version":"1.29.5"},"digest":"sha256:514a9c2814250e61396ef4d6125ece1a8fbb3b0964a2ab441e9f7acf0b66b8b5","mediaType":"application\/vnd.oci.image.manifest.v1+json","platform":{"architecture":"amd64","os":"linux"},"size":2290},{"annotations":{"com.docker.official-images.bashbrew.arch":"amd64","vnd.docker.reference.digest":"sha256:514a9c2814250e61396ef4d6125ece1a8fbb3b0964a2ab441e9f7acf0b66b8b5","vnd.docker.reference.type":"attestation-manifest"},"digest":"sha256:32923807439461f47e92b606f5fe670b1791b407c62a6b4648b38f7659c034be","mediaType":"application\/vnd.oci.image.manifest.v1+json","platform":{"architecture":"unknown","os":"unknown"},"size":841}],"mediaType":"application\/vnd.oci.image.index.v1+json","schemaVersion":2}`
	indexManifestContentType = "application/vnd.oci.image.index.v1+json" // "application/vnd.docker.distribution.manifest.list.v2+json"
	manifestDigest           = "sha256:4f10484d1c1bb13e3956b4de1cd42db8e0f14a75be1617b60f2de3cd59c803c6"
	manifestContent          = `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d","size":1069},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","digest":"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4","size":32}]}`
	manifestContentType      = "application/vnd.docker.distribution.manifest.v2+json"
	blobDigest               = "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	blobContent, _           = base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)
	blobContentType          = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	configDigest             = "sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d"
	configContent            = `{"architecture":"amd64","config":{"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/true"],"ArgsEscaped":true,"Image":"sha256:9bd8b88dc68b80cffe126cc820e4b52c6e558eb3b37680bfee8e5f3ed7b8c257"},"container":"b89fe92a887d55c0961f02bdfbfd8ac3ddf66167db374770d2d9e9fab3311510","container_config":{"Hostname":"b89fe92a887d","Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/true\"]"],"ArgsEscaped":true,"Image":"sha256:9bd8b88dc68b80cffe126cc820e4b52c6e558eb3b37680bfee8e5f3ed7b8c257"},"created":"2022-01-01T00:00:00.000000000Z","docker_version":"20.10.12","history":[{"created":"2022-01-01T00:00:00.000000000Z","created_by":"/bin/sh -c #(nop) COPY file:0e7589b0c800daaf6fa460d2677101e4676dd9491980210cb345480e513f3602 in /true "},{"created":"2022-01-01T00:00:00.000000001Z","created_by":"/bin/sh -c #(nop)  CMD [\"/true\"]","empty_layer":true}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:0ff3b91bdf21ecdf2f2f3d4372c2098a14dbe06cd678e8f0a85fd4902d00e2e2"]}}`
	configContentType        = "application/vnd.docker.container.image.v1+json"
	dockerAPIVersion         = "registry/2.0"
)

func MockForgejoRegistryServer() *httptest.Server {
	registryRoute := http.NewServeMux()
	srv := httptest.NewUnstartedServer(registryRoute)
	addr := srv.Listener.Addr()
	authHeader := "Bearer realm=" +
		"\"http://" + addr.String() + "/token\"" +
		",service=" + "\"container_registry\"" + ",scope=" + "\"*\""

	registryRoute.HandleFunc("/v2/",
		func(res http.ResponseWriter, req *http.Request) {
			authHeader := req.Header.Get("Authorization")
			if strings.Contains(authHeader, "Bearer") {
				setAuthHeader(res, authHeader)
				res.WriteHeader(http.StatusOK)
			} else {
				setAuthHeader(res, authHeader)
				_, _ = res.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED","message":""}]}`))
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/token",
		func(res http.ResponseWriter, req *http.Request) {
			hasScope := strings.Contains(req.URL.RawQuery, "scope=")
			hasService := strings.Contains(req.URL.RawQuery, "service=")
			hasBasic := strings.Contains(req.Header.Get("Authorization"), "Basic")
			if hasScope && hasService || hasBasic {
				res.Header().Add("content-type", "application/json")
				_, _ = res.Write([]byte(`{"token":"asd342adsdf34985udfng"}`))
			} else {
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}/manifests/{reference}",
		func(res http.ResponseWriter, req *http.Request) {
			ref := req.PathValue("reference")
			if req.Header.Get("Authorization") == "Bearer asd342adsdf34985udfng" {
				switch decideManifestResponse(req, ref) {
				case "HEADindex":
					setResponseHeader(res, indexManifestContentType, indexManifestDigest)
					res.WriteHeader(http.StatusOK)
				case "GETindex":
					setResponseHeader(res, indexManifestContentType, indexManifestDigest)
					_, _ = res.Write([]byte(indexManifestContent))
					res.WriteHeader(http.StatusOK)
				case "HEADmanifest":
					setResponseHeader(res, manifestContentType, manifestDigest)
					res.WriteHeader(http.StatusOK)
				case "GETmanifest":
					setResponseHeader(res, manifestContentType, manifestDigest)
					_, _ = res.Write([]byte(manifestContent))
					res.WriteHeader(http.StatusOK)
				default:
					res.WriteHeader(http.StatusNotFound)
				}
			} else {
				setAuthHeader(res, authHeader)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}/blobs/{digest}",
		func(res http.ResponseWriter, req *http.Request) {
			dig := req.PathValue("digest")
			if req.Header.Get("Authorization") == "Bearer asd342adsdf34985udfng" {
				switch decideBlobResponse(req, dig) {
				case "HEADconfig":
					setResponseHeader(res, configContentType, configDigest)
					res.WriteHeader(http.StatusOK)
				case "GETconfig":
					setResponseHeader(res, configContentType, configDigest)
					_, _ = res.Write([]byte(configContent))
					res.WriteHeader(http.StatusOK)
				case "HEADblob":
					setResponseHeader(res, blobContentType, blobDigest)
					res.WriteHeader(http.StatusOK)
				case "GETblob":
					setResponseHeader(res, blobContentType, blobDigest)
					_, _ = res.Write(blobContent)
					res.WriteHeader(http.StatusOK)
				default:
					res.WriteHeader(http.StatusNotFound)
				}
			} else {
				setAuthHeader(res, authHeader)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}/tags/list",
		func(res http.ResponseWriter, req *http.Request) {
			img := req.PathValue("image")
			content := `{"name":"` + img + `","tags":["latest"]}`
			if req.Header.Get("Authorization") != "Bearer asd342adsdf34985udfng" {
				switch req.Method {
				case "GET":
					res.Header().Add("content-type", "application/json")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					_, _ = res.Write([]byte(content))
				default:
					res.WriteHeader(http.StatusNotFound)
				}
			} else {
				setAuthHeader(res, authHeader)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	srv.Start()
	return srv
}

func decideManifestResponse(req *http.Request, ref string) string {
	res := ""
	switch req.Method {
	case "HEAD":
		switch ref {
		case indexManifestDigest:
			res = "HEADindex"
		case manifestDigest, "latest": // latest or manifest digest
			res = "HEADmanifest"
		}
	case "GET":
		switch ref {
		case indexManifestDigest:
			res = "GETindex"
		case manifestDigest, "latest":
			res = "GETmanifest"
		}
	}
	return res
}

func decideBlobResponse(req *http.Request, ref string) string {
	res := ""
	switch req.Method {
	case "HEAD":
		switch ref {
		case configDigest:
			res = "HEADconfig"
		case blobDigest: // only blob digest
			res = "HEADblob"
		}
	case "GET":
		switch ref {
		case configDigest:
			res = "GETconfig"
		case blobDigest:
			res = "GETblob"
		}
	}
	return res
}

func setAuthHeader(res http.ResponseWriter, authHeaderVal string) {
	res.Header().Add("docker-distribution-api-version", dockerAPIVersion)
	res.Header().Add("content-type", "application/json")
	res.Header().Add("www-authenticate", authHeaderVal)
}

func setResponseHeader(res http.ResponseWriter, vals ...string) {
	ct := vals[0]
	dig := vals[1]
	res.Header().Add("content-type", ct)
	res.Header().Add("docker-content-digest", dig)
	res.Header().Add("docker-distribution-api-version", dockerAPIVersion)
	res.Header().Add("etag", dig)
}
