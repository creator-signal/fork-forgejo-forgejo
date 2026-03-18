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

func TestSyncAllForks_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, RepoID: 100}
	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetDefaultBranch", mock.Anything, int64(100)).Return("main", nil)

	doer := &user_model.User{ID: 50, Name: "teacher"}
	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)

	submissions := []*Submission{
		{ID: 1, AssignmentID: 1, UserID: 200, StudentRepoID: 301},
		{ID: 2, AssignmentID: 1, UserID: 201, StudentRepoID: 302},
	}
	mockRepo.On("GetSubmissions", mock.Anything, int64(1)).Return(submissions, nil)

	fork1 := &repo_model.Repository{ID: 301, Name: "student1-hw1", IsFork: true}
	fork2 := &repo_model.Repository{ID: 302, Name: "student2-hw1", IsFork: true}
	mockForker.On("GetRepositoryByID", mock.Anything, int64(301)).Return(fork1, nil)
	mockForker.On("GetRepositoryByID", mock.Anything, int64(302)).Return(fork2, nil)

	mockForker.On("SyncFork", mock.Anything, doer, fork1, "main").Return(nil)
	mockForker.On("SyncFork", mock.Anything, doer, fork2, "main").Return(nil)

	mockRepo.On("CreateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*SyncForkTask)
			task.ID = 1
		}).Return(nil)
	mockRepo.On("UpdateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).Return(nil)

	task, err := svc.SyncAllForks(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, 2, task.TotalRepos)
	assert.Equal(t, 2, task.Synced)
	assert.Equal(t, 0, task.Skipped)
	assert.Equal(t, 0, task.Failed)
	assert.Equal(t, "done", task.Status)
}

func TestSyncAllForks_SkipsNoRepoID(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, RepoID: 100}
	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetDefaultBranch", mock.Anything, int64(100)).Return("main", nil)

	doer := &user_model.User{ID: 50, Name: "teacher"}
	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)

	submissions := []*Submission{
		{ID: 1, AssignmentID: 1, UserID: 200, StudentRepoID: 0}, // no repo
	}
	mockRepo.On("GetSubmissions", mock.Anything, int64(1)).Return(submissions, nil)

	mockRepo.On("CreateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*SyncForkTask)
			task.ID = 1
		}).Return(nil)
	mockRepo.On("UpdateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).Return(nil)

	task, err := svc.SyncAllForks(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.Equal(t, 1, task.TotalRepos)
	assert.Equal(t, 0, task.Synced)
	assert.Equal(t, 1, task.Skipped)
	assert.Equal(t, 0, task.Failed)
	assert.Equal(t, "done", task.Status)
}

func TestSyncAllForks_SyncError(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, RepoID: 100}
	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetDefaultBranch", mock.Anything, int64(100)).Return("main", nil)

	doer := &user_model.User{ID: 50, Name: "teacher"}
	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)

	submissions := []*Submission{
		{ID: 1, AssignmentID: 1, UserID: 200, StudentRepoID: 301},
		{ID: 2, AssignmentID: 1, UserID: 201, StudentRepoID: 302},
	}
	mockRepo.On("GetSubmissions", mock.Anything, int64(1)).Return(submissions, nil)

	fork1 := &repo_model.Repository{ID: 301, Name: "student1-hw1", IsFork: true}
	fork2 := &repo_model.Repository{ID: 302, Name: "student2-hw1", IsFork: true}
	mockForker.On("GetRepositoryByID", mock.Anything, int64(301)).Return(fork1, nil)
	mockForker.On("GetRepositoryByID", mock.Anything, int64(302)).Return(fork2, nil)

	// First sync fails
	mockForker.On("SyncFork", mock.Anything, doer, fork1, "main").Return(fmt.Errorf("push rejected"))
	// Second sync succeeds
	mockForker.On("SyncFork", mock.Anything, doer, fork2, "main").Return(nil)

	mockRepo.On("CreateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*SyncForkTask)
			task.ID = 1
		}).Return(nil)
	mockRepo.On("UpdateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).Return(nil)

	task, err := svc.SyncAllForks(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.Equal(t, 2, task.TotalRepos)
	assert.Equal(t, 1, task.Synced)
	assert.Equal(t, 0, task.Skipped)
	assert.Equal(t, 1, task.Failed)
	assert.Equal(t, "error", task.Status)
	assert.Contains(t, task.ErrorLog, "student1-hw1")
	assert.Contains(t, task.ErrorLog, "push rejected")
}

func TestSyncAllForks_EmptySubmissions(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	mockUsers := new(MockUserCreator)
	svc := NewService(mockRepo, mockForker, mockUsers)

	assignment := &Assignment{ID: 1, RepoID: 100}
	mockRepo.On("GetAssignmentByID", mock.Anything, int64(1)).Return(assignment, nil)
	mockForker.On("GetDefaultBranch", mock.Anything, int64(100)).Return("main", nil)

	doer := &user_model.User{ID: 50, Name: "teacher"}
	mockUsers.On("GetUserByID", mock.Anything, int64(50)).Return(doer, nil)

	mockRepo.On("GetSubmissions", mock.Anything, int64(1)).Return([]*Submission{}, nil)

	mockRepo.On("CreateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).
		Run(func(args mock.Arguments) {
			task := args.Get(1).(*SyncForkTask)
			task.ID = 1
		}).Return(nil)
	mockRepo.On("UpdateSyncForkTask", mock.Anything, mock.AnythingOfType("*edu.SyncForkTask")).Return(nil)

	task, err := svc.SyncAllForks(context.Background(), 1, 50)
	assert.NoError(t, err)
	assert.Equal(t, 0, task.TotalRepos)
	assert.Equal(t, "done", task.Status)
}
