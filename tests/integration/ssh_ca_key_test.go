// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestUserSuppliedSSHCAKeys(t *testing.T) {
	onApplicationRun(t, testUserSuppliedSSHCAKeys)
}

func endorseKey(t *testing.T, identityKeyFile, signingKeyFile string, certType uint32, principals []string, validitySeconds int) {
	dataIdentityKey, err := os.ReadFile(identityKeyFile + ".pub")
	require.NoError(t, err)
	identityKey, _, _, _, err := ssh.ParseAuthorizedKey(dataIdentityKey)
	require.NoError(t, err)
	dataPrivKey, err := os.ReadFile(signingKeyFile)
	require.NoError(t, err)
	ca, err := ssh.ParsePrivateKey(dataPrivKey)
	require.NoError(t, err)
	cert := &ssh.Certificate{
		Key:             identityKey, // SPKI "object" - key to which authority is delegated
		Serial:          uint64(1),
		CertType:        certType,
		ValidPrincipals: principals,
		ValidAfter:      0,
		ValidBefore:     uint64(time.Now().Add(time.Duration(validitySeconds) * time.Second).Unix()),
		// Permissions: a reasonable, standardesque collection
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				"permit-pty":              "",
				"permit-port-forwarding":  "",
				"permit-agent-forwarding": "",
				"permit-X11-forwarding":   "",
				"permit-user-rc":          "",
			},
		},
	}
	require.NoError(t, cert.SignCert(rand.Reader, ca))
	require.NoError(t, os.WriteFile(identityKeyFile+"-cert.pub", ssh.MarshalAuthorizedKey(cert), 0o700))
}

func testUserSuppliedSSHCAKeys(t *testing.T, u *url.URL) {
	defer test.MockVariableValue(&setting.SSH.EnableCertAuth, true)()

	reponame := "ssh-ca-test-repo"
	username := "user2"
	u.Path = fmt.Sprintf("%s/%s.git", username, reponame)

	// User configures this one as a normal key for pulling & pushing
	plainkey := fmt.Sprintf("%s-plainkey", username)

	// User configures this one as a CA key for signing other keys to allow them to pull & push
	cakey := fmt.Sprintf("%s-cakey", username)

	ctx := NewAPITestContext(t, username, reponame, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	t.Run("CreateRepository", doAPICreateRepository(ctx, nil, git.Sha1ObjectFormat))

	var cakeyID, plainkeyID int64
	withKeyFile(t, cakey, func(cakeyFile string) {
		withKeyFile(t, plainkey, func(plainkeyFile string) {
			// Start off with plainkey being the active key

			sshURL := createSSHUrl(ctx.GitPath(), u)

			// Shouldn't be able to clone initially as the keys aren't associated with the account yet
			t.Run("FailToClone", doGitCloneFail(sshURL))

			t.Run("CreatePlainKey", doAPICreateUserKey(ctx, plainkey, plainkeyFile, nil, func(t *testing.T, pk api.PublicKey) {
				plainkeyID = pk.ID
			}))

			t.Run("CloneWithPlainKey", doGitClone(t.TempDir(), sshURL))

			cakeyOptions := []string{
				`cert-authority`,
			}
			t.Run("CreateCAKey", doAPICreateUserKey(ctx, cakey, cakeyFile, cakeyOptions, func(t *testing.T, pk api.PublicKey) {
				cakeyID = pk.ID
			}))

			withActiveKey(t, cakeyFile, func() {
				// Not allowed to use a CA key directly to push/pull
				t.Run("FailToCloneWithCAKey", doGitCloneFail(sshURL))
			})

			withKeyFile(t, fmt.Sprintf("%s-ephemeralkey", username), func(ephemeralkeyFile string) {
				t.Run("FailToCloneWithEphemeralKey", doGitCloneFail(sshURL))
				endorseKey(t, ephemeralkeyFile, cakeyFile, ssh.HostCert, []string{username}, 60)
				t.Run("FailToCloneWithEphemeralKeyAndHostCert", doGitCloneFail(sshURL))
				endorseKey(t, ephemeralkeyFile, cakeyFile, ssh.UserCert, []string{username}, 60)
				t.Run("CloneWithEphemeralKeyAndUserCert", doGitClone(t.TempDir(), sshURL))
				endorseKey(t, ephemeralkeyFile, plainkeyFile, ssh.UserCert, []string{username}, 60)
				t.Run("FailToCloneWithEphemeralKeySignedByPlainNonCAKey", doGitCloneFail(sshURL))
				endorseKey(t, ephemeralkeyFile, cakeyFile, ssh.UserCert, []string{username}, -60)
				t.Run("FailToCloneWithEphemeralKeyExpiredCert", doGitCloneFail(sshURL))
				defer test.MockVariableValue(&setting.SSH.EnableCertAuth, false)()
				endorseKey(t, ephemeralkeyFile, cakeyFile, ssh.UserCert, []string{username}, 60)
				t.Run("FailToCloneWithEphemeralKeyAfterDisablingCertAuth", doGitCloneFail(sshURL))
			})

			t.Run("DeletePlainKey", doAPIDeleteUserKey(ctx, plainkeyID))
			t.Run("DeleteCAKey", doAPIDeleteUserKey(ctx, cakeyID))
		})
	})
}
