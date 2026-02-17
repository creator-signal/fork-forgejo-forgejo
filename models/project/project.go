// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"fmt"
	"html/template"
	"strconv"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

type (
	// CardConfig is used to identify the type of column card that is being used
	CardConfig struct {
		CardType    CardType
		Translation string
	}

	// Type is used to identify the type of project in question and ownership
	Type uint8
)

const (
	// TypeIndividual is a type of project column that is owned by an individual
	TypeIndividual Type = iota + 1

	// TypeRepository is a project that is tied to a repository
	TypeRepository

	// TypeOrganization is a project that is tied to an organisation
	TypeOrganization
)

// ErrProjectNotExist represents a "ProjectNotExist" kind of error.
type ErrProjectNotExist struct {
	ID     int64
	RepoID int64
}

// IsErrProjectNotExist checks if an error is a ErrProjectNotExist
func IsErrProjectNotExist(err error) bool {
	_, ok := err.(ErrProjectNotExist)
	return ok
}

func (err ErrProjectNotExist) Error() string {
	return fmt.Sprintf("projects does not exist [id: %d]", err.ID)
}

func (err ErrProjectNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrProjectColumnNotExist represents a "ProjectColumnNotExist" kind of error.
type ErrProjectColumnNotExist struct {
	ColumnID int64
}

// IsErrProjectColumnNotExist checks if an error is a ErrProjectColumnNotExist
func IsErrProjectColumnNotExist(err error) bool {
	_, ok := err.(ErrProjectColumnNotExist)
	return ok
}

func (err ErrProjectColumnNotExist) Error() string {
	return fmt.Sprintf("project column does not exist [id: %d]", err.ColumnID)
}

func (err ErrProjectColumnNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrProjectCardNotExist represents a "ProjectCardNotExist" kind of error.
type ErrProjectCardNotExist struct {
	CardID    int64
	ProjectID int64
	IssueID   int64
}

// IsErrProjectCardNotExist checks if an error is a ErrProjectCardNotExist
func IsErrProjectCardNotExist(err error) bool {
	_, ok := err.(ErrProjectCardNotExist)
	return ok
}

func (err ErrProjectCardNotExist) Error() string {
	if err.CardID > 0 {
		return fmt.Sprintf("project card does not exist [card_id: %d]", err.CardID)
	}
	return fmt.Sprintf("project card does not exist [project_id: %d, issue_id: %d]", err.ProjectID, err.IssueID)
}

func (err ErrProjectCardNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrCardAlreadyInProject represents a "CardAlreadyInProject" kind of error.
type ErrCardAlreadyInProject struct {
	ProjectID int64
	IssueID   int64
}

// IsErrCardAlreadyInProject checks if an error is a ErrCardAlreadyInProject
func IsErrCardAlreadyInProject(err error) bool {
	_, ok := err.(ErrCardAlreadyInProject)
	return ok
}

func (err ErrCardAlreadyInProject) Error() string {
	return fmt.Sprintf("issue already exists in project [project_id: %d, issue_id: %d]", err.ProjectID, err.IssueID)
}

func (err ErrCardAlreadyInProject) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrSomeCardsNotExist represents a "SomeCardsNotExist" kind of error.
type ErrSomeCardsNotExist struct {
	MissingCount int
}

// IsErrSomeCardsNotExist checks if an error is a ErrSomeCardsNotExist
func IsErrSomeCardsNotExist(err error) bool {
	_, ok := err.(ErrSomeCardsNotExist)
	return ok
}

func (err ErrSomeCardsNotExist) Error() string {
	return fmt.Sprintf("some cards do not exist [missing: %d]", err.MissingCount)
}

func (err ErrSomeCardsNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrCardNotInProjectRepo represents a "CardNotInProjectRepo" kind of error.
type ErrCardNotInProjectRepo struct {
	IssueID       int64
	ProjectRepoID int64
}

// IsErrCardNotInProjectRepo checks if an error is a ErrCardNotInProjectRepo
func IsErrCardNotInProjectRepo(err error) bool {
	_, ok := err.(ErrCardNotInProjectRepo)
	return ok
}

func (err ErrCardNotInProjectRepo) Error() string {
	return fmt.Sprintf("card issue %d does not belong to project repository %d", err.IssueID, err.ProjectRepoID)
}

func (err ErrCardNotInProjectRepo) Unwrap() error {
	return util.ErrInvalidArgument
}

// Project represents a project
type Project struct {
	ID           int64                  `xorm:"pk autoincr"`
	Title        string                 `xorm:"INDEX NOT NULL"`
	Description  string                 `xorm:"TEXT"`
	OwnerID      int64                  `xorm:"INDEX"`
	Owner        *user_model.User       `xorm:"-"`
	RepoID       int64                  `xorm:"INDEX"`
	Repo         *repo_model.Repository `xorm:"-"`
	CreatorID    int64                  `xorm:"NOT NULL"`
	IsClosed     bool                   `xorm:"INDEX"`
	TemplateType TemplateType           `xorm:"'board_type'"` // TODO: rename the column to template_type
	CardType     CardType
	Type         Type

	RenderedContent template.HTML `xorm:"-"`

	CreatedUnix    timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix    timeutil.TimeStamp `xorm:"INDEX updated"`
	ClosedDateUnix timeutil.TimeStamp
}

// Ghost Project is a project which has been deleted
const GhostProjectID = -1

func (p *Project) IsGhost() bool {
	return p.ID == GhostProjectID
}

func (p *Project) LoadOwner(ctx context.Context) (err error) {
	if p.Owner != nil {
		return nil
	}
	p.Owner, err = user_model.GetUserByID(ctx, p.OwnerID)
	return err
}

func (p *Project) LoadRepo(ctx context.Context) (err error) {
	if p.RepoID == 0 || p.Repo != nil {
		return nil
	}
	p.Repo, err = repo_model.GetRepositoryByID(ctx, p.RepoID)
	return err
}

func ProjectLinkForOrg(org *user_model.User, projectID int64) string { //nolint
	return fmt.Sprintf("%s/-/projects/%d", org.HomeLink(), projectID)
}

func ProjectLinkForRepo(repo *repo_model.Repository, projectID int64) string { //nolint
	return fmt.Sprintf("%s/projects/%d", repo.Link(), projectID)
}

// Link returns the project's relative URL.
func (p *Project) Link(ctx context.Context) string {
	if p.OwnerID > 0 {
		err := p.LoadOwner(ctx)
		if err != nil {
			log.Error("LoadOwner: %v", err)
			return ""
		}
		return ProjectLinkForOrg(p.Owner, p.ID)
	}
	if p.RepoID > 0 {
		err := p.LoadRepo(ctx)
		if err != nil {
			log.Error("LoadRepo: %v", err)
			return ""
		}
		return ProjectLinkForRepo(p.Repo, p.ID)
	}
	return ""
}

func (p *Project) IconName() string {
	if p.IsRepositoryProject() {
		return "octicon-project"
	}
	return "octicon-project-symlink"
}

func (p *Project) IsOrganizationProject() bool {
	return p.Type == TypeOrganization
}

func (p *Project) IsRepositoryProject() bool {
	return p.Type == TypeRepository
}

func (p *Project) CanBeAccessedByOwnerRepo(ownerID int64, repo *repo_model.Repository) bool {
	if p.Type == TypeRepository {
		return repo != nil && p.RepoID == repo.ID // if a project belongs to a repository, then its OwnerID is 0 and can be ignored
	}
	return p.OwnerID == ownerID && p.RepoID == 0
}

func (p *Project) State() api.StateType {
	if p.IsClosed {
		return api.StateClosed
	}
	return api.StateOpen
}

func init() {
	db.RegisterModel(new(Project))
}

// GetCardConfig retrieves the types of configurations project column cards could have
//
//llu:returnsTrKey
func GetCardConfig() []CardConfig {
	return []CardConfig{
		{CardTypeTextOnly, "repo.projects.card_type.text_only"},
		{CardTypeImagesAndText, "repo.projects.card_type.images_and_text"},
	}
}

// IsTypeValid checks if a project type is valid
func IsTypeValid(p Type) bool {
	switch p {
	case TypeIndividual, TypeRepository, TypeOrganization:
		return true
	default:
		return false
	}
}

// SearchOptions are options for GetProjects
type SearchOptions struct {
	db.ListOptions
	OwnerID  int64
	RepoID   int64
	IsClosed optional.Option[bool]
	OrderBy  db.SearchOrderBy
	Type     Type
	Title    string
}

func (opts SearchOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if has, value := opts.IsClosed.Get(); has {
		cond = cond.And(builder.Eq{"is_closed": value})
	}

	if opts.Type > 0 {
		cond = cond.And(builder.Eq{"type": opts.Type})
	}
	if opts.OwnerID > 0 {
		cond = cond.And(builder.Eq{"owner_id": opts.OwnerID})
	}

	if len(opts.Title) != 0 {
		cond = cond.And(db.BuildCaseInsensitiveLike("title", opts.Title))
	}
	return cond
}

func (opts SearchOptions) ToOrders() string {
	return opts.OrderBy.String()
}

// ValidProjectSortTypes contains all valid sort type values for projects
var ValidProjectSortTypes = []string{
	"oldest", "newest", "alphabetically", "reversealphabetically",
	"recentupdate", "leastupdate", "",
}

// IsValidSortType returns true if the given sort type is valid for projects
func IsValidSortType(sortType string) bool {
	for _, valid := range ValidProjectSortTypes {
		if sortType == valid {
			return true
		}
	}
	return false
}

func GetSearchOrderByBySortType(sortType string) db.SearchOrderBy {
	switch sortType {
	case "oldest":
		return db.SearchOrderByOldest
	case "recentupdate":
		return db.SearchOrderByRecentUpdated
	case "leastupdate":
		return db.SearchOrderByLeastUpdated
	case "alphabetically":
		return "title ASC"
	case "reversealphabetically":
		return "title DESC"
	case "newest", "":
		return db.SearchOrderByNewest
	default:
		// This should not happen if IsValidSortType is called first
		return db.SearchOrderByNewest
	}
}

// NewProject creates a new Project
// The title will be cut off at 255 characters if it's longer than 255 characters.
func NewProject(ctx context.Context, p *Project) error {
	if !IsTemplateTypeValid(p.TemplateType) {
		p.TemplateType = TemplateTypeNone
	}

	if !IsCardTypeValid(p.CardType) {
		p.CardType = CardTypeTextOnly
	}

	if !IsTypeValid(p.Type) {
		return util.NewInvalidArgumentErrorf("project type is not valid")
	}

	p.Title, _ = util.SplitStringAtByteN(p.Title, 255)

	return db.WithTx(ctx, func(ctx context.Context) error {
		if err := db.Insert(ctx, p); err != nil {
			return err
		}

		if p.RepoID > 0 {
			if _, err := db.Exec(ctx, "UPDATE `repository` SET num_projects = num_projects + 1 WHERE id = ?", p.RepoID); err != nil {
				return err
			}
		}

		return createDefaultColumnsForProject(ctx, p)
	})
}

// GetProjectByID returns the projects in a repository
func GetProjectByID(ctx context.Context, id int64) (*Project, error) {
	p := new(Project)

	has, err := db.GetEngine(ctx).ID(id).Get(p)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrProjectNotExist{ID: id}
	}

	return p, nil
}

// GetProjectsByIDs fetches multiple projects by their IDs in a single query
func GetProjectsByIDs(ctx context.Context, projectIDs []int64) (map[int64]*Project, error) {
	result := make(map[int64]*Project, len(projectIDs))
	if len(projectIDs) == 0 {
		return result, nil
	}
	projects := make([]*Project, 0, len(projectIDs))
	if err := db.GetEngine(ctx).In("id", projectIDs).Find(&projects); err != nil {
		return nil, err
	}
	for _, p := range projects {
		result[p.ID] = p
	}
	return result, nil
}

// GetProjectForRepoByID returns the projects in a repository
func GetProjectForRepoByID(ctx context.Context, repoID, id int64) (*Project, error) {
	p := new(Project)
	has, err := db.GetEngine(ctx).Where("id=? AND repo_id=?", id, repoID).Get(p)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrProjectNotExist{ID: id}
	}
	return p, nil
}

// GetProjectForUserByID returns the project by id that belongs to the specified user.
func GetProjectForUserByID(ctx context.Context, uid, id int64) (*Project, error) {
	p := new(Project)
	has, err := db.GetEngine(ctx).Where("id=? AND owner_id=?", id, uid).Get(p)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrProjectNotExist{ID: id}
	}
	return p, nil
}

// GetProjectForOrgByID returns a project by ID with org ownership validation
func GetProjectForOrgByID(ctx context.Context, orgID, id int64) (*Project, error) {
	p := new(Project)
	has, err := db.GetEngine(ctx).Where("id=? AND owner_id=? AND type=?", id, orgID, TypeOrganization).Get(p)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrProjectNotExist{ID: id}
	}
	return p, nil
}

