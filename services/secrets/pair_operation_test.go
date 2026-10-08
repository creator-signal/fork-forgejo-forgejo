// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package secrets

import (
	"encoding/json"
	"strings"
	"testing"

	secret_model "forgejo.org/models/secret"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/require"
)

func TestSecretPairClosedRequest(t *testing.T) {
	operationID := strings.Repeat("a", 64)
	input := api.ActionSecretPairRequest{
		Schema: secret_model.PairRequestSchema, Purpose: secret_model.PairPurpose, OperationID: operationID,
		Binding: api.ActionSecretPairBinding{OwnershipID: strings.Repeat("b", 64), TransactionID: strings.Repeat("c", 64), Nonce: strings.Repeat("d", 64)},
		Secrets: api.ActionSecretPairMaterial{Username: "vm-image-publisher", Password: "cs-vm-artifact-"+strings.Repeat("e", 43)},
	}
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	valid := string(raw)
	parsed, err := ParseSecretPairRequest(strings.NewReader(valid), operationID)
	require.NoError(t, err)
	require.Equal(t, input, *parsed)
	for name, body := range map[string]string{
		"duplicate root": strings.Replace(valid, `"schema":`, `"schema":"ignored","schema":`, 1),
		"escaped duplicate": strings.Replace(valid, `"nonce":`, `"\u006eonce":"ignored","nonce":`, 1),
		"unknown root": strings.Replace(valid, `{`, `{"authority":true,`, 1),
		"unknown nested": strings.Replace(valid, `"binding":{`, `"binding":{"proof":"none",`, 1),
		"third secret": strings.Replace(valid, `"secrets":{`, `"secrets":{"THIRD":"none",`, 1),
		"missing": strings.Replace(valid, `"purpose":"vm-artifact-publisher",`, ``, 1),
		"case alias": strings.Replace(valid, `"operationId"`, `"OperationId"`, 1),
		"null binding": strings.Replace(valid, `"ownershipId":"`+strings.Repeat("b", 64)+`"`, `"ownershipId":null`, 1),
		"array": `[]`,
		"trailing document": valid+`{}`,
		"upper identity": strings.Replace(valid, operationID, strings.Repeat("A", 64), 1),
		"short password": strings.Replace(valid, strings.Repeat("e", 43), strings.Repeat("e", 42), 1),
		"newline password": strings.Replace(valid, strings.Repeat("e", 43), strings.Repeat("e", 43)+`\n`, 1),
		"username": strings.Replace(valid, "vm-image-publisher", "other-user", 1),
		"oversized": strings.Repeat(" ", SecretPairRequestMaxBytes)+valid,
		"invalid UTF8": valid+string([]byte{255}),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseSecretPairRequest(strings.NewReader(body), operationID)
			require.ErrorIs(t, err, secret_model.ErrPairInvalid)
		})
	}
	_, err = ParseSecretPairRequest(strings.NewReader(valid), strings.Repeat("f", 64))
	require.ErrorIs(t, err, secret_model.ErrPairInvalid)
}
