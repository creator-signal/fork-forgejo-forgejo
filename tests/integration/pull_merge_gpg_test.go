// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"crypto"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/require"
)

// TestPullRebaseWithGlobalGPGSigningEnabled tests that rebase-then-fast-forward
// merge works correctly when the system has commit.gpgsign=true configured globally.
// This reproduces the bug where git rebase fails with "gpg failed to sign the data"
// because the temporary merge repository inherits commit.gpgsign=true but lacks
// a configured user.signingkey.
func TestPullRebaseWithGlobalGPGSigningEnabled(t *testing.T) {
	// Set up a temporary GNUPGHOME to avoid messing with the existing GPG keyring
	tmpDir := t.TempDir()
	require.NoError(t, os.Chmod(tmpDir, 0o700))
	t.Setenv("GNUPGHOME", tmpDir)

	// Import the testing GPG key for the instance
	instanceKeyPair, err := importTestingKey()
	require.NoError(t, err)
	instanceKeyID := instanceKeyPair.PrimaryKey.KeyIdShortString()

	// Set up global git config to match production environment
	// Key point: commit.gpgsign=true but NO user.signingkey
	tmpGitConfig := filepath.Join(t.TempDir(), "gitconfig")
	gitConfigContent := `[user]
	name = Forgejo Signing Bot
	email = invalid
[commit]
	gpgsign = true
[gpg]
	format = openpgp
`
	require.NoError(t, os.WriteFile(tmpGitConfig, []byte(gitConfigContent), 0o600))
	t.Setenv("GIT_CONFIG_GLOBAL", tmpGitConfig)

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		defer tests.PrintCurrentTest(t)()

		// Generate and import a GPG key for user2
		// This is critical to reproduce the bug: if instance and user share the same key,
		// the bug won't appear
		userKeyPair := generateGPGKey(t, "user2", "user2@example.com")
		userKeyID := userKeyPair.PrimaryKey.KeyIdShortString()
		importGPGKey(t, userKeyPair)

		// Configure instance-level GPG signing to match production settings
		defer test.MockVariableValue(&setting.Repository.Signing.SigningName, "Forgejo Signing Bot")()
		defer test.MockVariableValue(&setting.Repository.Signing.SigningEmail, "invalid")()
		defer test.MockVariableValue(&setting.Repository.Signing.SigningKey, instanceKeyID)()
		defer test.MockVariableValue(&setting.Repository.Signing.Format, "openpgp")()
		defer test.MockProtect(&setting.Repository.Signing.InitialCommit)()
		defer test.MockProtect(&setting.Repository.Signing.CRUDActions)()
		defer test.MockProtect(&setting.Repository.Signing.Merges)()

		// Set signing rules to match production
		setting.Repository.Signing.InitialCommit = []string{"pubkey", "twofa"}
		setting.Repository.Signing.CRUDActions = []string{"pubkey", "twofa", "parentsigned"}
		setting.Repository.Signing.Merges = []string{"pubkey", "twofa", "basesigned", "commitssigned"}

		// Ensure git is initialized with the new signing configuration
		require.NoError(t, git.InitFull(t.Context()))

		// Set up user2 with 2FA and GPG key
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
		session := loginUser(t, "user2")

		// Enable 2FA for user2
		t.Cleanup(func() {
			unittest.AssertSuccessfulDelete(t, &auth_model.WebAuthnCredential{UserID: user2.ID})
		})
		unittest.AssertSuccessfulInsert(t, &auth_model.WebAuthnCredential{UserID: user2.ID})

		// Get a token for user2 (needed because 2FA disables password auth and to add GPG key)
		// Note: The session is still valid even after enabling 2FA via database insertion
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Add user2's GPG key to their profile
		addGPGKeyToProfile(t, session, token, "user2@example.com")

		// Get the test repository
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerName: "user2", Name: "repo1"})

		// Clone the repository locally
		dstPath := t.TempDir()
		u.Path = "/user2/repo1.git"
		u.User = url.UserPassword("user2", token)

		_, _, err = git.NewCommand(git.DefaultContext, "clone").AddDynamicArguments(u.String(), dstPath).RunStdString(&git.RunOpts{})
		require.NoError(t, err)

		// Configure git in the cloned repo to sign commits with user2's key
		_, _, err = git.NewCommand(git.DefaultContext, "config", "user.name", "user2").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		_, _, err = git.NewCommand(git.DefaultContext, "config", "user.email", "user2@example.com").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		_, _, err = git.NewCommand(git.DefaultContext, "config", "user.signingkey").AddDynamicArguments(userKeyID).RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		_, _, err = git.NewCommand(git.DefaultContext, "config", "commit.gpgsign", "true").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)

		// Add a merge message template to trigger the commit --amend in rebase merge
		// Without this template, the commit message won't be amended and the bug won't trigger
		templateDir := filepath.Join(dstPath, ".forgejo", "default_merge_message")
		require.NoError(t, os.MkdirAll(templateDir, 0o755))
		templateContent := "Merged {{.HeadBranch}} into {{.BaseBranch}}"
		require.NoError(t, os.WriteFile(filepath.Join(templateDir, "REBASE_TEMPLATE.md"), []byte(templateContent), 0o644))

		// Commit and push the template to master
		_, _, err = git.NewCommand(git.DefaultContext, "add", ".forgejo").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		_, _, err = git.NewCommand(git.DefaultContext, "commit", "-m", "Add merge message template").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)
		_, _, err = git.NewCommand(git.DefaultContext, "push", "origin", "master").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)

		// Create a feature branch from current master
		_, _, err = git.NewCommand(git.DefaultContext, "checkout", "-b", "feature-branch").RunStdString(&git.RunOpts{Dir: dstPath})
		require.NoError(t, err)

		// Create diverging branches by adding different files to each
		createAndPushCommit(t, dstPath, "master", "base-file.txt", "Content on base branch\n", "Add base-file.txt")
		createAndPushCommit(t, dstPath, "feature-branch", "feature-file.txt", "Content on feature branch\n", "Add feature-file.txt")

		// Verify the commits are signed (required for basesigned and commitssigned conditions)
		masterSig, _, _ := git.NewCommand(git.DefaultContext, "log", "-1", "--show-signature", "master").RunStdString(&git.RunOpts{Dir: dstPath})
		require.Contains(t, masterSig, "gpg:", "Master commit should be GPG signed")

		featureSig, _, _ := git.NewCommand(git.DefaultContext, "log", "-1", "--show-signature", "feature-branch").RunStdString(&git.RunOpts{Dir: dstPath})
		require.Contains(t, featureSig, "gpg:", "Feature commit should be GPG signed")

		// Remove the user's GPG key from the keyring to properly mimic production environment
		// Note: GPG requires the full 40-character fingerprint for --delete-keys in batch mode
		userFingerprint := fmt.Sprintf("%X", userKeyPair.PrimaryKey.Fingerprint)
		_, _, err = process.GetManager().Exec(
			"gpg --batch --yes --delete-secret-keys "+userFingerprint,
			"gpg", "--batch", "--yes", "--delete-secret-keys", userFingerprint,
		)
		require.NoError(t, err)
		_, _, err = process.GetManager().Exec(
			"gpg --batch --yes --delete-keys "+userFingerprint,
			"gpg", "--batch", "--yes", "--delete-keys", userFingerprint,
		)
		require.NoError(t, err)

		// Create a pull request via API using our existing session and token
		apiCtx := APITestContext{
			Session:  session,
			Token:    token,
			Username: "user2",
			Reponame: repo.Name,
		}
		pr, err := doAPICreatePullRequest(
			apiCtx,
			"user2",
			repo.Name,
			"master",
			"feature-branch",
		)(t)
		require.NoError(t, err)

		// Attempt to merge the PR with rebase-then-fast-forward strategy
		// Without the fix, this will fail with "gpg failed to sign the data"
		// in the web UI if git < 2.44. For git >= 2.44, it will also fail
		// with "[E] Unable to amend commit message: exit status 128" on the
		// console.
		testPullMerge(t, session, "user2", repo.Name, fmt.Sprintf("%d", pr.Index), repo_model.MergeStyleRebase, false)

		// Verify the merge succeeded by reloading the PR from database and checking HasMerged
		prDB := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{
			BaseRepoID: repo.ID,
			HeadRepoID: repo.ID,
			Index:      pr.Index,
		})
		require.True(t, prDB.HasMerged, "PR should be merged successfully")

		// Verify the merge commit on master is signed by the instance key
		repoPath := repo_model.RepoPath("user2", repo.Name)
		mergeCommitID := prDB.MergedCommitID
		require.NotEmpty(t, mergeCommitID, "PR should have a merged commit ID")

		// Get the commit signature
		stdout, _, err := git.NewCommand(git.DefaultContext, "log", "-1", "--show-signature").
			AddDynamicArguments(mergeCommitID).
			RunStdString(&git.RunOpts{Dir: repoPath})
		require.NoError(t, err)

		// Verify the commit is GPG signed by checking for signature markers
		require.Contains(t, stdout, "gpg:", "Merge commit should have a GPG signature")
		require.Contains(t, stdout, instanceKeyID, "Merge commit should be signed by the instance key")
	})
}

