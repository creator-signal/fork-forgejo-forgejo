package edu

import (
	"context"
	"fmt"
	"time"
)

func (s *service) CreateCourse(ctx context.Context, creatorID int64, opts CreateCourseOptions) (*Course, error) {
	if opts.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if opts.StartUnix != 0 && opts.EndUnix != 0 && opts.EndUnix <= opts.StartUnix {
		return nil, fmt.Errorf("end date must be after start date")
	}

	now := time.Now().Unix()
	course := &Course{
		Name:        opts.Name,
		Description: opts.Description,
		CreatorID:   creatorID,
		OrgID:       opts.OrgID,
		StartUnix:   opts.StartUnix,
		EndUnix:     opts.EndUnix,
		CreatedUnix: now,
		UpdatedUnix: now,
	}

	if err := s.repo.CreateCourse(ctx, course); err != nil {
		return nil, fmt.Errorf("create course: %w", err)
	}

	return course, nil
}

func (s *service) GetCourseByID(ctx context.Context, id int64) (*Course, error) {
	return s.repo.GetCourseByID(ctx, id)
}

func (s *service) GetCoursesForUser(ctx context.Context, userID int64) ([]*Course, error) {
	return s.repo.GetCoursesByUser(ctx, userID)
}

func (s *service) UpdateCourse(ctx context.Context, course *Course) error {
	if course.Name == "" {
		return fmt.Errorf("name is required")
	}
	if course.StartUnix != 0 && course.EndUnix != 0 && course.EndUnix <= course.StartUnix {
		return fmt.Errorf("end date must be after start date")
	}

	course.UpdatedUnix = time.Now().Unix()
	return s.repo.UpdateCourse(ctx, course)
}

func (s *service) DeleteCourse(ctx context.Context, id int64) error {
	return s.repo.DeleteCourse(ctx, id)
}

func (s *service) EnrollUser(ctx context.Context, courseID, userID int64, role RoleType) error {
	enrollment := &CourseEnrollment{
		CourseID:    courseID,
		UserID:      userID,
		Role:        role,
		CreatedUnix: time.Now().Unix(),
	}
	return s.repo.EnrollUser(ctx, enrollment)
}

func (s *service) GetEnrollments(ctx context.Context, courseID int64) ([]*CourseEnrollment, error) {
	return s.repo.GetEnrollments(ctx, courseID)
}

func (s *service) RemoveEnrollment(ctx context.Context, courseID, userID int64) error {
	return s.repo.RemoveEnrollment(ctx, courseID, userID)
}

func (s *service) GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error) {
	return s.repo.GetAssignmentsByCourse(ctx, courseID)
}
