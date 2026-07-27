// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/services/forms"
	"forgejo.org/tests"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestInstanceSigning(t *testing.T) {
	t.Cleanup(func() {
		// Cannot use t.Context(), it is in the done state.
		require.NoError(t, git.InitFull(context.Background()))
	})

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Repository.Signing.SigningName, "UwU")()
		defer test.MockVariableValue(&setting.Repository.Signing.SigningEmail, "fox@example.com")()
		defer test.MockProtect(&setting.Repository.Signing.InitialCommit)()
		defer test.MockProtect(&setting.Repository.Signing.CRUDActions)()

		t.Run("SSH", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			signingKeyPath := path.Join(setting.AppWorkPath, "tests/integration/ssh-signing-key")
			pubKeyPath := signingKeyPath + ".pub"

			pubKeyContent, err := os.ReadFile(pubKeyPath)
			require.NoError(t, err)

			pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyContent)
			require.NoError(t, err)
			require.NoError(t, os.Chmod(signingKeyPath, 0o600))
			defer test.MockVariableValue(&setting.SSHInstanceKey, pubKey)()
			defer test.MockVariableValue(&setting.Repository.Signing.Format, "ssh")()
			defer test.MockVariableValue(&setting.Repository.Signing.SigningKey, signingKeyPath)()

			// Ensure the git config is updated with the new signing format.
			require.NoError(t, git.InitFull(t.Context()))

			forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
				u2 := *u
				testCRUD(t, &u2, "ssh", objectFormat)
			})
		})

		t.Run("PGP", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Use a new GNUPGPHOME to avoid messing with the existing GPG keyring.
			tmpDir := t.TempDir()
			require.NoError(t, os.Chmod(tmpDir, 0o700))
			t.Setenv("GNUPGHOME", tmpDir)

			rootKeyPair, err := importTestingKey()
			require.NoError(t, err)
			defer test.MockVariableValue(&setting.Repository.Signing.SigningKey, rootKeyPair.PrimaryKey.KeyIdShortString())()
			defer test.MockVariableValue(&setting.Repository.Signing.Format, "openpgp")()

			// Ensure the git config is updated with the new signing format.
			require.NoError(t, git.InitFull(t.Context()))

			forEachObjectFormat(t, func(t *testing.T, objectFormat git.ObjectFormat) {
				u2 := *u
				testCRUD(t, &u2, "pgp", objectFormat)
			})
		})
	})
}

