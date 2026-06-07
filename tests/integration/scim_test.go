package integration

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSCIMToken = "test-scim-bearer-token"

type scimUserResponse struct {
	ID          string `json:"id"`
	ExternalID  string `json:"externalId"`
	UserName    string `json:"userName"`
	DisplayName string `json:"displayName"`
	Active      bool   `json:"active"`
	Emails      []struct {
		Value   string `json:"value"`
		Primary bool   `json:"primary"`
	} `json:"emails"`
}

type scimListResponse struct {
	TotalResults int                `json:"totalResults"`
	Resources    []scimUserResponse `json:"Resources"`
}

func addSCIMAuthSource(t *testing.T, name string) *auth_model.Source {
	t.Helper()
	return addAuthSource(t, map[string]string{
		"type":                fmt.Sprintf("%d", auth_model.OAuth2),
		"name":                name,
		"is_active":           "on",
		"oauth2_provider":     "gitlab",
		"oauth2_scim_enabled": "on",
		"oauth2_scim_token":   testSCIMToken,
	})
}

func newSCIMRequestWithBody(t *testing.T, method, url, token string, body any) *RequestWrapper {
	t.Helper()
	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err)
	req := NewRequestWithBody(t, method, url, bytes.NewBuffer(jsonBytes)).
		SetHeader("Content-Type", "application/scim+json")
	if token != "" {
		req = req.SetHeader("Authorization", "Bearer "+token)
	}
	return req
}

func newSCIMRequest(t *testing.T, method, url, token string) *RequestWrapper {
	t.Helper()
	req := NewRequest(t, method, url)
	if token != "" {
		req = req.SetHeader("Authorization", "Bearer "+token)
	}
	return req
}

func scimCreateUserBody(userName, externalID, email string, active bool) map[string]any {
	return map[string]any{
		"schemas":    []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName":   userName,
		"externalId": externalID,
		"emails":     []map[string]any{{"value": email, "primary": true}},
		"active":     active,
	}
}

func TestSCIM_UnknownProviderReturns404(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	MakeRequest(t, NewRequest(t, "GET", "/api/scim/nonexistent-provider/v2/Users"), http.StatusNotFound)
}

func TestSCIM_MissingTokenReturns401(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-auth-missing")

	MakeRequest(t, NewRequest(t, "GET", "/api/scim/scim-auth-missing/v2/Users"), http.StatusUnauthorized)
}

func TestSCIM_WrongTokenReturns401(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-auth-wrong")

	MakeRequest(t, newSCIMRequest(t, "GET", "/api/scim/scim-auth-wrong/v2/Users", "wrong-token"), http.StatusUnauthorized)
}

func TestSCIM_CreateUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-create")

	body := scimCreateUserBody("scimcreateuser", "ext-create-001", "scimcreateuser@example.com", true)
	resp := MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-create/v2/Users", testSCIMToken, body), http.StatusCreated)

	var created scimUserResponse
	DecodeJSON(t, resp, &created)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "scimcreateuser", created.UserName)
	assert.Equal(t, "ext-create-001", created.ExternalID)
	assert.True(t, created.Active)
	require.Len(t, created.Emails, 1)
	assert.Equal(t, "scimcreateuser@example.com", created.Emails[0].Value)
}

func TestSCIM_CreateUserMissingExternalIDReturns400(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-val-extid")

	body := map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "scimvaluser",
		"emails":   []map[string]any{{"value": "val@example.com", "primary": true}},
	}
	MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-val-extid/v2/Users", testSCIMToken, body), http.StatusBadRequest)
}

func TestSCIM_CreateDuplicateUserUpserts(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-dup")

	body := scimCreateUserBody("scimdupuser", "ext-dup-001", "scimdupuser@example.com", true)
	MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-dup/v2/Users", testSCIMToken, body), http.StatusCreated)
	MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-dup/v2/Users", testSCIMToken, body), http.StatusOK)
}

