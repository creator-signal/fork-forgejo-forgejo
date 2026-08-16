// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package method

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/models/db"
	forgefed_model "forgejo.org/models/forgefed"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/auth"
	"forgejo.org/services/federation"

	"github.com/42wim/httpsig"
	httpsign9421 "github.com/yaronf/httpsign"
	"golang.org/x/crypto/ssh"
)

// Ensure the struct implements the interface.
var (
	_ auth.Method = &HTTPSign{}
)

// HTTPSign implements the Auth interface and authenticates requests (API requests
// only) by looking for http signature data in the "Signature" header.
// more information can be found on https://github.com/go-fed/httpsig
type HTTPSign struct{}

// Verify extracts and validates HTTPsign from the Signature header of the request and returns
// the corresponding user object on successful validation.
// Returns nil if header is empty or validation fails.
func (h *HTTPSign) Verify(req *http.Request, w http.ResponseWriter, _ auth.SessionStore) auth.MethodOutput {
	if len(req.Header.Get("Signature")) == 0 {
		return &auth.AuthenticationNotAttempted{}
	}

	var (
		publicKey *asymkey_model.PublicKey
		err       error
	)

	if len(req.Header.Get("X-Ssh-Certificate")) != 0 {
		// Handle Signature signed by SSH certificates
		if len(setting.SSH.TrustedUserCAKeys) == 0 {
			return &auth.AuthenticationNotAttempted{}
		}

		publicKey, err = VerifyCert(req)
		if err != nil {
			log.Debug("VerifyCert on request from %s: failed: %v", req.RemoteAddr, err)
			log.Warn("Failed authentication attempt from %s", req.RemoteAddr)
			return &auth.AuthenticationNotAttempted{} // 401 is not expected on signature validation miss; return not attempted
		}
	} else if len(req.Header.Get("Signature-Input")) != 0 && setting.Federation.UseRFC9421 {
		if publicKey, err = VerifyPubKeyRFC9421(req, false, activitypub.ClientKeySSH); err != nil {
			log.Debug("VerifyPubKey9421 on request from %s: failed: %v", req.RemoteAddr, err)
			log.Warn("Failed authentication attempt from %s", req.RemoteAddr)
			return &auth.AuthenticationNotAttempted{} // 401 is not expected on signature validation miss; return not attempted
		}
	} else if len(req.Header.Get("Signature")) != 0 {
		// Handle Signature signed by Public Key
		publicKey, err = VerifyPubKey(req)
		if err != nil {
			log.Debug("VerifyPubKey on request from %s: failed: %v", req.RemoteAddr, err)
			log.Warn("Failed authentication attempt from %s", req.RemoteAddr)
			return &auth.AuthenticationNotAttempted{} // 401 is not expected on signature validation miss; return not attempted
		}
	} else {
		log.Warn("No valid authorization headers found: %v", req.Header)
		return &auth.AuthenticationNotAttempted{} // 401 is not expected on signature validation miss; return not attempted
	}

	u, userErr := user_model.GetUserByID(req.Context(), publicKey.OwnerID)
	if userErr == nil {
		log.Trace("HTTP Sign: Logged in user %-v", u)
		return &auth.AuthenticationSuccess{Result: &httpSignAuthenticationResult{user: u}}
	}

	host, hostErr := forgefed_model.GetFederationHost(req.Context(), publicKey.OwnerID)
	if hostErr == nil {
		log.Trace("HTTP Sign: Logged in host %-v", host.AsURL())
		return &auth.AuthenticationSuccess{Result: &httpSignAuthenticationResult{host: host}}
	}

	return &auth.AuthenticationError{Error: fmt.Errorf("httpsign GetUserByID: %w, GetFederationHost: %w", userErr, hostErr)}
}

func VerifyPubKey(r *http.Request) (*asymkey_model.PublicKey, error) {
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return nil, fmt.Errorf("httpsig.NewVerifier failed: %s", err)
	}

	keyID := verifier.KeyId()

	publicKeys, err := db.Find[asymkey_model.PublicKey](r.Context(), asymkey_model.FindPublicKeyOptions{
		Fingerprint: keyID,
	})
	if err != nil {
		return nil, err
	}

	if len(publicKeys) == 0 {
		return nil, fmt.Errorf("no public key found for keyid %s", keyID)
	}

	sshPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeys[0].Content))
	if err != nil {
		return nil, err
	}

	if err := doVerify(verifier, []ssh.PublicKey{sshPublicKey}); err != nil {
		return nil, err
	}

	return publicKeys[0], nil
}

