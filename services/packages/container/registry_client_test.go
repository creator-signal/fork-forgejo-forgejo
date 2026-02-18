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

const (
	regName        string = "someRegistry"
	remoteUser     string = "someUser"
	remoteURL      string = "https://registry.example.com"
	remoteHost     string = "registry.example.com"
	remotePassword string = "somePassword"
	remoteToken    string = "someToken"
)

func Test_NewClient(t *testing.T) {
	rr := &rr_model.RemoteRegistry{
		Name:       regName,
		OwnerID:    int64(2),
		RemoteURL:  remoteURL,
		RemoteUser: remoteUser,
	}
	rr.SetRemoteToken(remoteToken)
	rr.SetRemotePassword(remotePassword)

	crc, err := NewContainerRegistryClient(rr)
	require.NoError(t, err)
	assert.NotEmpty(t, crc.httpClient)
	assert.NotEmpty(t, crc.RegClient)
	assert.NotEmpty(t, crc.RemoteRegistry)
	assert.NotEqual(t, remotePassword, string(crc.RemoteRegistry.RemotePassword))
	assert.NotEqual(t, remoteToken, string(crc.RemoteRegistry.RemoteToken))

	pass, err := rr.GetRemotePassword()
	require.NoError(t, err)
	token, err := rr.GetRemoteToken()
	require.NoError(t, err)

	assert.Equal(t, remotePassword, pass)
	assert.Equal(t, remoteToken, token)
}

func Test_NewRef(t *testing.T) {
	rr := &rr_model.RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteHost: remoteHost,
		RemotePort: 443,
		RemoteUser: remoteUser,
	}

	imageName := "myorg/test-image:latest"

	crc, err := NewContainerRegistryClient(rr, imageName)
	require.NoError(t, err)

	assert.Equal(t, "registry.example.com/myorg/test-image:latest", crc.Reference.Reference)
}

func Test_PingRegistry(t *testing.T) {
	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rrOpts := api.CreateRemoteRegistryOption{
		Name:      regName,
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
		Name:      regName,
		RemoteURL: server.URL,
	}

	rr := rr_model.RemoteRegistry{
		Name:       rrOpts.Name,
		OwnerID:    int64(2),
		RemoteURL:  rrOpts.RemoteURL,
		RemoteUser: remoteUser,
	}
	rr.SetRemoteToken(remoteToken)
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
		Name:      regName,
		RemoteURL: server.URL,
	}

	rr := rr_model.RemoteRegistry{
		Name:        rrOpts.Name,
		RemoteURL:   rrOpts.RemoteURL,
		RemoteUser:  "",
		RemoteToken: []byte{},
	}
	crc, err := NewContainerRegistryClient(&rr)
	require.NoError(t, err)
	connected, err := crc.RemoteRegistryConnected(t.Context())
	require.NoError(t, err)

	assert.True(t, connected)
}
