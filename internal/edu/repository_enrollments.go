package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) EnrollUser(ctx context.Context, enrollment *CourseEnrollment) error {
	_, err := db.GetEngine(ctx).Insert(enrollment)
	if err != nil {
		return fmt.Errorf("insert enrollment: %w", err)
	}
	return nil
}

func (r *xormRepository) GetEnrollment(ctx context.Context, courseID, userID int64) (*CourseEnrollment, error) {
	e := &CourseEnrollment{}
	has, err := db.GetEngine(ctx).Where("course_id = ? AND user_id = ?", courseID, userID).Get(e)
	if err != nil {
		return nil, fmt.Errorf("get enrollment: %w", err)
	}
	if !has {
		return nil, nil
	}
	return e, nil
}

func (r *xormRepository) GetEnrollments(ctx context.Context, courseID int64) ([]*CourseEnrollment, error) {
	var enrollments []*CourseEnrollment
	err := db.GetEngine(ctx).Where("course_id = ?", courseID).OrderBy("created_unix ASC").Find(&enrollments)
	if err != nil {
		return nil, fmt.Errorf("find enrollments: %w", err)
	}
	return enrollments, nil
}

func (r *xormRepository) RemoveEnrollment(ctx context.Context, courseID, userID int64) error {
	_, err := db.GetEngine(ctx).Where("course_id = ? AND user_id = ?", courseID, userID).Delete(&CourseEnrollment{})
	if err != nil {
		return fmt.Errorf("delete enrollment: %w", err)
	}
	return nil
}
