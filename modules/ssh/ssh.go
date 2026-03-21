// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package ssh

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"

	asymkey_model "forgejo.org/models/asymkey"
	"forgejo.org/models/db"
	"forgejo.org/models/user"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
)

func getExitStatusFromError(err error) int {
	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 1
	}

	waitStatus, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		// This is a fallback and should at least let us return something useful
		// when running on Windows, even if it isn't completely accurate.
		if exitErr.Success() {
			return 0
		}

		return 1
	}

	return waitStatus.ExitStatus()
}

func sessionHandler(session ssh.Session) {
	keyID := session.ConnPermissions().Extensions["forgejo-key-id"]

	command := session.RawCommand()

	logger.Trace("SSH: Payload: %v", command)

	args := []string{"--config=" + setting.CustomConf, "serv", "key-" + keyID}
	logger.Trace("SSH: Arguments: %v", args)

	ctx, cancel := context.WithCancel(session.Context())
	defer cancel()

	gitProtocol := ""
	for _, env := range session.Environ() {
		if strings.HasPrefix(env, "GIT_PROTOCOL=") {
			_, gitProtocol, _ = strings.Cut(env, "=")
			break
		}
	}

	cmd := exec.CommandContext(ctx, setting.AppPath, args...)
	cmd.Env = append(
		os.Environ(),
		"SSH_ORIGINAL_COMMAND="+command,
		"SKIP_MINWINSVC=1",
		"GIT_PROTOCOL="+gitProtocol,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("SSH: StdoutPipe: %v", err)
		return
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("SSH: StderrPipe: %v", err)
		return
	}
	defer stderr.Close()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Error("SSH: StdinPipe: %v", err)
		return
	}
	defer stdin.Close()

	process.SetSysProcAttribute(cmd)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	if err = cmd.Start(); err != nil {
		logger.Error("SSH: Start: %v", err)
		return
	}

	go func() {
		defer stdin.Close()
		if _, err := io.Copy(stdin, session); err != nil {
			logger.Error("Failed to write session to stdin. %s", err)
		}
	}()

	go func() {
		defer wg.Done()
		defer stdout.Close()
		if _, err := io.Copy(session, stdout); err != nil {
			logger.Error("Failed to write stdout to session. %s", err)
		}
	}()

	go func() {
		defer wg.Done()
		defer stderr.Close()
		if _, err := io.Copy(session.Stderr(), stderr); err != nil {
			logger.Error("Failed to write stderr to session. %s", err)
		}
	}()

	// Ensure all the output has been written before we wait on the command
	// to exit.
	wg.Wait()

	// Wait for the command to exit and log any errors we get
	err = cmd.Wait()
	if err != nil {
		// Cannot use errors.Is here because ExitError doesn't implement Is
		// Thus errors.Is will do equality test NOT type comparison
		if _, ok := err.(*exec.ExitError); !ok {
			logger.Error("SSH: Wait: %v", err)
		}
	}

	if err := session.Exit(getExitStatusFromError(err)); err != nil && !errors.Is(err, io.EOF) {
		logger.Error("Session failed to exit. %s", err)
	}
}

func publicKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	if logger.LevelEnabled(log.DEBUG) { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		logger.Debug("Handle Public Key: Fingerprint: %s from %s", gossh.FingerprintSHA256(key), ctx.RemoteAddr())
	}

	var loginOk bool
	if ctx.User() != setting.SSH.BuiltinServerUser {
		logger.Warn("Invalid SSH username %s - must use %s for all git operations via ssh", ctx.User(), setting.SSH.BuiltinServerUser)
		loginOk = false
	} else if cert, ok := key.(*gossh.Certificate); ok {
		loginOk = authenticationWithCert(ctx, cert)
	} else {
		loginOk = authenticationWithPlainKey(ctx, key)
	}

	if !loginOk {
		logger.Warn("Failed authentication attempt from %s", ctx.RemoteAddr())
	}
	return loginOk
}

