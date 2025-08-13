// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	secret_model "forgejo.org/models/secret"
	"forgejo.org/modules/actions"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/shared"
	"forgejo.org/routers/api/v1/utils"
	actions_service "forgejo.org/services/actions"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	secret_service "forgejo.org/services/secrets"
)

// ListActionsSecrets list an repo's actions secrets
func (Action) ListActionsSecrets(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/secrets repository repoListActionsSecrets
	// ---
	// summary: List an repo's actions secrets
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/SecretList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repo := ctx.Repo.Repository

	opts := &secret_model.FindSecretsOptions{
		RepoID:      repo.ID,
		ListOptions: utils.GetListOptions(ctx),
	}

	secrets, count, err := db.FindAndCount[secret_model.Secret](ctx, opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	apiSecrets := make([]*api.Secret, len(secrets))
	for k, v := range secrets {
		apiSecrets[k] = &api.Secret{
			Name:    v.Name,
			Created: v.CreatedUnix.AsTime(),
		}
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, apiSecrets)
}

// create or update one secret of the repository
func (Action) CreateOrUpdateSecret(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/actions/secrets/{secretname} repository updateRepoSecret
	// ---
	// summary: Create or Update a secret value in a repository
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: secretname
	//   in: path
	//   description: name of the secret
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateOrUpdateSecretOption"
	// responses:
	//   "201":
	//     description: response when creating a secret
	//   "204":
	//     description: response when updating a secret
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repo := ctx.Repo.Repository

	opt := web.GetForm(ctx).(*api.CreateOrUpdateSecretOption)

	_, created, err := secret_service.CreateOrUpdateSecret(ctx, 0, repo.ID, ctx.Params("secretname"), opt.Data)
	if err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "CreateOrUpdateSecret", err)
		} else if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "CreateOrUpdateSecret", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "CreateOrUpdateSecret", err)
		}
		return
	}

	if created {
		ctx.Status(http.StatusCreated)
	} else {
		ctx.Status(http.StatusNoContent)
	}
}

// DeleteSecret delete one secret of the repository
func (Action) DeleteSecret(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/actions/secrets/{secretname} repository deleteRepoSecret
	// ---
	// summary: Delete a secret in a repository
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repository
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: secretname
	//   in: path
	//   description: name of the secret
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     description: delete one secret of the organization
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	repo := ctx.Repo.Repository

	err := secret_service.DeleteSecretByName(ctx, 0, repo.ID, ctx.Params("secretname"))
	if err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "DeleteSecret", err)
		} else if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "DeleteSecret", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteSecret", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetVariable get a repo-level variable
func (Action) GetVariable(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/variables/{variablename} repository getRepoVariable
	// ---
	// summary: Get a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//			"$ref": "#/responses/ActionVariable"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"
	v, err := actions_service.GetVariable(ctx, actions_model.FindVariablesOpts{
		RepoID: ctx.Repo.Repository.ID,
		Name:   ctx.Params("variablename"),
	})
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetVariable", err)
		}
		return
	}

	variable := &api.ActionVariable{
		OwnerID: v.OwnerID,
		RepoID:  v.RepoID,
		Name:    v.Name,
		Data:    v.Data,
	}

	ctx.JSON(http.StatusOK, variable)
}

// DeleteVariable delete a repo-level variable
func (Action) DeleteVariable(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/actions/variables/{variablename} repository deleteRepoVariable
	// ---
	// summary: Delete a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     description: response when deleting a variable
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	if err := actions_service.DeleteVariableByName(ctx, 0, ctx.Repo.Repository.ID, ctx.Params("variablename")); err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "DeleteVariableByName", err)
		} else if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "DeleteVariableByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "DeleteVariableByName", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// CreateVariable create a repo-level variable
