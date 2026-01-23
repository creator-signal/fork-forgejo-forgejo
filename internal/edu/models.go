package edu

import (
	"time"
)

// Assignment represents a task assigned to students.
type Assignment struct {
	ID           int64  `json:"id" xorm:"pk autoincr"`
	RepoID       int64  `json:"repo_id" xorm:"INDEX NOT NULL"`
	Title        string `json:"title" xorm:"VARCHAR(255) NOT NULL"`
	Description  string `json:"description" xorm:"TEXT"`
	DeadlineUnix int64  `json:"deadline_unix"`
	CreatedUnix  int64  `json:"created_unix" xorm:"created"`
	UpdatedUnix  int64  `json:"updated_unix" xorm:"updated"`
}

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
	CreatedUnix   int64            `json:"created_unix" xorm:"created"`
	UpdatedUnix   int64            `json:"updated_unix" xorm:"updated"`
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
