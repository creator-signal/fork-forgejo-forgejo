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
				res.Write(unauthorizedContent)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/token",
		func(res http.ResponseWriter, req *http.Request) {
			hasScope := strings.Contains(req.URL.RawQuery, "scope=")
			hasService := strings.Contains(req.URL.RawQuery, "service=")
			if hasScope && hasService {
				res.Header().Add("content-type", "application/json")
				res.Write([]byte(`{"token":"asd342adsdf34985udfng"}`))
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
					res.Header().Add("docker-content-digest", "sha256:4f10484d1c1bb13e3956b4de1cd42db8e0f14a75be1617b60f2de3cd59c803c6")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Header().Add("etag", "sha256:c25049d7428c0e7176de521dea90f8d47a29b7acc2e40b67d557cd79c8c6a92d")
					res.WriteHeader(http.StatusOK)
				case "GET":
					manifestContent := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d","size":1069},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","digest":"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4","size":32}]}`
					res.Header().Add("content-type", "application/vnd.docker.distribution.manifest.v2+json")
					res.Header().Add("docker-content-digest", "sha256:4f10484d1c1bb13e3956b4de1cd42db8e0f14a75be1617b60f2de3cd59c803c6")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Header().Add("etag", "sha256:c25049d7428c0e7176de521dea90f8d47a29b7acc2e40b67d557cd79c8c6a92d")
					res.Write([]byte(manifestContent))
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

	registryRoute.HandleFunc("/v2/{org}/{image}//blobs/{digest}",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") == "Bearer asd342adsdf34985udfng" {
				switch req.Method {
				case "HEAD":
					res.Header().Add("content-type", "application/vnd.docker.image.rootfs.diff.tar.gzip")
					res.Header().Add("docker-content-digest", "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Header().Add("etag", "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4")
					res.WriteHeader(http.StatusOK)
				case "GET":
					blobContent, _ := base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)
					res.Header().Add("content-type", "application/vnd.docker.distribution.manifest.v2+json")
					res.Header().Add("etag", "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4")
					res.Header().Add("docker-content-digest", "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Write([]byte(blobContent))
					res.WriteHeader(http.StatusOK)
				default:
					res.WriteHeader(http.StatusBadRequest)
				}
			} else {
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{org}/{image}//tags/list",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "" {
				switch req.Method {
				case "GET":
					manifestContent := `{"name":"docker-test-org/forgejo-test","tags":["latest"]}`
					res.Header().Add("content-type", "application/json")
					res.Header().Add("docker-distribution-api-version", "registry/2.0")
					res.Write([]byte(manifestContent))
				default:
					res.WriteHeader(http.StatusNotFound)
				}
			} else {
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	srv.Start()
	return srv
}
