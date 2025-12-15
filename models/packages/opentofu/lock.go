package opentofu

import (
	"context"
	"fmt"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/util"
)

// ErrStateLockNotExist represents a "StateLockNotExist" kind of error.
//
// It is returned when no state lock exists for a given package.
type ErrStateLockNotExist struct {
	PackageName string
	OwnerID     int64
}

// IsErrStateLockNotExist checks if an error is of type ErrStateLockNotExist.
func IsErrStateLockNotExist(err error) bool {
	_, ok := err.(ErrStateLockNotExist)
	return ok
}

func (err ErrStateLockNotExist) Error() string {
	return fmt.Sprintf("package %s is not locked [OwnerID: %d]", err.PackageName, err.OwnerID)
}

func (err ErrStateLockNotExist) Unwrap() error {
	return util.ErrNotExist
}

// ErrStateLockAlreadyExist represents a "StateLockAlreadyExist" kind of error.
//
// It is returned when a state lock already exists for a given package.
type ErrStateLockAlreadyExist struct {
	PackageName string
	OwnerID     int64
}

// IsErrStateLockAlreadyExist checks if an error is of type
// StateLockAlreadyExist.
func IsErrStateLockAlreadyExist(err error) bool {
	_, ok := err.(ErrStateLockAlreadyExist)
	return ok
}

func (err ErrStateLockAlreadyExist) Error() string {
	return fmt.Sprintf("package %s is already locked [OwnerID: %d]", err.PackageName, err.OwnerID)
}

func (err ErrStateLockAlreadyExist) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrInvalidLockID represents a "InvalidLockID" kind of error.
//
// It is returned when a given lock ID does not match the lock ID stored in
// the database.
type ErrInvalidLockID struct {
	PackageName string
	OwnerID     int64
	LockID      string
}

// IsErrInvalidLockID checks if an error is of type ErrInvalidLockID.
func IsErrInvalidLockID(err error) bool {
	_, ok := err.(ErrInvalidLockID)
	return ok
}

func (err ErrInvalidLockID) Error() string {
	return fmt.Sprintf("lock ID %s is invalid [PackageName: %s, OwnerID: %d]", err.LockID, err.PackageName, err.OwnerID)
}

func (err ErrInvalidLockID) Unwrap() error {
	return util.ErrPermissionDenied
}

// StateLock represents a state lock preventing an OpenTofu/Terraform state file
// from being updating if another user is already updating it.
//
// The state file to lock is identified by the combination of a package name and
// its owner ID. A unique index composed of these two columns is set to avoid
// potential race conditions.
type StateLock struct {
	ID int64 `xorm:"pk autoincr" json:"-"`

	// The name of the locked package.
	PackageName string `xorm:"UNIQUE(state_lock) NOT NULL" json:"-"`

	// ID of the package owner (either a user or an organisation).
	OwnerID int64 `xorm:"UNIQUE(state_lock) NOT NULL" json:"-"`

	// The lock ID itself.
	//
	// The lock ID is a string that can take any form as long as it is random. At
	// the time of writing these lines, OpenTofu sends a UUID v4:
	//
	// https://github.com/opentofu/opentofu/blob/22910f2b01f7ade5aa9ec7505f9e24dfe65e5e6c/internal/states/statemgr/locker.go#L164-L174
	//
	// Given that the maximum size of a lock ID is not standardised, a trade-off has
	// been made to prevent a malicious client from sending a large lock ID value
	// while being agnostic to the client implementation.
	LockID string `xorm:"VARCHAR(80) NOT NULL" json:"ID"`

	// The client software operation for which the lock request has been made (e.g. plan, apply).
	Operation string `xorm:"VARCHAR(40)" json:"Operation"`

	// Optional extra info provided by the client software.
	Info string `xorm:"VARCHAR(300)" json:"Info,omitempty"`

	// Name of the Forgejo user having performed the lock request.
	UserName string `xorm:"NOT NULL" json:"Who"`

	// Version of the client software having performed the lock request.
	ClientVersion string `xorm:"VARCHAR(40)" json:"Version,omitempty"`

	// Timestamp of the lock creation.
	CreatedUnix time.Time `xorm:"created NOT NULL" json:"Created"`

	// Package path.
	Path string `xorm:"-" json:"Path"`
}

// TableName sets the table name for the Lock struct
func (s *StateLock) TableName() string {
	return "package_opentofu_state_locks"
}

// init registers the Lock struct as table in the database
func init() {
	db.RegisterModel(new(StateLock))
}

// GetLock returns the state lock, if any, based on the package name and its
// owner ID.
//
// Returns ErrStateLockNotExist if the state lock is not found.
func GetLock(ctx context.Context, packageName string, ownerID int64) (*StateLock, error) {
	var lock StateLock

	has, err := db.GetEngine(ctx).Where(
		"package_name = ?",
		packageName,
	).And(
		"owner_id = ?",
		ownerID,
	).Get(&lock)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrStateLockNotExist{packageName, ownerID}
	}

	return &lock, nil
}

// Lock locks an OpenTofu/Terraform state file package by inserting the lock
// info given as parameter into the database.
//
// The package to lock is identified by its name and its owner ID.
//
// If the package is already locked, the lock info from the database are
// returned in addition to ErrStateLockAlreadyExist.
func Lock(ctx context.Context, lockInfo *StateLock) (*StateLock, error) {
	lock, err := GetLock(ctx, lockInfo.PackageName, lockInfo.OwnerID)
	if err != nil {
		if !IsErrStateLockNotExist(err) {
			return nil, err
		}
	} else {
		return lock, ErrStateLockAlreadyExist{lockInfo.PackageName, lockInfo.OwnerID}
	}

	_, err = db.GetEngine(ctx).Insert(lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

// Unlock unlocks an OpenTofu/Terraform state file package by removing the lock
// ID given as parameter from the database.
//
// The package to unlock is identified by its name and its owner ID. The lock ID
// needs to match the value stored in the database.
//
// Returns ErrStateLockNotExist if the state lock does not exist.
func Unlock(ctx context.Context, packageName string, ownerID int64, lockID string) error {
	lock, err := GetLock(ctx, packageName, ownerID)
	if err != nil {
		return err
	}

	if lockID != lock.LockID {
		return ErrInvalidLockID{
			LockID:      lockID,
			PackageName: packageName,
			OwnerID:     ownerID,
		}
	}

	_, err = db.GetEngine(ctx).Where(
		"package_name = ?",
		lock.PackageName,
	).And(
		"owner_id = ?",
		ownerID,
	).Delete(lock)
	if err != nil {
		return err
	}

	return nil
}