// UpdateProject updates project properties
func UpdateProject(ctx context.Context, p *Project) error {
	if !IsCardTypeValid(p.CardType) {
		p.CardType = CardTypeTextOnly
	}

	p.Title, _ = util.SplitStringAtByteN(p.Title, 255)
	_, err := db.GetEngine(ctx).ID(p.ID).Cols(
		"title",
		"description",
		"card_type",
		"is_closed",
		"closed_date_unix",
	).Update(p)
	return err
}

func updateRepositoryProjectCount(ctx context.Context, repoID int64) error {
	if _, err := db.GetEngine(ctx).Exec(builder.Update(
		builder.Eq{
			"`num_projects`": builder.Select("count(*)").From("`project`").
				Where(builder.Eq{"`project`.`repo_id`": repoID}.
					And(builder.Eq{"`project`.`type`": TypeRepository})),
		}).From("`repository`").Where(builder.Eq{"id": repoID})); err != nil {
		return err
	}

	if _, err := db.GetEngine(ctx).Exec(builder.Update(
		builder.Eq{
			"`num_closed_projects`": builder.Select("count(*)").From("`project`").
				Where(builder.Eq{"`project`.`repo_id`": repoID}.
					And(builder.Eq{"`project`.`type`": TypeRepository}).
					And(builder.Eq{"`project`.`is_closed`": true})),
		}).From("`repository`").Where(builder.Eq{"id": repoID})); err != nil {
		return err
	}
	return nil
}

