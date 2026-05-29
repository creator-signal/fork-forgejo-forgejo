// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"errors"
	"fmt"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/util"
)

// ErrUserOwnRepos represents a "UserOwnRepos" kind of error.
type ErrUserOwnRepos struct {
	UID int64
}

// IsErrUserOwnRepos checks if an error is a ErrUserOwnRepos.
func IsErrUserOwnRepos(err error) bool {
	return errors.Is(err, ErrUserOwnRepos{})
}

func (err ErrUserOwnRepos) Error() string {
	return fmt.Sprintf("user still has ownership of repositories [uid: %d]", err.UID)
}

func (err ErrUserOwnRepos) Is(target error) bool {
	_, ok := target.(ErrUserOwnRepos)
	return ok
}

// ErrUserHasOrgs represents a "UserHasOrgs" kind of error.
type ErrUserHasOrgs struct {
	UID int64
}

// IsErrUserHasOrgs checks if an error is a ErrUserHasOrgs.
func IsErrUserHasOrgs(err error) bool {
	return errors.Is(err, ErrUserHasOrgs{})
}

func (err ErrUserHasOrgs) Error() string {
	return fmt.Sprintf("user still has membership of organizations [uid: %d]", err.UID)
}

func (err ErrUserHasOrgs) Is(target error) bool {
	_, ok := target.(ErrUserHasOrgs)
	return ok
}

// ErrUserOwnPackages notifies that the user (still) owns the packages.
type ErrUserOwnPackages struct {
	UID int64
}

// IsErrUserOwnPackages checks if an error is an ErrUserOwnPackages.
func IsErrUserOwnPackages(err error) bool {
	return errors.Is(err, ErrUserOwnPackages{})
}

func (err ErrUserOwnPackages) Error() string {
	return fmt.Sprintf("user still has ownership of packages [uid: %d]", err.UID)
}

func (err ErrUserOwnPackages) Is(target error) bool {
	_, ok := target.(ErrUserOwnPackages)
	return ok
}

// ErrDeleteLastAdminUser represents a "DeleteLastAdminUser" kind of error.
type ErrDeleteLastAdminUser struct {
	UID int64
}

// IsErrDeleteLastAdminUser checks if an error is a ErrDeleteLastAdminUser.
func IsErrDeleteLastAdminUser(err error) bool {
	return errors.Is(err, ErrDeleteLastAdminUser{})
}

func (err ErrDeleteLastAdminUser) Error() string {
	return fmt.Sprintf("can not delete the last admin user [uid: %d]", err.UID)
}

func (err ErrDeleteLastAdminUser) Is(target error) bool {
	_, ok := target.(ErrDeleteLastAdminUser)
	return ok
}

// ErrNoPendingRepoTransfer is an error type for repositories without a pending
// transfer request
type ErrNoPendingRepoTransfer struct {
	RepoID int64
}

func (err ErrNoPendingRepoTransfer) Error() string {
	return fmt.Sprintf("repository doesn't have a pending transfer [repo_id: %d]", err.RepoID)
}

func (err ErrNoPendingRepoTransfer) Is(target error) bool {
	_, ok := target.(ErrNoPendingRepoTransfer)
	return ok
}

// IsErrNoPendingTransfer is an error type when a repository has no pending
// transfers
func IsErrNoPendingTransfer(err error) bool {
	return errors.Is(err, ErrNoPendingRepoTransfer{})
}

func (err ErrNoPendingRepoTransfer) Unwrap() error {
	return util.ErrNotExist
}

// ErrRepoTransferInProgress represents the state of a repository that has an
// ongoing transfer
type ErrRepoTransferInProgress struct {
	Uname string
	Name  string
}

// IsErrRepoTransferInProgress checks if an error is a ErrRepoTransferInProgress.
func IsErrRepoTransferInProgress(err error) bool {
	return errors.Is(err, ErrRepoTransferInProgress{})
}

func (err ErrRepoTransferInProgress) Error() string {
	return fmt.Sprintf("repository is already being transferred [uname: %s, name: %s]", err.Uname, err.Name)
}

func (err ErrRepoTransferInProgress) Is(target error) bool {
	_, ok := target.(ErrRepoTransferInProgress)
	return ok
}

func (err ErrRepoTransferInProgress) Unwrap() error {
	return util.ErrAlreadyExist
}

