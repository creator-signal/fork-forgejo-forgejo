// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package hostmatcher

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ManualDialContext provides tools to manually moderate outgoing network connections.  When interacting with a remote
// server through subprocesses (such as git), we can't simply configure a `http.Transport.DialContext` -- different
// approaches are needed for different internal processes and subprocesses.
type ManualDialContext interface {
	// Verifies if the target of the manual dial context, passed when the object is created, is valid and accessible per
	// the policies provided.
	Check() error
	// Configures a git command so that access to the outgoing network connection is limited to the already validated
	// resources of the dial context.  addArg and addDynamicArg are closures which typically reference [*git.Command]'s
	// AddArgument and AddDynamicArgument methods, but can't be directly referenced to avoid a cyclical import.
	ConfigGitCommand(addArg, addDynamicArg func(string)) error
}

var ErrManualDialContextCheckFailed = errors.New("host failed to pass Check()")

// remoteManualDialContext is a typical implementation of ManualDialContext with an allowlist, denylist, and an attempt
// to access a specific hostname or IP.
type remoteManualDialContext struct {
	hostName  string
	port      string
	allowList *HostMatchList
	blockList *HostMatchList

	addrList []net.IP
	checked  checkedState
}

type checkedState int

const (
	// Check has not been invoked.
	checkedUnchecked checkedState = iota
	// Check has been invoked and succeeded.
	checkedSuccessfully
	// Check has been invoked and failed.
	checkedWithFailure
)

// When the server is attempting to access `hostName`, create a ManualDialContext which will validate that it conforms
// to the given `allowList` and `blockList`.
func NewRemoteManualDialContext(url *url.URL, allowList, blockList *HostMatchList) ManualDialContext {
	hostName := url.Hostname()
	port := url.Port()
	if port == "" {
		switch url.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}

	// some users only use proxy, there is no DNS resolver. it's safe to ignore the LookupIP error
	addrList, _ := net.LookupIP(hostName)
	return &remoteManualDialContext{
		hostName:  hostName,
		port:      port,
		allowList: allowList,
		blockList: blockList,
		addrList:  addrList,
	}
}

func (m *remoteManualDialContext) Check() error {
	m.checked = checkedUnchecked

	var ipAllowed bool
	var ipBlocked bool
	for _, addr := range m.addrList {
		ipAllowed = ipAllowed || m.allowList.MatchIPAddr(addr)
		ipBlocked = ipBlocked || m.blockList.MatchIPAddr(addr)
	}
	// if we have an allow-list, check the allow-list before return to get the more accurate error
	if !m.allowList.IsEmpty() {
		if !m.allowList.MatchHostName(m.hostName) && !ipAllowed {
			m.checked = checkedWithFailure
			return ErrManualDialContextCheckFailed
		}
	}

	// otherwise, we always follow the blocked list
	var blockedError error
	if m.blockList.MatchHostName(m.hostName) || ipBlocked {
		blockedError = ErrManualDialContextCheckFailed
	}
	if blockedError != nil {
		m.checked = checkedWithFailure
		return blockedError
	}

	m.checked = checkedSuccessfully
	return nil
}

func (m *remoteManualDialContext) ConfigGitCommand(addArg, addDynamicArg func(string)) error {
	if m.checked != checkedSuccessfully {
		return errors.New("must invoke and respect Check() before calling ConfigGitCommand()")
	} else if len(m.addrList) == 0 {
		return fmt.Errorf("no addresses found for remote %q", m.hostName)
	}

	sepList := strings.Builder{}
	for i, addr := range m.addrList {
		if addr.To4() == nil {
			// ipv6 address; wrap with brackets per https://curl.se/libcurl/c/CURLOPT_RESOLVE.html
			sepList.WriteString("[")
			sepList.WriteString(addr.String())
			sepList.WriteString("]")
		} else {
			sepList.WriteString(addr.String())
		}
		if i != len(m.addrList)-1 {
			sepList.WriteString(",")
		}
	}

	addArg("-c")
	addDynamicArg(fmt.Sprintf("http.curloptResolve=%s:%s:%s", m.hostName, m.port, sepList.String()))
	return nil
}

// When the server is attempting to access a resource that optionally does not need dial context checking, a nil dial
// context can be created for a consistent API.
func NewNilManualDialContext() ManualDialContext {
	return &nilManualDialContext{}
}

type nilManualDialContext struct{}

func (*nilManualDialContext) Check() error {
	return nil
}

func (*nilManualDialContext) ConfigGitCommand(addArg, addDynamicArg func(string)) error {
	return nil
}
