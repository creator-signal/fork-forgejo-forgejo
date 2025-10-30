// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"strings"
	"testing"

	"forgejo.org/modules/json"
	"forgejo.org/tests"

	swagger_spec "github.com/go-openapi/spec"
	"github.com/stretchr/testify/require"
)

func getSwagger(t *testing.T) *swagger_spec.Swagger {
	t.Helper()

	resp := MakeRequest(t, NewRequest(t, "GET", "/swagger.v1.json"), http.StatusOK)

	swagger := new(swagger_spec.Swagger)

	decoder := json.NewDecoder(resp.Body)
	require.NoError(t, decoder.Decode(swagger))

	return swagger
}

func checkSwaggerMethodResponse(t *testing.T, path string, method *swagger_spec.Operation, name string, statusCode int, responseType string) {
	t.Helper()

	if method == nil {
		return
	}

	val, ok := method.Responses.StatusCodeResponses[statusCode]
	if !ok {
		t.Errorf("%s %s is missing response status code %d in swagger", name, path, statusCode)
		return
	}

	if responseType != val.Ref.String() {
		t.Errorf("%s %s has %s response type for %d in swagger (expected %s)", name, path, val.Ref.String(), statusCode, responseType)
	}
}

func checkSwaggerPathResponse(t *testing.T, paths map[string]swagger_spec.PathItem, pathMatch string, statusCode int, responseType string) {
	t.Helper()

	for pathName, pathData := range paths {
		if pathName != pathMatch {
			continue
		}

		checkSwaggerMethodResponse(t, pathName, pathData.Get, "GET", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Put, "PUT", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Post, "POST", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Patch, "PATCH", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Delete, "DELETE", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Options, "OPTIONS", statusCode, responseType)

		return
	}
}

func checkSwaggerRouteResponse(t *testing.T, paths map[string]swagger_spec.PathItem, prefix string, statusCode int, responseType string) {
	t.Helper()

	for pathName, pathData := range paths {
		if !strings.HasPrefix(pathName, prefix) {
			continue
		}

		checkSwaggerMethodResponse(t, pathName, pathData.Get, "GET", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Put, "PUT", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Post, "POST", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Patch, "PATCH", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Delete, "DELETE", statusCode, responseType)
		checkSwaggerMethodResponse(t, pathName, pathData.Options, "OPTIONS", statusCode, responseType)
	}
}

func TestSwaggerUserRoute(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	swagger := getSwagger(t)

	checkSwaggerPathResponse(t, swagger.Paths.Paths, "/user", http.StatusUnauthorized, "#/responses/unauthorized")
	checkSwaggerRouteResponse(t, swagger.Paths.Paths, "/user/", http.StatusUnauthorized, "#/responses/unauthorized")
}

func TestSwaggerUsersRoute(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	swagger := getSwagger(t)

	checkSwaggerRouteResponse(t, swagger.Paths.Paths, "/users/{username}", http.StatusNotFound, "#/responses/notFound")
}

func TestSwaggerPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	swagger := getSwagger(t)

	allowList := map[string]struct{} {
		"repoGetWikiPageRevisions": {},
	};

	for _, pathData := range swagger.Paths.Paths {
		if pathData.Get == nil {
			continue
		}

		if _, allowed := allowList[pathData.Get.ID]; allowed {
			continue
		}

		var hasPageParam bool
		var hasLimitParam bool
		for i := range pathData.Get.Parameters {
			parameter := pathData.Get.Parameters[i]
			if parameter.Name == "page" {
				hasPageParam = true;
			} else if parameter.Name == "limit" {
				hasLimitParam = true;
			}
		}

		var hasTotalCountHeader bool
		var hasLinkHeader bool
		for responseCode, responseMaybe := range pathData.Get.Responses.StatusCodeResponses {
			var response swagger_spec.Response
			if len(responseMaybe.Ref.String()) > 0 {
				res, err := swagger_spec.ResolveResponse(swagger, responseMaybe.Ref)
				if err != nil {
					t.Errorf("failed to resolve response reference [%s]", err.Error())
				}
				response = *res
			} else {
				response = responseMaybe
			}
			if responseCode < 200 || responseCode >= 300 {
				continue
			}
			for headerName, _ := range response.Headers {
				if headerName == "X-Total-Count" {
					hasTotalCountHeader = true;
				} else if headerName == "Link" {
					hasLinkHeader = true;
				}
			}
		}

		hasAny := hasPageParam || hasLimitParam || hasLinkHeader || hasTotalCountHeader
		hasAll := hasPageParam && hasLimitParam && hasLinkHeader && hasTotalCountHeader
		if hasAny && !hasAll {
			t.Errorf("%s\t has some pagination fields but not all: page (%t) limit (%t) Link (%t) X-Total-Count (%t)", pathData.Get.ID, hasPageParam, hasLimitParam, hasLinkHeader, hasTotalCountHeader)
		}
	}
}
