// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"forgejo.org/models/unit"
	"forgejo.org/routers/api/v1/shared"
	"forgejo.org/services/context"
	project_service "forgejo.org/services/project"
)

// Create handler instance for repository projects
var repoProjectHandler = &shared.ProjectAPIHandler[project_service.RepoOwner]{
	GetOwner: func(ctx *context.APIContext) project_service.RepoOwner {
		return project_service.RepoOwner{Repo: ctx.Repo.Repository}
	},
	CanWrite: func(ctx *context.APIContext) bool {
		return ctx.Repo.CanWrite(unit.TypeProjects)
	},
	CanRead: func(ctx *context.APIContext) bool {
		return ctx.Repo.CanRead(unit.TypeProjects)
	},
}

// ListProjects lists the projects in a repository
func ListProjects(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects repository repoListProjects
	// ---
	// summary: List a repository's projects
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

	repoProjectHandler.ListProjects(ctx)
}

// GetProject gets a project by its ID or title
func GetProject(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id} repository repoGetProject
	// ---
	// summary: Get a project
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

	repoProjectHandler.GetProject(ctx)
}

// CreateProject creates a project for a repository
func CreateProject(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects repository repoCreateProject
	// ---
	// summary: Create a project
	// consumes:
	// - application/json
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

	repoProjectHandler.CreateProject(ctx)
}

// UpdateProject updates a project
func UpdateProject(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id} repository repoEditProject
	// ---
	// summary: Update a project
	// consumes:
	// - application/json
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

	repoProjectHandler.UpdateProject(ctx)
}

// DeleteProject deletes a project
func DeleteProject(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/projects/{id} repository repoDeleteProject
	// ---
	// summary: Delete a project
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

	repoProjectHandler.DeleteProject(ctx)
}

// ListProjectColumns lists the columns in a project
func ListProjectColumns(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id}/columns repository repoListProjectColumns
	// ---
	// summary: Get columns of a project
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

	repoProjectHandler.ListProjectColumns(ctx)
}

// CreateProjectColumn creates a new project column
func CreateProjectColumn(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects/{id}/columns repository repoCreateProjectColumn
	// ---
	// summary: Create a project column
	// consumes:
	// - application/json
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

	repoProjectHandler.CreateProjectColumn(ctx)
}

// UpdateProjectColumn updates a project column
func UpdateProjectColumn(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id}/columns/{column_id} repository repoEditProjectColumn
	// ---
	// summary: Update a project column
	// consumes:
	// - application/json
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

	repoProjectHandler.UpdateProjectColumn(ctx)
}

// DeleteProjectColumn deletes a project column
func DeleteProjectColumn(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/projects/{id}/columns/{column_id} repository repoDeleteProjectColumn
	// ---
	// summary: Delete a project column
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

	repoProjectHandler.DeleteProjectColumn(ctx)
}

// MoveColumns moves columns in a project
func MoveColumns(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects/{id}/columns/move repository repoMoveColumns
	// ---
	// summary: Move project columns
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

	repoProjectHandler.MoveColumns(ctx)
}

// ListColumnCards lists the cards in a project column
func ListColumnCards(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/projects/{id}/columns/{column_id}/cards repository repoListColumnCards
	// ---
	// summary: List cards in a project column
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

	repoProjectHandler.ListColumnCards(ctx)
}

// AddCardToColumn adds a card to a project column
func AddCardToColumn(ctx *context.APIContext) {
	// swagger:operation POST /repos/{owner}/{repo}/projects/{id}/columns/{column_id}/cards repository repoCreateCard
	// ---
	// summary: Add a card to a project column
	// consumes:
	// - application/json
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

	repoProjectHandler.AddCardToColumn(ctx)
}

// ReorderCardsInColumn reorders cards in a project column
func ReorderCardsInColumn(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id}/columns/{column_id}/cards/reorder repository repoReorderCards
	// ---
	// summary: Reorder cards in a project column
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

	repoProjectHandler.ReorderCardsInColumn(ctx)
}

// MoveProjectCard moves a project card to a different column or position
func MoveProjectCard(ctx *context.APIContext) {
	// swagger:operation PATCH /repos/{owner}/{repo}/projects/{id}/cards/{card_id} repository repoMoveCard
	// ---
	// summary: Move a project card
	// consumes:
	// - application/json
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

	repoProjectHandler.MoveProjectCard(ctx)
}

// DeleteProjectCard deletes a project card
func DeleteProjectCard(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/projects/{id}/cards/{card_id} repository repoDeleteCard
	// ---
	// summary: Delete a project card
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

	repoProjectHandler.DeleteProjectCard(ctx)
}
