package edu

import (
	"time"
)

// Course represents an educational course (учебный поток).
type Course struct {
	ID          int64  `json:"id" xorm:"pk autoincr"`
	Name        string `json:"name" xorm:"VARCHAR(255) NOT NULL"`
	Description string `json:"description" xorm:"TEXT"`
	CreatorID   int64  `json:"creator_id" xorm:"INDEX NOT NULL"`
	OrgID       int64  `json:"org_id" xorm:"INDEX"`
	StartUnix   int64  `json:"start_unix"`
	EndUnix     int64  `json:"end_unix"`
	CreatedUnix int64  `json:"created_unix" xorm:"created"`
	UpdatedUnix int64  `json:"updated_unix" xorm:"updated"`
}

func (*Course) TableName() string { return "edu_courses" }

func (c *Course) StartDate() time.Time {
	return time.Unix(c.StartUnix, 0)
}

func (c *Course) EndDate() time.Time {
	return time.Unix(c.EndUnix, 0)
}

func (c *Course) IsActive() bool {
	if c.EndUnix == 0 {
		return true
	}
	return time.Now().Unix() < c.EndUnix
}

// CourseEnrollment represents a user's participation in a course.
type CourseEnrollment struct {
	ID          int64    `json:"id" xorm:"pk autoincr"`
	CourseID    int64    `json:"course_id" xorm:"INDEX NOT NULL"`
	UserID      int64    `json:"user_id" xorm:"INDEX NOT NULL"`
	Role        RoleType `json:"role" xorm:"VARCHAR(20) NOT NULL"`
	CreatedUnix int64    `json:"created_unix" xorm:"created"`
}

func (*CourseEnrollment) TableName() string { return "edu_course_enrollments" }

// Assignment represents a task assigned to students.
type Assignment struct {
	ID           int64  `json:"id" xorm:"pk autoincr"`
	CourseID     int64  `json:"course_id" xorm:"INDEX"`
	RepoID       int64  `json:"repo_id" xorm:"INDEX NOT NULL"`
	Title        string `json:"title" xorm:"VARCHAR(255) NOT NULL"`
	Description  string `json:"description" xorm:"TEXT"`
	DeadlineUnix int64  `json:"deadline_unix"`
	CreatedUnix  int64  `json:"created_unix" xorm:"created"`
	UpdatedUnix  int64  `json:"updated_unix" xorm:"updated"`
}

func (*Assignment) TableName() string { return "edu_assignments" }

// Deadline returns the deadline as a time.Time object.
func (a *Assignment) Deadline() time.Time {
	return time.Unix(a.DeadlineUnix, 0)
}

// SubmissionStatus represents the status of a student's work.
type SubmissionStatus string

const (
	StatusStarted   SubmissionStatus = "started"
	StatusSubmitted SubmissionStatus = "submitted"
	StatusGraded    SubmissionStatus = "graded"
	StatusPassed    SubmissionStatus = "passed"
	StatusFailed    SubmissionStatus = "failed"
)

// Submission represents a student's attempt at an assignment.
type Submission struct {
	ID            int64            `json:"id" xorm:"pk autoincr"`
	AssignmentID  int64            `json:"assignment_id" xorm:"INDEX NOT NULL"`
	UserID        int64            `json:"user_id" xorm:"INDEX NOT NULL"`
	StudentRepoID int64            `json:"student_repo_id"`
	Status        SubmissionStatus `json:"status" xorm:"VARCHAR(50) NOT NULL DEFAULT 'started'"`
	Grade         int              `json:"grade" xorm:"DEFAULT -1"`
	Comment       string           `json:"comment" xorm:"TEXT"`
	GradedByID    int64            `json:"graded_by_id" xorm:"INDEX"`
	GradedUnix    int64            `json:"graded_unix"`
	CreatedUnix   int64            `json:"created_unix" xorm:"created"`
	UpdatedUnix   int64            `json:"updated_unix" xorm:"updated"`
}

func (*Submission) TableName() string { return "edu_submissions" }

// IsGraded returns true if the submission has been graded.
func (s *Submission) IsGraded() bool {
	return s.Grade >= 0
}

// TestResult represents the outcome of a CI run for a submission.
type TestResult struct {
	ID           int64  `json:"id" xorm:"pk autoincr"`
	SubmissionID int64  `json:"submission_id" xorm:"INDEX NOT NULL"`
	CommitSHA    string `json:"commit_sha" xorm:"VARCHAR(64) NOT NULL"`
	Score        int    `json:"score" xorm:"DEFAULT 0"`
	Details      string `json:"details" xorm:"TEXT"`
	CreatedUnix  int64  `json:"created_unix" xorm:"created"`
}

func (*TestResult) TableName() string { return "edu_test_results" }

