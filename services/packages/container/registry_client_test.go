package container

import (
	"net/http"
	"testing"

	remote_registry_model "forgejo.org/models/remote_registry"
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
	rr := &remote_registry_model.RemoteRegistry{
		Name:       regName,
		OwnerID:    int64(2),
		RemoteURL:  remoteURL,
		RemoteHost: remoteHost,
		RemoteUser: remoteUser,
	}
	rr.SetRemoteToken(remoteToken)
	rr.SetRemotePassword(remotePassword)

	crc, err := NewContainerRegistryClient(rr, "test/image", "latest")
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
	assert.Equal(t, remoteHost+"/test/image:latest", crc.Reference.Reference)

	_, err = NewContainerRegistryClient(rr, "library/nginx", "sha256:341bf0f3ce6c5277d6002cf6e1fb0319fa4252add24ab6a0e262e0056d313208")
	require.NoError(t, err)
}

func Test_NewRef(t *testing.T) {
	rr := &remote_registry_model.RemoteRegistry{
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

	rr := remote_registry_model.RemoteRegistry{
		Name:      regName,
		RemoteURL: server.URL,
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

	rr := remote_registry_model.RemoteRegistry{
		Name:       regName,
		OwnerID:    int64(2),
		RemoteURL:  server.URL,
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

	rr := remote_registry_model.RemoteRegistry{
		Name:        regName,
		RemoteURL:   server.URL,
		RemoteUser:  "",
		RemoteToken: []byte{},
	}
	crc, err := NewContainerRegistryClient(&rr)
	require.NoError(t, err)
	connected, err := crc.RemoteRegistryConnected(t.Context())
	require.NoError(t, err)

	assert.True(t, connected)
}
