package edu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateCourse(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		opts := CreateCourseOptions{
			Name:        "Test Course",
			Description: "A test course",
			StartUnix:   1000,
			EndUnix:     2000,
		}

		mockRepo.On("CreateCourse", ctx, mock.MatchedBy(func(c *Course) bool {
			return c.Name == opts.Name && c.CreatorID == int64(1)
		})).Return(nil).Once()
		mockRepo.On("EnrollUser", ctx, mock.MatchedBy(func(e *CourseEnrollment) bool {
			return e.UserID == int64(1) && e.Role == RoleTeacher
		})).Return(nil).Once()

		course, err := service.CreateCourse(ctx, 1, opts)
		assert.NoError(t, err)
		assert.NotNil(t, course)
		assert.Equal(t, opts.Name, course.Name)
		assert.Equal(t, int64(1), course.CreatorID)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty name", func(t *testing.T) {
		opts := CreateCourseOptions{}
		_, err := service.CreateCourse(ctx, 1, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("End before start", func(t *testing.T) {
		opts := CreateCourseOptions{
			Name:      "Bad dates",
			StartUnix: 2000,
			EndUnix:   1000,
		}
		_, err := service.CreateCourse(ctx, 1, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "end date must be after start date")
	})
}

func TestGetCoursesForUser(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	expected := []*Course{{ID: 1, Name: "C1"}, {ID: 2, Name: "C2"}}
	mockRepo.On("GetCoursesByUser", ctx, int64(5)).Return(expected, nil)

	result, err := service.GetCoursesForUser(ctx, 5)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestUpdateCourse(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		course := &Course{ID: 1, Name: "Updated", StartUnix: 1000, EndUnix: 2000}
		mockRepo.On("UpdateCourse", ctx, mock.MatchedBy(func(c *Course) bool {
			return c.ID == int64(1) && c.Name == "Updated"
		})).Return(nil).Once()

		err := service.UpdateCourse(ctx, course)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty name", func(t *testing.T) {
		course := &Course{ID: 1, Name: ""}
		err := service.UpdateCourse(ctx, course)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestEnrollUser(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	mockRepo.On("GetEnrollment", ctx, int64(1), int64(5)).Return(nil, nil)
	mockRepo.On("EnrollUser", ctx, mock.MatchedBy(func(e *CourseEnrollment) bool {
		return e.CourseID == int64(1) && e.UserID == int64(5) && e.Role == RoleStudent
	})).Return(nil)
	mockRepo.On("GetCourseByID", ctx, mock.AnythingOfType("int64")).Return(&Course{OrgID: 0, TasksMasterRepoID: 0}, nil).Maybe()

	err := service.EnrollUser(ctx, EnrollUserOptions{CourseID: 1, UserID: 5, Role: RoleStudent})
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestEnrollUser_StoresGroup(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	mockRepo.On("GetEnrollment", ctx, int64(1), int64(7)).Return(nil, nil)
	mockRepo.On("EnrollUser", ctx, mock.MatchedBy(func(e *CourseEnrollment) bool {
		return e.GroupName == "se241" && e.Role == RoleStudent
	})).Return(nil).Once()
	mockRepo.On("GetCourseByID", ctx, mock.AnythingOfType("int64")).Return(&Course{OrgID: 0, TasksMasterRepoID: 0}, nil).Maybe()

	err := service.EnrollUser(ctx, EnrollUserOptions{
		CourseID:  1,
		UserID:    7,
		Role:      RoleStudent,
		GroupName: "se241",
	})
	assert.NoError(t, err)
}

func TestRemoveEnrollment(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	mockRepo.On("GetEnrollment", ctx, int64(1), int64(5)).Return(&CourseEnrollment{
		ID: 7, CourseID: 1, UserID: 5, Role: RoleStudent, StudentForkRepoID: 0,
	}, nil)
	mockRepo.On("RemoveEnrollment", ctx, int64(1), int64(5)).Return(nil)

	err := service.RemoveEnrollment(ctx, 1, 5)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateCourse_StoresTasksMasterRepoID(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	opts := CreateCourseOptions{
		Name:              "Cxx",
		TasksMasterRepoID: 101,
	}
	mockRepo.On("CreateCourse", ctx, mock.MatchedBy(func(c *Course) bool {
		return c.TasksMasterRepoID == 101
	})).Return(nil).Once()
	mockRepo.On("EnrollUser", ctx, mock.Anything).Return(nil).Once()

	course, err := service.CreateCourse(ctx, 1, opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(101), course.TasksMasterRepoID)
}

func TestGetAssignmentsByCourse(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	expected := []*Assignment{{ID: 1, CourseID: 10, Title: "A1"}}
	mockRepo.On("GetAssignmentsByCourse", ctx, int64(10)).Return(expected, nil)

	result, err := service.GetAssignmentsByCourse(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}