// RoleType represents the role of a user in the educational system.
type RoleType string

const (
	RoleStudent RoleType = "student"
	RoleTeacher RoleType = "teacher"
	RoleAdmin   RoleType = "admin"
)

// UserRole represents the global role of a user.
type UserRole struct {
	ID          int64    `json:"id" xorm:"pk autoincr"`
	UserID      int64    `json:"user_id" xorm:"INDEX UNIQUE NOT NULL"`
	Role        RoleType `json:"role" xorm:"VARCHAR(20) NOT NULL"`
	CreatedUnix int64    `json:"created_unix" xorm:"created"`
	UpdatedUnix int64    `json:"updated_unix" xorm:"updated"`
}

// ImportDraft represents a CSV import session.
type ImportDraft struct {
	ID          int64  `json:"id" xorm:"pk autoincr"`
	CourseID    int64  `json:"course_id" xorm:"INDEX NOT NULL"`
	CreatorID   int64  `json:"creator_id" xorm:"NOT NULL"`
	Status      string `json:"status" xorm:"VARCHAR(20) NOT NULL DEFAULT 'draft'"`
	RawCSV      string `json:"raw_csv" xorm:"TEXT"`
	CreatedUnix int64  `json:"created_unix" xorm:"created"`
}

func (*ImportDraft) TableName() string { return "edu_import_draft" }

// ImportDraftRow represents a single row within an import draft.
type ImportDraftRow struct {
	ID          int64  `json:"id" xorm:"pk autoincr"`
	DraftID     int64  `json:"draft_id" xorm:"INDEX NOT NULL"`
	FullName    string `json:"full_name" xorm:"VARCHAR(255)"`
	Email       string `json:"email" xorm:"VARCHAR(255)"`
	Group       string `json:"group_name" xorm:"VARCHAR(100)"`
	Username    string `json:"username" xorm:"VARCHAR(255)"`
	Role        string `json:"role" xorm:"VARCHAR(20) NOT NULL DEFAULT 'student'"`
	Status      string `json:"status" xorm:"VARCHAR(20) NOT NULL DEFAULT 'pending'"`
	ErrorMsg    string `json:"error_msg" xorm:"TEXT"`
	CreatedUnix int64  `json:"created_unix" xorm:"created"`
}

func (*ImportDraftRow) TableName() string { return "edu_import_draft_row" }

// ImportResult holds the result of executing an import.
type ImportResult struct {
	TotalRows    int
	Created      int
	AlreadyExist int
	Errors       int
	Credentials  []UserCredential
}

// UserCredential holds generated credentials for a newly created user.
type UserCredential struct {
	Username string
	Password string
	FullName string
	Email    string
}

// BulkForkTask tracks a mass fork operation for an assignment.
type BulkForkTask struct {
	ID           int64  `json:"id" xorm:"pk autoincr"`
	AssignmentID int64  `json:"assignment_id" xorm:"INDEX NOT NULL"`
	CreatorID    int64  `json:"creator_id" xorm:"NOT NULL"`
	TotalUsers   int    `json:"total_users" xorm:"NOT NULL DEFAULT 0"`
	Completed    int    `json:"completed" xorm:"NOT NULL DEFAULT 0"`
	Failed       int    `json:"failed" xorm:"NOT NULL DEFAULT 0"`
	Status       string `json:"status" xorm:"VARCHAR(20) NOT NULL DEFAULT 'pending'"`
	ErrorLog     string `json:"error_log" xorm:"TEXT"`
	CreatedUnix  int64  `json:"created_unix" xorm:"created"`
	UpdatedUnix  int64  `json:"updated_unix" xorm:"updated"`
}

// SyncForkTask tracks a mass fork sync operation for an assignment.
type SyncForkTask struct {
	ID           int64  `json:"id" xorm:"pk autoincr"`
	AssignmentID int64  `json:"assignment_id" xorm:"INDEX NOT NULL"`
	CreatorID    int64  `json:"creator_id" xorm:"NOT NULL"`
	TotalRepos   int    `json:"total_repos" xorm:"NOT NULL DEFAULT 0"`
	Synced       int    `json:"synced" xorm:"NOT NULL DEFAULT 0"`
	Skipped      int    `json:"skipped" xorm:"NOT NULL DEFAULT 0"`
	Failed       int    `json:"failed" xorm:"NOT NULL DEFAULT 0"`
	Status       string `json:"status" xorm:"VARCHAR(20) NOT NULL DEFAULT 'pending'"`
	ErrorLog     string `json:"error_log" xorm:"TEXT"`
	CreatedUnix  int64  `json:"created_unix" xorm:"created"`
	UpdatedUnix  int64  `json:"updated_unix" xorm:"updated"`
}
