// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgejo.org/modules/structs"
	shared "forgejo.org/routers/api/v1/shared"
)

// SecretList
// swagger:response SecretList
type swaggerResponseSecretList struct {
	// in:body
	Body []api.Secret `json:"body"`
}

// Secret
// swagger:response Secret
type swaggerResponseSecret struct {
	// in:body
	Body api.Secret `json:"body"`
}

// ActionVariable
// swagger:response ActionVariable
type swaggerResponseActionVariable struct {
	// in:body
	Body api.ActionVariable `json:"body"`
}

// VariableList
// swagger:response VariableList
type swaggerResponseVariableList struct {
	// in:body
	Body []api.ActionVariable `json:"body"`
}

// RunJobList is a list of action run jobs
// swagger:response RunJobList
type swaggerRunJobList struct {
	// in:body
	Body []*api.ActionRunJobResponse `json:"body"`
}

// DispatchWorkflowRun is a Workflow Run after dispatching
// swagger:response DispatchWorkflowRun
type swaggerDispatchWorkflowRun struct {
	// in:body
	Body *api.DispatchWorkflowRun `json:"body"`
}

// RegistrationToken is a string used to register a runner with a server
// swagger:response RegistrationToken
type swaggerRegistrationToken struct {
	// in: body
	Body shared.RegistrationToken `json:"body"`
}

// ActionRunJob is a detailed action run job with steps
// swagger:response ActionRunJob
type swaggerActionRunJob struct {
	// in:body
	Body *api.ActionRunJobResponse `json:"body"`
}