func (Action) CreateVariable(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/actions/variables/{variablename} repository createRepoVariable
	// ---
	// summary: Create a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateVariableOption"
	// responses:
	//   "201":
	//     description: response when creating a repo-level variable
	//   "204":
	//     description: response when creating a repo-level variable
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	opt := web.GetForm(ctx).(*api.CreateVariableOption)

	repoID := ctx.Repo.Repository.ID
	variableName := ctx.Params("variablename")

	v, err := actions_service.GetVariable(ctx, actions_model.FindVariablesOpts{
		RepoID: repoID,
		Name:   variableName,
	})
	if err != nil && !errors.Is(err, util.ErrNotExist) {
		ctx.Error(http.StatusInternalServerError, "GetVariable", err)
		return
	}
	if v != nil && v.ID > 0 {
		ctx.Error(http.StatusConflict, "VariableNameAlreadyExists", util.NewAlreadyExistErrorf("variable name %s already exists", variableName))
		return
	}

	if _, err := actions_service.CreateVariable(ctx, 0, repoID, variableName, opt.Value); err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "CreateVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "CreateVariable", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UpdateVariable update a repo-level variable
func (Action) UpdateVariable(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/actions/variables/{variablename} repository updateRepoVariable
	// ---
	// summary: Update a repo-level variable
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: variablename
	//   in: path
	//   description: name of the variable
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/UpdateVariableOption"
	// responses:
	//   "201":
	//     description: response when updating a repo-level variable
	//   "204":
	//     description: response when updating a repo-level variable
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	opt := web.GetForm(ctx).(*api.UpdateVariableOption)

	v, err := actions_service.GetVariable(ctx, actions_model.FindVariablesOpts{
		RepoID: ctx.Repo.Repository.ID,
		Name:   ctx.Params("variablename"),
	})
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetVariable", err)
		}
		return
	}

	if opt.Name == "" {
		opt.Name = ctx.Params("variablename")
	}
	if _, err := actions_service.UpdateVariable(ctx, v.ID, 0, ctx.Repo.Repository.ID, opt.Name, opt.Value); err != nil {
		if errors.Is(err, util.ErrInvalidArgument) {
			ctx.Error(http.StatusBadRequest, "UpdateVariable", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "UpdateVariable", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ListVariables list repo-level variables
func (Action) ListVariables(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/variables repository getRepoVariablesList
	// ---
	// summary: Get repo-level variables list
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: name of the owner
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repository
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//		 "$ref": "#/responses/VariableList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	vars, count, err := db.FindAndCount[actions_model.ActionVariable](ctx, &actions_model.FindVariablesOpts{
		RepoID:      ctx.Repo.Repository.ID,
		ListOptions: utils.GetListOptions(ctx),
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FindVariables", err)
		return
	}

	variables := make([]*api.ActionVariable, len(vars))
	for i, v := range vars {
		variables[i] = &api.ActionVariable{
			OwnerID: v.OwnerID,
			RepoID:  v.RepoID,
			Name:    v.Name,
		}
	}

	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, variables)
}

// GetRegistrationToken returns the token to register repo runners
func (Action) GetRegistrationToken(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runners/registration-token repository repoGetRunnerRegistrationToken
	// ---
	// summary: Get a repository's actions runner registration token
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/RegistrationToken"

	shared.GetRegistrationToken(ctx, 0, ctx.Repo.Repository.ID)
}

// SearchActionRunJobs return a list of actions jobs filtered by the provided parameters
func (Action) SearchActionRunJobs(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runners/jobs repository repoSearchRunJobs
	// ---
	// summary: Search for repository's action jobs according filter conditions
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: labels
	//   in: query
	//   description: a comma separated list of run job labels to search for
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/RunJobList"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	shared.GetActionRunJobs(ctx, 0, ctx.Repo.Repository.ID)
}

var _ actions_service.API = new(Action)

// Action implements actions_service.API
type Action struct{}

// NewAction creates a new Action service
func NewAction() actions_service.API {
	return Action{}
}

// ListActionTasks list all the actions of a repository
func ListActionTasks(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/tasks repository ListActionTasks
	// ---
	// summary: List a repository's action tasks
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results, default maximum page size is 50
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/TasksList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "409":
	//     "$ref": "#/responses/conflict"
	//   "422":
	//     "$ref": "#/responses/validationError"

	tasks, total, err := db.FindAndCount[actions_model.ActionTask](ctx, &actions_model.FindTaskOptions{
		ListOptions: utils.GetListOptions(ctx),
		RepoID:      ctx.Repo.Repository.ID,
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListActionTasks", err)
		return
	}

	res := new(api.ActionTaskResponse)
	res.TotalCount = total

	res.Entries = make([]*api.ActionTask, len(tasks))
	for i := range tasks {
		convertedTask, err := convert.ToActionTask(ctx, tasks[i])
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "ToActionTask", err)
			return
		}
		res.Entries[i] = convertedTask
	}

	ctx.JSON(http.StatusOK, &res)
}