// ChangeProjectStatus changes the status of the specified project to the state
// specified via the `isClosed` argument.
func ChangeProjectStatus(ctx context.Context, p *Project, isClosed bool) error {
	if p.IsClosed == isClosed {
		return nil
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		p.IsClosed = isClosed
		if isClosed {
			p.ClosedDateUnix = timeutil.TimeStampNow()
		} else {
			p.ClosedDateUnix = 0
		}
		count, err := db.GetEngine(ctx).ID(p.ID).Cols("is_closed", "closed_date_unix").Update(p)
		if err != nil {
			return err
		}
		if count < 1 {
			return nil
		}

		return updateRepositoryProjectCount(ctx, p.RepoID)
	})
}

// DeleteProjectByID deletes a project from a repository. if it's not in a database
// transaction, it will start a new database transaction
func DeleteProjectByID(ctx context.Context, id int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		p, err := GetProjectByID(ctx, id)
		if err != nil {
			if IsErrProjectNotExist(err) {
				return nil
			}
			return err
		}

		if err := deleteProjectIssuesByProjectID(ctx, id); err != nil {
			return err
		}

		if err := deleteColumnByProjectID(ctx, id); err != nil {
			return err
		}

		if _, err = db.GetEngine(ctx).ID(p.ID).Delete(new(Project)); err != nil {
			return err
		}

		return updateRepositoryProjectCount(ctx, p.RepoID)
	})
}

