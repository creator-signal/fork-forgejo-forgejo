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
	"forgejo.org/modules/ssh"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

type principalMatchingTestCase = struct {
	Allowed  string
	Supplied string
	Matched  bool
}

func variousPrincipalMatchingTestCases(userName, otherUser string) []principalMatchingTestCase {
	userAndOther := userName + "," + otherUser
	unrelated := userName + "_," + otherUser + "_"
	repeated := userName + "," + userName

	return []principalMatchingTestCase{
		{"", "", false},
		{"", userName, true},
		{"", otherUser, false},
		{"", userAndOther, true},
		{"", unrelated, false},

		{userName, "", false},
		{userName, userName, true},
		{userName, otherUser, false},
		{userName, userAndOther, true},
		{userName, unrelated, false},

		{otherUser, "", false},
		{otherUser, userName, false},
		{otherUser, otherUser, true},
		{otherUser, userAndOther, true},
		{otherUser, unrelated, false},

		{userAndOther, "", false},
		{userAndOther, userName, true},
		{userAndOther, otherUser, true},
		{userAndOther, userAndOther, true},
		{userAndOther, unrelated, false},

		{userName, strings.ToUpper(userName), false},

		{userAndOther, userName[1 : len(userName)-1], false},
		{userAndOther, userName[:len(userName)/2], false},
		{userAndOther, userName[len(userName)/2:], false},
		{userAndOther, otherUser[len(otherUser)/2:], false},

		{repeated, userName, true},
		{userName, repeated, true},
	}
}

// This is, strictly, a unit test and should be in modules/ssh/ssh_test.go, but it's here so we can
// share variousPrincipalMatchingTestCases with the E2E tests elsewhere in this file
func TestCertificatePrincipalMatching(t *testing.T) {
	userName := "username"

	for _, c := range append(variousPrincipalMatchingTestCases(userName, "garbage"), []principalMatchingTestCase{
		{"a,b,c", "c", true},
		{"a,c,b", "c", true},
		{"c,a,b", "c", true},
		{"a,d", "a,b,c", true},
		{"d,b", "a,b,c", true},
		{"v,w", "x,y,z", false},
	}...) {
		allowed := ssh.SplitPrincipals(c.Allowed)
		supplied := ssh.SplitPrincipals(c.Supplied)
		if len(allowed) == 0 {
			allowed = []string{userName}
		}
		_, found := ssh.FindMatchingPrincipal(supplied, allowed)
		assert.Equal(t, c.Matched, found, "Expected that for allowed %q and supplied %q, found should be %v", c.Allowed, c.Supplied, c.Matched)
	}
}

func TestUserSuppliedSSHCAKeys(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		testUserSuppliedSSHCAKeyBasics(t, u)
		testCAKeyProvisioning(t, u)
		testPrincipalMatchingE2E(t, u)
		testTamperedCertificates(t, u)
	})
}

func endorseKey(t *testing.T, identityKeyFile, signingKeyFile string, certType uint32, principals []string, validitySeconds int) {
	endorseKeyWithValidityOffset(t, identityKeyFile, signingKeyFile, certType, principals, validitySeconds, 0, nil)
}

