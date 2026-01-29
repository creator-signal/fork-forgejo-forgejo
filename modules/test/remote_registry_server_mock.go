package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
)

func MockRegistryServer() *httptest.Server {
	registryRoute := http.NewServeMux()

	srv := httptest.NewUnstartedServer(registryRoute)
	addr := srv.Listener.Addr()

	registryRoute.HandleFunc("/v2/",
		func(res http.ResponseWriter, req *http.Request) {
			authHeader := req.Header.Get("Authorization")
			if strings.Contains(authHeader, "Bearer") {
				res.WriteHeader(http.StatusOK)
			} else {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				headerVal := "Bearer realm=" + "\"http://" + addr.String() + "/auth\"" + ",service=" + "\"" + addr.String() + "\""
				res.Header().Add("www-authenticate", headerVal)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/auth",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "" {
				res.WriteHeader(http.StatusOK)
			} else {
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/v2/{name}/manifests/{reference}",
		func(res http.ResponseWriter, r *http.Request) {
			res.WriteHeader(http.StatusOK)
		})

	srv.Start()
	return srv
}
