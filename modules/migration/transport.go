// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package migration

import "net/http"

type CustomTransport struct {
	*http.Transport
}

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", "Forgejo/16.0.1 (Migration Service)")
	return t.Transport.RoundTrip(req)
}
