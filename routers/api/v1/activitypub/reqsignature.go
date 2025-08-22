// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activitypub

import (
	"net/http"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	services_context "forgejo.org/services/context"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
)

func verifyHTTPUserOrInstanceSignature(ctx services_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		log.Debug("Request %q requires HTTPUserOrInstanceSignature. Verification failed: %v", r.URL.Path, err)
		return false, err
	}

	log.Debug("Request %q requires HTTPUserOrInstanceSignature. Signed by KeyId: %v", r.URL.Path, v.KeyId())
	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateFederatedUserKey(ctx, v.KeyId())
	if err != nil || pubKey == nil {
		pubKey, err = federation.FindOrCreateFederationHostKey(ctx, v.KeyId())
		if err != nil {
			log.Debug("Request %q requires HTTPUserOrInstanceSignature. Verification failed: %v", r.URL.Path, err)
			return false, err
		}
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		log.Debug("Request %q requires HTTPUserOrInstanceSignature. Verification failed: %v", r.URL.Path, err)
		return false, err
	}
	log.Debug("Request %q requires HTTPUserOrInstanceSignature. Signature was valid.", r.URL.Path)
	return true, nil
}

func verifyHTTPUserSignature(ctx services_context.APIContext) (authenticated bool, err error) {
	if !setting.Federation.SignatureEnforced {
		return true, nil
	}

	r := ctx.Req

	// 1. Figure out what key we need to verify
	v, err := httpsig.NewVerifier(r)
	if err != nil {
		log.Debug("Request %q requires HTTPUserSignature. Verification failed: %v", r.URL.Path, err)
		return false, err
	}

	log.Debug("Request %q requires HTTPUserSignature. Signed by KeyId: %v", r.URL.Path, v.KeyId())
	signatureAlgorithm := httpsig.Algorithm(setting.Federation.SignatureAlgorithms[0])
	pubKey, err := federation.FindOrCreateFederatedUserKey(ctx, v.KeyId())
	if err != nil {
		log.Debug("Request %q requires HTTPUserSignature. Verification failed: %v", r.URL.Path, err)
		return false, err
	}

	err = v.Verify(pubKey, signatureAlgorithm)
	if err != nil {
		log.Debug("Request %q requires HTTPUserSignature. Verification failed: %v", r.URL.Path, err)
		return false, err
	}
	log.Debug("Request %q requires HTTPUserSignature. Signature was valid.", r.URL.Path)
	return true, nil
}

// ReqHTTPSignature function
func ReqHTTPUserOrInstanceSignature() func(ctx *services_context.APIContext) {
	return func(ctx *services_context.APIContext) {
		if authenticated, err := verifyHTTPUserOrInstanceSignature(*ctx); err != nil {
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}

// ReqHTTPUserSignature function
func ReqHTTPUserSignature() func(ctx *services_context.APIContext) {
	return func(ctx *services_context.APIContext) {
		if authenticated, err := verifyHTTPUserSignature(*ctx); err != nil {
			ctx.Error(http.StatusBadRequest, "reqSignature", "request signature verification failed")
		} else if !authenticated {
			ctx.Error(http.StatusForbidden, "reqSignature", "request signature verification failed")
		}
	}
}
