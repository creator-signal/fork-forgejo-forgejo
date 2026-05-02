package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/organization"
	"forgejo.org/models/perm"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
)

// CreateAssignmentOptions contains options for creating a new assignment.
type CreateAssignmentOptions struct {
	CourseID         int64
	TaskName         string
	AllowedFilesGlob string
	Title            string
	Description      string
	DeadlineUnix     int64
}

// CreateCourseOptions contains options for creating a new course.
type CreateCourseOptions struct {
	Name        string
	Description string
	OrgID       int64
	StartUnix   int64
	EndUnix     int64
}

// EducationalService defines the business logic for the educational module.
type EducationalService interface {
	CreateAssignment(ctx context.Context, opts CreateAssignmentOptions) (*Assignment, error)
	GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error)
	GetAssignmentByCourseAndTask(ctx context.Context, courseID int64, taskName string) (*Assignment, error)
	GetAssignmentsForUser(ctx context.Context, userID int64) ([]*Assignment, error)
	UpdateAssignment(ctx context.Context, assignment *Assignment) error
	DeleteAssignment(ctx context.Context, id int64) error
	GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error)
	JoinAssignment(ctx context.Context, doer *user_model.User, assignmentID int64) (*Submission, error)

	CreateCourse(ctx context.Context, creatorID int64, opts CreateCourseOptions) (*Course, error)
	GetCourseByID(ctx context.Context, id int64) (*Course, error)
	GetCoursesForUser(ctx context.Context, userID int64) ([]*Course, error)
	UpdateCourse(ctx context.Context, course *Course) error
	DeleteCourse(ctx context.Context, id int64) error
	EnrollUser(ctx context.Context, courseID, userID int64, role RoleType) error
	GetEnrollments(ctx context.Context, courseID int64) ([]*CourseEnrollment, error)
	RemoveEnrollment(ctx context.Context, courseID, userID int64) error
	GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error)

	UploadCSV(ctx context.Context, courseID, creatorID int64, data []byte, mapping CSVColumnMapping) (*ImportDraft, error)
	GetImportDraft(ctx context.Context, id int64) (*ImportDraft, []*ImportDraftRow, error)
	UpdateDraftRow(ctx context.Context, rowID int64, username, email string) error
	ExecuteImport(ctx context.Context, draftID int64, doerID int64, defaultRole RoleType) (*ImportResult, error)
	DeleteImportDraft(ctx context.Context, id int64) error

	BulkForkForAssignment(ctx context.Context, assignmentID, doerID int64) (*BulkForkTask, error)
	GetBulkForkTask(ctx context.Context, taskID int64) (*BulkForkTask, error)
	GetBulkForkTaskByAssignment(ctx context.Context, assignmentID int64) (*BulkForkTask, error)

	SyncAllForks(ctx context.Context, assignmentID, doerID int64) (*SyncForkTask, error)
	GetSyncForkTask(ctx context.Context, taskID int64) (*SyncForkTask, error)
	GetSyncForkTaskByAssignment(ctx context.Context, assignmentID int64) (*SyncForkTask, error)

	GetTestResults(ctx context.Context, submissionID int64) ([]*TestResult, error)
	GetLatestTestResult(ctx context.Context, submissionID int64) (*TestResult, error)
	GradeSubmission(ctx context.Context, submissionID int64, grade int, comment string, gradedByID int64) error
	ResetToAutoGrade(ctx context.Context, submissionID int64) error
}

// RepoForker abstracts the repository forking, retrieval, and sync logic.
type RepoForker interface {
	ForkRepositoryAndUpdates(ctx context.Context, doer, owner *user_model.User, opts ForkRepoOptions) (*repo_model.Repository, error)
	GetRepositoryByID(ctx context.Context, id int64) (*repo_model.Repository, error)
	SyncFork(ctx context.Context, doer *user_model.User, forkRepo *repo_model.Repository, branch string) error
	GetDefaultBranch(ctx context.Context, repoID int64) (string, error)
}

// UserCreator abstracts user creation and lookup for import and bulk operations.
type UserCreator interface {
	CreateUser(ctx context.Context, username, email, password, fullName string) error
	GetUserByName(ctx context.Context, name string) (*user_model.User, error)
	GetUserByID(ctx context.Context, id int64) (*user_model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*user_model.User, error)
}

// OrgManager abstracts organization team management for enrollment.
type OrgManager interface {
	EnsureTeam(ctx context.Context, orgID int64, teamName string, accessMode perm.AccessMode) (*organization.Team, error)
	AddTeamMember(ctx context.Context, team *organization.Team, userID int64) error
	RemoveTeamMember(ctx context.Context, team *organization.Team, userID int64) error
	GetTeam(ctx context.Context, orgID int64, name string) (*organization.Team, error)
}

// ForkRepoOptions is a subset of options needed for forking.
type ForkRepoOptions struct {
	BaseRepo *repo_model.Repository
	Name     string
}

// private implementation
type service struct {
	repo   Repository
	forker RepoForker
	users  UserCreator
	orgs   OrgManager
}

