// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package shared

import (
	"fmt"
	"net/http"
	"strconv"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	project_service "forgejo.org/services/project"
)

// ProjectAPIHandler provides generic project operations for any owner type
type ProjectAPIHandler[T project_service.Owner] struct {
	GetOwner func(*context.APIContext) T
	CanWrite func(*context.APIContext) bool
	CanRead  func(*context.APIContext) bool
}

// ListProjects lists projects for any owner type
func (h *ProjectAPIHandler[T]) ListProjects(ctx *context.APIContext) {
	if !h.CanRead(ctx) {
		ctx.Error(http.StatusForbidden, "ListProjects", "no read permission")
		return
	}

	owner := h.GetOwner(ctx)
	listOptions := utils.GetListOptions(ctx)

	// Parse query parameters
	state := ctx.FormString("state")
	keyword := ctx.FormString("q")
	sortType := ctx.FormString("sort")

	// Validate sort type
	if sortType != "" && !project_model.IsValidSortType(sortType) {
		ctx.Error(http.StatusUnprocessableEntity, "InvalidSortType", fmt.Errorf("invalid sort type: %s", sortType))
		return
	}

	// Determine IsClosed filter based on state parameter
	var isClosed optional.Option[bool]
	switch state {
	case "closed":
		isClosed = optional.Some(true)
	case "all":
		isClosed = optional.None[bool]()
	default: // "open" or empty
		isClosed = optional.Some(false)
	}

	opts := project_model.SearchOptions{
		ListOptions: listOptions,
		Type:        owner.GetProjectType(),
		Title:       keyword,
		OrderBy:     project_model.GetSearchOrderByBySortType(sortType),
		IsClosed:    isClosed,
	}

	// Set the appropriate ID field
	if owner.GetProjectType() == project_model.TypeRepository {
		opts.RepoID = owner.GetRepoID()
	} else {
		opts.OwnerID = owner.GetOwnerID()
	}

	// Use existing generic db function
	projects, total, err := db.FindAndCount[project_model.Project](ctx, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FindProjects", err)
		return
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, convert.ToAPIProjectList(ctx, projects))
}

// GetProject gets a single project
func (h *ProjectAPIHandler[T]) GetProject(ctx *context.APIContext) {
	if !h.CanRead(ctx) {
		ctx.Error(http.StatusForbidden, "GetProject", "no read permission")
		return
	}

	owner := h.GetOwner(ctx)

	// Support both ID and title
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	if !project_service.ValidateProjectOwner(project, owner) {
		ctx.NotFound("Project", nil)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProject(ctx, project))
}