func endorseKeyWithValidityOffset(
	t *testing.T,
	identityKeyFile, signingKeyFile string,
	certType uint32,
	principals []string,
	validitySeconds int,
	validityWindowOffset int,
	tamperFn func(cert *gossh.Certificate),
) {
	dataIdentityKey, err := os.ReadFile(identityKeyFile + ".pub")
	require.NoError(t, err)
	identityKey, _, _, _, err := gossh.ParseAuthorizedKey(dataIdentityKey)
	require.NoError(t, err)
	dataPrivKey, err := os.ReadFile(signingKeyFile)
	require.NoError(t, err)
	ca, err := gossh.ParsePrivateKey(dataPrivKey)
	require.NoError(t, err)
	validAfter := uint64(time.Now().Add(time.Duration(validityWindowOffset) * time.Second).Unix())
	cert := &gossh.Certificate{
		Key:             identityKey, // SPKI "object" - key to which authority is delegated
		Serial:          uint64(1),
		CertType:        certType,
		ValidPrincipals: principals,
		ValidAfter:      validAfter,
		ValidBefore:     uint64(int64(validAfter) + int64(validitySeconds)),
		// Permissions: a reasonable, standardesque collection
		Permissions: gossh.Permissions{
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
	if tamperFn != nil {
		tamperFn(cert)
	}
	require.NoError(t, os.WriteFile(identityKeyFile+"-cert.pub", gossh.MarshalAuthorizedKey(cert), 0o700))
}

type sshKeyFixture = struct {
	name     string
	fileName string
}

type sshCAFixture = struct {
	ctx                   APITestContext
	relativeURL, cloneURL *url.URL
	plainkey, cakey       sshKeyFixture
}

// When callback is called, the plain key is the active key.
func sshCASetup(t *testing.T, u *url.URL, callback func(t *testing.T, f *sshCAFixture)) {
	defer test.MockVariableValue(&setting.SSH.EnableCertAuth, true)()

	repoName := "ssh-ca-test-repo"
	userName := "user2"

	relativeURL := &url.URL{}
	*relativeURL = *u
	relativeURL.Path = fmt.Sprintf("%s/%s.git", userName, repoName)

	// User configures this one as a normal key for pulling & pushing
	plainkeyName := fmt.Sprintf("%s-plainkey", userName)

	// User configures this one as a CA key for signing other keys to allow them to pull & push
	cakeyName := fmt.Sprintf("%s-cakey", userName)

	ctx := NewAPITestContext(t, userName, repoName, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

	t.Run("CreateRepository", doAPICreateRepository(ctx, nil, git.Sha1ObjectFormat))
	defer doAPIDeleteRepository(ctx)(t)
	cloneURL := createSSHUrl(ctx.GitPath(), u)

	withKeyFile(t, cakeyName, func(cakeyFile string) {
		withKeyFile(t, plainkeyName, func(plainkeyFile string) {
			callback(t, &sshCAFixture{
				ctx:         ctx,
				relativeURL: relativeURL,
				cloneURL:    cloneURL,
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
		t.Run("FailToClone", doGitCloneFail(f.cloneURL))

		withRegisteredKey(t, f.ctx, f.plainkey, nil, func(t *testing.T, plainKey api.PublicKey) {
			// Can clone using the plain key
			t.Run("CloneWithPlainKey", doGitClone(t.TempDir(), f.cloneURL))
			withRegisteredKey(t, f.ctx, f.cakey, caOptions(""), func(t *testing.T, caKey api.PublicKey) {
				withActiveKey(t, f.cakey.fileName, func() {
					// Not allowed to use a CA key directly to push/pull
					t.Run("FailToCloneWithCAKey", doGitCloneFail(f.cloneURL))
				})

				userName := f.ctx.Username
				withKeyFile(t, fmt.Sprintf("%s-ephemeralkey", userName), func(ephemeralkeyFile string) {
					// This test must appear first, before ephemeralkeyFile is ever endorsed
					t.Run("FailToCloneWithUnsignedEphemeralKey", doGitCloneFail(f.cloneURL))

					withKeyFile(t, fmt.Sprintf("%s-unregCA", userName), func(unregFile string) {
						withActiveKey(t, ephemeralkeyFile, func() {
							endorseKey(t, ephemeralkeyFile, unregFile, gossh.UserCert, []string{userName}, 60)
							t.Run("FailToCloneWithUnregisteredCAKey", doGitCloneFail(f.cloneURL))
						})
					})

					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, gossh.HostCert, []string{userName}, 60)
					t.Run("FailToCloneWithEphemeralKeyAndHostCert", doGitCloneFail(f.cloneURL))

					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, []string{userName}, 60)
					t.Run("CloneWithEphemeralKeyAndUserCert", doGitClone(t.TempDir(), f.cloneURL))

					endorseKey(t, ephemeralkeyFile, f.plainkey.fileName, gossh.UserCert, []string{userName}, 60)
					t.Run("FailToCloneWithEphemeralKeySignedByPlainNonCAKey", doGitCloneFail(f.cloneURL))

					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, []string{userName}, -60)
					t.Run("FailToCloneWithEphemeralKeyExpiredCert", doGitCloneFail(f.cloneURL))

					endorseKeyWithValidityOffset(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, []string{userName}, 60, 60, nil)
					t.Run("FailToCloneWithFutureValidityCert", doGitCloneFail(f.cloneURL))

					defer test.MockVariableValue(&setting.SSH.EnableCertAuth, false)()
					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, []string{userName}, 60)
					t.Run("FailToCloneWithEphemeralKeyAfterDisablingCertAuth", doGitCloneFail(f.cloneURL))
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

func testPrincipalMatchingE2E(t *testing.T, u *url.URL) {
	sshCASetup(t, u, func(t *testing.T, f *sshCAFixture) {
		for _, c := range variousPrincipalMatchingTestCases(f.ctx.Username, "not-"+f.ctx.Username+"-not") {
			withRegisteredKey(t, f.ctx, f.cakey, caOptions(c.Allowed), func(t *testing.T, caKey api.PublicKey) {
				withKeyFile(t, fmt.Sprintf("%s-ephemeralkey", f.ctx.Username), func(ephemeralkeyFile string) {
					endorseKey(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, ssh.SplitPrincipals(c.Supplied), 60)
					desc := fmt.Sprintf("allowed=%s supplied=%s ok=%v", c.Allowed, c.Supplied, c.Matched)
					if c.Matched {
						t.Run(desc, doGitClone(t.TempDir(), f.cloneURL))
					} else {
						t.Run(desc, doGitCloneFail(f.cloneURL))
					}
				})
			})
		}
	})
}

func testTamperedCertificates(t *testing.T, u *url.URL) {
	sshCASetup(t, u, func(t *testing.T, f *sshCAFixture) {
		withRegisteredKey(t, f.ctx, f.cakey, caOptions(""), func(t *testing.T, caKey api.PublicKey) {
			withKeyFile(t, fmt.Sprintf("%s-ephemeralkey", f.ctx.Username), func(ephemeralkeyFile string) {
				// Make sure an unmodified certificate is accepted first
				endorseKeyWithValidityOffset(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, []string{f.ctx.Username}, 60, 0, nil)
				for _, c := range []struct {
					desc     string
					tamperFn func(cert *gossh.Certificate)
				}{
					{"nonce", func(cert *gossh.Certificate) { cert.Nonce[len(cert.Nonce)-1]++ }},
					{"principals", func(cert *gossh.Certificate) { cert.ValidPrincipals = []string{"invalid"} }},
					{"signature", func(cert *gossh.Certificate) { cert.Signature.Blob[len(cert.Signature.Blob)/2]++ }},
					{"validity1", func(cert *gossh.Certificate) { cert.ValidAfter += 60; cert.ValidBefore += 60 }},
					{"validity2", func(cert *gossh.Certificate) { cert.ValidAfter -= 600; cert.ValidBefore -= 600 }},
					{"signingkey", func(cert *gossh.Certificate) { cert.SignatureKey = cert.Key }},
				} {
					endorseKeyWithValidityOffset(t, ephemeralkeyFile, f.cakey.fileName, gossh.UserCert, []string{f.ctx.Username}, 60, 0, c.tamperFn)
					t.Run("Tamper with "+c.desc, doGitCloneFail(f.cloneURL))
				}
			})
		})
	})
}
