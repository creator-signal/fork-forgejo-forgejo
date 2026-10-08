// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

// ActionSecretPairBinding identifies the original private operation intent.
// swagger:model
type ActionSecretPairBinding struct {
	OwnershipID   string `json:"ownershipId"`
	TransactionID string `json:"transactionId"`
	Nonce         string `json:"nonce"`
}

// ActionSecretPairMaterial is the sole managed pair. Values are never returned.
// swagger:model
type ActionSecretPairMaterial struct {
	Username string `json:"ZOT_VM_ARTIFACT_USERNAME"`
	Password string `json:"ZOT_VM_ARTIFACT_PASSWORD"`
}

// ActionSecretPairRequest is accepted only by the bounded, closed parser.
// swagger:model
type ActionSecretPairRequest struct {
	Schema      string                   `json:"schema"`
	Purpose     string                   `json:"purpose"`
	OperationID string                   `json:"operationId"`
	Binding     ActionSecretPairBinding  `json:"binding"`
	Secrets     ActionSecretPairMaterial `json:"secrets"`
}

// ActionSecretPairResult contains no secret material or secret-derived hashes.
// swagger:model
type ActionSecretPairResult struct {
	Schema      string                  `json:"schema"`
	OperationID string                  `json:"operationId"`
	Binding     ActionSecretPairBinding `json:"binding"`
	State       string                  `json:"state"`
	Replayed    bool                    `json:"replayed"`
}
