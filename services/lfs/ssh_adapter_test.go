package lfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCapability(t *testing.T) {
	assert.Equal(t, "version=1\n", string(GetCapabilityAdvertisement()))
}

func TestCheckVersionCommand(t *testing.T) {
	assert.False(t, CheckVersionCommand([]byte("version 2\n")))
	assert.True(t, CheckVersionCommand([]byte("version 1\n")))
}
