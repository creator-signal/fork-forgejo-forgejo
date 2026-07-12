// Copyright 2026 The Forgejo Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package hostmatcher

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllowBlockList(t *testing.T) {
	rmdc := remoteManualDialContext{}

	init := func(allow, block string, local bool) {
		// Replica of migrations/allowlist.Init
		blockList := ParseSimpleMatchList("migrations.BLOCKED_DOMAINS", block)
		allowList := ParseSimpleMatchList("migrations.ALLOWED_DOMAINS/ALLOW_LOCALNETWORKS", allow)
		if allowList.IsEmpty() {
			// the default policy is that migration module can access external hosts
			allowList.AppendBuiltin(MatchBuiltinExternal)
		}
		if local {
			allowList.AppendBuiltin(MatchBuiltinPrivate)
			allowList.AppendBuiltin(MatchBuiltinLoopback)
		}

		rmdc.allowList = allowList
		rmdc.blockList = blockList
	}

	checkByAllowBlockList := func(hostName string, addrList []net.IP) error {
		rmdc.hostName = hostName
		rmdc.addrList = addrList
		return rmdc.Check()
	}

	// default, allow all external, block none, no local networks
	init("", "", false)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.Error(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// allow all including local networks (it could lead to SSRF in production)
	init("", "", true)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// allow wildcard, block some subdomains. if the domain name is allowed, then the local network check is skipped
	init("*.domain.com", "blocked.domain.com", false)
	require.NoError(t, checkByAllowBlockList("sub.domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.NoError(t, checkByAllowBlockList("sub.domain.com", []net.IP{net.ParseIP("127.0.0.1")}))
	require.Error(t, checkByAllowBlockList("blocked.domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.Error(t, checkByAllowBlockList("sub.other.com", []net.IP{net.ParseIP("1.2.3.4")}))

	// allow wildcard (it could lead to SSRF in production)
	init("*", "", false)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// local network can still be blocked
	init("*", "127.0.0.*", false)
	require.NoError(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("1.2.3.4")}))
	require.Error(t, checkByAllowBlockList("domain.com", []net.IP{net.ParseIP("127.0.0.1")}))

	// reset
	init("", "", false)
}

func TestRemoteManualDialContext(t *testing.T) {
	allowList := ParseSimpleMatchList("", "example.com")
	blockList := ParseSimpleMatchList("", "example.org")

	t.Run("working ConfigGitCommand", func(t *testing.T) {
		// Manually create mdc so that we don't have to deal with `NewRemoteManualDialContext`'s `LookupIP`, which might
		// change if we put a real domain in here.
		mdc := &remoteManualDialContext{
			hostName:  "example.com",
			port:      "443",
			allowList: allowList,
			blockList: blockList,
			addrList: []net.IP{
				net.ParseIP("127.0.0.1"),
				net.ParseIP("::1"),
			},
		}
		require.NoError(t, mdc.Check())

		argContent := []string{}
		argDynamic := []bool{}
		err := mdc.ConfigGitCommand(
			func(arg string) {
				argContent = append(argContent, arg)
				argDynamic = append(argDynamic, false)
			},
			func(arg string) {
				argContent = append(argContent, arg)
				argDynamic = append(argDynamic, true)
			},
		)
		require.NoError(t, err)

		assert.Equal(t, []string{"-c", "http.curloptResolve=example.com:443:127.0.0.1,[::1]"}, argContent)
		assert.Equal(t, []bool{false, true}, argDynamic)
	})

	t.Run("ConfigGitCommand no addrList", func(t *testing.T) {
		mdc := &remoteManualDialContext{
			hostName:  "example.com",
			port:      "443",
			allowList: allowList,
			blockList: blockList,
			addrList:  []net.IP{},
		}
		require.NoError(t, mdc.Check())

		err := mdc.ConfigGitCommand(nil, nil)
		require.ErrorContains(t, err, "no addresses found for remote \"example.com\"")
	})

	t.Run("ConfigGitCommand not checked", func(t *testing.T) {
		mdc := &remoteManualDialContext{
			hostName:  "example.com",
			port:      "443",
			allowList: allowList,
			blockList: blockList,
			addrList:  []net.IP{},
		}
		err := mdc.ConfigGitCommand(nil, nil)
		require.ErrorContains(t, err, "must invoke and respect Check() before calling ConfigGitCommand()")
	})
}