func authenticationWithCert(ctx ssh.Context, cert *gossh.Certificate) bool {
	if logger.LevelEnabled(log.DEBUG) { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		logger.Debug("Handle Certificate: %s Fingerprint: %s is a certificate", ctx.RemoteAddr(), gossh.FingerprintSHA256(cert))
	}

	if cert.CertType != gossh.UserCert {
		logger.Warn("Certificate Rejected: Not a user certificate")
		return false
	}

	var userCAs []*asymkey_model.PublicKey
	var err error
	if setting.SSH.EnableCertAuth {
		signerFingerprint := gossh.FingerprintSHA256(cert.SignatureKey)
		userCAs, err = db.Find[asymkey_model.PublicKey](ctx, asymkey_model.FindPublicKeyOptions{
			Fingerprint: signerFingerprint,
			KeyTypes:    []asymkey_model.KeyType{asymkey_model.KeyTypeUser},
			IsCA:        sql.NullBool{Bool: true, Valid: true},
		})
		if err != nil {
			logger.Error("Failed retrieving user CAs for fingerprint %s: %v", signerFingerprint, err.Error())
			return false
		}
	}

	var activeKeyID int64
	var authenticated bool
	if len(userCAs) == 1 {
		activeKeyID, authenticated = authenticationWithUserSuppliedCA(ctx, cert, userCAs[0])
	} else {
		activeKeyID, authenticated = authenticationWithTrustedCA(ctx, cert)
	}

	if authenticated {
		if ctx.Permissions().Extensions == nil {
			ctx.Permissions().Extensions = map[string]string{}
		}
		ctx.Permissions().Extensions["forgejo-key-id"] = strconv.FormatInt(activeKeyID, 10)
	}
	return authenticated
}

func authenticationWithUserSuppliedCA(ctx ssh.Context, cert *gossh.Certificate, userCA *asymkey_model.PublicKey) (keyID int64, ok bool) {
	allowedPrincipals := SplitPrincipals(userCA.Principals)
	if len(allowedPrincipals) == 0 {
		owner, err := user.GetUserByID(ctx, userCA.OwnerID)
		if err != nil {
			logger.Error("Failed to retrieve owner for user SSH CA key ID %d: %v", userCA.ID, err)
			return 0, false
		}
		allowedPrincipals = []string{owner.Name}
	}

	principal, matchedPrincipal := FindMatchingPrincipal(cert.ValidPrincipals, allowedPrincipals)
	if !matchedPrincipal {
		logger.Error("No principal listed in certificate is also listed in CA key's allowed principals list")
		return 0, false
	}

	c := &gossh.CertChecker{}
	err := c.CheckCert(principal, cert)
	if err != nil {
		logger.Error("Certificate rejected: %v", err)
		return 0, false
	}

	if logger.LevelEnabled(log.DEBUG) { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		logger.Debug("Successfully authenticated: %s Certificate Fingerprint: %s Principal: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(cert), principal)
	}

	return userCA.ID, true
}

// Exposed for testing
func SplitPrincipals(principals string) []string {
	if principals == "" {
		return nil
	}
	return strings.Split(principals, ",")
}

// Exposed for testing
func FindMatchingPrincipal(suppliedPrincipals, allowedPrincipals []string) (string, bool) {
	for _, suppliedPrincipal := range suppliedPrincipals {
		if slices.Contains(allowedPrincipals, suppliedPrincipal) {
			return suppliedPrincipal, true
		}
	}
	return "", false
}

func authenticationWithTrustedCA(ctx ssh.Context, cert *gossh.Certificate) (keyID int64, ok bool) {
	if len(setting.SSH.TrustedUserCAKeys) == 0 {
		logger.Warn("Certificate Rejected: No trusted certificate authorities for this server")
		return 0, false
	}

	// look for the exact principal
principalLoop:
	for _, principal := range cert.ValidPrincipals {
		pkey, err := asymkey_model.SearchPublicKeyByContentExact(ctx, principal)
		if err != nil {
			if asymkey_model.IsErrKeyNotExist(err) {
				logger.Debug("Principal Rejected: %s Unknown Principal: %s", ctx.RemoteAddr(), principal)
				continue principalLoop
			}
			logger.Error("SearchPublicKeyByContentExact: %v", err)
			return 0, false
		}

		c := &gossh.CertChecker{
			IsUserAuthority: func(auth gossh.PublicKey) bool {
				marshaled := auth.Marshal()
				for _, k := range setting.SSH.TrustedUserCAKeysParsed {
					if bytes.Equal(marshaled, k.Marshal()) {
						return true
					}
				}

				return false
			},
		}

		// check the CA of the cert
		if !c.IsUserAuthority(cert.SignatureKey) {
			if logger.LevelEnabled(log.DEBUG) {
				logger.Debug("Principal Rejected: %s Untrusted Authority Signature Fingerprint %s for Principal: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(cert.SignatureKey), principal)
			}
			continue principalLoop
		}

		// validate the cert for this principal
		if err := c.CheckCert(principal, cert); err != nil {
			// User is presenting an invalid certificate - STOP any further processing
			logger.Error("Invalid Certificate KeyID %s with Signature Fingerprint %s presented for Principal: %s from %s", cert.KeyId, gossh.FingerprintSHA256(cert.SignatureKey), principal, ctx.RemoteAddr())
			return 0, false
		}

		if logger.LevelEnabled(log.DEBUG) { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
			logger.Debug("Successfully authenticated: %s Certificate Fingerprint: %s Principal: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(cert), principal)
		}

		return pkey.ID, true
	}

	logger.Warn("From %s Fingerprint: %s is a certificate, but no valid principals found", ctx.RemoteAddr(), gossh.FingerprintSHA256(cert))
	return 0, false
}

func authenticationWithPlainKey(ctx ssh.Context, key ssh.PublicKey) bool {
	if logger.LevelEnabled(log.DEBUG) { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		logger.Debug("Handle Public Key: %s Fingerprint: %s is not a certificate", ctx.RemoteAddr(), gossh.FingerprintSHA256(key))
	}

	pkey, err := asymkey_model.SearchPublicKeyByContent(ctx, strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key))))
	if err != nil {
		if asymkey_model.IsErrKeyNotExist(err) {
			logger.Warn("Unknown public key: %s from %s", gossh.FingerprintSHA256(key), ctx.RemoteAddr())
			return false
		}
		logger.Error("SearchPublicKeyByContent: %v", err)
		return false
	}

	// We deny use of CA keys as plain authentication keys directly. They're only allowed to sign certificates.
	if pkey.IsCA {
		logger.Warn("Rejecting direct use of CA key: %s from %s", gossh.FingerprintSHA256(key), ctx.RemoteAddr())
		return false
	}

	if logger.LevelEnabled(log.DEBUG) { // <- FingerprintSHA256 is kinda expensive so only calculate it if necessary
		logger.Debug("Successfully authenticated: %s Public Key Fingerprint: %s", ctx.RemoteAddr(), gossh.FingerprintSHA256(key))
	}
	if ctx.Permissions().Extensions == nil {
		ctx.Permissions().Extensions = map[string]string{}
	}
	ctx.Permissions().Extensions["forgejo-key-id"] = strconv.FormatInt(pkey.ID, 10)

	return true
}

