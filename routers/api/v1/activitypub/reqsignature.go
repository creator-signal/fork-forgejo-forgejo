// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"
	"net/url"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	app_context "forgejo.org/services/context"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
)

func verifyHTTPSignature(ctx app_context.APIContext) (authenticated bool, err error) {
	// Verify that the canonical domain for federation is accessed regardless of
	// if signatures are actually checked or not.

	// This check does not cover the instance actor but that does not contain any
	// potentially private information which should only be accessed via the
	// canonical instance domain.
	appURL, err := url.Parse(setting.AppURL)
	if err != nil {
		return false, err
	}

	if ctx.Req.Host != appURL.Host {
		log.Error("%s", ctx.Req.Host)
		return false, nil
	}

	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		log.Debug("For %q verification failed: %v", r.URL.Path, err)
		return false, err
	}

	log.Debug("Verify %q, signed by KeyId: %v", r.URL.Path, v.KeyId())
	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateActorKey(ctx, v.KeyId())
	if err != nil {
		return false, err
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		log.Debug("For %q verification failed: %v", r.URL.Path, err)
		return false, err
	}
	return true, nil
}

// ReqHTTPSignature function
func ReqHTTPSignature() func(ctx *app_context.APIContext) {
	return func(ctx *app_context.APIContext) {
		if authenticated, err := verifyHTTPSignature(*ctx); err != nil {
			log.Warn("verifyHttpSignature failed: %v", err)
			ctx.Error(http.StatusBadRequest, "reqSignature", "request verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request verification failed")
		}
	}
}
