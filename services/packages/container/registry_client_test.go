package container

import (
	"testing"

	rr_model "forgejo.org/models/remote_registry"
	api "forgejo.org/modules/structs"
	mock_server "forgejo.org/modules/test"
	"github.com/stretchr/testify/assert"
)

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
	resp, err := PingRemoteRegistry(t.Context(), &rr)

	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}