// CreateProject creates a new project
func (h *ProjectAPIHandler[T]) CreateProject(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "CreateProject", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	form := web.GetForm(ctx).(*api.CreateProjectOption)

	opts := project_service.CreateProjectOptions{
		Title:        form.Title,
		Description:  form.Body,
		TemplateType: project_model.TemplateType(form.TemplateType),
		CanWrite:     h.CanWrite(ctx),
	}

	project, err := project_service.CreateProjectGeneric(ctx, owner, ctx.Doer, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateProject", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIProject(ctx, project))
}

// UpdateProject updates a project
func (h *ProjectAPIHandler[T]) UpdateProject(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "UpdateProject", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	if !project_service.ValidateProjectOwner(project, owner) {
		ctx.NotFound("Project", nil)
		return
	}

	form := web.GetForm(ctx).(*api.EditProjectOption)
	opts := project_service.UpdateProjectOptions{
		Title:       form.Title,
		Description: form.Body,
	}

	if form.State != nil {
		isClosed := *form.State == api.StateClosed
		opts.IsClosed = &isClosed
	}

	if err = project_service.UpdateProject(ctx, project, opts); err != nil {
		ctx.Error(http.StatusInternalServerError, "UpdateProject", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProject(ctx, project))
}

// DeleteProject deletes a project
func (h *ProjectAPIHandler[T]) DeleteProject(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "DeleteProject", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	if !project_service.ValidateProjectOwner(project, owner) {
		ctx.NotFound("Project", nil)
		return
	}

	if err := project_service.DeleteProject(ctx, project); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteProject", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// ListProjectColumns lists columns in a project
func (h *ProjectAPIHandler[T]) ListProjectColumns(ctx *context.APIContext) {
	if !h.CanRead(ctx) {
		ctx.Error(http.StatusForbidden, "ListProjectColumns", "no read permission")
		return
	}

	owner := h.GetOwner(ctx)
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	if !project_service.ValidateProjectOwner(project, owner) {
		ctx.NotFound("Project", nil)
		return
	}

	columns, err := project.GetColumns(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetProjectColumns", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumnList(ctx, columns))
}

// CreateProjectColumn creates a new column in a project
func (h *ProjectAPIHandler[T]) CreateProjectColumn(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "CreateProjectColumn", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	if !project_service.ValidateProjectOwner(project, owner) {
		ctx.NotFound("Project", nil)
		return
	}

	form := web.GetForm(ctx).(*api.CreateProjectColumnOption)

	opts := project_service.CreateColumnOptions{
		Title: form.Title,
	}

	column, err := project_service.CreateColumn(ctx, project, ctx.Doer, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateColumn", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIProjectColumn(ctx, column))
}

// UpdateProjectColumn updates a project column
func (h *ProjectAPIHandler[T]) UpdateProjectColumn(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "UpdateProjectColumn", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	_, column := h.getProjectAndColumn(ctx, owner)
	if ctx.Written() {
		return
	}

	form := web.GetForm(ctx).(*api.EditProjectColumnOption)

	opts := project_service.UpdateColumnOptions{
		Title: form.Title,
	}

	if err := project_service.UpdateColumn(ctx, column, opts); err != nil {
		ctx.Error(http.StatusInternalServerError, "UpdateColumn", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectColumn(ctx, column))
}

// DeleteProjectColumn deletes a project column
func (h *ProjectAPIHandler[T]) DeleteProjectColumn(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "DeleteProjectColumn", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	_, column := h.getProjectAndColumn(ctx, owner)
	if ctx.Written() {
		return
	}

	if err := project_service.DeleteColumn(ctx, column); err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteColumn", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// Helper methods

// getProjectByIDOrTitle gets a project by ID or title
func (h *ProjectAPIHandler[T]) getProjectByIDOrTitle(ctx *context.APIContext, idOrTitle string, owner T) (*project_model.Project, error) {
	switch owner.GetProjectType() {
	case project_model.TypeRepository:
		return project_model.GetProjectForRepoByIDOrTitle(ctx, owner.GetRepoID(), idOrTitle)
	case project_model.TypeOrganization:
		return project_model.GetProjectForOrgByIDOrTitle(ctx, owner.GetOwnerID(), idOrTitle)
	default:
		return nil, project_model.ErrProjectNotExist{ID: 0}
	}
}

// getProjectAndColumn gets both project and column with validation
func (h *ProjectAPIHandler[T]) getProjectAndColumn(ctx *context.APIContext, owner T) (*project_model.Project, *project_model.Column) {
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return nil, nil
	}

	if !project_service.ValidateProjectOwner(project, owner) {
		ctx.NotFound("Project", nil)
		return nil, nil
	}

	columnID := ctx.ParamsInt64("column_id")
	column, err := project_model.GetColumn(ctx, columnID)
	if err != nil {
		ctx.NotFoundOrServerError("GetColumn", project_model.IsErrProjectColumnNotExist, err)
		return nil, nil
	}

	if column.ProjectID != project.ID {
		ctx.Error(http.StatusUnprocessableEntity, "ProjectColumn", "column does not belong to project")
		return nil, nil
	}

	return project, column
}

// ListColumnCards lists cards in a project column
func (h *ProjectAPIHandler[T]) ListColumnCards(ctx *context.APIContext) {
	owner := h.GetOwner(ctx)
	project, column := h.getProjectAndColumn(ctx, owner)
	if project == nil || column == nil {
		return
	}

	projectIssues, err := project_model.GetProjectCardsInColumn(ctx, column.ID, db.ListOptions{})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetProjectCardsInColumn", err)
		return
	}

	// Collect issue IDs and batch fetch issues
	issueIDs := make([]int64, len(projectIssues))
	for i, pi := range projectIssues {
		issueIDs[i] = pi.IssueID
	}

	issues, err := issues_model.GetIssuesByIDs(ctx, issueIDs)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetIssuesByIDs", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectCardList(ctx, ctx.Doer, projectIssues, issues))
}

// AddCardToColumn adds a card to a project column
func (h *ProjectAPIHandler[T]) AddCardToColumn(ctx *context.APIContext) {
	owner := h.GetOwner(ctx)
	_, column := h.getProjectAndColumn(ctx, owner)
	if column == nil {
		return
	}

	form := web.GetForm(ctx).(*api.AddCardToColumnOption)

	// Add the issue to the project column using service layer
	projectIssue, err := project_service.AddCardToColumn(ctx, column, form.IssueID, 0)
	if err != nil {
		if project_model.IsErrCardAlreadyInProject(err) {
			ctx.Error(http.StatusUnprocessableEntity, "AddCardToColumn", "Issue is already in this project")
		} else if project_model.IsErrCardNotInProjectRepo(err) {
			ctx.Error(http.StatusUnprocessableEntity, "AddCardToColumn", "Issue does not belong to project repository")
		} else {
			ctx.Error(http.StatusInternalServerError, "AddCardToColumn", err)
		}
		return
	}

	// Load the issue for API response
	issue, err := issues_model.GetIssueByID(ctx, form.IssueID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetIssueByID", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToAPIProjectCard(ctx, ctx.Doer, projectIssue, issue))
}

// ReorderCardsInColumn reorders cards in a project column
func (h *ProjectAPIHandler[T]) ReorderCardsInColumn(ctx *context.APIContext) {
	owner := h.GetOwner(ctx)
	_, column := h.getProjectAndColumn(ctx, owner)
	if column == nil {
		return
	}

	form := web.GetForm(ctx).(*api.ReorderCardsInColumnOption)

	// Validate that card positions are provided
	if len(form.CardPositions) == 0 {
		ctx.Error(http.StatusBadRequest, "EmptyCardPositions", "At least one card position must be provided")
		return
	}

	// Convert API card positions (CardID) to service card positions (IssueID)
	cardPositions := make([]project_service.CardPosition, 0, len(form.CardPositions))
	for _, cardPos := range form.CardPositions {
		// Find the issue ID for this card ID
		projectIssue := &project_model.ProjectIssue{}
		has, err := db.GetEngine(ctx).ID(cardPos.CardID).Get(projectIssue)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetProjectIssue", err)
			return
		}
		if !has {
			ctx.Error(http.StatusNotFound, "CardNotFound", fmt.Sprintf("Card %d not found", cardPos.CardID))
			return
		}
		cardPositions = append(cardPositions, project_service.CardPosition{
			IssueID: projectIssue.IssueID,
			Sorting: cardPos.Position,
		})
	}

	// Use service layer for consistency with web frontend and to get validation
	if err := project_service.ReorderCardsInColumn(ctx, column, cardPositions); err != nil {
		if project_model.IsErrSomeCardsNotExist(err) {
			ctx.Error(http.StatusNotFound, "ReorderCardsInColumn", err)
		} else if project_model.IsErrCardNotInProjectRepo(err) {
			ctx.Error(http.StatusUnprocessableEntity, "ReorderCardsInColumn", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "ReorderCardsInColumn", err)
		}
		return
	}

	ctx.Status(http.StatusNoContent)
}

// MoveProjectCard moves a project card to a different column or position
func (h *ProjectAPIHandler[T]) MoveProjectCard(ctx *context.APIContext) {
	owner := h.GetOwner(ctx)
	_, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	cardID, err := strconv.ParseInt(ctx.Params("card_id"), 10, 64)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "ParseCardID", "Invalid card ID")
		return
	}

	form := web.GetForm(ctx).(*api.MoveProjectCardOption)

	// Handle pointer fields from form
	var columnID, position int64
	if form.ColumnID != nil {
		columnID = *form.ColumnID
	}
	if form.Position != nil {
		position = *form.Position
	}

	// Move the card to the new column and position
	if err := project_model.MoveCardToColumn(ctx, cardID, columnID, position); err != nil {
		if project_model.IsErrProjectCardNotExist(err) {
			ctx.NotFound("MoveProjectCard", err)
		} else if project_model.IsErrProjectColumnNotExist(err) {
			ctx.Error(http.StatusUnprocessableEntity, "MoveProjectCard", "Invalid column")
		} else {
			ctx.Error(http.StatusInternalServerError, "MoveCardToColumn", err)
		}
		return
	}

	// Get the updated project issue
	projectIssue := &project_model.ProjectIssue{}
	has, err := db.GetEngine(ctx).ID(cardID).Get(projectIssue)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetProjectIssue", err)
		return
	}
	if !has {
		ctx.NotFound("GetProjectIssue", nil)
		return
	}

	// Load the issue
	issue, err := issues_model.GetIssueByID(ctx, projectIssue.IssueID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetIssueByID", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToAPIProjectCard(ctx, ctx.Doer, projectIssue, issue))
}

// DeleteProjectCard deletes a project card
func (h *ProjectAPIHandler[T]) DeleteProjectCard(ctx *context.APIContext) {
	owner := h.GetOwner(ctx)
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	cardID, err := strconv.ParseInt(ctx.Params("card_id"), 10, 64)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "ParseCardID", "Invalid card ID")
		return
	}

	// Get the project issue to find the issue ID
	projectIssue := &project_model.ProjectIssue{}
	has, err := db.GetEngine(ctx).ID(cardID).Get(projectIssue)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetProjectIssue", err)
		return
	}
	if !has {
		ctx.NotFound("GetProjectIssue", nil)
		return
	}

	// Verify the card belongs to this project
	if projectIssue.ProjectID != project.ID {
		ctx.NotFound("DeleteProjectCard", nil)
		return
	}

	// Remove the issue from the project using service layer
	if err := project_service.RemoveCardFromProject(ctx, project, projectIssue.IssueID); err != nil {
		ctx.Error(http.StatusInternalServerError, "RemoveCardFromProject", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// MoveColumns moves columns in a project
func (h *ProjectAPIHandler[T]) MoveColumns(ctx *context.APIContext) {
	if !h.CanWrite(ctx) {
		ctx.Error(http.StatusForbidden, "MoveColumns", "no write permission")
		return
	}

	owner := h.GetOwner(ctx)
	project, err := h.getProjectByIDOrTitle(ctx, ctx.Params("id"), owner)
	if err != nil {
		ctx.NotFoundOrServerError("GetProject", project_model.IsErrProjectNotExist, err)
		return
	}

	form := web.GetForm(ctx).(*api.MoveProjectColumnsOption)

	// Validate that all column positions are provided
	if len(form.Columns) == 0 {
		ctx.Error(http.StatusBadRequest, "EmptyColumnPositions", "At least one column position must be provided")
		return
	}

	// Convert to the format expected by the model layer
	sortedColumnIDs := make(map[int64]int64)
	for _, columnPos := range form.Columns {
		sortedColumnIDs[columnPos.Position] = columnPos.ColumnID
	}

	// Verify all columns belong to the project
	for _, columnPos := range form.Columns {
		column, err := project_model.GetColumn(ctx, columnPos.ColumnID)
		if err != nil {
			if project_model.IsErrProjectColumnNotExist(err) {
				ctx.Error(http.StatusNotFound, "GetColumn", fmt.Sprintf("Column %d not found", columnPos.ColumnID))
			} else {
				ctx.Error(http.StatusInternalServerError, "GetColumn", err)
			}
			return
		}
		if column.ProjectID != project.ID {
			ctx.Error(http.StatusBadRequest, "InvalidColumn", fmt.Sprintf("Column %d does not belong to project %d", columnPos.ColumnID, project.ID))
			return
		}
	}

	// Move the columns
	if err := project_model.MoveColumnsOnProject(ctx, project, sortedColumnIDs); err != nil {
		ctx.Error(http.StatusInternalServerError, "MoveColumnsOnProject", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