func testCRUD(t *testing.T, u *url.URL, signingFormat string, objectFormat git.ObjectFormat) {
	t.Helper()
	setting.Repository.Signing.CRUDActions = []string{"never"}
	setting.Repository.Signing.InitialCommit = []string{"never"}

	username := "user2"
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
	baseAPITestContext := NewAPITestContext(t, username, "repo1", auth_model.AccessTokenScopeReadRepository)
	u.Path = baseAPITestContext.GitPath()

	suffix := "-" + signingFormat + "-" + objectFormat.Name()

	t.Run("Unsigned-Initial", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
		t.Run("CheckMasterBranchUnsigned", func(t *testing.T) {
			branch := doAPIGetBranch(testCtx, "master")(t)
			assert.NotNil(t, branch.Commit)
			assert.NotNil(t, branch.Commit.Verification)
			assert.False(t, branch.Commit.Verification.Verified)
			assert.Empty(t, branch.Commit.Verification.Signature)
		})
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "master", "never", "unsigned-never.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "never", "never2", "unsigned-never2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-ParentSigned", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"parentsigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "master", "parentsigned", "signed-parent.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
		t.Run("CreateCRUDFile-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "parentsigned", "parentsigned2", "signed-parent2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-Never", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.InitialCommit = []string{"never"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "parentsigned", "parentsigned-never", "unsigned-never2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-Always", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"always"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-Always", crudActionCreateFile(
			t, testCtx, user, "master", "always", "signed-always.txt", func(t *testing.T, response api.FileResponse) {
				require.NotNil(t, response.Verification)
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
		t.Run("CreateCRUDFile-ParentSigned-always", crudActionCreateFile(
			t, testCtx, user, "parentsigned", "parentsigned-always", "signed-parent2.txt", func(t *testing.T, response api.FileResponse) {
				require.NotNil(t, response.Verification)
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("Unsigned-Initial-CRUD-ParentSigned", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"parentsigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateCRUDFile-Always-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "always", "always-parentsigned", "signed-always-parentsigned.txt", func(t *testing.T, response api.FileResponse) {
				require.NotNil(t, response.Verification)
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("AlwaysSign-Pubkey", func(t *testing.T) {
		setting.Repository.Signing.InitialCommit = []string{"pubkey"}

		t.Run("Has publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, username, "initial-pubkey"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CheckMasterBranchSigned", func(t *testing.T) {
				branch := doAPIGetBranch(testCtx, "master")(t)
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.True(t, branch.Commit.Verification.Verified)
				assert.Equal(t, "fox@example.com", branch.Commit.Verification.Signer.Email)
			})
		})

		t.Run("No publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, "user4", "initial-no-pubkey"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CheckMasterBranchSigned", func(t *testing.T) {
				branch := doAPIGetBranch(testCtx, "master")(t)
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.False(t, branch.Commit.Verification.Verified)
			})
		})
	})

	t.Run("AlwaysSign-Twofa", func(t *testing.T) {
		setting.Repository.Signing.InitialCommit = []string{"twofa"}

		t.Run("Has 2fa", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Cleanup(func() {
				unittest.AssertSuccessfulDelete(t, &auth_model.WebAuthnCredential{UserID: user.ID})
			})

			testCtx := NewAPITestContext(t, username, "initial-2fa"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			unittest.AssertSuccessfulInsert(t, &auth_model.WebAuthnCredential{UserID: user.ID})

			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CheckMasterBranchSigned", func(t *testing.T) {
				branch := doAPIGetBranch(testCtx, "master")(t)
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.True(t, branch.Commit.Verification.Verified)
				assert.Equal(t, "fox@example.com", branch.Commit.Verification.Signer.Email)
			})
		})

		t.Run("No 2fa", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, "user4", "initial-no-2fa"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CheckMasterBranchSigned", func(t *testing.T) {
				branch := doAPIGetBranch(testCtx, "master")(t)
				require.NotNil(t, branch.Commit)
				require.NotNil(t, branch.Commit.Verification)
				assert.False(t, branch.Commit.Verification.Verified)
			})
		})
	})

	t.Run("AlwaysSign-Initial", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.InitialCommit = []string{"always"}

		testCtx := NewAPITestContext(t, username, "initial-always"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
		t.Run("CheckMasterBranchSigned", func(t *testing.T) {
			branch := doAPIGetBranch(testCtx, "master")(t)
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.True(t, branch.Commit.Verification.Verified)
			assert.Equal(t, "fox@example.com", branch.Commit.Verification.Signer.Email)
		})
	})

	t.Run("AlwaysSign-Initial-CRUD-Never", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"never"}

		testCtx := NewAPITestContext(t, username, "initial-always-never"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
		t.Run("CreateCRUDFile-Never", crudActionCreateFile(
			t, testCtx, user, "master", "never", "unsigned-never.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	})

	t.Run("AlwaysSign-Initial-CRUD-ParentSigned-On-Always", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"parentsigned"}

		testCtx := NewAPITestContext(t, username, "initial-always-parent"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
		t.Run("CreateCRUDFile-ParentSigned", crudActionCreateFile(
			t, testCtx, user, "master", "parentsigned", "signed-parent.txt", func(t *testing.T, response api.FileResponse) {
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("AlwaysSign-Initial-CRUD-Pubkey", func(t *testing.T) {
		setting.Repository.Signing.CRUDActions = []string{"pubkey"}

		t.Run("Has publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, username, "initial-always-pubkey"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CreateCRUDFile-Pubkey", crudActionCreateFile(
				t, testCtx, user, "master", "pubkey", "signed-pubkey.txt", func(t *testing.T, response api.FileResponse) {
					assert.True(t, response.Verification.Verified)
					assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
				}))
		})

		t.Run("No publickey", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, "user4", "initial-always-no-pubkey"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CreateCRUDFile-Pubkey", crudActionCreateFile(
				t, testCtx, user, "master", "pubkey", "unsigned-pubkey.txt", func(t *testing.T, response api.FileResponse) {
					assert.False(t, response.Verification.Verified)
				}))
		})
	})

	t.Run("AlwaysSign-Initial-CRUD-Twofa", func(t *testing.T) {
		setting.Repository.Signing.CRUDActions = []string{"twofa"}

		t.Run("Has 2fa", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Cleanup(func() {
				unittest.AssertSuccessfulDelete(t, &auth_model.WebAuthnCredential{UserID: user.ID})
			})

			testCtx := NewAPITestContext(t, username, "initial-always-twofa"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			unittest.AssertSuccessfulInsert(t, &auth_model.WebAuthnCredential{UserID: user.ID})
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CreateCRUDFile-Twofa", crudActionCreateFile(
				t, testCtx, user, "master", "twofa", "signed-twofa.txt", func(t *testing.T, response api.FileResponse) {
					assert.True(t, response.Verification.Verified)
					assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
				}))
		})

		t.Run("No 2fa", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, "user4", "initial-always-no-twofa"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
			t.Run("CreateCRUDFile-Pubkey", crudActionCreateFile(
				t, testCtx, user, "master", "twofa", "unsigned-twofa.txt", func(t *testing.T, response api.FileResponse) {
					assert.False(t, response.Verification.Verified)
				}))
		})
	})

	t.Run("AlwaysSign-Initial-CRUD-Always", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.CRUDActions = []string{"always"}

		testCtx := NewAPITestContext(t, username, "initial-always-always"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
		t.Run("CreateCRUDFile-Always", crudActionCreateFile(
			t, testCtx, user, "master", "always", "signed-always.txt", func(t *testing.T, response api.FileResponse) {
				assert.True(t, response.Verification.Verified)
				assert.Equal(t, "fox@example.com", response.Verification.Signer.Email)
			}))
	})

	t.Run("UnsignedMerging", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"commitssigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "never2")(t)
			require.NoError(t, err)
			t.Run("MergePR", doAPIMergePullRequest(testCtx, testCtx.Username, testCtx.Reponame, pr.Index))
		})
		t.Run("CheckMasterBranchUnsigned", func(t *testing.T) {
			branch := doAPIGetBranch(testCtx, "master")(t)
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.False(t, branch.Commit.Verification.Verified)
			assert.Empty(t, branch.Commit.Verification.Signature)
		})
	})

	t.Run("BaseSignedMerging", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"basesigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "parentsigned2")(t)
			require.NoError(t, err)
			t.Run("MergePR", doAPIMergePullRequest(testCtx, testCtx.Username, testCtx.Reponame, pr.Index))
		})
		t.Run("CheckMasterBranchUnsigned", func(t *testing.T) {
			branch := doAPIGetBranch(testCtx, "master")(t)
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.False(t, branch.Commit.Verification.Verified)
			assert.Empty(t, branch.Commit.Verification.Signature)
		})
	})

	t.Run("CommitsSignedMerging", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"commitssigned"}

		testCtx := NewAPITestContext(t, username, "initial-unsigned"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		t.Run("CreatePullRequest", func(t *testing.T) {
			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "always-parentsigned")(t)
			require.NoError(t, err)
			t.Run("MergePR", doAPIMergePullRequest(testCtx, testCtx.Username, testCtx.Reponame, pr.Index))
		})
		t.Run("CheckMasterBranchUnsigned", func(t *testing.T) {
			branch := doAPIGetBranch(testCtx, "master")(t)
			require.NotNil(t, branch.Commit)
			require.NotNil(t, branch.Commit.Verification)
			assert.True(t, branch.Commit.Verification.Verified)
		})
	})

	assertCommitSignedByInstance := func(t *testing.T, commit api.Commit) {
		t.Helper()
		require.NotNil(t, commit.RepoCommit)
		require.NotNil(t, commit.RepoCommit.Verification)
		assert.True(t, commit.RepoCommit.Verification.Verified)
		require.NotNil(t, commit.RepoCommit.Verification.Signer)
		assert.Equal(t, "fox@example.com", commit.RepoCommit.Verification.Signer.Email)
	}
	assertCommitUnsigned := func(t *testing.T, commit api.Commit) {
		t.Helper()
		require.NotNil(t, commit.RepoCommit)
		require.NotNil(t, commit.RepoCommit.Verification)
		assert.False(t, commit.RepoCommit.Verification.Verified)
		assert.Empty(t, commit.RepoCommit.Verification.Signature)
	}
	// Creates a repository with a "head2" branch holding two unsigned commits,
	// so that rebase merges rewrite more than one commit.
	createRepoWithUnsignedBranch := func(t *testing.T, testCtx APITestContext) {
		t.Helper()
		t.Run("CreateRepository", doAPICreateRepository(testCtx, nil, objectFormat))
		t.Run("CreateHeadCommit1", crudActionCreateFile(
			t, testCtx, user, "master", "head", "head-1.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
		t.Run("CreateHeadCommit2", crudActionCreateFile(
			t, testCtx, user, "head", "head2", "head-2.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))
	}

	t.Run("AlwaysSignMerging-Rebase", func(t *testing.T) {
		setting.Repository.Signing.InitialCommit = []string{"never"}
		setting.Repository.Signing.CRUDActions = []string{"never"}
		setting.Repository.Signing.Merges = []string{"always"}

		t.Run("FastForward", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, username, "rebase-ff"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			createRepoWithUnsignedBranch(t, testCtx)

			// Advance master with a rebase message template: the pull request
			// gets behind (a real rebase happens) and the tip commit gets amended.
			t.Run("CreateRebaseTemplate", doAPICreateFile(testCtx, ".forgejo/default_merge_message/REBASE_TEMPLATE.md", &api.CreateFileOptions{
				FileOptions: api.FileOptions{
					BranchName: "master",
					Message:    "add rebase template",
					Author:     api.Identity{Name: user.FullName, Email: user.Email},
					Committer:  api.Identity{Name: user.FullName, Email: user.Email},
				},
				ContentBase64: base64.StdEncoding.EncodeToString([]byte("Rebased: ${CommitTitle}")),
			}))

			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "head2")(t)
			require.NoError(t, err)
			doAPIMergePullRequestForm(t, testCtx, testCtx.Username, testCtx.Reponame, pr.Index, &forms.MergePullRequestForm{
				Do: string(repo_model.MergeStyleRebase),
			})

			t.Run("CheckRebasedCommitsSigned", func(t *testing.T) {
				// The tip is the rebased head commit, amended with the template message.
				tip := doAPIGetCommit(testCtx, "master")(t)
				require.Len(t, tip.Files, 1)
				assert.Equal(t, "head-2.txt", tip.Files[0].Filename)
				assert.True(t, strings.HasPrefix(tip.RepoCommit.Message, "Rebased: "))
				assertCommitSignedByInstance(t, tip)

				require.Len(t, tip.Parents, 1)
				rebased := doAPIGetCommit(testCtx, tip.Parents[0].SHA)(t)
				require.Len(t, rebased.Files, 1)
				assert.Equal(t, "head-1.txt", rebased.Files[0].Filename)
				assertCommitSignedByInstance(t, rebased)

				// The commits were rebased onto the advanced master, whose own
				// commit is left untouched.
				require.Len(t, rebased.Parents, 1)
				oldMaster := doAPIGetCommit(testCtx, rebased.Parents[0].SHA)(t)
				require.Len(t, oldMaster.Files, 1)
				assert.Equal(t, ".forgejo/default_merge_message/REBASE_TEMPLATE.md", oldMaster.Files[0].Filename)
				assertCommitUnsigned(t, oldMaster)
			})
		})

		t.Run("FastForwardZeroBehind", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, username, "rebase-ff-0b"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			createRepoWithUnsignedBranch(t, testCtx)

			headBranch := doAPIGetBranch(testCtx, "head2")(t)
			require.NotNil(t, headBranch.Commit)

			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "head2")(t)
			require.NoError(t, err)
			doAPIMergePullRequestForm(t, testCtx, testCtx.Username, testCtx.Reponame, pr.Index, &forms.MergePullRequestForm{
				Do: string(repo_model.MergeStyleRebase),
			})

			t.Run("CheckCommitsRewrittenAndSigned", func(t *testing.T) {
				// Even though master could be fast-forwarded onto the unsigned
				// head commits, they are rewritten so that they carry a signature.
				tip := doAPIGetCommit(testCtx, "master")(t)
				assert.NotEqual(t, headBranch.Commit.ID, tip.SHA)
				require.Len(t, tip.Files, 1)
				assert.Equal(t, "head-2.txt", tip.Files[0].Filename)
				assertCommitSignedByInstance(t, tip)

				require.Len(t, tip.Parents, 1)
				rebased := doAPIGetCommit(testCtx, tip.Parents[0].SHA)(t)
				require.Len(t, rebased.Files, 1)
				assert.Equal(t, "head-1.txt", rebased.Files[0].Filename)
				assertCommitSignedByInstance(t, rebased)
			})
		})

		t.Run("MergeCommit", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testCtx := NewAPITestContext(t, username, "rebase-mc"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
			createRepoWithUnsignedBranch(t, testCtx)
			t.Run("AdvanceMaster", crudActionCreateFile(
				t, testCtx, user, "master", "", "master-ahead.txt", func(t *testing.T, response api.FileResponse) {
					assert.False(t, response.Verification.Verified)
				}))

			pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "head2")(t)
			require.NoError(t, err)
			doAPIMergePullRequestForm(t, testCtx, testCtx.Username, testCtx.Reponame, pr.Index, &forms.MergePullRequestForm{
				Do: string(repo_model.MergeStyleRebaseMerge),
			})

			t.Run("CheckRebasedCommitsSigned", func(t *testing.T) {
				// The merge commit was already signed before rebase signing was
				// supported; the rebased commits below it have to be signed too.
				tip := doAPIGetCommit(testCtx, "master")(t)
				require.Len(t, tip.Parents, 2)
				assertCommitSignedByInstance(t, tip)

				rebasedHead := doAPIGetCommit(testCtx, tip.Parents[1].SHA)(t)
				require.Len(t, rebasedHead.Files, 1)
				assert.Equal(t, "head-2.txt", rebasedHead.Files[0].Filename)
				assertCommitSignedByInstance(t, rebasedHead)

				require.Len(t, rebasedHead.Parents, 1)
				rebased := doAPIGetCommit(testCtx, rebasedHead.Parents[0].SHA)(t)
				require.Len(t, rebased.Files, 1)
				assert.Equal(t, "head-1.txt", rebased.Files[0].Filename)
				assertCommitSignedByInstance(t, rebased)

				oldMaster := doAPIGetCommit(testCtx, tip.Parents[0].SHA)(t)
				require.Len(t, oldMaster.Files, 1)
				assert.Equal(t, "master-ahead.txt", oldMaster.Files[0].Filename)
				assertCommitUnsigned(t, oldMaster)
			})
		})
	})

	t.Run("NeverSignMerging-Rebase", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		setting.Repository.Signing.Merges = []string{"never"}

		testCtx := NewAPITestContext(t, username, "rebase-never"+suffix, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)
		createRepoWithUnsignedBranch(t, testCtx)
		t.Run("AdvanceMaster", crudActionCreateFile(
			t, testCtx, user, "master", "", "master-ahead.txt", func(t *testing.T, response api.FileResponse) {
				assert.False(t, response.Verification.Verified)
			}))

		pr, err := doAPICreatePullRequest(testCtx, testCtx.Username, testCtx.Reponame, "master", "head2")(t)
		require.NoError(t, err)
		doAPIMergePullRequestForm(t, testCtx, testCtx.Username, testCtx.Reponame, pr.Index, &forms.MergePullRequestForm{
			Do: string(repo_model.MergeStyleRebase),
		})

		t.Run("CheckRebasedCommitsUnsigned", func(t *testing.T) {
			tip := doAPIGetCommit(testCtx, "master")(t)
			require.Len(t, tip.Files, 1)
			assert.Equal(t, "head-2.txt", tip.Files[0].Filename)
			assertCommitUnsigned(t, tip)

			require.Len(t, tip.Parents, 1)
			rebased := doAPIGetCommit(testCtx, tip.Parents[0].SHA)(t)
			require.Len(t, rebased.Files, 1)
			assert.Equal(t, "head-1.txt", rebased.Files[0].Filename)
			assertCommitUnsigned(t, rebased)
		})
	})
}