// DispatchWorkflow dispatches a workflow
func DispatchWorkflow(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/actions/workflows/{workflowfilename}/dispatches repository DispatchWorkflow
	// ---
	// summary: Dispatches a workflow
	// consumes:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: workflowfilename
	//   in: path
	//   description: name of the workflow
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/DispatchWorkflowOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/DispatchWorkflowRun"
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	opt := web.GetForm(ctx).(*api.DispatchWorkflowOption)
	name := ctx.Params("workflowfilename")

	if len(opt.Ref) == 0 {
		ctx.Error(http.StatusBadRequest, "ref", "ref is empty")
		return
	} else if len(name) == 0 {
		ctx.Error(http.StatusBadRequest, "workflowfilename", "workflow file name is empty")
		return
	}

	workflow, err := actions_service.GetWorkflowFromCommit(ctx.Repo.GitRepo, opt.Ref, name)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetWorkflowFromCommit", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetWorkflowFromCommit", err)
		}
		return
	}

	inputGetter := func(key string) string {
		return opt.Inputs[key]
	}

	run, jobs, err := workflow.Dispatch(ctx, inputGetter, ctx.Repo.Repository, ctx.Doer)
	if err != nil {
		if actions_service.IsInputRequiredErr(err) {
			ctx.Error(http.StatusBadRequest, "workflow.Dispatch", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "workflow.Dispatch", err)
		}
		return
	}

	workflowRun := &api.DispatchWorkflowRun{
		ID:        run.ID,
		RunNumber: run.Index,
		Jobs:      jobs,
	}

	if opt.ReturnRunInfo {
		ctx.JSON(http.StatusCreated, workflowRun)
	} else {
		ctx.JSON(http.StatusNoContent, nil)
	}
}

// ListActionRuns return a filtered list of ActionRun
func ListActionRuns(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runs repository ListActionRuns
	// ---
	// summary: List a repository's action runs
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results, default maximum page size is 50
	//   type: integer
	// - name: event
	//   in: query
	//   description: Returns workflow run triggered by the specified events. For example, `push`, `pull_request` or `workflow_dispatch`.
	//   type: array
	//   items:
	//     type: string
	// - name: status
	//   in: query
	//   description: |
	//     Returns workflow runs with the check run status or conclusion that is specified. For example, a conclusion can be success or a status can be in_progress. Only Forgejo Actions can set a status of waiting, pending, or requested.
	//   type: array
	//   items:
	//     type: string
	//     enum: [unknown, waiting, running, success, failure, cancelled, skipped, blocked]
	// - name: run_number
	//   in: query
	//   description: |
	//     Returns the workflow run associated with the run number.
	//   type: integer
	//   format: int64
	// - name: head_sha
	//   in: query
	//   description: Only returns workflow runs that are associated with the specified head_sha.
	//   type: string
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActionRunList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	statusStrs := ctx.FormStrings("status")
	statuses := make([]actions_model.Status, len(statusStrs))
	for i, s := range statusStrs {
		if status, exists := actions_model.StatusFromString(s); exists {
			statuses[i] = status
		} else {
			ctx.Error(http.StatusBadRequest, "StatusFromString", fmt.Sprintf("unknown status: %s", s))
			return
		}
	}

	runs, total, err := db.FindAndCount[actions_model.ActionRun](ctx, &actions_model.FindRunJobOptions{
		ListOptions: utils.GetListOptions(ctx),
		OwnerID:     ctx.Repo.Owner.ID,
		RepoID:      ctx.Repo.Repository.ID,
		Events:      ctx.FormStrings("event"),
		Statuses:    statuses,
		RunNumber:   ctx.FormInt64("run_number"),
		CommitSHA:   ctx.FormString("head_sha"),
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListActionRuns", err)
		return
	}

	res := new(api.ListActionRunResponse)
	res.TotalCount = total

	res.Entries = make([]*api.ActionRun, len(runs))
	for i, r := range runs {
		if err := r.LoadAttributes(ctx); err != nil {
			ctx.Error(http.StatusInternalServerError, "LoadAttributes", err)
			return
		}
		cr := convert.ToActionRun(ctx, r, ctx.Doer)
		res.Entries[i] = cr
	}

	ctx.JSON(http.StatusOK, &res)
}

// GetActionRun get one action instance
func GetActionRun(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runs/{run_id} repository ActionRun
	// ---
	// summary: Get an action run
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: run_id
	//   in: path
	//   description: id of the action run
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActionRun"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	run, err := actions_model.GetRunByID(ctx, ctx.ParamsInt64(":run_id"))
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "GetRunById", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetRunByID", err)
		}
		return
	}

	// Action runs lives in its own table, therefore we check that the
	// run with the requested ID is owned by the repository
	if ctx.Repo.Repository.ID != run.RepoID {
		ctx.Error(http.StatusNotFound, "GetRunById", util.ErrNotExist)
		return
	}

	if err := run.LoadAttributes(ctx); err != nil {
		ctx.Error(http.StatusInternalServerError, "LoadAttributes", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToActionRun(ctx, run, ctx.Doer))
}

