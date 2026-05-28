// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding_test

import (
	"testing"

	funding_service "forgejo.org/services/funding"

	"github.com/stretchr/testify/assert"
)

func TestIsFundingConfig(t *testing.T) {
	assert.True(t, funding_service.IsFundingConfig(".forgejo/FUNDING.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/FUNDING.yml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/Funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/Funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/fundING.yml"))

	assert.True(t, funding_service.IsFundingConfig(".github/FUNDING.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".github/FUNDING.yml"))
	assert.True(t, funding_service.IsFundingConfig(".github/Funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".github/Funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".github/funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".github/funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".github/fundING.yml"))

	assert.True(t, funding_service.IsFundingConfig("FUNDING.yaml"))
	assert.True(t, funding_service.IsFundingConfig("FUNDING.yml"))
	assert.True(t, funding_service.IsFundingConfig("Funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig("Funding.yml"))
	assert.True(t, funding_service.IsFundingConfig("funding.yml"))
	assert.True(t, funding_service.IsFundingConfig("funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig("fundING.yml"))

	assert.False(t, funding_service.IsFundingConfig("README.md"))
	assert.False(t, funding_service.IsFundingConfig(".gitea/FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig("custom/FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig(".forgejo/_FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig(".forgejo/.FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig(".forgejo/FUNDING.yml."))
}
