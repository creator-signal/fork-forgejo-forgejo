package edu

import (
	"time"
)

// Assignment represents a task assigned to students.
type Assignment struct {
	ID           int64  `json:"id"`
	RepoID       int64  `json:"repo_id"` // ID of the template repository
	Title        string `json:"title"`
	Description  string `json:"description"`
	DeadlineUnix int64  `json:"deadline_unix"`
	CreatedUnix  int64  `json:"created_unix"`
	UpdatedUnix  int64  `json:"updated_unix"`
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
	ID            int64            `json:"id"`
	AssignmentID  int64            `json:"assignment_id"`
	UserID        int64            `json:"user_id"`         // Student ID
	StudentRepoID int64            `json:"student_repo_id"` // ID of the student's fork/repo
	Status        SubmissionStatus `json:"status"`
	CreatedUnix   int64            `json:"created_unix"`
	UpdatedUnix   int64            `json:"updated_unix"`
}

// TestResult represents the outcome of a CI run for a submission.
type TestResult struct {
	ID           int64  `json:"id"`
	SubmissionID int64  `json:"submission_id"`
	CommitSHA    string `json:"commit_sha"`
	Score        int    `json:"score"`
	Details      string `json:"details"` // JSON or plain text logs
	CreatedUnix  int64  `json:"created_unix"`
}
