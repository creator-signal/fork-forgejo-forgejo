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
	return db.WithTx(ctx, func(ctx context.Context) error {
		e := db.GetEngine(ctx)

		// 1. Find all assignments for this course
		var assignments []*Assignment
		if err := e.Where("course_id = ?", id).Find(&assignments); err != nil {
			return fmt.Errorf("find assignments for course: %w", err)
		}

		for _, a := range assignments {
			// 1a. Collect submission IDs for this assignment
			var submissions []*Submission
			if err := e.Where("assignment_id = ?", a.ID).Find(&submissions); err != nil {
				return fmt.Errorf("find submissions for assignment %d: %w", a.ID, err)
			}

			// 1b. Delete test results for all submissions
			for _, sub := range submissions {
				if _, err := e.Where("submission_id = ?", sub.ID).Delete(&TestResult{}); err != nil {
					return fmt.Errorf("delete test results for submission %d: %w", sub.ID, err)
				}
			}

			// 1c. Delete submissions
			if _, err := e.Where("assignment_id = ?", a.ID).Delete(&Submission{}); err != nil {
				return fmt.Errorf("delete submissions for assignment %d: %w", a.ID, err)
			}

			// 1d. Delete sync fork tasks
			if _, err := e.Where("assignment_id = ?", a.ID).Delete(&SyncForkTask{}); err != nil {
				return fmt.Errorf("delete sync fork tasks for assignment %d: %w", a.ID, err)
			}
		}

		// 1e. Delete init-forks tasks for this course
		if _, err := e.Where("course_id = ?", id).Delete(&InitForksTask{}); err != nil {
			return fmt.Errorf("delete init forks tasks for course: %w", err)
		}

		// 2. Delete all assignments
		if _, err := e.Where("course_id = ?", id).Delete(&Assignment{}); err != nil {
			return fmt.Errorf("delete assignments for course: %w", err)
		}

		// 3. Delete import draft rows (via drafts belonging to this course)
		var drafts []*ImportDraft
		if err := e.Where("course_id = ?", id).Find(&drafts); err != nil {
			return fmt.Errorf("find import drafts for course: %w", err)
		}
		for _, d := range drafts {
			if _, err := e.Where("draft_id = ?", d.ID).Delete(&ImportDraftRow{}); err != nil {
				return fmt.Errorf("delete import draft rows for draft %d: %w", d.ID, err)
			}
		}

		// 4. Delete import drafts
		if _, err := e.Where("course_id = ?", id).Delete(&ImportDraft{}); err != nil {
			return fmt.Errorf("delete import drafts for course: %w", err)
		}

		// 5. Delete enrollments
		if _, err := e.Where("course_id = ?", id).Delete(&CourseEnrollment{}); err != nil {
			return fmt.Errorf("delete enrollments for course: %w", err)
		}

		// 6. Delete the course itself
		if _, err := e.ID(id).Delete(&Course{}); err != nil {
			return fmt.Errorf("delete course: %w", err)
		}

		return nil
	})
}

func (r *xormRepository) GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error) {
	var assignments []*Assignment
	err := db.GetEngine(ctx).Where("course_id = ?", courseID).OrderBy("created_unix DESC").Find(&assignments)
	if err != nil {
		return nil, fmt.Errorf("find assignments by course: %w", err)
	}
	return assignments, nil
}