func crudActionCreateFile(_ *testing.T, ctx APITestContext, user *user_model.User, from, to, path string, callback ...func(*testing.T, api.FileResponse)) func(*testing.T) {
	return doAPICreateFile(ctx, path, &api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    from,
			NewBranchName: to,
			Message:       fmt.Sprintf("from:%s to:%s path:%s", from, to, path),
			Author: api.Identity{
				Name:  user.FullName,
				Email: user.Email,
			},
			Committer: api.Identity{
				Name:  user.FullName,
				Email: user.Email,
			},
		},
		ContentBase64: base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "This is new text for %s", path)),
	}, callback...)
}

func importTestingKey() (*openpgp.Entity, error) {
	keyringFilePath := path.Join(setting.AppWorkPath, "tests/integration/private-testing.key")
	if _, _, err := process.GetManager().Exec("gpg --import "+keyringFilePath, "gpg", "--import", keyringFilePath); err != nil {
		return nil, err
	}
	keyringFile, err := os.Open(keyringFilePath)
	if err != nil {
		return nil, err
	}
	defer keyringFile.Close()

	block, err := armor.Decode(keyringFile)
	if err != nil {
		return nil, err
	}

	keyring, err := openpgp.ReadKeyRing(block.Body)
	if err != nil {
		return nil, fmt.Errorf("Keyring access failed: '%w'", err)
	}

	// There should only be one entity in this file.
	return keyring[0], nil
}
