// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import "time"

// HTTPClient represents configuration for HTTP client pooling
var HTTPClient = struct {
	MaxIdleConns          int           `ini:"MAX_IDLE_CONNS"`
	MaxIdleConnsPerHost   int           `ini:"MAX_IDLE_CONNS_PER_HOST"`
	IdleConnTimeout       time.Duration `ini:"IDLE_CONN_TIMEOUT"`
	TLSHandshakeTimeout   time.Duration `ini:"TLS_HANDSHAKE_TIMEOUT"`
	ExpectContinueTimeout time.Duration `ini:"EXPECT_CONTINUE_TIMEOUT"`
	DialTimeout           time.Duration `ini:"DIAL_TIMEOUT"`
	KeepAlive             time.Duration `ini:"KEEP_ALIVE"`
	DefaultTimeout        time.Duration `ini:"DEFAULT_TIMEOUT"`
	ForceHTTP2            bool          `ini:"FORCE_HTTP2"`
}{
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   10,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	DialTimeout:           30 * time.Second,
	KeepAlive:             30 * time.Second,
	DefaultTimeout:        60 * time.Second,
	ForceHTTP2:            true,
}

func loadHTTPClientFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("http_client")

	HTTPClient.MaxIdleConns = sec.Key("MAX_IDLE_CONNS").MustInt(100)
	HTTPClient.MaxIdleConnsPerHost = sec.Key("MAX_IDLE_CONNS_PER_HOST").MustInt(10)
	HTTPClient.IdleConnTimeout = sec.Key("IDLE_CONN_TIMEOUT").MustDuration(90 * time.Second)
	HTTPClient.TLSHandshakeTimeout = sec.Key("TLS_HANDSHAKE_TIMEOUT").MustDuration(10 * time.Second)
	HTTPClient.ExpectContinueTimeout = sec.Key("EXPECT_CONTINUE_TIMEOUT").MustDuration(1 * time.Second)
	HTTPClient.DialTimeout = sec.Key("DIAL_TIMEOUT").MustDuration(30 * time.Second)
	HTTPClient.KeepAlive = sec.Key("KEEP_ALIVE").MustDuration(30 * time.Second)
	HTTPClient.DefaultTimeout = sec.Key("DEFAULT_TIMEOUT").MustDuration(60 * time.Second)
	HTTPClient.ForceHTTP2 = sec.Key("FORCE_HTTP2").MustBool(true)
}
