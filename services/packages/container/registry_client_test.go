package container

import (
	"net/http"
	"testing"

	rr_model "forgejo.org/models/remote_registry"
	api "forgejo.org/modules/structs"
	mock_server "forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewClient(t *testing.T) {
	rr := &rr_model.RemoteRegistry{
		Name:        "someRegistry",
		RemoteURL:   "registry.example.com",
		RemoteUser:  "someUser",
		RemoteToken: "someToken",
	}

	crc := NewContainerRegistryClient(rr)
	assert.NotEmpty(t, crc.httpClient)
	assert.NotEmpty(t, crc.remoteRegistry)
}

func Test_PingRegistry(t *testing.T) {
	server := mock_server.MockRegistryServer()
	defer server.Close()

	rrOpts := api.CreateRemoteRegistryOption{
		Name:      "testreg",
		RemoteURL: server.URL,
	}

	rr := rr_model.RemoteRegistry{
		Name:      rrOpts.Name,
		RemoteURL: rrOpts.RemoteURL,
	}

	crc := NewContainerRegistryClient(&rr)
	resp, err := crc.PingRemoteRegistry(t.Context())

	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func Test_AuthenticateRegistry(t *testing.T) {
	server := mock_server.MockRegistryServer()
	defer server.Close()

	rrOpts := api.CreateRemoteRegistryOption{
		Name:      "testreg",
		RemoteURL: server.URL,
	}

	rr := rr_model.RemoteRegistry{
		Name:        rrOpts.Name,
		RemoteURL:   rrOpts.RemoteURL,
		RemoteUser:  "someUser",
		RemoteToken: "someToken",
	}
	crc := NewContainerRegistryClient(&rr)
	resp, err := crc.PingRemoteRegistry(t.Context())
	require.NoError(t, err)

	authResp, err := crc.AuthenticateRemoteRegistry(t.Context(), resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, authResp.StatusCode)
}
