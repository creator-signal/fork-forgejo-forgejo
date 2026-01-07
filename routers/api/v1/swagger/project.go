// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgejo.org/modules/structs"
)

// Project
// swagger:response Project
type swaggerResponseProject struct {
	// in:body
	Body api.Project `json:"body"`
}

// ProjectList
// swagger:response ProjectList
type swaggerResponseProjectList struct {
	// in:body
	Body []api.Project `json:"body"`

	// The total number of projects
	TotalCount int64 `json:"X-Total-Count"`
}

// ProjectColumn
// swagger:response ProjectColumn
type swaggerResponseProjectColumn struct {
	// in:body
	Body api.ProjectColumn `json:"body"`
}

// ProjectColumnList
// swagger:response ProjectColumnList
type swaggerResponseProjectColumnList struct {
	// in:body
	Body []api.ProjectColumn `json:"body"`
}

// ProjectCard
// swagger:response ProjectCard
type swaggerResponseProjectCard struct {
	// in:body
	Body api.ProjectCard `json:"body"`
}

// ProjectCardList
// swagger:response ProjectCardList
type swaggerResponseProjectCardList struct {
	// in:body
	Body []api.ProjectCard `json:"body"`
}
