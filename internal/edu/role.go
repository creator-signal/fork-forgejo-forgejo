package edu

import (
	"context"
	"fmt"
)

// GetUserRole returns the role of a user. If no role is set, returns empty string (which implies standard user).
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

// SetUserRole sets the role for a user.
func SetUserRole(ctx context.Context, userID int64, role RoleType) error {
	sess := GetSQLRunner(ctx)

	// Check if role exists
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

// EnsureUserRole ensures a user has at least the specified role.
// Useful for auto-assignment logic if needed.
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

// IsTeacher checks if the user is a teacher or admin.
func IsTeacher(ctx context.Context, userID int64) (bool, error) {
	role, err := GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return role == RoleTeacher || role == RoleAdmin, nil
}

// IsStudent checks if the user is a student.
func IsStudent(ctx context.Context, userID int64) (bool, error) {
	role, err := GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return role == RoleStudent, nil
}

// GetUserByName finds a user by name (simple wrapper to avoid importing models/user in routers if we want to keep logic here)
// Actually we need to import models/user here if we want to use it, or pass sql runner.
// But models/user is where User struct is.
// Using raw SQL for minimal dependency or just assume `user_model` is clear.
// Let's use xorm from context.
func GetUserByName(ctx context.Context, name string) (*UserStub, error) {
	// We need a minimal User struct to scan into if we don't want to import the huge User model.
	// Or we can just return a Stub.
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
