// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"
)

// Project represents a kanban board
type Project struct {
	ID           int64               `json:"id"`
	Title        string              `json:"title"`
	Body         string              `json:"body"`
	State        StateType           `json:"state"`
	TemplateType ProjectTemplateType `json:"template_type"`
	CardType     ProjectCardType     `json:"card_type"`
	Type         ProjectType         `json:"type"`

	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated *time.Time `json:"updated_at"`
	// swagger:strfmt date-time
	Closed *time.Time `json:"closed_at"`

	// Counts
	ColumnCount  int `json:"column_count"`
	OpenIssues   int `json:"open_issues"`
	ClosedIssues int `json:"closed_issues"`

	// Repository info
	Repository *Repository `json:"repository,omitempty"`
}

// ProjectMeta basic project information
// swagger:model
type ProjectMeta struct {
	ID    int64       `json:"id"`
	Title string      `json:"title"`
	State StateType   `json:"state"`
	Type  ProjectType `json:"type"`
}

// ProjectColumn represents a column in a kanban board
type ProjectColumn struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Color   string `json:"color"`
	Default bool   `json:"default"`
	Sorting int8   `json:"sorting"`

	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
	// swagger:strfmt date-time
	Updated *time.Time `json:"updated_at"`

	// Counts
	CardCount int `json:"card_count"`
}

// ProjectCard represents an issue/card on a kanban board
type ProjectCard struct {
	ID      int64          `json:"id"`
	Sorting int64          `json:"sorting"`
	Issue   *Issue         `json:"issue"`
	Column  *ProjectColumn `json:"column,omitempty"`
	Project *Project       `json:"project,omitempty"`
}

// ProjectType represents the type of project
type ProjectType int

const (
	// ProjectTypeIndividual is a project that is owned by an individual
	ProjectTypeIndividual ProjectType = iota + 1
	// ProjectTypeRepository is a project that is tied to a repository
	ProjectTypeRepository
	// ProjectTypeOrganization is a project that is tied to an organization
	ProjectTypeOrganization
)

// ProjectTemplateType represents a project template type
type ProjectTemplateType int

const (
	// ProjectTemplateTypeNone is a project template type that has no predefined columns
	ProjectTemplateTypeNone ProjectTemplateType = iota
	// ProjectTemplateTypeBasicKanban is a project template type that has basic predefined columns
	ProjectTemplateTypeBasicKanban
	// ProjectTemplateTypeBugTriage is a project template type that has predefined columns suited to hunting down bugs
	ProjectTemplateTypeBugTriage
)

// ProjectCardType represents a project column card type
type ProjectCardType int

const (
	// ProjectCardTypeTextOnly is a project column card type that is text only
	ProjectCardTypeTextOnly ProjectCardType = iota
	// ProjectCardTypeImagesAndText is a project column card type that has images and text
	ProjectCardTypeImagesAndText
)

// CreateProjectOption represents the options for creating a project
type CreateProjectOption struct {
	// required: true
	Title        string              `json:"title" binding:"Required"`
	Body         string              `json:"body"`
	TemplateType ProjectTemplateType `json:"template_type"`
}

// EditProjectOption represents the options for editing a project
type EditProjectOption struct {
	Title *string    `json:"title"`
	Body  *string    `json:"body"`
	State *StateType `json:"state"`
}

// CreateProjectColumnOption represents the options for creating a project column
type CreateProjectColumnOption struct {
	// required: true
	Title string `json:"title" binding:"Required"`
	Color string `json:"color"`
}

// EditProjectColumnOption represents the options for editing a project column
type EditProjectColumnOption struct {
	Title *string `json:"title"`
	Color *string `json:"color"`
}

// ColumnPosition represents a column and its new position
type ColumnPosition struct {
	// required: true
	ColumnID int64 `json:"column_id" binding:"Required"`
	// Position is 0-indexed
	Position int64 `json:"position" binding:"min=0"`
}

// MoveProjectColumnsOption represents the options for reordering project columns
type MoveProjectColumnsOption struct {
	// required: true
	Columns []ColumnPosition `json:"columns" binding:"Required"`
}

// AddCardToColumnOption represents the options for adding a card to a column
type AddCardToColumnOption struct {
	// required: true
	IssueID  int64  `json:"issue_id" binding:"Required"`
	Position *int64 `json:"position"`
}

// MoveProjectCardOption represents the options for moving a project card
type MoveProjectCardOption struct {
	ColumnID *int64 `json:"column_id"`
	Position *int64 `json:"position"`
}

// ReorderCardsInColumnOption represents the options for reordering cards within a column
type ReorderCardsInColumnOption struct {
	// required: true
	CardPositions []CardPosition `json:"card_positions" binding:"Required"`
}

// CardPosition represents a card's new position
type CardPosition struct {
	// required: true
	CardID int64 `json:"card_id" binding:"Required"`
	// required: true
	Position int64 `json:"position" binding:"min=0"`
}

// ProjectSearchOptions represents the options for searching projects
type ProjectSearchOptions struct {
	Query string       `json:"query"`
	State *StateType   `json:"state"`
	Type  *ProjectType `json:"type"`
	Sort  string       `json:"sort"`
}