// GetActionJob get a specific job of a run
func GetActionJob(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runs/{run}/jobs/{job} repository repoGetActionJob
	// ---
	// summary: Get a specific job of a workflow run
	// produces:
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: run
	//   in: path
	//   description: index of the workflow run
	//   type: integer
	//   required: true
	// - name: job
	//   in: path
	//   description: index of the job within the run
	//   type: integer
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ActionJob"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	runIndex := ctx.ParamsInt64(":run")
	jobIndex := ctx.ParamsInt64(":job")

	job, jobs := getRunJobsForAPI(ctx, runIndex, jobIndex)
	if ctx.Written() {
		return
	}

	// Convert job to API response
	resp, err := convert.ToActionJobResponse(ctx, job, jobs)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ToActionJobResponse", err)
		return
	}

	ctx.JSON(http.StatusOK, resp)
}

// GetActionJobLogs get logs of a specific job
func GetActionJobLogs(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/actions/runs/{run}/jobs/{job}/logs repository repoGetActionJobLogs
	// ---
	// summary: Get logs for a workflow job
	// produces:
	// - text/plain
	// - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: run
	//   in: path
	//   description: index of the workflow run
	//   type: integer
	//   required: true
	// - name: job
	//   in: path
	//   description: index of the job within the run
	//   type: integer
	//   required: true
	// - name: tail
	//   in: query
	//   description: Return the last N lines from the log (cannot be combined with head)
	//   type: integer
	// - name: head
	//   in: query
	//   description: Return the first N lines from the log (cannot be combined with tail)
	//   type: integer
	// - name: offset
	//   in: query
	//   description: Line offset to start from (0-based, can be combined with head or tail)
	//   type: integer
	// - name: format
	//   in: query
	//   description: Response format (text or json, defaults to text)
	//   type: string
	//   enum: [text, json]
	// responses:
	//   "200":
	//     description: Job logs as plain text or JSON array
	//     schema:
	//       type: string
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	runIndex := ctx.ParamsInt64(":run")
	jobIndex := ctx.ParamsInt64(":job")

	job, _ := getRunJobsForAPI(ctx, runIndex, jobIndex)
	if ctx.Written() {
		return
	}

	if job.TaskID == 0 {
		ctx.NotFound("Job has not started", nil)
		return
	}

	task, err := actions_model.GetTaskByID(ctx, job.TaskID)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.NotFound("Task not found", err)
			return
		}
		ctx.Error(http.StatusInternalServerError, "GetTaskByID", err)
		return
	}

	if task.LogExpired {
		ctx.NotFound("Logs have been cleaned up", nil)
		return
	}

	// Parse query parameters
	tail := ctx.FormInt64("tail")
	head := ctx.FormInt64("head")
	offset := ctx.FormInt64("offset")
	format := ctx.FormString("format")
	if format == "" {
		format = "text"
	}

	// Validate parameters - tail and head are mutually exclusive
	if tail > 0 && head > 0 {
		ctx.Error(http.StatusBadRequest, "InvalidParameters", "Cannot specify both 'tail' and 'head' parameters")
		return
	}

	// Validate format parameter
	if format != "text" && format != "json" {
		ctx.Error(http.StatusBadRequest, "InvalidFormat", "Format must be 'text' or 'json'")
		return
	}

	// Handle different modes
	if tail > 0 || head > 0 || offset > 0 || format == "json" {
		// Use partial/structured log retrieval
		if err := servePartialLogs(ctx, task, tail, head, offset, format); err != nil {
			if errors.Is(err, util.ErrInvalidArgument) {
				ctx.Error(http.StatusBadRequest, "InvalidParameters", err.Error())
			} else if errors.Is(err, os.ErrNotExist) {
				ctx.Error(http.StatusNotFound, "LogsNotFound", "Log file not found")
			} else {
				ctx.Error(http.StatusInternalServerError, "ServePartialLogs", err)
			}
		}
		return
	}

	// Default: serve complete log file
	reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ctx.Error(http.StatusNotFound, "LogsNotFound", "Log file not found")
		} else {
			ctx.Error(http.StatusInternalServerError, "OpenLogs", err)
		}
		return
	}
	defer reader.Close()

	workflowName := job.Run.WorkflowID
	if p := strings.Index(workflowName, "."); p > 0 {
		workflowName = workflowName[0:p]
	}

	ctx.ServeContent(reader, &context.ServeHeaderOptions{
		Filename:           fmt.Sprintf("%v-%v-%v.log", workflowName, job.Name, task.ID),
		ContentLength:      &task.LogSize,
		ContentType:        "text/plain",
		ContentTypeCharset: "utf-8",
		Disposition:        "inline", // Use inline for API, attachment for web UI download
	})
}

