package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

// xormRepository is the Xorm-based implementation of Repository.
type xormRepository struct{}

// NewRepository creates a new instance of Repository.
func NewRepository() Repository {
	return &xormRepository{}
}

func (r *xormRepository) CreateAssignment(ctx context.Context, a *Assignment) error {
	_, err := db.GetEngine(ctx).Insert(a)
	if err != nil {
		return fmt.Errorf("insert assignment: %w", err)
	}
	return nil
}

func (r *xormRepository) GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error) {
	var submissions []*Submission
	err := db.GetEngine(ctx).Where("assignment_id = ?", assignmentID).Find(&submissions)
	if err != nil {
		return nil, fmt.Errorf("find submissions: %w", err)
	}
	return submissions, nil
}

func (r *xormRepository) GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error) {
	a := &Assignment{}
	has, err := db.GetEngine(ctx).ID(id).Get(a)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	if !has {
		return nil, nil
	}
	return a, nil
}