// ErrInvalidCloneAddr represents a "InvalidCloneAddr" kind of error.
type ErrInvalidCloneAddr struct {
	Host               string
	IsURLError         bool
	IsInvalidPath      bool
	IsProtocolInvalid  bool
	IsPermissionDenied bool
	HasCredentials     bool
	LocalPath          bool
}

// IsErrInvalidCloneAddr checks if an error is a ErrInvalidCloneAddr.
func IsErrInvalidCloneAddr(err error) bool {
	return errors.Is(err, &ErrInvalidCloneAddr{})
}

func (err *ErrInvalidCloneAddr) Error() string {
	if err.IsInvalidPath {
		return fmt.Sprintf("migration/cloning from '%s' is not allowed: the provided path is invalid", err.Host)
	}
	if err.IsProtocolInvalid {
		return fmt.Sprintf("migration/cloning from '%s' is not allowed: the provided url protocol is not allowed", err.Host)
	}
	if err.IsPermissionDenied {
		return fmt.Sprintf("migration/cloning from '%s' is not allowed.", err.Host)
	}
	if err.IsURLError {
		return fmt.Sprintf("migration/cloning from '%s' is not allowed: the provided url is invalid", err.Host)
	}
	if err.HasCredentials {
		return fmt.Sprintf("migration/cloning from '%s' is not allowed: the provided url contains credentials", err.Host)
	}

	return fmt.Sprintf("migration/cloning from '%s' is not allowed", err.Host)
}

func (err *ErrInvalidCloneAddr) Unwrap() error {
	return util.ErrInvalidArgument
}

func (err *ErrInvalidCloneAddr) Is(target error) bool {
	_, ok := target.(*ErrInvalidCloneAddr)
	return ok
}

// ErrInvalidTagName represents a "InvalidTagName" kind of error.
type ErrInvalidTagName struct {
	TagName string
}

// IsErrInvalidTagName checks if an error is a ErrInvalidTagName.
func IsErrInvalidTagName(err error) bool {
	return errors.Is(err, ErrInvalidTagName{})
}

func (err ErrInvalidTagName) Error() string {
	return fmt.Sprintf("release tag name is not valid [tag_name: %s]", err.TagName)
}

func (err ErrInvalidTagName) Unwrap() error {
	return util.ErrInvalidArgument
}

func (err ErrInvalidTagName) Is(target error) bool {
	_, ok := target.(ErrInvalidTagName)
	return ok
}

// ErrProtectedTagName represents a "ProtectedTagName" kind of error.
type ErrProtectedTagName struct {
	TagName string
}

// IsErrProtectedTagName checks if an error is a ErrProtectedTagName.
func IsErrProtectedTagName(err error) bool {
	return errors.Is(err, ErrProtectedTagName{})
}

func (err ErrProtectedTagName) Error() string {
	return fmt.Sprintf("release tag name is protected [tag_name: %s]", err.TagName)
}

func (err ErrProtectedTagName) Unwrap() error {
	return util.ErrPermissionDenied
}

func (err ErrProtectedTagName) Is(target error) bool {
	_, ok := target.(ErrProtectedTagName)
	return ok
}

// ErrRepoFileAlreadyExists represents a "RepoFileAlreadyExist" kind of error.
type ErrRepoFileAlreadyExists struct {
	Path string
}

// IsErrRepoFileAlreadyExists checks if an error is a ErrRepoFileAlreadyExists.
func IsErrRepoFileAlreadyExists(err error) bool {
	return errors.Is(err, ErrRepoFileAlreadyExists{})
}

func (err ErrRepoFileAlreadyExists) Error() string {
	return fmt.Sprintf("repository file already exists [path: %s]", err.Path)
}

func (err ErrRepoFileAlreadyExists) Unwrap() error {
	return util.ErrAlreadyExist
}

func (err ErrRepoFileAlreadyExists) Is(target error) bool {
	_, ok := target.(ErrRepoFileAlreadyExists)
	return ok
}

// ErrRepoFileDoesNotExist represents a "RepoFileDoesNotExist" kind of error.
type ErrRepoFileDoesNotExist struct {
	Path string
	Name string
}

// IsErrRepoFileDoesNotExist checks if an error is a ErrRepoDoesNotExist.
func IsErrRepoFileDoesNotExist(err error) bool {
	return errors.Is(err, ErrRepoFileDoesNotExist{})
}

