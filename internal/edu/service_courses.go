package edu

import (
	"context"
	"errors"
	"fmt"
	"time"

	"forgejo.org/models/perm"
	"forgejo.org/modules/log"
)

// ErrUserAlreadyEnrolled is returned when trying to enroll a user who is already enrolled in the course.
var ErrUserAlreadyEnrolled = errors.New("user is already enrolled in this course")

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

	// Auto-enroll the creator as teacher
	enrollment := &CourseEnrollment{
		CourseID:    course.ID,
		UserID:      creatorID,
		Role:        RoleTeacher,
		CreatedUnix: now,
	}
	if err := s.repo.EnrollUser(ctx, enrollment); err != nil {
		return nil, fmt.Errorf("enroll creator: %w", err)
	}

	if err := s.addToOrgTeam(ctx, course.ID, creatorID, RoleTeacher); err != nil {
		log.Error("Failed to add creator %d to org team for course %d: %v", creatorID, course.ID, err)
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
	// Check if the user is already enrolled to avoid UNIQUE constraint violation
	existing, err := s.repo.GetEnrollment(ctx, courseID, userID)
	if err != nil {
		return fmt.Errorf("check existing enrollment: %w", err)
	}
	if existing != nil {
		return ErrUserAlreadyEnrolled
	}

	enrollment := &CourseEnrollment{
		CourseID:    courseID,
		UserID:      userID,
		Role:        role,
		CreatedUnix: time.Now().Unix(),
	}
	if err := s.repo.EnrollUser(ctx, enrollment); err != nil {
		return err
	}

	if err := s.addToOrgTeam(ctx, courseID, userID, role); err != nil {
		log.Error("Failed to add user %d to org team for course %d: %v", userID, courseID, err)
	}

	return nil
}

func (s *service) GetEnrollments(ctx context.Context, courseID int64) ([]*CourseEnrollment, error) {
	return s.repo.GetEnrollments(ctx, courseID)
}

func (s *service) RemoveEnrollment(ctx context.Context, courseID, userID int64) error {
	if err := s.repo.RemoveEnrollment(ctx, courseID, userID); err != nil {
		return err
	}

	if err := s.removeFromOrgTeam(ctx, courseID, userID); err != nil {
		log.Error("Failed to remove user %d from org team for course %d: %v", userID, courseID, err)
	}

	return nil
}

func (s *service) GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error) {
	return s.repo.GetAssignmentsByCourse(ctx, courseID)
}

func eduTeamName(courseID int64, role RoleType) string {
	switch role {
	case RoleTeacher, RoleAdmin:
		return fmt.Sprintf("edu-course-%d-teachers", courseID)
	case RoleTA:
		return fmt.Sprintf("edu-course-%d-ta", courseID)
	default:
		return fmt.Sprintf("edu-course-%d-students", courseID)
	}
}

func eduAccessMode(role RoleType) perm.AccessMode {
	switch role {
	case RoleTeacher, RoleAdmin:
		return perm.AccessModeAdmin
	case RoleTA:
		return perm.AccessModeRead
	default:
		return perm.AccessModeWrite
	}
}

// addToOrgTeam ensures the user is in the right org team for the course.
// If the course has no OrgID or the org manager is not configured, this is a no-op.
func (s *service) addToOrgTeam(ctx context.Context, courseID, userID int64, role RoleType) error {
	if s.orgs == nil {
		return nil
	}

	course, err := s.repo.GetCourseByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("get course for team mapping: %w", err)
	}
	if course == nil || course.OrgID == 0 {
		return nil
	}

	teamName := eduTeamName(courseID, role)
	accessMode := eduAccessMode(role)

	team, err := s.orgs.EnsureTeam(ctx, course.OrgID, teamName, accessMode)
	if err != nil {
		return fmt.Errorf("ensure team %s: %w", teamName, err)
	}

	if err := s.orgs.AddTeamMember(ctx, team, userID); err != nil {
		return fmt.Errorf("add team member: %w", err)
	}

	return nil
}

// removeFromOrgTeam removes a user from the org teams for the course.
func (s *service) removeFromOrgTeam(ctx context.Context, courseID, userID int64) error {
	if s.orgs == nil {
		return nil
	}

	course, err := s.repo.GetCourseByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("get course for team removal: %w", err)
	}
	if course == nil || course.OrgID == 0 {
		return nil
	}

	// Try to remove from all team types
	for _, role := range []RoleType{RoleStudent, RoleTA, RoleTeacher} {
		teamName := eduTeamName(courseID, role)
		team, err := s.orgs.GetTeam(ctx, course.OrgID, teamName)
		if err != nil {
			continue // team doesn't exist, nothing to remove from
		}
		if err := s.orgs.RemoveTeamMember(ctx, team, userID); err != nil {
			log.Error("Failed to remove user %d from team %s: %v", userID, teamName, err)
		}
	}

	return nil
}