// Repository defines the data access layer interface.
type Repository interface {
	CreateAssignment(ctx context.Context, assignment *Assignment) error
	GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error)
	GetAssignmentByCourseAndTask(ctx context.Context, courseID int64, taskName string) (*Assignment, error)
	GetAssignmentsForUser(ctx context.Context, userID int64) ([]*Assignment, error)
	UpdateAssignment(ctx context.Context, assignment *Assignment) error
	DeleteAssignment(ctx context.Context, id int64) error
	CreateSubmission(ctx context.Context, submission *Submission) error
	GetSubmission(ctx context.Context, assignmentID, userID int64) (*Submission, error)
	GetSubmissionByID(ctx context.Context, id int64) (*Submission, error)
	GetSubmissionByEnrollmentAssignment(ctx context.Context, enrollmentID, assignmentID int64) (*Submission, error)
	GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error)
	UpdateSubmission(ctx context.Context, submission *Submission) error

	CreateCourse(ctx context.Context, course *Course) error
	GetCourseByID(ctx context.Context, id int64) (*Course, error)
	GetCoursesByCreator(ctx context.Context, creatorID int64) ([]*Course, error)
	GetCoursesByUser(ctx context.Context, userID int64) ([]*Course, error)
	UpdateCourse(ctx context.Context, course *Course) error
	DeleteCourse(ctx context.Context, id int64) error
	GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error)

	EnrollUser(ctx context.Context, enrollment *CourseEnrollment) error
	GetEnrollment(ctx context.Context, courseID, userID int64) (*CourseEnrollment, error)
	GetEnrollmentByCourseUser(ctx context.Context, courseID, userID int64) (*CourseEnrollment, error)
	GetEnrollments(ctx context.Context, courseID int64) ([]*CourseEnrollment, error)
	RemoveEnrollment(ctx context.Context, courseID, userID int64) error

	CreateImportDraft(ctx context.Context, draft *ImportDraft) error
	GetImportDraft(ctx context.Context, id int64) (*ImportDraft, error)
	CreateImportDraftRows(ctx context.Context, rows []*ImportDraftRow) error
	GetImportDraftRows(ctx context.Context, draftID int64) ([]*ImportDraftRow, error)
	UpdateImportDraftRow(ctx context.Context, row *ImportDraftRow) error
	UpdateImportDraft(ctx context.Context, draft *ImportDraft) error
	DeleteImportDraft(ctx context.Context, id int64) error

	CreateInitForksTask(ctx context.Context, task *InitForksTask) error
	GetInitForksTask(ctx context.Context, id int64) (*InitForksTask, error)
	GetInitForksTaskByCourse(ctx context.Context, courseID int64) (*InitForksTask, error)
	UpdateInitForksTask(ctx context.Context, task *InitForksTask) error

	CreateSyncForkTask(ctx context.Context, task *SyncForkTask) error
	GetSyncForkTask(ctx context.Context, id int64) (*SyncForkTask, error)
	GetSyncForkTaskByAssignment(ctx context.Context, assignmentID int64) (*SyncForkTask, error)
	UpdateSyncForkTask(ctx context.Context, task *SyncForkTask) error

	CreateTestResult(ctx context.Context, tr *TestResult) error
	GetTestResultsBySubmission(ctx context.Context, submissionID int64) ([]*TestResult, error)
	GetLatestTestResult(ctx context.Context, submissionID int64) (*TestResult, error)
	GradeSubmission(ctx context.Context, submissionID int64, grade int, comment string, gradedByID int64) error
	AutoGradeSubmission(ctx context.Context, submissionID int64, grade int) error
	ResetToAutoGrade(ctx context.Context, submissionID int64, grade int) error
}

var globalService EducationalService

// GetService returns the singleton edu service instance. Must be called after Init().
func GetService() EducationalService {
	return globalService
}

// NewService creates a new instance of EducationalService.
// If the provided UserCreator also implements OrgManager, it is used for org team management.
func NewService(repo Repository, forker RepoForker, users ...UserCreator) EducationalService {
	s := &service{repo: repo, forker: forker}
	if len(users) > 0 {
		s.users = users[0]
		if o, ok := users[0].(OrgManager); ok {
			s.orgs = o
		}
	}
	return s
}

func (s *service) CreateAssignment(ctx context.Context, opts CreateAssignmentOptions) (*Assignment, error) {
	if opts.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	assignment := &Assignment{
		CourseID:         opts.CourseID,
		TaskName:         opts.TaskName,
		AllowedFilesGlob: opts.AllowedFilesGlob,
		Title:            opts.Title,
		Description:      opts.Description,
		DeadlineUnix:     opts.DeadlineUnix,
	}

	if err := s.repo.CreateAssignment(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	return assignment, nil
}

func (s *service) GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error) {
	return s.repo.GetAssignmentByID(ctx, id)
}

func (s *service) GetAssignmentByCourseAndTask(ctx context.Context, courseID int64, taskName string) (*Assignment, error) {
	return s.repo.GetAssignmentByCourseAndTask(ctx, courseID, taskName)
}

func (s *service) GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error) {
	return s.repo.GetSubmissions(ctx, assignmentID)
}