func TestSCIM_GetUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-get")

	body := scimCreateUserBody("scimgetuser", "ext-get-001", "scimgetuser@example.com", true)
	createResp := MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-get/v2/Users", testSCIMToken, body), http.StatusCreated)
	var created scimUserResponse
	DecodeJSON(t, createResp, &created)

	resp := MakeRequest(t, newSCIMRequest(t, "GET", "/api/scim/scim-get/v2/Users/"+created.ID, testSCIMToken), http.StatusOK)
	var fetched scimUserResponse
	DecodeJSON(t, resp, &fetched)
	assert.Equal(t, created.ID, fetched.ID)
	assert.Equal(t, "scimgetuser", fetched.UserName)
	assert.Equal(t, "ext-get-001", fetched.ExternalID)
}

func TestSCIM_GetNonExistentUserReturns404(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-notfound")

	MakeRequest(t, newSCIMRequest(t, "GET", "/api/scim/scim-notfound/v2/Users/99999", testSCIMToken), http.StatusNotFound)
}

func TestSCIM_ListUsers(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-list")

	body := scimCreateUserBody("scimlistuser", "ext-list-001", "scimlistuser@example.com", true)
	MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-list/v2/Users", testSCIMToken, body), http.StatusCreated)

	resp := MakeRequest(t, newSCIMRequest(t, "GET", "/api/scim/scim-list/v2/Users", testSCIMToken), http.StatusOK)
	var list scimListResponse
	DecodeJSON(t, resp, &list)
	assert.Equal(t, 1, list.TotalResults)
	require.Len(t, list.Resources, 1)
	assert.Equal(t, "scimlistuser", list.Resources[0].UserName)
}

func TestSCIM_ReplaceUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-replace")

	createBody := scimCreateUserBody("scimreplaceuser", "ext-replace-001", "replace@example.com", true)
	createResp := MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-replace/v2/Users", testSCIMToken, createBody), http.StatusCreated)
	var created scimUserResponse
	DecodeJSON(t, createResp, &created)

	replaceBody := map[string]any{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName":    "scimreplaceuser",
		"externalId":  "ext-replace-001",
		"displayName": "Replace User",
		"emails":      []map[string]any{{"value": "replace-new@example.com", "primary": true}},
		"active":      false,
	}
	resp := MakeRequest(t, newSCIMRequestWithBody(t, "PUT", "/api/scim/scim-replace/v2/Users/"+created.ID, testSCIMToken, replaceBody), http.StatusOK)
	var updated scimUserResponse
	DecodeJSON(t, resp, &updated)
	assert.Equal(t, "Replace User", updated.DisplayName)
	assert.False(t, updated.Active)

	u := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "scimreplaceuser"})
	assert.False(t, u.IsActive)
	assert.True(t, u.ProhibitLogin)
}

func TestSCIM_CrossSourceIsolation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-source-a")
	addSCIMAuthSource(t, "scim-source-b")

	body := scimCreateUserBody("scimisolationuser", "ext-isolation-001", "scimisolationuser@example.com", true)
	createResp := MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-source-a/v2/Users", testSCIMToken, body), http.StatusCreated)
	var created scimUserResponse
	DecodeJSON(t, createResp, &created)

	// User belongs to source A — source B must not be able to see or delete it.
	MakeRequest(t, newSCIMRequest(t, "GET", "/api/scim/scim-source-b/v2/Users/"+created.ID, testSCIMToken), http.StatusNotFound)
	MakeRequest(t, newSCIMRequest(t, "DELETE", "/api/scim/scim-source-b/v2/Users/"+created.ID, testSCIMToken), http.StatusNotFound)
}

func TestSCIM_DeleteUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	addSCIMAuthSource(t, "scim-delete")

	body := scimCreateUserBody("scimdeleteuser", "ext-delete-001", "scimdeleteuser@example.com", true)
	createResp := MakeRequest(t, newSCIMRequestWithBody(t, "POST", "/api/scim/scim-delete/v2/Users", testSCIMToken, body), http.StatusCreated)
	var created scimUserResponse
	DecodeJSON(t, createResp, &created)

	MakeRequest(t, newSCIMRequest(t, "DELETE", "/api/scim/scim-delete/v2/Users/"+created.ID, testSCIMToken), http.StatusNoContent)

	MakeRequest(t, newSCIMRequest(t, "GET", "/api/scim/scim-delete/v2/Users/"+created.ID, testSCIMToken), http.StatusNotFound)
}
