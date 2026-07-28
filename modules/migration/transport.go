// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import "net/http"

type CustomTransport struct {
	*http.Transport
}

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", "Forgejo-migration-http-client/1.1")
	return t.Transport.RoundTrip(req)
}