// servePartialLogs serves partial or formatted logs based on query parameters
func servePartialLogs(ctx *context.APIContext, task *actions_model.ActionTask, tail, head, offset int64, format string) error {
	reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("log file not found: %w", err)
		}
		return err
	}
	defer reader.Close()

	var lines []string
	var timestamps []time.Time

	if tail > 0 {
		// Tail mode: read last N lines
		lines, timestamps, err = readTailLines(reader, tail, offset)
	} else if head > 0 {
		// Head mode: read first N lines from offset
		lines, timestamps, err = readHeadLines(ctx, task, head, offset)
	} else if offset > 0 {
		// Offset only: read all lines from offset
		lines, timestamps, err = readHeadLines(ctx, task, -1, offset)
	} else if format == "json" {
		// JSON format with no filters: read all lines
		lines, timestamps, err = readHeadLines(ctx, task, -1, 0)
	}

	if err != nil {
		return err
	}

	// Format and serve the response
	if format == "json" {
		type LogLine struct {
			LineNumber int64     `json:"line_number"`
			Timestamp  time.Time `json:"timestamp"`
			Content    string    `json:"content"`
		}

		jsonLines := make([]LogLine, len(lines))
		startLine := offset
		if tail > 0 && len(lines) > 0 {
			// For tail mode, calculate the starting line number
			// This is approximate since we don't know total lines without reading the whole file
			startLine = offset
			if offset == 0 {
				// If no offset, we need to count total lines (expensive but necessary for accuracy)
				totalLines, _ := countTotalLines(reader)
				startLine = totalLines - int64(len(lines))
			}
		}

		for i, line := range lines {
			jsonLines[i] = LogLine{
				LineNumber: startLine + int64(i) + 1, // 1-based line numbers
				Content:    line,
			}
			if i < len(timestamps) {
				jsonLines[i].Timestamp = timestamps[i]
			}
		}

		ctx.JSON(http.StatusOK, jsonLines)
	} else {
		// Plain text format
		ctx.Resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, line := range lines {
			if _, err := fmt.Fprintln(ctx.Resp, line); err != nil {
				return err
			}
		}
	}

	return nil
}

// readHeadLines reads N lines from the log starting at line offset (0-based)
func readHeadLines(ctx *context.APIContext, task *actions_model.ActionTask, limit, offset int64) ([]string, []time.Time, error) {
	reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()

	// Seek to beginning
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, nil, err
	}

	scanner := bufio.NewScanner(reader)
	maxLineSize := 64*1024 + 100 // MaxLineSize + timestamp format length
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

	var lines []string
	var timestamps []time.Time
	var currentLine int64

	for scanner.Scan() {
		// Skip lines before offset
		if currentLine < offset {
			currentLine++
			continue
		}

		// Stop if we've read enough lines
		if limit >= 0 && int64(len(lines)) >= limit {
			break
		}

		lineText := scanner.Text()
		t, content, err := actions.ParseLog(lineText)
		if err == nil {
			lines = append(lines, content)
			timestamps = append(timestamps, t)
		}
		currentLine++
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return lines, timestamps, nil
}

