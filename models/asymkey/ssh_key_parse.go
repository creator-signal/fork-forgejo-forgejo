// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"forgejo.org/modules/log"
	"forgejo.org/modules/process"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"

	"golang.org/x/crypto/ssh"
)

//  ____  __.             __________
// |    |/ _|____ ___.__. \______   \_____ _______  ______ ___________
// |      <_/ __ <   |  |  |     ___/\__  \\_  __ \/  ___// __ \_  __ \
// |    |  \  ___/\___  |  |    |     / __ \|  | \/\___ \\  ___/|  | \/
// |____|__ \___  > ____|  |____|    (____  /__|  /____  >\___  >__|
//         \/   \/\/                      \/           \/     \/
//
// This file contains functions for parsing ssh-keys
//
// TODO: Consider if these functions belong in models - no other models function call them or are called by them
// They may belong in a service or a module

const ssh2keyStart = "---- BEGIN SSH2 PUBLIC KEY ----"

func extractTypeFromBase64Key(key string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(b) < 4 {
		return "", fmt.Errorf("invalid key format: %w", err)
	}

	keyLength := int(binary.BigEndian.Uint32(b))
	if len(b) < 4+keyLength {
		return "", fmt.Errorf("invalid key format: not enough length %d", keyLength)
	}

	return string(b[4 : 4+keyLength]), nil
}

// parseKeyString parses any key string in OpenSSH or SSH2 format to clean OpenSSH string (RFC4253).
// Also returns key options (e.g. from OpenSSH authorized_keys lines) if they are present.
func parseKeyString(content string) (string, []string, error) {
	// remove whitespace at start and end
	content = strings.TrimSpace(content)

	switch {
	case strings.HasPrefix(content, ssh2keyStart):
		// Convert SSH2 file format to OpenSSH format

		// Transform all legal line endings to a single "\n".
		content = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(content)

		lines := strings.Split(content, "\n")
		continuationLine := false

		var keyContent string
		for _, line := range lines {
			// Skip lines that:
			// 1) are a continuation of the previous line,
			// 2) contain ":" as that are comment lines
			// 3) contain "-" as that are begin and end tags
			if continuationLine || strings.ContainsAny(line, ":-") {
				continuationLine = strings.HasSuffix(line, "\\")
			} else {
				keyContent += line
			}
		}

		keyType, err := extractTypeFromBase64Key(keyContent)
		if err != nil {
			return "", nil, fmt.Errorf("extractTypeFromBase64Key: %w", err)
		}

		content = keyType + " " + keyContent
	case strings.Contains(content, "-----BEGIN"):
		// Convert PEM Keys to OpenSSH format

		// Transform all legal line endings to a single "\n".
		content = strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(content)

		block, _ := pem.Decode([]byte(content))
		if block == nil {
			return "", nil, errors.New("failed to parse PEM block containing the public key")
		}
		if strings.Contains(block.Type, "PRIVATE") {
			return "", nil, ErrKeyIsPrivate
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			var pk rsa.PublicKey
			_, err2 := asn1.Unmarshal(block.Bytes, &pk)
			if err2 != nil {
				return "", nil, fmt.Errorf("failed to parse DER encoded public key as either PKIX or PEM RSA Key: %v %w", err, err2)
			}
			pub = &pk
		}

		sshKey, err := ssh.NewPublicKey(pub)
		if err != nil {
			return "", nil, fmt.Errorf("unable to convert to ssh public key: %w", err)
		}
		content = string(ssh.MarshalAuthorizedKey(sshKey))
	default:
		// Just try OpenSSH format.
	}

	// Parse OpenSSH format.

	// Remove all newlines
	content = strings.NewReplacer("\r\n", "", "\n", "").Replace(content)

	sshKey, comment, options, _, err := ssh.ParseAuthorizedKey([]byte(content))
	if err != nil {
		return "", nil, fmt.Errorf("invalid ssh public key: %w", err)
	}

	validated := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshKey)))
	if comment != "" {
		validated = validated + " " + comment
	}

	return validated, options, nil
}

// CheckPublicKeyString checks if the given public key string is recognized by SSH.
// It returns the actual public key line on success, along with any key options that were present.
func CheckPublicKeyString(content string) (_ string, options []string, err error) {
	content, options, err = parseKeyString(content)
	if err != nil {
		return "", nil, err
	}

	content = strings.TrimRight(content, "\n\r")
	if strings.ContainsAny(content, "\n\r") {
		return "", nil, util.NewInvalidArgumentErrorf("only a single line with a single key please")
	}

	// remove any unnecessary whitespace now
	content = strings.TrimSpace(content)

	if !setting.SSH.MinimumKeySizeCheck {
		return content, nil, nil
	}

	var (
		fnName  string
		keyType string
		length  int
	)
	if len(setting.SSH.KeygenPath) == 0 {
		fnName = "SSHNativeParsePublicKey"
		keyType, length, err = SSHNativeParsePublicKey(content)
	} else {
		fnName = "SSHKeyGenParsePublicKey"
		keyType, length, err = SSHKeyGenParsePublicKey(content)
	}
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", fnName, err)
	}
	log.Trace("Key info [native: %v]: %s-%d", setting.SSH.StartBuiltinServer, keyType, length)

	if minLen, found := setting.SSH.MinimumKeySizes[keyType]; found && length >= minLen {
		return content, options, nil
	} else if found && length < minLen {
		return "", nil, fmt.Errorf("key length is not enough: got %d, needs %d", length, minLen)
	}
	return "", nil, fmt.Errorf("key type is not allowed: %s", keyType)
}

