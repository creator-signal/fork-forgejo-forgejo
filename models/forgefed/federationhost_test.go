// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/federation_key"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FederationHostValidation(t *testing.T) {
	sut := FederationHost{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederationHost{
		HostFqdn: "",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid: HostFqdn empty")
	}

	sut = FederationHost{
		HostFqdn: strings.Repeat("fill", 64),
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid: HostFqdn too long (len=256)")
	}

	sut = FederationHost{
		HostFqdn:       "host.do.main",
		NodeInfo:       NodeInfo{},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid: NodeInfo invalid")
	}

	sut = FederationHost{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now().Add(1 * time.Hour),
		HostPort:       443,
		HostSchema:     "https",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid: Future timestamp")
	}

	sut = FederationHost{
		HostFqdn: "hOst.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid: HostFqdn lower case")
	}
}

func Test_FederationHostKeyIDValidation(t *testing.T) {
	sut := FederationHost{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}

	res, err := validation.IsValid(sut)
	assert.True(t, res)
	require.NoError(t, err)

	hostURL := sut.AsURL()
	keyID, err := federation_key.NewKeyID(fmt.Sprintf("%v#main-key", hostURL.String()))
	require.NoError(t, err)

	keyURL, err := keyID.IRI().URL()
	require.NoError(t, err)

	assert.Equal(t, keyURL.Scheme, hostURL.Scheme)
	assert.Equal(t, keyURL.Host, hostURL.Host)

	require.NoError(t, sut.ValidateKeyID(keyID))
}

func Test_FederationHostInvalidKeyIDValidation(t *testing.T) {
	sut := FederationHost{
		HostFqdn: "host.do.main",
		NodeInfo: NodeInfo{
			SoftwareName: "forgejo",
		},
		LatestActivity: time.Now(),
		HostPort:       443,
		HostSchema:     "https",
	}

	res, err := validation.IsValid(sut)
	assert.True(t, res)
	require.NoError(t, err)

	hostURL := sut.AsURL()

	badSchemeKeyID, err := federation_key.NewKeyID(fmt.Sprintf("http://%v#main-key", hostURL.Host))
	require.NoError(t, err)

	require.Error(t, sut.ValidateKeyID(badSchemeKeyID))

	badHostKeyID, err := federation_key.NewKeyID(fmt.Sprintf("%v://bad.host#main-key", hostURL.Scheme))
	require.NoError(t, err)

	require.Error(t, sut.ValidateKeyID(badHostKeyID))
}
