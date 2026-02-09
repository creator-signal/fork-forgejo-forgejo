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

	crc, err := NewContainerRegistryClient(rr)
	require.NoError(t, err)
	assert.NotEmpty(t, crc.httpClient)
	assert.NotEmpty(t, crc.RegClient)
	assert.NotEmpty(t, crc.RemoteRegistry)
}

func Test_NewRef(t *testing.T) {
	rr := &rr_model.RemoteRegistry{
		Name:        "someRegistry",
		RemoteURL:   "https://registry.example.com",
		RemoteHost:  "registry.example.com",
		RemotePort:  443,
		RemoteUser:  "someUser",
		RemoteToken: "someToken",
	}

	imageName := "myorg/test-image:latest"

	crc, err := NewContainerRegistryClient(rr)
	require.NoError(t, err)

	ref, err := crc.NewRef(imageName)
	require.NoError(t, err)
	assert.Equal(t, "registry.example.com/myorg/test-image:latest", ref.Reference)
}

func Test_PingRegistry(t *testing.T) {
	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rrOpts := api.CreateRemoteRegistryOption{
		Name:      "testreg",
		RemoteURL: server.URL,
	}

	rr := rr_model.RemoteRegistry{
		Name:      rrOpts.Name,
		RemoteURL: rrOpts.RemoteURL,
	}

	crc, err := NewContainerRegistryClient(&rr)
	require.NoError(t, err)
	resp, err := crc.PingRemoteRegistry(t.Context())

	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func Test_AuthenticateRegistry(t *testing.T) {
	server := mock_server.MockForgejoRegistryServer()
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
	crc, err := NewContainerRegistryClient(&rr)
	require.NoError(t, err)
	resp, err := crc.PingRemoteRegistry(t.Context())
	require.NoError(t, err)

	authResp, err := crc.AuthenticateRemoteRegistry(t.Context(), resp)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, authResp.StatusCode)
}

func Test_RemoteRegistryConnectedNoAuthInfo(t *testing.T) {
	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rrOpts := api.CreateRemoteRegistryOption{
		Name:      "testreg",
		RemoteURL: server.URL,
	}

	rr := rr_model.RemoteRegistry{
		Name:        rrOpts.Name,
		RemoteURL:   rrOpts.RemoteURL,
		RemoteUser:  "",
		RemoteToken: "",
	}
	crc, err := NewContainerRegistryClient(&rr)
	require.NoError(t, err)
	connected, err := crc.RemoteRegistryConnected(t.Context())
	require.NoError(t, err)

	assert.True(t, connected)
}
