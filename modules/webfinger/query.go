// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webfinger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"golang.org/x/net/idna"
)

// unescape sub-delims escaped by go and escape gen-delims not escaped
//
// https://cs.opensource.google/go/go/+/refs/tags/go1.25.6:src/net/url/url.go;l=145
func acctEscape(input string) string {
	encoded := url.PathEscape(input)
	encoded = strings.ReplaceAll(encoded, "%3B", ";")
	encoded = strings.ReplaceAll(encoded, "%2C", ",")

	encoded = strings.ReplaceAll(encoded, "#", "%23")

	return encoded
}

func Query(ctx context.Context, actor string) (*JRD, error) {
	url, err := BuildURL(actor)
	if err != nil {
		return nil, err
	}

	log.Trace("Built webfinger URL: %s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if setting.IsProd || !errors.Is(err, http.ErrSchemeMismatch) {
			return nil, err
		}

		url.Scheme = "http"
		req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
		if err != nil {
			return nil, err
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, Malformed{extraInfo: "Got non-okay status code"}
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/jrd+json") && !strings.Contains(contentType, "application/json") {
		return nil, Malformed{extraInfo: "Got non-JSON content type"}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jrd JRD
	err = json.Unmarshal(body, &jrd)
	if err != nil {
		return nil, err
	}

	log.Debug("Fetched %v", jrd)

	return &jrd, nil
}

func BuildURL(actor string) (*url.URL, error) {
	encoded := acctEscape(fmt.Sprintf("acct:%s", actor))
	userActor, err := ParseUserActor(encoded)
	if err != nil {
		return nil, err
	}

	domain, err := idna.ToASCII(userActor.Host)
	if err != nil {
		return nil, err
	}

	var rawURL string
	if userActor.Port.Has() {
		rawURL = fmt.Sprintf("https://%s:%d/.well-known/webfinger", domain, userActor.Port.Value())
	} else {
		rawURL = fmt.Sprintf("https://%s/.well-known/webfinger", domain)
	}

	webfingerURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	query := webfingerURL.Query()

	var resource string
	if userActor.Port.Has() {
		resource = fmt.Sprintf("acct:%s@%s:%d", userActor.User, domain, userActor.Port.Value())
	} else {
		resource = fmt.Sprintf("acct:%s@%s", userActor.User, domain)
	}

	query.Add("resource", resource)
	webfingerURL.RawQuery = query.Encode()

	return webfingerURL, nil
}
