// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migrations

import (
	"crypto/tls"
	"net/http"

	"forgejo.org/modules/hostmatcher"
	"forgejo.org/modules/httplib"
	"forgejo.org/modules/setting"
)

// NewMigrationHTTPClient returns a HTTP client for migration
func NewMigrationHTTPClient() *http.Client {
	// Use the new HTTP client pool for migration operations
	baseClient := httplib.GetMigrationClient()
	baseTransport := baseClient.Transport.(*http.Transport)

	// Create a custom transport with migration-specific settings
	migrationTransport := &http.Transport{
		Proxy:                 baseTransport.Proxy,
		DialContext:           hostmatcher.NewDialContext("migration", allowList, blockList, setting.Proxy.ProxyURLFixed),
		ForceAttemptHTTP2:     baseTransport.ForceAttemptHTTP2,
		MaxIdleConns:          baseTransport.MaxIdleConns,
		MaxIdleConnsPerHost:   baseTransport.MaxIdleConnsPerHost,
		IdleConnTimeout:       baseTransport.IdleConnTimeout,
		TLSHandshakeTimeout:   baseTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: baseTransport.ExpectContinueTimeout,
		DisableKeepAlives:     baseTransport.DisableKeepAlives,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: setting.Migrations.SkipTLSVerify},
	}

	return &http.Client{
		Transport: migrationTransport,
		Timeout:   baseClient.Timeout,
	}
}

// NewMigrationHTTPTransport returns a HTTP transport for migration
func NewMigrationHTTPTransport() *http.Transport {
	// Use the new HTTP client pool for migration operations
	baseClient := httplib.GetMigrationClient()
	baseTransport := baseClient.Transport.(*http.Transport)

	return &http.Transport{
		Proxy:                 baseTransport.Proxy,
		DialContext:           hostmatcher.NewDialContext("migration", allowList, blockList, setting.Proxy.ProxyURLFixed),
		ForceAttemptHTTP2:     baseTransport.ForceAttemptHTTP2,
		MaxIdleConns:          baseTransport.MaxIdleConns,
		MaxIdleConnsPerHost:   baseTransport.MaxIdleConnsPerHost,
		IdleConnTimeout:       baseTransport.IdleConnTimeout,
		TLSHandshakeTimeout:   baseTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: baseTransport.ExpectContinueTimeout,
		DisableKeepAlives:     baseTransport.DisableKeepAlives,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: setting.Migrations.SkipTLSVerify},
	}
}