func (err ErrRepoFileDoesNotExist) Error() string {
	return fmt.Sprintf("repository file does not exist [path: %s]", err.Path)
}

func (err ErrRepoFileDoesNotExist) Unwrap() error {
	return util.ErrNotExist
}

func (err ErrRepoFileDoesNotExist) Is(target error) bool {
	_, ok := target.(ErrRepoFileDoesNotExist)
	return ok
}

// ErrFilenameInvalid represents a "FilenameInvalid" kind of error.
type ErrFilenameInvalid struct {
	Path string
}

// IsErrFilenameInvalid checks if an error is an ErrFilenameInvalid.
func IsErrFilenameInvalid(err error) bool {
	return errors.Is(err, ErrFilenameInvalid{})
}

func (err ErrFilenameInvalid) Error() string {
	return fmt.Sprintf("path contains a malformed path component [path: %s]", err.Path)
}

func (err ErrFilenameInvalid) Unwrap() error {
	return util.ErrInvalidArgument
}

func (err ErrFilenameInvalid) Is(target error) bool {
	_, ok := target.(ErrFilenameInvalid)
	return ok
}

// ErrUserCannotCommit represents "UserCannotCommit" kind of error.
type ErrUserCannotCommit struct {
	UserName string
}

// IsErrUserCannotCommit checks if an error is an ErrUserCannotCommit.
func IsErrUserCannotCommit(err error) bool {
	return errors.Is(err, ErrUserCannotCommit{})
}

func (err ErrUserCannotCommit) Error() string {
	return fmt.Sprintf("user cannot commit to repo [user: %s]", err.UserName)
}

func (err ErrUserCannotCommit) Unwrap() error {
	return util.ErrPermissionDenied
}

func (err ErrUserCannotCommit) Is(target error) bool {
	_, ok := target.(ErrUserCannotCommit)
	return ok
}

// ErrFilePathInvalid represents a "FilePathInvalid" kind of error.
type ErrFilePathInvalid struct {
	Message string
	Path    string
	Name    string
	Type    git.EntryMode
}

// IsErrFilePathInvalid checks if an error is an ErrFilePathInvalid.
func IsErrFilePathInvalid(err error) bool {
	return errors.Is(err, ErrFilePathInvalid{})
}

func (err ErrFilePathInvalid) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return fmt.Sprintf("path is invalid [path: %s]", err.Path)
}

func (err ErrFilePathInvalid) Unwrap() error {
	return util.ErrInvalidArgument
}

func (err ErrFilePathInvalid) Is(target error) bool {
	_, ok := target.(ErrFilePathInvalid)
	return ok
}

// ErrFilePathProtected represents a "FilePathProtected" kind of error.
type ErrFilePathProtected struct {
	Message string
	Path    string
}

// IsErrFilePathProtected checks if an error is an ErrFilePathProtected.
func IsErrFilePathProtected(err error) bool {
	return errors.Is(err, ErrFilePathProtected{})
}

func (err ErrFilePathProtected) Error() string {
	if err.Message != "" {
		return err.Message
	}
	return fmt.Sprintf("path is protected and can not be changed [path: %s]", err.Path)
}

func (err ErrFilePathProtected) Unwrap() error {
	return util.ErrPermissionDenied
}

func (err ErrFilePathProtected) Is(target error) bool {
	_, ok := target.(ErrFilePathProtected)
	return ok
}

// ErrDisallowedToMerge represents an error that a branch is protected and the current user is not allowed to modify it.
type ErrDisallowedToMerge struct {
	Reason string
}

// IsErrDisallowedToMerge checks if an error is an ErrDisallowedToMerge.
func IsErrDisallowedToMerge(err error) bool {
	return errors.Is(err, ErrDisallowedToMerge{})
}

func (err ErrDisallowedToMerge) Error() string {
	return fmt.Sprintf("not allowed to merge [reason: %s]", err.Reason)
}

func (err ErrDisallowedToMerge) Unwrap() error {
	return util.ErrPermissionDenied
}

func (err ErrDisallowedToMerge) Is(target error) bool {
	_, ok := target.(ErrDisallowedToMerge)
	return ok
}

// ErrTagAlreadyExists represents an error that tag with such name already exists.
type ErrTagAlreadyExists struct {
	TagName string
}

