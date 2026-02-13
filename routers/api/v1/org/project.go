// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package org

import (
	"forgejo.org/models/perm"
	"forgejo.org/models/unit"
	"forgejo.org/routers/api/v1/shared"
	"forgejo.org/services/context"
	project_service "forgejo.org/services/project"
)

// Create handler instance for organization projects
var orgProjectHandler = &shared.ProjectAPIHandler[project_service.OrgOwner]{
	GetOwner: func(ctx *context.APIContext) project_service.OrgOwner {
		return project_service.OrgOwner{Org: ctx.Org.Organization}
	},
	CanWrite: func(ctx *context.APIContext) bool {
		if ctx.Org.Organization == nil || ctx.Doer == nil {
			return false
		}
		return ctx.Org.Organization.UnitPermission(ctx, ctx.Doer, unit.TypeProjects) >= perm.AccessModeWrite
	},
	CanRead: func(ctx *context.APIContext) bool {
		if ctx.Org.Organization == nil || ctx.Doer == nil {
			return false
		}
		return ctx.Org.Organization.UnitPermission(ctx, ctx.Doer, unit.TypeProjects) >= perm.AccessModeRead
	},
}

// ListProjects lists the projects in an organization
func ListProjects(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects organization orgListProjects
	// ---
	// summary: List an organization's projects
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: state
	//   in: query
	//   description: whether project is open or closed
	//   type: string
	//   enum: [closed, open, all]
	// - name: q
	//   in: query
	//   description: search projects by title
	//   type: string
	// - name: sort
	//   in: query
	//   description: sort projects by created time, alphabetically, etc
	//   type: string
	//   enum: [newest, oldest, alphabetically, reversealphabetically, recentupdate, leastupdate]
	// - name: since
	//   in: query
	//   description: Only show projects updated after the given time (RFC 3339 format)
	//   type: string
	//   format: date-time
	// - name: before
	//   in: query
	//   description: Only show projects updated before the given time (RFC 3339 format)
	//   type: string
	//   format: date-time
	// - name: created_by
	//   in: query
	//   description: Only show projects created by the given user
	//   type: string
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
	//     "$ref": "#/responses/ProjectList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.ListProjects(ctx)
}

// GetProject gets a project by its ID or title
func GetProject(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{id} organization orgGetProject
	// ---
	// summary: Get a project
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project to get, identified by ID and if not available by title
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.GetProject(ctx)
}

// CreateProject creates a project for an organization
func CreateProject(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects organization orgCreateProject
	// ---
	// summary: Create a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateProjectOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	orgProjectHandler.CreateProject(ctx)
}

// UpdateProject updates a project
func UpdateProject(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{id} organization orgEditProject
	// ---
	// summary: Update a project
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project, identified by ID and if not available by title
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditProjectOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/Project"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	orgProjectHandler.UpdateProject(ctx)
}

// DeleteProject deletes a project
func DeleteProject(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{id} organization orgDeleteProject
	// ---
	// summary: Delete a project
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project, identified by ID and if not available by title
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.DeleteProject(ctx)
}

// ListProjectColumns lists the columns in a project
func ListProjectColumns(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{id}/columns organization orgListProjectColumns
	// ---
	// summary: Get columns of a project
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumnList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.ListProjectColumns(ctx)
}

// CreateProjectColumn creates a new project column
func CreateProjectColumn(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{id}/columns organization orgCreateProjectColumn
	// ---
	// summary: Create a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateProjectColumnOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/ProjectColumn"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	orgProjectHandler.CreateProjectColumn(ctx)
}

// UpdateProjectColumn updates a project column
func UpdateProjectColumn(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{id}/columns/{column_id} organization orgEditProjectColumn
	// ---
	// summary: Update a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/EditProjectColumnOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectColumn"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	orgProjectHandler.UpdateProjectColumn(ctx)
}

// DeleteProjectColumn deletes a project column
func DeleteProjectColumn(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{id}/columns/{column_id} organization orgDeleteProjectColumn
	// ---
	// summary: Delete a project column
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.DeleteProjectColumn(ctx)
}

// MoveColumns moves columns in a project
func MoveColumns(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{id}/columns/move organization orgMoveColumns
	// ---
	// summary: Move project columns
	// consumes:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MoveProjectColumnsOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.MoveColumns(ctx)
}

// ListColumnCards lists the cards in a project column
func ListColumnCards(ctx *context.APIContext) {
	// swagger:operation GET /orgs/{org}/projects/{id}/columns/{column_id}/cards organization orgListColumnCards
	// ---
	// summary: Get cards in a project column
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectCardList"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.ListColumnCards(ctx)
}

// AddCardToColumn adds a card to a project column
func AddCardToColumn(ctx *context.APIContext) {
	// swagger:operation POST /orgs/{org}/projects/{id}/columns/{column_id}/cards organization orgCreateCard
	// ---
	// summary: Create a card in a project column
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/AddCardToColumnOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/ProjectCard"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	orgProjectHandler.AddCardToColumn(ctx)
}

// ReorderCardsInColumn reorders cards in a project column
func ReorderCardsInColumn(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{id}/columns/{column_id}/cards/reorder organization orgReorderCards
	// ---
	// summary: Reorder cards in a project column
	// consumes:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: column_id
	//   in: path
	//   description: id of the column
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/ReorderCardsInColumnOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.ReorderCardsInColumn(ctx)
}

// MoveProjectCard moves a project card to a different column or position
func MoveProjectCard(ctx *context.APIContext) {
	// swagger:operation PATCH /orgs/{org}/projects/{id}/cards/{card_id} organization orgMoveCard
	// ---
	// summary: Move a project card
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: card_id
	//   in: path
	//   description: id of the card
	//   type: integer
	//   format: int64
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/MoveProjectCardOption"
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectCard"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	orgProjectHandler.MoveProjectCard(ctx)
}

// DeleteProjectCard deletes a project card
func DeleteProjectCard(ctx *context.APIContext) {
	// swagger:operation DELETE /orgs/{org}/projects/{id}/cards/{card_id} organization orgDeleteCard
	// ---
	// summary: Delete a project card
	// parameters:
	// - name: org
	//   in: path
	//   description: name of the organization
	//   type: string
	//   required: true
	// - name: id
	//   in: path
	//   description: id of the project
	//   type: string
	//   required: true
	// - name: card_id
	//   in: path
	//   description: id of the card
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "404":
	//     "$ref": "#/responses/notFound"

	orgProjectHandler.DeleteProjectCard(ctx)
}
