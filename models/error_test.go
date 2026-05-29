package models

import (
	"fmt"
	"testing"
)

func Test_IsErrMethods(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  bool
		fn    func(err error) bool
	}{
		{
			name:  "IsErrUserOwnRepos_direct_returns_true",
			input: ErrUserOwnRepos{UID: 1},
			want:  true,
			fn:    IsErrUserOwnRepos,
		},
		{
			name:  "IsErrUserOwnRepos_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrUserOwnRepos: %w", ErrUserOwnRepos{UID: 2}),
			want:  true,
			fn:    IsErrUserOwnRepos,
		},
		{
			name:  "IsErrUserHasOrgs_direct_returns_true",
			input: ErrUserHasOrgs{UID: 1},
			want:  true,
			fn:    IsErrUserHasOrgs,
		},
		{
			name:  "IsErruserHasOrgs_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrUserHasOrgs: %w", ErrUserHasOrgs{UID: 2}),
			want:  true,
			fn:    IsErrUserHasOrgs,
		},
		{
			name:  "IsErrUserOwnPackages_direct_returns_true",
			input: ErrUserOwnPackages{UID: 1},
			want:  true,
			fn:    IsErrUserOwnPackages,
		},
		{
			name:  "IsErrUserOwnPackages_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrUserOwnPackages: %w", ErrUserOwnPackages{UID: 2}),
			want:  true,
			fn:    IsErrUserOwnPackages,
		},
		{
			name:  "IsErrDeleteLastAdminUser_direct_returns_true",
			input: ErrDeleteLastAdminUser{UID: 1},
			want:  true,
			fn:    IsErrDeleteLastAdminUser,
		},
		{
			name:  "IsErrUserOwnPackages_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrDeleteLastAdminUser: %w", ErrDeleteLastAdminUser{UID: 2}),
			want:  true,
			fn:    IsErrDeleteLastAdminUser,
		},

		{
			name:  "IsErrNoPendingTransfer_direct_returns_true",
			input: ErrNoPendingRepoTransfer{RepoID: 1},
			want:  true,
			fn:    IsErrNoPendingTransfer,
		},
		{
			name:  "IsErrNoPendingTransfer_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrNoPendingRepoTransfer: %w", ErrNoPendingRepoTransfer{RepoID: 2}),
			want:  true,
			fn:    IsErrNoPendingTransfer,
		},
		{
			name:  "IsErrRepoTransferInProgress_direct_returns_true",
			input: ErrRepoTransferInProgress{Uname: "direct", Name: "other_name"},
			want:  true,
			fn:    IsErrRepoTransferInProgress,
		},
		{
			name:  "IsErrRepoTransferInProgress_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrRepoTransferInProgress: %w", ErrRepoTransferInProgress{Uname: "wrapped", Name: "other_name"}),
			want:  true,
			fn:    IsErrRepoTransferInProgress,
		},
		{
			name:  "IsErrInvalidCloneAddr_direct_returns_true",
			input: &ErrInvalidCloneAddr{},
			want:  true,
			fn:    IsErrInvalidCloneAddr,
		},
		{
			name:  "IsErrInvalidCloneAddr_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrInvalidCloneAddr: %w", &ErrInvalidCloneAddr{}),
			want:  true,
			fn:    IsErrInvalidCloneAddr,
		},
		{
			name:  "IsErrInvalidTagName_direct_returns_true",
			input: ErrInvalidTagName{TagName: "v0.0.0"},
			want:  true,
			fn:    IsErrInvalidTagName,
		},
		{
			name:  "IsErrInvalidTagName_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrInvalidTagName: %w", ErrInvalidTagName{TagName: "v0.0.0-wrapped"}),
			want:  true,
			fn:    IsErrInvalidTagName,
		},
		{
			name:  "IsErrProtectedTagName_direct_returns_true",
			input: ErrProtectedTagName{TagName: "v0.0.0"},
			want:  true,
			fn:    IsErrProtectedTagName,
		},
		{
			name:  "IsErrProtectedTagName_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrProtectedTagName: %w", ErrProtectedTagName{TagName: "v0.0.0-wrapped"}),
			want:  true,
			fn:    IsErrProtectedTagName,
		},
		{
			name:  "IsErrRepoFileAlreadyExists_direct_returns_true",
			input: ErrRepoFileAlreadyExists{Path: "/fake/path/file/exists"},
			want:  true,
			fn:    IsErrRepoFileAlreadyExists,
		},
		{
			name:  "IsErrRepoFileAlreadyExists_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrRepoFileAlreadyExists: %w", ErrRepoFileAlreadyExists{Path: "/fake/path/wrapped/file/exists"}),
			want:  true,
			fn:    IsErrRepoFileAlreadyExists,
		},
		{
			name:  "IsErrRepoFileDoesNotExist_direct_returns_true",
			input: ErrRepoFileDoesNotExist{Path: "/fake/path/file/does_not/exist"},
			want:  true,
			fn:    IsErrRepoFileDoesNotExist,
		},
		{
			name:  "IsErrRepoFileDoesNotExist_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrRepoFileDoesNotExist: %w", ErrRepoFileDoesNotExist{Path: "/fake/path/wrapped/file/does_not/exist"}),
			want:  true,
			fn:    IsErrRepoFileDoesNotExist,
		},
		{
			name:  "IsErrFilenameInvalid_direct_returns_true",
			input: ErrFilenameInvalid{Path: "/fake/path/filename/invalid"},
			want:  true,
			fn:    IsErrFilenameInvalid,
		},
		{
			name:  "IsErrFilenameInvalid_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrFilenameInvalid: %w", ErrFilenameInvalid{Path: "/fake/path/wrapped/filename/invalid"}),
			want:  true,
			fn:    IsErrFilenameInvalid,
		},
		{
			name:  "IsErrUserCannotCommit_direct_returns_true",
			input: ErrUserCannotCommit{UserName: "UserCannotCommit"},
			want:  true,
			fn:    IsErrUserCannotCommit,
		},
		{
			name:  "IsErrUserCannotCommit_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrUserCannotCommit: %w", ErrUserCannotCommit{UserName: "wrapped_UserCannotCommit"}),
			want:  true,
			fn:    IsErrUserCannotCommit,
		},
		{
			name:  "IsErrFilePathInvalid_direct_returns_true",
			input: ErrFilePathInvalid{Path: "FilePathInvalid"},
			want:  true,
			fn:    IsErrFilePathInvalid,
		},
		{
			name:  "IsErrFilePathInvalid_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrFilePathInvalid: %w", ErrFilePathInvalid{Path: "wrapped_FilePathInvalid"}),
			want:  true,
			fn:    IsErrFilePathInvalid,
		},
		{
			name:  "IsErrFilePathProtected_direct_returns_true",
			input: ErrFilePathProtected{Path: "/FilePathProtected"},
			want:  true,
			fn:    IsErrFilePathProtected,
		},
		{
			name:  "IsErrFilePathProtected_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrFilePathProtected: %w", ErrFilePathProtected{Path: "/wrapped/FilePathProtected"}),
			want:  true,
			fn:    IsErrFilePathProtected,
		},
		{
			name:  "IsErrDisallowedToMerge_direct_returns_true",
			input: ErrDisallowedToMerge{Reason: "Fake_Reason_DisallowedToMerge"},
			want:  true,
			fn:    IsErrDisallowedToMerge,
		},
		{
			name:  "IsErrDisallowedToMerge_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrDisallowedToMerge: %w", ErrDisallowedToMerge{Reason: "wrapped_Reason_DisallowedToMerge"}),
			want:  true,
			fn:    IsErrDisallowedToMerge,
		},
		{
			name:  "IsErrTagAlreadyExists_direct_returns_true",
			input: ErrTagAlreadyExists{TagName: "TagNameAlreadyExists"},
			want:  true,
			fn:    IsErrTagAlreadyExists,
		},
		{
			name:  "IsErrTagAlreadyExists_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrTagAlreadyExists: %w", ErrTagAlreadyExists{TagName: "wrapped_TagAlreadyExists"}),
			want:  true,
			fn:    IsErrTagAlreadyExists,
		},
		{
			name:  "IsErrSHADoesNotMatch_direct_returns_true",
			input: ErrSHADoesNotMatch{Path: "/SHADoesNotMatch/"},
			want:  true,
			fn:    IsErrSHADoesNotMatch,
		},
		{
			name:  "IsErrSHADoesNotMatch_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrSHADoesNotMatch: %w", ErrSHADoesNotMatch{Path: "/wrapped/SHADoesNotMatch"}),
			want:  true,
			fn:    IsErrSHADoesNotMatch,
		},
		{
			name:  "IsErrSHANotFound_direct_returns_true",
			input: ErrSHANotFound{SHA: "SHANotFound"},
			want:  true,
			fn:    IsErrSHANotFound,
		},
		{
			name:  "IsErrSHANotFound_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrSHANotFound: %w", ErrSHANotFound{SHA: "WrappedSHANotFound"}),
			want:  true,
			fn:    IsErrSHANotFound,
		},
		{
			name:  "IsErrCommitIDDoesNotMatch_direct_returns_true",
			input: ErrCommitIDDoesNotMatch{GivenCommitID: "CommitIDDoesNotMatch"},
			want:  true,
			fn:    IsErrCommitIDDoesNotMatch,
		},
		{
			name:  "IsErrCommitIDDoesNotMatch_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrCommitIDDoesNotMatch: %w", ErrCommitIDDoesNotMatch{GivenCommitID: "WrappedCommitIDDoesNotMatch"}),
			want:  true,
			fn:    IsErrCommitIDDoesNotMatch,
		},
		{
			name:  "IsErrSHAOrCommitIDNotProvided_direct_returns_true",
			input: ErrSHAOrCommitIDNotProvided{},
			want:  true,
			fn:    IsErrSHAOrCommitIDNotProvided,
		},
		{
			name:  "IsErrSHAOrCommitIDNotProvided_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrSHAOrCommitIDNotProvided: %w", ErrSHAOrCommitIDNotProvided{}),
			want:  true,
			fn:    IsErrSHAOrCommitIDNotProvided,
		},
		{
			name:  "IsErrInvalidMergeStyle_direct_returns_true",
			input: ErrInvalidMergeStyle{ID: 1},
			want:  true,
			fn:    IsErrInvalidMergeStyle,
		},
		{
			name:  "IsErrInvalidMergeStyle_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrInvalidMergeStyle: %w", ErrInvalidMergeStyle{ID: 2}),
			want:  true,
			fn:    IsErrInvalidMergeStyle,
		},
		{
			name:  "IsErrMergeConflicts_direct_returns_true",
			input: ErrMergeConflicts{StdOut: "merge_conflict_std_out"},
			want:  true,
			fn:    IsErrMergeConflicts,
		},
		{
			name:  "IsErrMergeConflicts_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrMergeConflicts: %w", ErrMergeConflicts{StdOut: "wrapped_merge_conflict_std_out"}),
			want:  true,
			fn:    IsErrMergeConflicts,
		},
		{
			name:  "IsErrMergeUnrelatedHistories_direct_returns_true",
			input: ErrMergeUnrelatedHistories{StdOut: "MergeUnrelatedHistories_std_out"},
			want:  true,
			fn:    IsErrMergeUnrelatedHistories,
		},
		{
			name:  "IsErrMergeUnrelatedHistories_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrMergeUnrelatedHistories: %w", ErrMergeUnrelatedHistories{StdOut: "wrapped_MergeUnrelatedHistories_std_out"}),
			want:  true,
			fn:    IsErrMergeUnrelatedHistories,
		},
		{
			name:  "IsErrMergeDivergingFastForwardOnly_direct_returns_true",
			input: ErrMergeDivergingFastForwardOnly{StdOut: "MergeDivergingFastForwardOnly_std_out"},
			want:  true,
			fn:    IsErrMergeDivergingFastForwardOnly,
		},
		{
			name:  "IsErrMergeDivergingFastForwardOnly_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrMergeDivergingFastForwardOnly: %w", ErrMergeDivergingFastForwardOnly{StdOut: "wrapped_MergeDivergingFastForwardOnly_std_out"}),
			want:  true,
			fn:    IsErrMergeDivergingFastForwardOnly,
		},
		{
			name:  "IsErrRebaseConflicts_direct_returns_true",
			input: ErrRebaseConflicts{StdOut: "RebaseConflicts_std_out"},
			want:  true,
			fn:    IsErrRebaseConflicts,
		},
		{
			name:  "IsErrRebaseConflicts_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrRebaseConflicts: %w", ErrRebaseConflicts{StdOut: "wrapped_RebaseConflicts_std_out"}),
			want:  true,
			fn:    IsErrRebaseConflicts,
		},
		{
			name:  "IsErrPullRequestHasMerged_direct_returns_true",
			input: ErrPullRequestHasMerged{ID: 1},
			want:  true,
			fn:    IsErrPullRequestHasMerged,
		},
		{
			name:  "IsErrPullRequestHasMerged_wrapped_returns_true",
			input: fmt.Errorf("wrapped_ErrPullRequestHasMerged: %w", ErrPullRequestHasMerged{ID: 2}),
			want:  true,
			fn:    IsErrPullRequestHasMerged,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn(tc.input)
			if got != tc.want {
				t.Errorf("wanted %v, got %v", tc.want, got)
			}
		})
	}
}
