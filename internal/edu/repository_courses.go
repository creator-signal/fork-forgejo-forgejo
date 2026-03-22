package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateCourse(ctx context.Context, c *Course) error {
	_, err := db.GetEngine(ctx).Insert(c)
	if err != nil {
		return fmt.Errorf("insert course: %w", err)
	}
	return nil
}

func (r *xormRepository) GetCourseByID(ctx context.Context, id int64) (*Course, error) {
	c := &Course{}
	has, err := db.GetEngine(ctx).ID(id).Get(c)
	if err != nil {
		return nil, fmt.Errorf("get course: %w", err)
	}
	if !has {
		return nil, nil
	}
	return c, nil
}

func (r *xormRepository) GetCoursesByCreator(ctx context.Context, creatorID int64) ([]*Course, error) {
	var courses []*Course
	err := db.GetEngine(ctx).Where("creator_id = ?", creatorID).OrderBy("created_unix DESC").Find(&courses)
	if err != nil {
		return nil, fmt.Errorf("find courses by creator: %w", err)
	}
	return courses, nil
}

func (r *xormRepository) GetCoursesByUser(ctx context.Context, userID int64) ([]*Course, error) {
	var courses []*Course
	err := db.GetEngine(ctx).
		Join("INNER", "edu_course_enrollments", "edu_course_enrollments.course_id = edu_courses.id").
		Where("edu_course_enrollments.user_id = ?", userID).
		OrderBy("edu_courses.created_unix DESC").
		Find(&courses)
	if err != nil {
		return nil, fmt.Errorf("find courses by user: %w", err)
	}
	return courses, nil
}

func (r *xormRepository) UpdateCourse(ctx context.Context, c *Course) error {
	_, err := db.GetEngine(ctx).ID(c.ID).Cols("name", "description", "start_unix", "end_unix", "updated_unix").Update(c)
	if err != nil {
		return fmt.Errorf("update course: %w", err)
	}
	return nil
}

func (r *xormRepository) DeleteCourse(ctx context.Context, id int64) error {
	_, err := db.GetEngine(ctx).ID(id).Delete(&Course{})
	if err != nil {
		return fmt.Errorf("delete course: %w", err)
	}
	return nil
}

func (r *xormRepository) GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error) {
	var assignments []*Assignment
	err := db.GetEngine(ctx).Where("course_id = ?", courseID).OrderBy("created_unix DESC").Find(&assignments)
	if err != nil {
		return nil, fmt.Errorf("find assignments by course: %w", err)
	}
	return assignments, nil
}
