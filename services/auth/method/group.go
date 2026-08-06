// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package method

import (
	stdctx "context"
	"errors"
	"fmt"
	"net/http"

	"forgejo.org/services/auth"
)

// isRequestCanceled reports whether the request was abandoned by the client, either because err carries a context
// cancellation or deadline, or because the request context is already done. Both mean the response can no longer be
// delivered, so the failure did not originate on the server.
func isRequestCanceled(req *http.Request, err error) bool {
	if errors.Is(err, stdctx.Canceled) || errors.Is(err, stdctx.DeadlineExceeded) {
		return true
	}
	return req != nil && req.Context().Err() != nil
}

// Ensure the struct implements the interface.
var (
	_ auth.Method = &Group{}
)

// Group implements the Auth interface with serval Auth.
type Group struct {
	methods []auth.Method
}

// NewGroup creates a new auth group
func NewGroup(methods ...auth.Method) *Group {
	return &Group{
		methods: methods,
	}
}

// Add adds a new method to group
func (b *Group) Add(method auth.Method) {
	b.methods = append(b.methods, method)
}

func (b *Group) Verify(req *http.Request, w http.ResponseWriter, sess auth.SessionStore) auth.MethodOutput {
	var incorrectCredentials []error
	var interactiveReauthenticationPossible bool

	for _, m := range b.methods {
		output := m.Verify(req, w, sess)

		switch v := output.(type) {
		case *auth.AuthenticationError:
			// A canceled or timed out request context means the client went away while the method was talking to the
			// database, not that authentication failed. Report it as such so callers do not treat it as an internal
			// error. See https://codeberg.org/forgejo/forgejo/issues/13782
			if isRequestCanceled(req, v.Error) {
				return &auth.AuthenticationCancelled{Error: v.Error}
			}
			return v

		case *auth.AuthenticationSuccess:
			return v

		case *auth.AuthenticationNotAttempted:
			// Move on to the next supported authentication method.
			interactiveReauthenticationPossible = interactiveReauthenticationPossible || v.InteractiveReauthenticationPossible
			continue

		case *auth.AuthenticationAttemptedIncorrectCredential:
			// Move on to the next supported authentication method, but keep a record of this error.  If none of the
			// other methods are able to authenticate the user, we'll report this as an incorrect credential (401) case.
			incorrectCredentials = append(incorrectCredentials, v.Error)
			continue

		default:
			return &auth.AuthenticationError{Error: fmt.Errorf("unexpected result from Method.Verify on method %v: %v", m, v)}
		}
	}

	if len(incorrectCredentials) != 0 {
		return &auth.AuthenticationAttemptedIncorrectCredential{Error: errors.Join(incorrectCredentials...)}
	}

	return &auth.AuthenticationNotAttempted{InteractiveReauthenticationPossible: interactiveReauthenticationPossible}
}
