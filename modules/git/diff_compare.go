// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package git

import "bytes"

// CheckIfDiffDiffers returns if the diff of the firstCommitID and
// secondCommitID with the merge base of the base branch has changed.
//
// Informally it checks if the following two diffs are exactly the same in their
// contents, thus ignoring different commit IDs, headers and messages:
// 1. git diff $(git merge-base baseReference firstCommitID) firstCommitID
// 2. git diff $(git merge-base baseReference secondCommitID) secondCommitID
func (repo *Repository) CheckIfDiffDiffers(baseReference, firstCommitID, secondCommitID string, env []string) (hasChanged bool, err error) {
	// Use git-diff-tree(1) to output the difference of tree objects between the
	// commits and the base branch using the merge-base between the two. It is
	// faster than doing a normal three-way diff via git-diff(1) as it compares
	// trees (equivalent to directories) instead of individual files.
	//
	// The raw output of the command contains the before and after hash for each tree that
	// changed in the diff. If two diffs are equal then so would be the raw output
	// of this command.

	firstDiff := &bytes.Buffer{}
	err = NewCommand(repo.Ctx, "diff-tree", "--raw", "--merge-base").
		AddDynamicArguments(baseReference, firstCommitID).
		Run(&RunOpts{Dir: repo.Path, Env: env, Stdout: firstDiff})
	if err != nil {
		return false, err
	}

	secondDiff := &bytes.Buffer{}
	err = NewCommand(repo.Ctx, "diff-tree", "--raw", "--merge-base").
		AddDynamicArguments(baseReference, secondCommitID).
		Run(&RunOpts{Dir: repo.Path, Env: env, Stdout: secondDiff})
	if err != nil {
		return false, err
	}

	return !bytes.Equal(firstDiff.Bytes(), secondDiff.Bytes()), nil
}