// generateGPGKey creates a new GPG key pair for testing.
func generateGPGKey(t *testing.T, name, email string) *openpgp.Entity {
	t.Helper()

	entity, err := openpgp.NewEntity(name, "test key", email, &packet.Config{
		DefaultHash: crypto.SHA256,
		Time:        func() time.Time { return time.Unix(0, 0) },
	})
	require.NoError(t, err)

	return entity
}

// importGPGKey imports a GPG key into the keyring.
func importGPGKey(t *testing.T, entity *openpgp.Entity) {
	t.Helper()

	// Write the key to a temporary file
	tmpFile := filepath.Join(t.TempDir(), "key.asc")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	w, err := armor.Encode(f, openpgp.PrivateKeyType, nil)
	require.NoError(t, err)

	err = entity.SerializePrivate(w, nil)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())

	// Import the key using gpg
	_, _, err = process.GetManager().Exec("gpg --import "+tmpFile, "gpg", "--import", tmpFile)
	require.NoError(t, err)
}

// addGPGKeyToProfile adds a GPG key to the user's Forgejo profile.
func addGPGKeyToProfile(t *testing.T, session *TestSession, token, email string) {
	t.Helper()

	// Export the public key by email
	pubKeyOutput, _, err := process.GetManager().Exec("gpg --armor --export "+email, "gpg", "--armor", "--export", email)
	require.NoError(t, err)
	require.NotEmpty(t, pubKeyOutput, "Failed to export public key for %s", email)

	// Add the GPG key to the user's profile via API
	req := NewRequestWithJSON(t, "POST", "/api/v1/user/gpg_keys", api.CreateGPGKeyOption{
		ArmoredKey: pubKeyOutput,
	}).AddTokenAuth(token)
	session.MakeRequest(t, req, http.StatusCreated)
}

// createAndPushCommit creates a file on a branch, commits it, and pushes to origin.
func createAndPushCommit(t *testing.T, repoPath, branch, filename, content, commitMessage string) {
	t.Helper()

	// Checkout the branch
	_, _, err := git.NewCommand(git.DefaultContext, "checkout").AddDynamicArguments(branch).RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)

	// Create and write the file
	filePath := filepath.Join(repoPath, filename)
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	// Add the file to git
	_, _, err = git.NewCommand(git.DefaultContext, "add").AddDynamicArguments(filename).RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)

	// Commit the changes
	_, _, err = git.NewCommand(git.DefaultContext, "commit").AddOptionFormat("--message=%s", commitMessage).RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)

	// Push to origin
	_, _, err = git.NewCommand(git.DefaultContext, "push", "origin").AddDynamicArguments(branch).RunStdString(&git.RunOpts{Dir: repoPath})
	require.NoError(t, err)
}
