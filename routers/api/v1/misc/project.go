// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"
	"strings"

	project_model "forgejo.org/models/project"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/context"
)

// templateKeyFromConfig extracts the template key (e.g. "basic_kanban") from
// a TemplateConfig's translation key (e.g. "repo.projects.type.basic_kanban").
func templateKeyFromConfig(cfg project_model.TemplateConfig) string {
	parts := strings.Split(cfg.Translation, ".")
	return parts[len(parts)-1]
}

// ListProjectTemplates returns the names of available project templates
func ListProjectTemplates(ctx *context.APIContext) {
	// swagger:operation GET /project/templates miscellaneous listProjectTemplates
	// ---
	// summary: Returns a list of all project template names
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectTemplateList"
	configs := project_model.GetTemplateConfigs()
	result := make([]string, 0, len(configs))
	for _, cfg := range configs {
		if cfg.TemplateType == project_model.TemplateTypeNone {
			continue
		}
		result = append(result, templateKeyFromConfig(cfg))
	}
	ctx.JSON(http.StatusOK, result)
}

// GetProjectTemplate returns the detail for a project template
func GetProjectTemplate(ctx *context.APIContext) {
	// swagger:operation GET /project/templates/{name} miscellaneous getProjectTemplate
	// ---
	// summary: Returns the columns of a project template
	// produces:
	// - application/json
	// parameters:
	// - name: name
	//   in: path
	//   description: name of the template
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/ProjectTemplateInfo"
	//   "404":
	//     "$ref": "#/responses/notFound"
	name := ctx.Params("name")

	configs := project_model.GetTemplateConfigs()
	for _, cfg := range configs {
		if cfg.TemplateType == project_model.TemplateTypeNone {
			continue
		}
		if templateKeyFromConfig(cfg) == name {
			ctx.JSON(http.StatusOK, buildProjectTemplate(cfg))
			return
		}
	}

	ctx.NotFound()
}

// buildProjectTemplate creates a ProjectTemplate response for a given config.
// The first column is always "Backlog" (the default for new issues), followed
// by the template-specific columns from settings.
func buildProjectTemplate(cfg project_model.TemplateConfig) api.ProjectTemplate {
	var items []string
	switch cfg.TemplateType {
	case project_model.TemplateTypeBasicKanban:
		items = setting.Project.ProjectBoardBasicKanbanType
	case project_model.TemplateTypeBugTriage:
		items = setting.Project.ProjectBoardBugTriageType
	}

	columns := make([]api.ProjectTemplateColumn, 0, 1+len(items))
	columns = append(columns, api.ProjectTemplateColumn{
		Title:   "Backlog",
		Default: true,
	})
	for _, title := range items {
		columns = append(columns, api.ProjectTemplateColumn{
			Title:   title,
			Default: false,
		})
	}

	return api.ProjectTemplate{
		Key:     templateKeyFromConfig(cfg),
		Columns: columns,
	}
}
