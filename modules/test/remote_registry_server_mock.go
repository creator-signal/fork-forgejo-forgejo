package test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
)

func MockForgejoRegistryServer() *httptest.Server {
	registryRoute := http.NewServeMux()
	srv := httptest.NewUnstartedServer(registryRoute)
	addr := srv.Listener.Addr()
	manifestDigest := "sha256:4f10484d1c1bb13e3956b4de1cd42db8e0f14a75be1617b60f2de3cd59c803c6"
	manifestContent := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d","size":1069},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","digest":"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4","size":32}]}`
	blobDigest := "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	blobContent, _ := base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)
	configDigest := "sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d"
	configContent := `{"architecture":"amd64","config":{"Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/true"],"ArgsEscaped":true,"Image":"sha256:9bd8b88dc68b80cffe126cc820e4b52c6e558eb3b37680bfee8e5f3ed7b8c257"},"container":"b89fe92a887d55c0961f02bdfbfd8ac3ddf66167db374770d2d9e9fab3311510","container_config":{"Hostname":"b89fe92a887d","Env":["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],"Cmd":["/bin/sh","-c","#(nop) ","CMD [\"/true\"]"],"ArgsEscaped":true,"Image":"sha256:9bd8b88dc68b80cffe126cc820e4b52c6e558eb3b37680bfee8e5f3ed7b8c257"},"created":"2022-01-01T00:00:00.000000000Z","docker_version":"20.10.12","history":[{"created":"2022-01-01T00:00:00.000000000Z","created_by":"/bin/sh -c #(nop) COPY file:0e7589b0c800daaf6fa460d2677101e4676dd9491980210cb345480e513f3602 in /true "},{"created":"2022-01-01T00:00:00.000000001Z","created_by":"/bin/sh -c #(nop)  CMD [\"/true\"]","empty_layer":true}],"os":"linux","rootfs":{"type":"layers","diff_ids":["sha256:0ff3b91bdf21ecdf2f2f3d4372c2098a14dbe06cd678e8f0a85fd4902d00e2e2"]}}`

	unauthorizedContent := []byte(`{"errors":[{"code":"UNAUTHORIZED","message":""}]}`)

	registryRoute.HandleFunc("/v2/",
		func(res http.ResponseWriter, req *http.Request) {
			authHeader := req.Header.Get("Authorization")
			if strings.Contains(authHeader, "Bearer") {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				res.Header().Add("content-type", "application/json")
				headerVal := "Bearer realm=" +
					"\"http://" + addr.String() + "/token\"" +
					",service=" + "\"container_registry\"" + ",scope=" + "\"*\""
				res.Header().Add("www-authenticate", headerVal)
				res.WriteHeader(http.StatusOK)
			} else {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				res.Header().Add("content-type", "application/json")
				headerVal := "Bearer realm=" +
					"\"http://" + addr.String() + "/token\"" +
					",service=" + "\"container_registry\"" + ",scope=" + "\"*\""
				res.Header().Add("www-authenticate", headerVal)
				_, _ = res.Write(unauthorizedContent)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/token",
		func(res http.ResponseWriter, req *http.Request) {
			hasScope := strings.Contains(req.URL.RawQuery, "scope=")
			hasService := strings.Contains(req.URL.RawQuery, "service=")
			if hasScope && hasService {
				res.Header().Add("content-type", "application/json")
				_, _ = res.Write([]byte(`{"token":"asd342adsdf34985udfng"}`))
			} else {
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}/manifests/{reference}",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") == "Bearer asd342adsdf34985udfng" {
				switch req.Method {
				case "HEAD":
					res.Header().Add("content-type", "application/vnd.docker.distribution.manifest.v2+json")
					res.Header().Add("docker-content-digest", manifestDigest)
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Header().Add("etag", manifestDigest)
					res.WriteHeader(http.StatusOK)
				case "GET":
					res.Header().Add("content-type", "application/vnd.docker.distribution.manifest.v2+json")
					res.Header().Add("docker-content-digest", manifestDigest)
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Header().Add("etag", manifestDigest)
					_, _ = res.Write([]byte(manifestContent))
					res.WriteHeader(http.StatusOK)
				default:
					res.WriteHeader(http.StatusBadRequest)
				}
			} else {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				res.Header().Add("content-type", "application/json")
				headerVal := "Bearer realm=" +
					"\"http://" + addr.String() + "/token\"" +
					",service=" + "\"container_registry\"" + ",scope=" + "\"*\""
				res.Header().Add("www-authenticate", headerVal)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}/blobs/{digest}",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") == "Bearer asd342adsdf34985udfng" {
				switch req.Method {
				case "HEAD":
					if strings.Contains(req.URL.Path, configDigest) {
						res.Header().Add("content-type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
						res.Header().Add("docker-content-digest", configDigest)
						res.Header().Add("docker-distribution-api-version", "registry/2.0")
						res.Header().Add("etag", configDigest)
						res.WriteHeader(http.StatusOK)
					} else {
						res.Header().Add("content-type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
						res.Header().Add("docker-content-digest", blobDigest)
						res.Header().Add("docker-distribution-api-version", "registry/2.0")
						res.Header().Add("etag", blobDigest)
						res.WriteHeader(http.StatusOK)
					}
				case "GET":
					if strings.Contains(req.URL.Path, configDigest) {
						res.Header().Add("content-type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
						res.Header().Add("docker-content-digest", configDigest)
						res.Header().Add("docker-distribution-api-version", "registry/2.0")
						res.Header().Add("etag", configDigest)
						_, _ = res.Write([]byte(configContent))
						res.WriteHeader(http.StatusOK)
					} else {
						res.Header().Add("content-type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
						res.Header().Add("docker-distribution-api-version", "registry/2.0")
						res.Header().Add("docker-content-digest", blobDigest)
						res.Header().Add("etag", blobDigest)
						_, _ = res.Write(blobContent)
						res.WriteHeader(http.StatusOK)
					}
				default:
					res.WriteHeader(http.StatusBadRequest)
				}
			} else {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				res.Header().Add("content-type", "application/json")
				headerVal := "Bearer realm=" +
					"\"http://" + addr.String() + "/token\"" +
					",service=" + "\"container_registry\"" + ",scope=" + "\"*\""
				res.Header().Add("www-authenticate", headerVal)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}/tags/list",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "" {
				switch req.Method {
				case "GET":
					manifestContent := `{"name":"docker-test-org/forgejo-test","tags":["latest"]}`
					res.Header().Add("content-type", "application/json")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					_, _ = res.Write([]byte(manifestContent))
				default:
					res.WriteHeader(http.StatusNotFound)
				}
			} else {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				res.Header().Add("content-type", "application/json")
				headerVal := "Bearer realm=" +
					"\"http://" + addr.String() + "/token\"" +
					",service=" + "\"container_registry\"" + ",scope=" + "\"*\""
				res.Header().Add("www-authenticate", headerVal)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	srv.Start()
	return srv
}