// readTailLines reads the last N lines, optionally ending at a given line offset
func readTailLines(reader io.ReadSeekCloser, tail, offset int64) ([]string, []time.Time, error) {
	if offset > 0 {
		// When offset is specified with tail, we read N lines ending at line offset
		// E.g., offset=10&tail=5 returns lines 6-10 (1-based)
		// Convert to 0-based: we want lines at indices 5-9
		startLine := offset - tail
		if startLine < 0 {
			startLine = 0
		}
		// Seek to beginning and skip to startLine
		if _, err := reader.Seek(0, io.SeekStart); err != nil {
			return nil, nil, err
		}

		scanner := bufio.NewScanner(reader)
		maxLineSize := 64*1024 + 100
		scanner.Buffer(make([]byte, maxLineSize), maxLineSize)

		var lines []string
		var timestamps []time.Time
		var currentLine int64

		for scanner.Scan() {
			if currentLine < startLine {
				currentLine++
				continue
			}
			if currentLine >= offset {
				break
			}

			lineText := scanner.Text()
			t, content, err := actions.ParseLog(lineText)
			if err == nil {
				lines = append(lines, content)
				timestamps = append(timestamps, t)
			}
			currentLine++
		}

		if err := scanner.Err(); err != nil {
			return nil, nil, err
		}

		return lines, timestamps, nil
	}

	// No offset: read last N lines from end of file
	// Strategy: seek backwards from end and read lines
	size, err := reader.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, nil, err
	}

	const chunkSize = 8192 // Read in 8KB chunks
	var lines []string
	var timestamps []time.Time
	var leftover []byte
	seekPos := size

	for seekPos > 0 && int64(len(lines)) < tail {
		// Calculate how much to read
		readSize := chunkSize
		if seekPos < chunkSize {
			readSize = int(seekPos)
		}
		seekPos -= int64(readSize)

		// Seek to position and read chunk
		if _, err := reader.Seek(seekPos, io.SeekStart); err != nil {
			return nil, nil, err
		}

		chunk := make([]byte, readSize)
		n, err := reader.Read(chunk)
		if err != nil && err != io.EOF {
			return nil, nil, err
		}
		chunk = chunk[:n]

		// Combine with leftover from previous iteration
		if leftover != nil {
			chunk = append(chunk, leftover...)
		}

		// Split into lines (process from end to beginning)
		chunkLines := bytes.Split(chunk, []byte("\n"))

		// The first element might be incomplete (partial line)
		if seekPos > 0 {
			leftover = chunkLines[0]
			chunkLines = chunkLines[1:]
		} else {
			leftover = nil
		}

		// Process lines in reverse order (since we're reading backwards)
		for i := len(chunkLines) - 1; i >= 0; i-- {
			if len(chunkLines[i]) > 0 {
				lineStr := string(chunkLines[i])
				// Parse timestamp from log format
				t, content, err := actions.ParseLog(lineStr)
				if err == nil {
					lines = append([]string{content}, lines...)
					timestamps = append([]time.Time{t}, timestamps...)
					if int64(len(lines)) >= tail {
						break
					}
				}
			}
		}
	}

	// Handle any remaining leftover as the first line
	if len(leftover) > 0 && int64(len(lines)) < tail {
		lineStr := string(leftover)
		t, content, err := actions.ParseLog(lineStr)
		if err == nil {
			lines = append([]string{content}, lines...)
			timestamps = append([]time.Time{t}, timestamps...)
		}
	}

	// Trim to exact number of requested lines if we got more
	if int64(len(lines)) > tail {
		startIdx := len(lines) - int(tail)
		lines = lines[startIdx:]
		timestamps = timestamps[startIdx:]
	}

	return lines, timestamps, nil
}

// countTotalLines counts the total number of lines in the log file
func countTotalLines(reader io.ReadSeekCloser) (int64, error) {
	// Save current position
	currentPos, err := reader.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	defer func() {
		_, _ = reader.Seek(currentPos, io.SeekStart)
	}()

	// Count lines from beginning
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(reader)
	var count int64
	for scanner.Scan() {
		count++
	}

	return count, scanner.Err()
}

// Helper function to get run jobs for API endpoints
func getRunJobsForAPI(ctx *context.APIContext, runIndex, jobIndex int64) (*actions_model.ActionRunJob, []*actions_model.ActionRunJob) {
	run, err := actions_model.GetRunByIndex(ctx, ctx.Repo.Repository.ID, runIndex)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			ctx.NotFound("Run not found", err)
			return nil, nil
		}
		ctx.Error(http.StatusInternalServerError, "GetRunByIndex", err)
		return nil, nil
	}
	run.Repo = ctx.Repo.Repository

	jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetRunJobsByRunID", err)
		return nil, nil
	}

	if len(jobs) == 0 {
		ctx.NotFound("No jobs found", nil)
		return nil, nil
	}

	for _, v := range jobs {
		v.Run = run
	}

	if jobIndex >= 0 && jobIndex < int64(len(jobs)) {
		return jobs[jobIndex], jobs
	}

	// If jobIndex is out of range, return 404
	ctx.NotFound("Job not found", nil)
	return nil, nil
}
