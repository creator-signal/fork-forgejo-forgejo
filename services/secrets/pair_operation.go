// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package secrets

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"

	secret_model "forgejo.org/models/secret"
	api "forgejo.org/modules/structs"
)

const SecretPairRequestMaxBytes = 4096

// ParseSecretPairRequest rejects duplicate, unknown and missing keys before
// decoding values. Generic JSON binding accepts duplicate/case-insensitive keys
// and is intentionally not used for this closed operation.
func ParseSecretPairRequest(reader io.Reader, operationID string) (*api.ActionSecretPairRequest, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, SecretPairRequestMaxBytes+1))
	if err != nil || len(raw) > SecretPairRequestMaxBytes || !utf8.Valid(raw) {
		return nil, secret_model.ErrPairInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := parsePairObject(decoder, "request"); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, secret_model.ErrPairInvalid
	}
	var request api.ActionSecretPairRequest
	if err := json.Unmarshal(raw, &request); err != nil || request.OperationID != operationID {
		return nil, secret_model.ErrPairInvalid
	}
	if err := secret_model.ValidateSecretPairRequest(&request); err != nil {
		return nil, err
	}
	return &request, nil
}

func parsePairObject(decoder *json.Decoder, object string) error {
	keys := map[string]string{}
	switch object {
	case "request":
		keys = map[string]string{"schema":"", "purpose":"", "operationId":"", "binding":"binding", "secrets":"secrets"}
	case "binding":
		keys = map[string]string{"ownershipId":"", "transactionId":"", "nonce":""}
	case "secrets":
		keys = map[string]string{secret_model.PairUsernameName:"", secret_model.PairPasswordName:""}
	default:
		return secret_model.ErrPairInvalid
	}
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return secret_model.ErrPairInvalid
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return secret_model.ErrPairInvalid
		}
		child, known := keys[key]
		if !known {
			return secret_model.ErrPairInvalid
		}
		seen[key] = true
		if child != "" {
			if err := parsePairObject(decoder, child); err != nil {
				return err
			}
		} else {
			value, err := decoder.Token()
			if _, ok := value.(string); err != nil || !ok {
				return secret_model.ErrPairInvalid
			}
		}
	}
	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') || len(seen) != len(keys) {
		return secret_model.ErrPairInvalid
	}
	return nil
}
