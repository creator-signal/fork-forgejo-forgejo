// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"strings"
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
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		testUserSuppliedSSHCAKeyBasics(t, u)
		testCAKeyProvisioning(t, u)
		// testPrincipalMatchingE2E(t, u)
	})
}

func endorseKey(t *testing.T, identityKeyFile, signingKeyFile string, certType uint32, principals []string, validitySeconds int) {
	endorseKeyWithValidityOffset(t, identityKeyFile, signingKeyFile, certType, principals, validitySeconds, 0)
}

func endorseKeyWithValidityOffset(t *testing.T, identityKeyFile, signingKeyFile string, certType uint32, principals []string, validitySeconds int, validityWindowOffset int) {
	dataIdentityKey, err := os.ReadFile(identityKeyFile + ".pub")
	require.NoError(t, err)
	identityKey, _, _, _, err := ssh.ParseAuthorizedKey(dataIdentityKey)
	require.NoError(t, err)
	dataPrivKey, err := os.ReadFile(signingKeyFile)
	require.NoError(t, err)
	ca, err := ssh.ParsePrivateKey(dataPrivKey)
	require.NoError(t, err)
	validAfter := uint64(time.Now().Add(time.Duration(validityWindowOffset) * time.Second).Unix())
	cert := &ssh.Certificate{
		Key:             identityKey, // SPKI "object" - key to which authority is delegated
		Serial:          uint64(1),
		CertType:        certType,
		ValidPrincipals: principals,
		ValidAfter:      validAfter,
		ValidBefore:     uint64(int64(validAfter) + int64(validitySeconds)),
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

type sshKeyFixture = struct {
	name     string
	fileName string
}

type sshCAFixture = struct {
	ctx                   APITestContext
	relativeUrl, cloneUrl *url.URL
	plainkey, cakey       sshKeyFixture
}

// When callback is called, the plain key is the active key.
func sshCASetup(t *testing.T, u *url.URL, callback func(t *testing.T, f *sshCAFixture)) {
	defer test.MockVariableValue(&setting.SSH.EnableCertAuth, true)()

	repoName := "ssh-ca-test-repo"
	userName := "user2"

	relativeUrl := &url.URL{}
	*relativeUrl = *u
	relativeUrl.Path = fmt.Sprintf("%s/%s.git", userName, repoName)

	// User configures this one as a normal key for pulling & pushing
	plainkeyName := fmt.Sprintf("%s-plainkey", userName)

	// User configures this one as a CA key for signing other keys to allow them to pull & push
	cakeyName := fmt.Sprintf("%s-cakey", userName)

	ctx := NewAPITestContext(t, userName, repoName, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	t.Run("CreateRepository", doAPICreateRepository(ctx, nil, git.Sha1ObjectFormat))
	defer doAPIDeleteRepository(ctx)(t)
	cloneUrl := createSSHUrl(ctx.GitPath(), u)

	withKeyFile(t, cakeyName, func(cakeyFile string) {
		withKeyFile(t, plainkeyName, func(plainkeyFile string) {
			callback(t, &sshCAFixture{
				ctx:         ctx,
				relativeUrl: relativeUrl,
				cloneUrl:    cloneUrl,
				plainkey: sshKeyFixture{
					name:     plainkeyName,
					fileName: plainkeyFile,
				},
				cakey: sshKeyFixture{
					name:     cakeyName,
					fileName: cakeyFile,
				},
			})
		})
	})
}

func withRegisteredKey(t *testing.T, ctx APITestContext, k sshKeyFixture, keyOptions []string, callback func(t *testing.T, pk api.PublicKey)) {
	t.Run("CreateKey "+k.name, doAPICreateUserKey(ctx, k.name, k.fileName, keyOptions, func(t *testing.T, pk api.PublicKey) {
		defer t.Run("DeleteKey "+k.name, doAPIDeleteUserKey(ctx, pk.ID))
		callback(t, pk)
	}))
}

func caOptions(principals string, emptyPrincipalsAllowed ...bool) []string {
	options := []string{`cert-authority`}
	if principals != "" || (len(emptyPrincipalsAllowed) > 0 && emptyPrincipalsAllowed[0]) {
		options = append(options, fmt.Sprintf(`principals="%s"`, principals))
	}
	return options
}

func testUserSuppliedSSHCAKeyBasics(t *testing.T, u *url.URL) {
	sshCASetup(t, u, func(t *testing.T, f *sshCAFixture) {
		// Shouldn't be able to clone initially as the keys aren't associated with the account yet
		t.Run("FailToClone", doGitCloneFail(f.cloneUrl))

		withRegisteredKey(t, f.ctx, f.plainkey, nil, func(t *testing.T, plainKey api.PublicKey) {
			// Can clone using the plain key
			t.Run("CloneWithPlainKey", doGitClone(t.TempDir(), f.cloneUrl))
			withRegisteredKey(t, f.ctx, f.cakey, caOptions(""), func(t *testing.T, caKey api.PublicKey) {
				withActiveKey(t, f.cakey.fileName, func() {
					// Not allowed to use a CA key directly to push/pull
					t.Run("FailToCloneWithCAKey", doGitCloneFail(f.cloneUrl))
				})

				userName := f.ctx.Username
				withKeyFile(t, fmt.Sprintf("%s-ephemeralkey", userName), func(ephemeralkeyFile string) {
					// This test must appear first, before ephemeralkeyFile is ever endorsed
					t.Run("FailToCloneWithUnsignedEphemeralKey", doGitCloneFail(f.cloneUrl))

					withKeyFile(t, fmt.Sprintf("%s-unregCA", userName), func(unregFile string) {
						withActiveKey(t, ephemeralkeyFile, func() {
							endorseKey(t, ephemeralkeyFile, unregFile, ssh.UserCert, []string{userName}, 60)
							t.Run("FailToCloneWithUnregisteredCAKey", doGitCloneFail(f.cloneUrl))
						})
					})

					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, ssh.HostCert, []string{userName}, 60)
					t.Run("FailToCloneWithEphemeralKeyAndHostCert", doGitCloneFail(f.cloneUrl))
					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, ssh.UserCert, []string{userName}, 60)
					t.Run("CloneWithEphemeralKeyAndUserCert", doGitClone(t.TempDir(), f.cloneUrl))
					endorseKey(t, ephemeralkeyFile, f.plainkey.fileName, ssh.UserCert, []string{userName}, 60)
					t.Run("FailToCloneWithEphemeralKeySignedByPlainNonCAKey", doGitCloneFail(f.cloneUrl))
					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, ssh.UserCert, []string{userName}, -60)
					t.Run("FailToCloneWithEphemeralKeyExpiredCert", doGitCloneFail(f.cloneUrl))

					endorseKeyWithValidityOffset(t, ephemeralkeyFile, f.cakey.fileName, ssh.UserCert, []string{userName}, 60, 60)
					t.Run("FailToCloneWithFutureValidityCert", doGitCloneFail(f.cloneUrl))

					defer test.MockVariableValue(&setting.SSH.EnableCertAuth, false)()
					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, ssh.UserCert, []string{userName}, 60)
					t.Run("FailToCloneWithEphemeralKeyAfterDisablingCertAuth", doGitCloneFail(f.cloneUrl))
				})
			})
		})
	})
}