// sshConnectionFailed logs a failed connection
// -  this mainly exists to give a nice function name in logging
func sshConnectionFailed(conn net.Conn, err error) {
	// Log the underlying error with a specific message
	logger.Warn("Failed connection from %s with error: %v", conn.RemoteAddr(), err)
	// Log with the standard failed authentication from message for simpler fail2ban configuration
	logger.Warn("Failed authentication attempt from %s", conn.RemoteAddr())
}

// Listen starts a SSH server listens on given port.
func Listen(host string, port int, ciphers, keyExchanges, macs []string) {
	srv := ssh.Server{
		Addr:             net.JoinHostPort(host, strconv.Itoa(port)),
		PublicKeyHandler: publicKeyHandler,
		Handler:          sessionHandler,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			config := &gossh.ServerConfig{}
			config.KeyExchanges = keyExchanges
			config.MACs = macs
			config.Ciphers = ciphers
			return config
		},
		ConnectionFailedCallback: sshConnectionFailed,
		// We need to explicitly disable the PtyCallback so text displays
		// properly.
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			return false
		},
	}

	keys := make([]string, 0, len(setting.SSH.ServerHostKeys))
	for _, key := range setting.SSH.ServerHostKeys {
		isExist, err := util.IsExist(key)
		if err != nil {
			log.Fatal("Unable to check if %s exists. Error: %v", setting.SSH.ServerHostKeys, err)
		}
		if isExist {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		filePath := filepath.Dir(setting.SSH.ServerHostKeys[0])

		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			logger.Error("Failed to create dir %s: %v", filePath, err)
		}

		err := GenKeyPair(setting.SSH.ServerHostKeys[0])
		if err != nil {
			log.Fatal("Failed to generate private key: %v", err)
		}
		logger.Trace("New private key is generated: %s", setting.SSH.ServerHostKeys[0])
		keys = append(keys, setting.SSH.ServerHostKeys[0])
	}

	for _, key := range keys {
		logger.Info("Adding SSH host key: %s", key)
		err := srv.SetOption(ssh.HostKeyFile(key))
		if err != nil {
			logger.Error("Failed to set Host Key. %s", err)
		}
	}

	go func() {
		_, _, finished := process.GetManager().AddTypedContext(graceful.GetManager().HammerContext(), "Service: Built-in SSH server", process.SystemProcessType, true)
		defer finished()
		listen(&srv)
	}()
}

// GenKeyPair make a pair of public and private keys for SSH access.
// Public key is encoded in the format for inclusion in an OpenSSH authorized_keys file.
// Private Key generated is PEM encoded
func GenKeyPair(keyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	f, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			logger.Error("Close: %v", err)
		}
	}()

	if err := pem.Encode(f, privateKeyPEM); err != nil {
		return err
	}

	// generate public key
	pub, err := gossh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	public := gossh.MarshalAuthorizedKey(pub)
	p, err := os.OpenFile(keyPath+".pub", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if err = p.Close(); err != nil {
			logger.Error("Close: %v", err)
		}
	}()
	_, err = p.Write(public)
	return err
}
