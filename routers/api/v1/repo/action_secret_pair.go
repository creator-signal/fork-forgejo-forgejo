// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"errors"
	"net/http"

	secret_model "forgejo.org/models/secret"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	secrets_service "forgejo.org/services/secrets"
)

// CreateSecretPairOperation creates or identically replays the sole managed pair.
func (Action) CreateSecretPairOperation(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/actions/secret-pair-operations/{operationId} repository createSecretPairOperation
	// ---
	// summary: Atomically create the operation-bound VM artifact Actions pair
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   type: string
	//   required: true
	// - name: operationId
	//   in: path
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   required: true
	//   schema:
	//     "$ref": "#/definitions/ActionSecretPairRequest"
	// responses:
	//   "200":
	//     description: Identical replay verified without writes
	//     schema:
	//       "$ref": "#/definitions/ActionSecretPairResult"
	//   "201":
	//     description: Operation and pair created in one transaction
	//     schema:
	//       "$ref": "#/definitions/ActionSecretPairResult"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/error"
	if !setting.Actions.Enabled {
		ctx.NotFound()
		return
	}
	request, err := secrets_service.ParseSecretPairRequest(ctx.Req.Body, ctx.Params("operationId"))
	if err != nil {
		ctx.Error(http.StatusBadRequest, "CreateSecretPairOperation", secret_model.ErrPairInvalid)
		return
	}
	result, err := secret_model.ApplySecretPairOperation(ctx, ctx.Repo().Repository.ID, request)
	if err != nil {
		if errors.Is(err, secret_model.ErrPairDisabled) {
			ctx.NotFound(); return
		}
		ctx.Error(http.StatusConflict, "CreateSecretPairOperation", secret_model.ErrPairConflict)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	ctx.JSON(status, result)
}
