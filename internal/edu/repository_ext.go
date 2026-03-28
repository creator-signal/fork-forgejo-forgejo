package edu

import (
	"context"
	"fmt"
	"time"

	"forgejo.org/models/db"
)

func (r *xormRepository) GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error) {
	var assignments []*Assignment
	err := db.GetEngine(ctx).Where("repo_id = ?", repoID).Find(&assignments)
	if err != nil {
		return nil, fmt.Errorf("find assignments by repo: %w", err)
	}
	return assignments, nil
}

func (r *xormRepository) GetAssignmentsForUser(ctx context.Context, userID int64) ([]*Assignment, error) {
	var assignments []*Assignment
	err := db.GetEngine(ctx).
		Join("INNER", "edu_course_enrollments", "edu_course_enrollments.course_id = edu_assignments.course_id").
		Join("INNER", "edu_courses", "edu_courses.id = edu_assignments.course_id").
		Where("edu_course_enrollments.user_id = ?", userID).
		And("(edu_courses.end_unix = 0 OR edu_courses.end_unix > ?)", time.Now().Unix()).
		OrderBy("edu_assignments.created_unix DESC").
		Find(&assignments)
	if err != nil {
		return nil, fmt.Errorf("find assignments for user: %w", err)
	}
	return assignments, nil
}

func (r *xormRepository) UpdateAssignment(ctx context.Context, a *Assignment) error {
	_, err := db.GetEngine(ctx).ID(a.ID).Cols("title", "description", "deadline_unix", "updated_unix").Update(a)
	if err != nil {
		return fmt.Errorf("update assignment: %w", err)
	}
	return nil
}

func (r *xormRepository) DeleteAssignment(ctx context.Context, id int64) error {
	// Delete related submissions first
	_, err := db.GetEngine(ctx).Where("assignment_id = ?", id).Delete(&Submission{})
	if err != nil {
		return fmt.Errorf("delete submissions: %w", err)
	}

	// Delete the assignment
	_, err = db.GetEngine(ctx).ID(id).Delete(&Assignment{})
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}

	return nil
}
