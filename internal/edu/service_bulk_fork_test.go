package edu

import (
	"context"
	"fmt"
	"testing"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBulkForkForAssignment_AllStudents(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, CourseID: 10, RepoID: 100}
	baseRepo := &repo_model.Repository{ID: 100, Name: "homework1"}

	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetRepositoryByID", mock.Anything, int64(100)).Return(baseRepo, nil)

	enrollments := []*CourseEnrollment{
		{ID: 1, CourseID: 10, UserID: 200, Role: RoleStudent},
		{ID: 2, CourseID: 10, UserID: 201, Role: RoleStudent},
	}
	mockRepo.On("GetEnrollments", mock.Anything, int64(10)).Return(enrollments, nil)

	mockRepo.On("CreateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*BulkForkTask)
			task.ID = 1
		}).Return(nil)

	doer := &user_model.User{ID: 50, Name: "teacher1"}
	student1 := &user_model.User{ID: 200, Name: "student1"}
	student2 := &user_model.User{ID: 201, Name: "student2"}

	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)
	mockUsers.On("GetUserByID", mock.Anything, int64(200)).Return(student1, nil)
	mockUsers.On("GetUserByID", mock.Anything, int64(201)).Return(student2, nil)

	// No existing submissions
	mockRepo.On("GetSubmission", mock.Anything, int64(1), int64(200)).Return(nil, nil)
	mockRepo.On("GetSubmission", mock.Anything, int64(1), int64(201)).Return(nil, nil)

	forkedRepo1 := &repo_model.Repository{ID: 301, Name: "student1-homework1"}
	forkedRepo2 := &repo_model.Repository{ID: 302, Name: "student2-homework1"}

	mockForker.On("ForkRepositoryAndUpdates", mock.Anything, doer, student1, ForkRepoOptions{BaseRepo: baseRepo, Name: "student1-homework1"}).Return(forkedRepo1, nil)
	mockForker.On("ForkRepositoryAndUpdates", mock.Anything, doer, student2, ForkRepoOptions{BaseRepo: baseRepo, Name: "student2-homework1"}).Return(forkedRepo2, nil)

	mockRepo.On("CreateSubmission", mock.Anything, mock.AnythingOfType("*edu.Submission")).Return(nil).Times(2)
	mockRepo.On("UpdateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).Return(nil)

	task, err := svc.BulkForkForAssignment(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, 2, task.TotalUsers)
	assert.Equal(t, 2, task.Completed)
	assert.Equal(t, 0, task.Failed)
	assert.Equal(t, "done", task.Status)
}

func TestBulkForkForAssignment_SkipsExistingSubmissions(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, CourseID: 10, RepoID: 100}
	baseRepo := &repo_model.Repository{ID: 100, Name: "hw1"}

	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetRepositoryByID", mock.Anything, int64(100)).Return(baseRepo, nil)

	enrollments := []*CourseEnrollment{
		{ID: 1, CourseID: 10, UserID: 200, Role: RoleStudent},
	}
	mockRepo.On("GetEnrollments", mock.Anything, int64(10)).Return(enrollments, nil)

	mockRepo.On("CreateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*BulkForkTask)
			task.ID = 1
		}).Return(nil)

	doer := &user_model.User{ID: 50, Name: "teacher"}
	student := &user_model.User{ID: 200, Name: "student1"}

	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)
	mockUsers.On("GetUserByID", mock.Anything, int64(200)).Return(student, nil)

	// Submission already exists
	existingSub := &Submission{ID: 10, AssignmentID: 1, UserID: 200}
	mockRepo.On("GetSubmission", mock.Anything, int64(1), int64(200)).Return(existingSub, nil)

	mockRepo.On("UpdateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).Return(nil)

	task, err := svc.BulkForkForAssignment(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.Equal(t, 1, task.TotalUsers)
	assert.Equal(t, 1, task.Completed)
	assert.Equal(t, 0, task.Failed)
	assert.Equal(t, "done", task.Status)

	// Fork should NOT have been called
	mockForker.AssertNotCalled(t, "ForkRepositoryAndUpdates", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestBulkForkForAssignment_ForkError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, CourseID: 10, RepoID: 100}
	baseRepo := &repo_model.Repository{ID: 100, Name: "hw1"}

	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetRepositoryByID", mock.Anything, int64(100)).Return(baseRepo, nil)

	enrollments := []*CourseEnrollment{
		{ID: 1, CourseID: 10, UserID: 200, Role: RoleStudent},
		{ID: 2, CourseID: 10, UserID: 201, Role: RoleStudent},
	}
	mockRepo.On("GetEnrollments", mock.Anything, int64(10)).Return(enrollments, nil)

	mockRepo.On("CreateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*BulkForkTask)
			task.ID = 1
		}).Return(nil)

	doer := &user_model.User{ID: 50, Name: "teacher"}
	student1 := &user_model.User{ID: 200, Name: "student1"}
	student2 := &user_model.User{ID: 201, Name: "student2"}

	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)
	mockUsers.On("GetUserByID", mock.Anything, int64(200)).Return(student1, nil)
	mockUsers.On("GetUserByID", mock.Anything, int64(201)).Return(student2, nil)

	mockRepo.On("GetSubmission", mock.Anything, int64(1), int64(200)).Return(nil, nil)
	mockRepo.On("GetSubmission", mock.Anything, int64(1), int64(201)).Return(nil, nil)

	// First fork fails
	mockForker.On("ForkRepositoryAndUpdates", mock.Anything, doer, student1, ForkRepoOptions{BaseRepo: baseRepo, Name: "student1-hw1"}).
		Return(nil, fmt.Errorf("disk full"))

	// Second fork succeeds
	forkedRepo2 := &repo_model.Repository{ID: 302, Name: "student2-hw1"}
	mockForker.On("ForkRepositoryAndUpdates", mock.Anything, doer, student2, ForkRepoOptions{BaseRepo: baseRepo, Name: "student2-hw1"}).
		Return(forkedRepo2, nil)

	mockRepo.On("CreateSubmission", mock.Anything, mock.AnythingOfType("*edu.Submission")).Return(nil)
	mockRepo.On("UpdateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).Return(nil)

	task, err := svc.BulkForkForAssignment(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.Equal(t, 2, task.TotalUsers)
	assert.Equal(t, 1, task.Completed)
	assert.Equal(t, 1, task.Failed)
	assert.Equal(t, "error", task.Status)
	assert.Contains(t, task.ErrorLog, "student1")
	assert.Contains(t, task.ErrorLog, "disk full")
}

func TestBulkForkForAssignment_EmptyCourse(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, CourseID: 10, RepoID: 100}
	baseRepo := &repo_model.Repository{ID: 100, Name: "hw1"}

	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetRepositoryByID", mock.Anything, int64(100)).Return(baseRepo, nil)

	// No enrollments
	mockRepo.On("GetEnrollments", mock.Anything, int64(10)).Return([]*CourseEnrollment{}, nil)

	mockRepo.On("CreateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*BulkForkTask)
			task.ID = 1
		}).Return(nil)
	mockRepo.On("UpdateBulkForkTask", mock.Anything, mock.AnythingOfType("*edu.BulkForkTask")).Return(nil)

	task, err := svc.BulkForkForAssignment(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.Equal(t, 0, task.TotalUsers)
	assert.Equal(t, "done", task.Status)
}
