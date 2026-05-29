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