// IsErrTagAlreadyExists checks if an error is an ErrTagAlreadyExists.
func IsErrTagAlreadyExists(err error) bool {
	return errors.Is(err, ErrTagAlreadyExists{})
}

func (err ErrTagAlreadyExists) Error() string {
	return fmt.Sprintf("tag already exists [name: %s]", err.TagName)
}

func (err ErrTagAlreadyExists) Unwrap() error {
	return util.ErrAlreadyExist
}

func (err ErrTagAlreadyExists) Is(target error) bool {
	_, ok := target.(ErrTagAlreadyExists)
	return ok
}

// ErrSHADoesNotMatch represents a "SHADoesNotMatch" kind of error.
type ErrSHADoesNotMatch struct {
	Path       string
	GivenSHA   string
	CurrentSHA string
}

// IsErrSHADoesNotMatch checks if an error is a ErrSHADoesNotMatch.
func IsErrSHADoesNotMatch(err error) bool {
	return errors.Is(err, ErrSHADoesNotMatch{})
}

func (err ErrSHADoesNotMatch) Error() string {
	return fmt.Sprintf("sha does not match [given: %s, expected: %s]", err.GivenSHA, err.CurrentSHA)
}

func (err ErrSHADoesNotMatch) Is(target error) bool {
	_, ok := target.(ErrSHADoesNotMatch)
	return ok
}

// ErrSHANotFound represents a "SHADoesNotMatch" kind of error.
type ErrSHANotFound struct {
	SHA string
}

// IsErrSHANotFound checks if an error is a ErrSHANotFound.
func IsErrSHANotFound(err error) bool {
	return errors.Is(err, ErrSHANotFound{})
}

func (err ErrSHANotFound) Error() string {
	return fmt.Sprintf("sha not found [%s]", err.SHA)
}

func (err ErrSHANotFound) Unwrap() error {
	return util.ErrNotExist
}

func (err ErrSHANotFound) Is(target error) bool {
	_, ok := target.(ErrSHANotFound)
	return ok
}

// ErrCommitIDDoesNotMatch represents a "CommitIDDoesNotMatch" kind of error.
type ErrCommitIDDoesNotMatch struct {
	GivenCommitID   string
	CurrentCommitID string
}

// IsErrCommitIDDoesNotMatch checks if an error is a ErrCommitIDDoesNotMatch.
func IsErrCommitIDDoesNotMatch(err error) bool {
	return errors.Is(err, ErrCommitIDDoesNotMatch{})
}

func (err ErrCommitIDDoesNotMatch) Error() string {
	return fmt.Sprintf("file CommitID does not match [given: %s, expected: %s]", err.GivenCommitID, err.CurrentCommitID)
}

func (err ErrCommitIDDoesNotMatch) Is(target error) bool {
	_, ok := target.(ErrCommitIDDoesNotMatch)
	return ok
}

// ErrSHAOrCommitIDNotProvided represents a "SHAOrCommitIDNotProvided" kind of error.
type ErrSHAOrCommitIDNotProvided struct{}

// IsErrSHAOrCommitIDNotProvided checks if an error is a ErrSHAOrCommitIDNotProvided.
func IsErrSHAOrCommitIDNotProvided(err error) bool {
	return errors.Is(err, ErrSHAOrCommitIDNotProvided{})
}

func (err ErrSHAOrCommitIDNotProvided) Error() string {
	return "a SHA or commit ID must be provided when updating a file"
}

func (err ErrSHAOrCommitIDNotProvided) Is(target error) bool {
	_, ok := target.(ErrSHAOrCommitIDNotProvided)
	return ok
}

// ErrInvalidMergeStyle represents an error if merging with disabled merge strategy
type ErrInvalidMergeStyle struct {
	ID    int64
	Style repo_model.MergeStyle
}

// IsErrInvalidMergeStyle checks if an error is a ErrInvalidMergeStyle.
func IsErrInvalidMergeStyle(err error) bool {
	return errors.Is(err, ErrInvalidMergeStyle{})
}

func (err ErrInvalidMergeStyle) Error() string {
	return fmt.Sprintf("merge strategy is not allowed or is invalid [repo_id: %d, strategy: %s]",
		err.ID, err.Style)
}

func (err ErrInvalidMergeStyle) Unwrap() error {
	return util.ErrInvalidArgument
}

