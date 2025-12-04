// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"forgejo.org/models/federation_key"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"
)

// FederationHost data type
// swagger:model
type FederationHost struct {
	ID             int64              `xorm:"pk autoincr"`
	HostFqdn       string             `xorm:"host_fqdn UNIQUE(federation_host) INDEX VARCHAR(255) NOT NULL"`
	HostPort       uint16             `xorm:" UNIQUE(federation_host) INDEX NOT NULL DEFAULT 443"`
	NodeInfo       NodeInfo           `xorm:"extends NOT NULL"`
	HostSchema     string             `xorm:"NOT NULL DEFAULT 'https'"`
	LatestActivity time.Time          `xorm:"NOT NULL"`
	PublicKeyID    sql.NullInt64      `xorm:"INDEX UNIQUE REFERENCES(federation_public_key, id)"`
	Created        timeutil.TimeStamp `xorm:"created"`
	Updated        timeutil.TimeStamp `xorm:"updated"`
}

// Factory function for FederationHost. Created struct is asserted to be valid.
func NewFederationHost(hostFqdn string, nodeInfo NodeInfo, port uint16, schema string) (FederationHost, error) {
	result := FederationHost{
		HostFqdn:   strings.ToLower(hostFqdn),
		NodeInfo:   nodeInfo,
		HostPort:   port,
		HostSchema: schema,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederationHost{}, err
	}
	return result, nil
}

func (host FederationHost) AsURL() url.URL {
	return url.URL{
		Scheme: host.HostSchema,
		Host:   fmt.Sprintf("%v:%v", host.HostFqdn, host.HostPort),
	}
}

// Validate collects error strings in a slice and returns this
func (host FederationHost) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(host.HostFqdn, "HostFqdn")...)
	result = append(result, validation.ValidateMaxLen(host.HostFqdn, 255, "HostFqdn")...)
	result = append(result, validation.ValidateNotEmpty(host.HostPort, "HostPort")...)
	result = append(result, validation.ValidateNotEmpty(host.HostSchema, "HostSchema")...)
	result = append(result, host.NodeInfo.Validate()...)
	if host.HostFqdn != strings.ToLower(host.HostFqdn) {
		result = append(result, fmt.Sprintf("HostFqdn has to be lower case but was: %v", host.HostFqdn))
	}
	if !host.LatestActivity.IsZero() && host.LatestActivity.After(time.Now().Add(10*time.Minute)) {
		result = append(result, fmt.Sprintf("Latest Activity cannot be in the far future: %v", host.LatestActivity))
	}

	return result
}

// ValidateKeyID checks that the provided ActivityPub key ID matches the host.
func (host FederationHost) ValidateKeyID(keyID federation_key.KeyID) error {
	keyURL, err := keyID.IRI().URL()
	if err != nil {
		return err
	}

	hostURL := host.AsURL()

	if keyURL.Scheme != hostURL.Scheme || keyURL.Host != hostURL.Host {
		return fmt.Errorf("invalid key ID for host, key URL: %v, host URL: %v", keyURL, hostURL)
	}

	return nil
}