func VerifyPubKeyRFC9421(r *http.Request, fetchKey bool, clientKeyType activitypub.ClientKeyType) (*asymkey_model.PublicKey, error) {
	if r == nil {
		return nil, fmt.Errorf("nil request")
	}

	sigHeaders := r.Header.Values("Signature")
	if len(sigHeaders) == 0 {
		return nil, fmt.Errorf("missing signature header")
	}

	hasBody := r.Header.Get("Content-Digest") != ""

	var fields httpsign9421.Fields
	var headers []string
	if hasBody {
		headers = append(headers, "Content-Digest")
	}
	switch r.Method {
	case http.MethodGet:
		headers = append(headers, setting.Federation.GetHeadersRFC9421...)
	case http.MethodPost:
		headers = append(headers, setting.Federation.PostHeadersRFC9421...)
	default:
		return nil, fmt.Errorf("unsupported request type: %v", r.Method)
	}

	fields = httpsign9421.Headers(headers...)
	sigNames, err := httpsign9421.RequestSignatureNames(r, false)
	if err != nil {
		return nil, fmt.Errorf("error getting signature names: %w", err)
	}

	if len(sigNames) == 0 {
		return nil, fmt.Errorf("no RFC 9421 signature headers found: %v", r.Header)
	}

	config := httpsign9421.NewVerifyConfig()
	config.SetAllowedAlgs(setting.Federation.SignatureAlgorithmsRFC9421)

	if hasBody {
		if err = activitypub.ValidateContentDigest(r.Header.Get("Content-Digest"), &r.Body, setting.Federation.DigestAlgorithms); err != nil {
			return nil, fmt.Errorf("invalid HTTP Content-Digest header: %v", err)
		}
	}

	ctx := r.Context()

	for _, name := range sigNames {
		msgDetails, err := httpsign9421.RequestDetails(name, r)
		if err != nil {
			log.Warn("no details for signature: %v, error: %v", name, err)
			continue
		}

		alg, err := setting.AlgorithmFromString(msgDetails.Alg)
		if err != nil {
			log.Debug("invalid HTTP message signature algorithm: %v", err)
			continue
		}

		if msgDetails.KeyID == nil {
			log.Warn("signature: %s nil key ID", name)
			continue
		}
		keyID := *msgDetails.KeyID

		log.Debug("verifyHttpMessageSignatures signature: %v, alg: %v, key ID: %v", name, alg, keyID)

		if fetchKey {
			if _, err = federation.FindOrCreateActorKey(ctx, keyID); err != nil {
				log.Debug("For %q verification failed: %v", r.URL.Path, err)
				return nil, err
			}
		}

		clientKey, err := activitypub.NewClientPublicKey(keyID, alg, clientKeyType).ClientKey(ctx)
		if err != nil {
			log.Warn("error creating client key: %v", err)
			continue
		}
		verifier, err := clientKey.VerifierRFC9421(config, fields)
		if err != nil {
			log.Warn("error creating HTTP message signature verifier: %v", err)
			continue
		}
		if err = httpsign9421.VerifyRequest(name, *verifier, r); err == nil {
			// return on first verified signature
			return &asymkey_model.PublicKey{OwnerID: clientKey.OwnerID()}, nil
		}
		log.Debug("error validating signature %s with key ID: %s", name, keyID)
	}

	return nil, fmt.Errorf("no valid HTTP Message Signature (RFC 9421) found")
}

// VerifyCert verifies the validity of the ssh certificate and returns the publickey of the signer
// We verify that the certificate is signed with the correct CA
// We verify that the http request is signed with the private key (of the public key mentioned in the certificate)
func VerifyCert(r *http.Request) (*asymkey_model.PublicKey, error) {
	// Get our certificate from the header
	bcert, err := base64.RawStdEncoding.DecodeString(r.Header.Get("x-ssh-certificate"))
	if err != nil {
		return nil, err
	}

	pk, err := ssh.ParsePublicKey(bcert)
	if err != nil {
		return nil, err
	}

	// Check if it's really a ssh certificate
	cert, ok := pk.(*ssh.Certificate)
	if !ok {
		return nil, errors.New("no certificate found")
	}

	c := &ssh.CertChecker{
		IsUserAuthority: func(auth ssh.PublicKey) bool {
			marshaled := auth.Marshal()

			for _, k := range setting.SSH.TrustedUserCAKeysParsed {
				if bytes.Equal(marshaled, k.Marshal()) {
					return true
				}
			}

			return false
		},
	}

	// check the CA of the cert
	if !c.IsUserAuthority(cert.SignatureKey) {
		return nil, errors.New("CA check failed")
	}

	// Create a verifier
	verifier, err := httpsig.NewVerifier(r)
	if err != nil {
		return nil, fmt.Errorf("httpsig.NewVerifier failed: %s", err)
	}

	// now verify that this request was signed with the private key that matches the certificate public key
	if err := doVerify(verifier, []ssh.PublicKey{cert.Key}); err != nil {
		return nil, err
	}

	// Now for each of the certificate valid principals
	for _, principal := range cert.ValidPrincipals {
		// Look in the db for the public key
		publicKey, err := asymkey_model.SearchPublicKeyByContentExact(r.Context(), principal)
		if asymkey_model.IsErrKeyNotExist(err) {
			// No public key matches this principal - try the next principal
			continue
		} else if err != nil {
			// this error will be a db error therefore we can't solve this and we should abort
			log.Error("SearchPublicKeyByContentExact: %v", err)
			return nil, err
		}

		// Validate the cert for this principal
		if err := c.CheckCert(principal, cert); err != nil {
			// however, because principal is a member of ValidPrincipals - if this fails then the certificate itself is invalid
			return nil, err
		}

		// OK we have a public key for a principal matching a valid certificate whose key has signed this request.
		return publicKey, nil
	}

	// No public key matching a principal in the certificate is registered in gitea
	return nil, errors.New("no valid principal found")
}

// doVerify iterates across the provided public keys attempting the verify the current request against each key in turn
func doVerify(verifier httpsig.Verifier, sshPublicKeys []ssh.PublicKey) error {
	for _, publicKey := range sshPublicKeys {
		cryptoPubkey := publicKey.(ssh.CryptoPublicKey).CryptoPublicKey()

		var algos []httpsig.Algorithm

		switch {
		case strings.HasPrefix(publicKey.Type(), "ssh-ed25519"):
			algos = []httpsig.Algorithm{httpsig.ED25519}
		case strings.HasPrefix(publicKey.Type(), "ssh-rsa"):
			algos = []httpsig.Algorithm{httpsig.RSA_SHA256, httpsig.RSA_SHA512}
		}
		for _, algo := range algos {
			if err := verifier.Verify(cryptoPubkey, algo); err == nil {
				return nil
			}
		}
	}

	return errors.New("verification failed")
}