func (err ErrInvalidMergeStyle) Is(target error) bool {
	_, ok := target.(ErrInvalidMergeStyle)
	return ok
}

// ErrMergeConflicts represents an error if merging fails with a conflict
type ErrMergeConflicts struct {
	Style  repo_model.MergeStyle
	StdOut string
	StdErr string
	Err    error
}

// IsErrMergeConflicts checks if an error is a ErrMergeConflicts.
func IsErrMergeConflicts(err error) bool {
	return errors.Is(err, ErrMergeConflicts{})
}

func (err ErrMergeConflicts) Error() string {
	return fmt.Sprintf("Merge Conflict Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

func (err ErrMergeConflicts) Is(target error) bool {
	_, ok := target.(ErrMergeConflicts)
	return ok
}

// ErrMergeUnrelatedHistories represents an error if merging fails due to unrelated histories
type ErrMergeUnrelatedHistories struct {
	Style  repo_model.MergeStyle
	StdOut string
	StdErr string
	Err    error
}

// IsErrMergeUnrelatedHistories checks if an error is a ErrMergeUnrelatedHistories.
func IsErrMergeUnrelatedHistories(err error) bool {
	return errors.Is(err, ErrMergeUnrelatedHistories{})
}

func (err ErrMergeUnrelatedHistories) Error() string {
	return fmt.Sprintf("Merge UnrelatedHistories Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

func (err ErrMergeUnrelatedHistories) Is(target error) bool {
	_, ok := target.(ErrMergeUnrelatedHistories)
	return ok
}

// ErrMergeDivergingFastForwardOnly represents an error if a fast-forward-only merge fails because the branches diverge
type ErrMergeDivergingFastForwardOnly struct {
	StdOut string
	StdErr string
	Err    error
}

// IsErrMergeDivergingFastForwardOnly checks if an error is a ErrMergeDivergingFastForwardOnly.
func IsErrMergeDivergingFastForwardOnly(err error) bool {
	return errors.Is(err, ErrMergeDivergingFastForwardOnly{})
}

func (err ErrMergeDivergingFastForwardOnly) Error() string {
	return fmt.Sprintf("Merge DivergingFastForwardOnly Error: %v: %s\n%s", err.Err, err.StdErr, err.StdOut)
}

func (err ErrMergeDivergingFastForwardOnly) Is(target error) bool {
	_, ok := target.(ErrMergeDivergingFastForwardOnly)
	return ok
}

// ErrRebaseConflicts represents an error if rebase fails with a conflict
type ErrRebaseConflicts struct {
	Style     repo_model.MergeStyle
	CommitSHA string
	StdOut    string
	StdErr    string
	Err       error
}

// IsErrRebaseConflicts checks if an error is a ErrRebaseConflicts.
func IsErrRebaseConflicts(err error) bool {
	return errors.Is(err, ErrRebaseConflicts{})
}

func (err ErrRebaseConflicts) Error() string {
	return fmt.Sprintf("Rebase Error: %v: Whilst Rebasing: %s\n%s\n%s", err.Err, err.CommitSHA, err.StdErr, err.StdOut)
}

func (err ErrRebaseConflicts) Is(target error) bool {
	_, ok := target.(ErrRebaseConflicts)
	return ok
}

// ErrPullRequestHasMerged represents a "PullRequestHasMerged"-error
type ErrPullRequestHasMerged struct {
	ID         int64
	IssueID    int64
	HeadRepoID int64
	BaseRepoID int64
	HeadBranch string
	BaseBranch string
}

// IsErrPullRequestHasMerged checks if an error is a ErrPullRequestHasMerged.
func IsErrPullRequestHasMerged(err error) bool {
	return errors.Is(err, ErrPullRequestHasMerged{})
}

// Error does pretty-printing :D
func (err ErrPullRequestHasMerged) Error() string {
	return fmt.Sprintf("pull request has merged [id: %d, issue_id: %d, head_repo_id: %d, base_repo_id: %d, head_branch: %s, base_branch: %s]",
		err.ID, err.IssueID, err.HeadRepoID, err.BaseRepoID, err.HeadBranch, err.BaseBranch)
}

func (err ErrPullRequestHasMerged) Is(target error) bool {
	_, ok := target.(ErrPullRequestHasMerged)
	return ok
}