func DeleteProjectByRepoID(ctx context.Context, repoID int64) error {
	switch {
	case setting.Database.Type.IsSQLite3():
		if _, err := db.GetEngine(ctx).Exec("DELETE FROM project_issue WHERE project_issue.id IN (SELECT project_issue.id FROM project_issue INNER JOIN project WHERE project.id = project_issue.project_id AND project.repo_id = ?)", repoID); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Exec("DELETE FROM project_board WHERE project_board.id IN (SELECT project_board.id FROM project_board INNER JOIN project WHERE project.id = project_board.project_id AND project.repo_id = ?)", repoID); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Table("project").Where("repo_id = ? ", repoID).Delete(&Project{}); err != nil {
			return err
		}
	case setting.Database.Type.IsPostgreSQL():
		if _, err := db.GetEngine(ctx).Exec("DELETE FROM project_issue USING project WHERE project.id = project_issue.project_id AND project.repo_id = ? ", repoID); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Exec("DELETE FROM project_board USING project WHERE project.id = project_board.project_id AND project.repo_id = ? ", repoID); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Table("project").Where("repo_id = ? ", repoID).Delete(&Project{}); err != nil {
			return err
		}
	default:
		if _, err := db.GetEngine(ctx).Exec("DELETE project_issue FROM project_issue INNER JOIN project ON project.id = project_issue.project_id WHERE project.repo_id = ? ", repoID); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Exec("DELETE project_board FROM project_board INNER JOIN project ON project.id = project_board.project_id WHERE project.repo_id = ? ", repoID); err != nil {
			return err
		}
		if _, err := db.GetEngine(ctx).Table("project").Where("repo_id = ? ", repoID).Delete(&Project{}); err != nil {
			return err
		}
	}

	return updateRepositoryProjectCount(ctx, repoID)
}

// GetProjectForRepoByIDOrTitle gets a project for a repository by ID and if not available by title
func GetProjectForRepoByIDOrTitle(ctx context.Context, repoID int64, idOrTitle string) (*Project, error) {
	// Try to parse as ID first
	if id, err := strconv.ParseInt(idOrTitle, 10, 64); err == nil {
		return GetProjectForRepoByID(ctx, repoID, id)
	}

	// Fall back to title lookup
	project := &Project{}
	has, err := db.GetEngine(ctx).Where("repo_id = ? AND type = ? AND title = ?", repoID, TypeRepository, idOrTitle).Get(project)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectNotExist{RepoID: repoID}
	}
	return project, nil
}

// GetProjectForOrgByIDOrTitle gets a project for an organization by ID and if not available by title
func GetProjectForOrgByIDOrTitle(ctx context.Context, orgID int64, idOrTitle string) (*Project, error) {
	// Try to parse as ID first
	if id, err := strconv.ParseInt(idOrTitle, 10, 64); err == nil {
		return GetProjectForOrgByID(ctx, orgID, id)
	}

	// Fall back to title lookup
	project := &Project{}
	has, err := db.GetEngine(ctx).Where("owner_id = ? AND type = ? AND title = ?", orgID, TypeOrganization, idOrTitle).Get(project)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectNotExist{ID: 0}
	}
	return project, nil
}