func testCAKeyProvisioning(t *testing.T, u *url.URL) {
	sshCASetup(t, u, func(t *testing.T, f *sshCAFixture) {
		// Register & delete
		withRegisteredKey(t, f.ctx, f.cakey, caOptions("firstRegistration"), func(t *testing.T, caKey api.PublicKey) {})

		// Then register...
		withRegisteredKey(t, f.ctx, f.cakey, caOptions("firstRegistration"), func(t *testing.T, caKey api.PublicKey) {
			// ... and try registering the same key again with a different name/options...
			f.ctx.ExpectedCode = 422 // Unprocessable entity (because duplicate key content)
			secondOptions := caOptions("secondRegistration")
			t.Run("FailToCreateSecondCAInstanceOfCAKey", doAPICreateUserKey(f.ctx, f.cakey.name+"2", f.cakey.fileName, secondOptions))
			// ... OK but let's also try registering it as a plain key...
			t.Run("FailToCreateSecondPlainInstanceOfCAKey", doAPICreateUserKey(f.ctx, f.cakey.name+"3", f.cakey.fileName, nil))
		})

		// plain key provisioned with SSH_ENABLE_CERT_AUTH true
		// plain key provisioned with SSH_ENABLE_CERT_AUTH false
		expectKeyRegistration(t, f.ctx, f.plainkey, nil, 0, 0)

		// CA key provisioned with principals absent SSH_ENABLE_CERT_AUTH true
		// CA key provisioned with principals absent SSH_ENABLE_CERT_AUTH false - rejected
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions(""), 0, 422)

		// CA key provisioned with valid principals and SSH_ENABLE_CERT_AUTH true
		// CA key provisioned with valid principals and SSH_ENABLE_CERT_AUTH false - rejected
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions("a,b,c"), 0, 422)
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions(f.ctx.Username), 0, 422)

		// CA key provisioned with present but empty principals and SSH_ENABLE_CERT_AUTH true - rejected
		// CA key provisioned with present but empty principals and SSH_ENABLE_CERT_AUTH false - rejected
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions("", true), 422, 422)

		// CA key provisioned with present but invalid principals and SSH_ENABLE_CERT_AUTH true - rejected
		// CA key provisioned with present but invalid principals and SSH_ENABLE_CERT_AUTH false - rejected
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions(`with, spaces, forbidden`), 422, 422)
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions(`with,\backslash,forbidden`), 422, 422)
		expectKeyRegistration(t, f.ctx, f.cakey, caOptions("with,nul\000,forbidden"), 422, 422)
	})
}

func expectKeyRegistration(t *testing.T, ctx APITestContext, k sshKeyFixture, keyOptions []string, expectedWithCAEnabled, expectedWithCADisabled int) {
	// Assumption: SSH_ENABLE_CERT_AUTH true on entry
	expectKeyRegistration1(t, ctx, k, keyOptions, expectedWithCAEnabled, true)
	defer test.MockVariableValue(&setting.SSH.EnableCertAuth, false)()
	expectKeyRegistration1(t, ctx, k, keyOptions, expectedWithCADisabled, false)
}

func expectKeyRegistration1(t *testing.T, ctx APITestContext, k sshKeyFixture, keyOptions []string, expectedCode int, isEnabled bool) {
	t.Run(
		fmt.Sprintf("SSH_ENABLED_CERT_AUTH=%v options=%s expectedCode=%d",
			isEnabled,
			strings.Join(keyOptions, ","),
			expectedCode),
		func(t *testing.T) {
			ctx.ExpectedCode = expectedCode
			if expectedCode == 0 {
				withRegisteredKey(t, ctx, k, keyOptions, func(t *testing.T, pk api.PublicKey) {})
			} else {
				doAPICreateUserKey(ctx, k.name, k.fileName, keyOptions)(t)
			}
		},
	)
}

// func testPrincipalMatchingE2E(t *testing.T, u *url.URL) {
// }
