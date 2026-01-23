package edu

import (
	"context"
	"fmt"
)

// GetUserRole returns the role of a user. If no role is set, returns empty string.
func GetUserRole(ctx context.Context, userID int64) (RoleType, error) {
	var role UserRole
	has, err := GetSQLRunner(ctx).Where("user_id = ?", userID).Get(&role)
	if err != nil {
		return "", err
	}
	if !has {
		return "", nil
	}
	return role.Role, nil
}

func SetUserRole(ctx context.Context, userID int64, role RoleType) error {
	sess := GetSQLRunner(ctx)

	var userRole UserRole
	has, err := sess.Where("user_id = ?", userID).Get(&userRole)
	if err != nil {
		return err
	}

	if has {
		userRole.Role = role
		_, err = sess.ID(userRole.ID).Cols("role").Update(&userRole)
	} else {
		userRole = UserRole{
			UserID: userID,
			Role:   role,
		}
		_, err = sess.Insert(&userRole)
	}
	return err
}

func EnsureUserRole(ctx context.Context, userID int64, role RoleType) error {
	current, err := GetUserRole(ctx, userID)
	if err != nil {
		return err
	}
	if current == "" {
		return SetUserRole(ctx, userID, role)
	}
	return nil
}

func IsTeacher(ctx context.Context, userID int64) (bool, error) {
	role, err := GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return role == RoleTeacher || role == RoleAdmin, nil
}

func IsStudent(ctx context.Context, userID int64) (bool, error) {
	role, err := GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return role == RoleStudent, nil
}

func GetUserByName(ctx context.Context, name string) (*UserStub, error) {
	type UserStub struct {
		ID   int64
		Name string
	}
	var u UserStub
	has, err := GetSQLRunner(ctx).Table("user").Where("lower_name = ?", name).Get(&u)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("user not found")
	}
	return &u, nil
}

type UserStub struct {
	ID   int64
	Name string
}