// Extracts `cert-authority` and `principals` options from authorized_keys key options.
// Yields isCA true if `cert-authority` present, and principals the string value directly
// from the options field.
func ParseCAOptions(options []string) (isCA bool, principals string, err error) {
	principalsPresent := false
	for _, opt := range options {
		optLC := strings.ToLower(opt)
		switch {
		case optLC == "cert-authority":
			isCA = true
		case strings.HasPrefix(opt, `principals="`):
			principalsPresent = true
			principals = opt[len(`principals="`) : len(opt)-1] // drop trailing quote too
		}
	}

	if isCA {
		if !setting.SSH.EnableCertAuth {
			return false, "", ErrSSHCADisabled{}
		}
		if principalsPresent {
			// We reject: empty string; anything with a control character; spaces, backslash, doublequote
			// This is because OpenSSH's parser is simple and well defined but WEIRD.
			// If we want to allow exotic principal strings we can do so later.
			principalsOk := principals != "" && !strings.ContainsFunc(principals, func(r rune) bool {
				return r <= 0x20 || r == '\\' || r == '"'
			})
			if !principalsOk {
				return false, "", ErrInvalidSSHCAPrincipals{InvalidPrincipals: principals}
			}
		}
	}

	return isCA, principals, nil
}

// SSHNativeParsePublicKey extracts the key type and length using the golang SSH library.
func SSHNativeParsePublicKey(keyLine string) (string, int, error) {
	fields := strings.Fields(keyLine)
	if len(fields) < 2 {
		return "", 0, fmt.Errorf("not enough fields in public key line: %s", keyLine)
	}

	raw, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return "", 0, err
	}

	pkey, err := ssh.ParsePublicKey(raw)
	if err != nil {
		if strings.Contains(err.Error(), "ssh: unknown key algorithm") {
			return "", 0, ErrKeyUnableVerify{err.Error()}
		}
		return "", 0, fmt.Errorf("ParsePublicKey: %w", err)
	}

	pkeyType := pkey.Type()
	if certPkey, ok := pkey.(*ssh.Certificate); ok {
		pkeyType = certPkey.Key.Type()
	}

	// The ssh library can parse the key, so next we find out what key exactly we have.
	switch pkeyType {
	case ssh.KeyAlgoDSA: //nolint:staticcheck
		rawPub := struct {
			Name       string
			P, Q, G, Y *big.Int
		}{}
		if err := ssh.Unmarshal(pkey.Marshal(), &rawPub); err != nil {
			return "", 0, err
		}
		// as per https://bugzilla.mindrot.org/show_bug.cgi?id=1647 we should never
		// see dsa keys != 1024 bit, but as it seems to work, we will not check here
		return "dsa", rawPub.P.BitLen(), nil // use P as per crypto/dsa/dsa.go (is L)
	case ssh.KeyAlgoRSA:
		rawPub := struct {
			Name string
			E    *big.Int
			N    *big.Int
		}{}
		if err := ssh.Unmarshal(pkey.Marshal(), &rawPub); err != nil {
			return "", 0, err
		}
		return "rsa", rawPub.N.BitLen(), nil // use N as per crypto/rsa/rsa.go (is bits)
	case ssh.KeyAlgoECDSA256:
		return "ecdsa", 256, nil
	case ssh.KeyAlgoECDSA384:
		return "ecdsa", 384, nil
	case ssh.KeyAlgoECDSA521:
		return "ecdsa", 521, nil
	case ssh.KeyAlgoED25519:
		return "ed25519", 256, nil
	case ssh.KeyAlgoSKECDSA256:
		return "ecdsa-sk", 256, nil
	case ssh.KeyAlgoSKED25519:
		return "ed25519-sk", 256, nil
	}
	return "", 0, fmt.Errorf("unsupported key length detection for type: %s", pkey.Type())
}

// writeTmpKeyFile writes key content to a temporary file
// and returns the name of that file, along with any possible errors.
func writeTmpKeyFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp(setting.SSH.KeyTestPath, "gitea_keytest")
	if err != nil {
		return "", fmt.Errorf("TempFile: %w", err)
	}
	defer tmpFile.Close()

	if _, err = tmpFile.WriteString(content); err != nil {
		return "", fmt.Errorf("WriteString: %w", err)
	}
	return tmpFile.Name(), nil
}

// SSHKeyGenParsePublicKey extracts key type and length using ssh-keygen.
func SSHKeyGenParsePublicKey(key string) (string, int, error) {
	tmpName, err := writeTmpKeyFile(key)
	if err != nil {
		return "", 0, fmt.Errorf("writeTmpKeyFile: %w", err)
	}
	defer func() {
		if err := util.Remove(tmpName); err != nil {
			log.Warn("Unable to remove temporary key file: %s: Error: %v", tmpName, err)
		}
	}()

	keygenPath := setting.SSH.KeygenPath
	if len(keygenPath) == 0 {
		keygenPath = "ssh-keygen"
	}

	stdout, stderr, err := process.GetManager().Exec("SSHKeyGenParsePublicKey", keygenPath, "-lf", tmpName)
	if err != nil {
		return "", 0, fmt.Errorf("fail to parse public key: %s - %s", err, stderr)
	}
	if strings.Contains(stdout, "is not a public key file") {
		return "", 0, ErrKeyUnableVerify{stdout}
	}

	fields := strings.Split(stdout, " ")
	if len(fields) < 4 {
		return "", 0, fmt.Errorf("invalid public key line: %s", stdout)
	}

	keyType := strings.Trim(fields[len(fields)-1], "()\r\n")
	length, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return "", 0, err
	}
	return strings.ToLower(keyType), int(length), nil
}
