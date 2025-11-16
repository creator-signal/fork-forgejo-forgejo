// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
)

// Git settings
var Git = struct {
	Path                 string
	HomePath             string
	DisableDiffHighlight bool

	MaxGitDiffLines           int
	MaxGitDiffLineCharacters  int
	MaxGitDiffFiles           int
	CommitsRangeSize          int // CommitsRangeSize the default commits range size
	BranchesRangeSize         int // BranchesRangeSize the default branches range size
	VerbosePush               bool
	VerbosePushDelay          time.Duration
	GCArgs                    []string `ini:"GC_ARGS" delim:" "`
	EnableAutoGitWireProtocol bool
	PullRequestPushMessage    bool
	DisablePartialClone       bool
	CredentialHelperPath      string
	Timeout                   struct {
		Default int
		Migrate int
		Mirror  int
		Clone   int
		Pull    int
		GC      int `ini:"GC"`
		Grep    int
	} `ini:"git.timeout"`
}{
	DisableDiffHighlight:      false,
	MaxGitDiffLines:           1000,
	MaxGitDiffLineCharacters:  5000,
	MaxGitDiffFiles:           100,
	CommitsRangeSize:          50,
	BranchesRangeSize:         20,
	VerbosePush:               true,
	VerbosePushDelay:          5 * time.Second,
	GCArgs:                    []string{},
	EnableAutoGitWireProtocol: true,
	PullRequestPushMessage:    true,
	DisablePartialClone:       false,
	Timeout: struct {
		Default int
		Migrate int
		Mirror  int
		Clone   int
		Pull    int
		GC      int `ini:"GC"`
		Grep    int
	}{
		Default: 360,
		Migrate: 600,
		Mirror:  300,
		Clone:   300,
		Pull:    300,
		GC:      60,
		Grep:    2,
	},
}

type GitConfigType struct {
	Options map[string]string // git config key is case-insensitive, always use lower-case
}

func (c *GitConfigType) SetOption(key, val string) {
	c.Options[strings.ToLower(key)] = val
}

func (c *GitConfigType) GetOption(key string) string {
	return c.Options[strings.ToLower(key)]
}

var GitConfig = GitConfigType{
	Options: make(map[string]string),
}

func loadGitFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("git")
	if err := sec.MapTo(&Git); err != nil {
		log.Fatal("Failed to map Git settings: %v", err)
	}

	secGitConfig := rootCfg.Section("git.config")
	GitConfig.Options = make(map[string]string)
	GitConfig.SetOption("diff.algorithm", "histogram")
	GitConfig.SetOption("core.logAllRefUpdates", "true")
	GitConfig.SetOption("gc.reflogExpire", "90")

	secGitReflog := rootCfg.Section("git.reflog")
	if secGitReflog.HasKey("ENABLED") {
		deprecatedSetting(rootCfg, "git.reflog", "ENABLED", "git.config", "core.logAllRefUpdates", "1.21")
		GitConfig.SetOption("core.logAllRefUpdates", secGitReflog.Key("ENABLED").In("true", []string{"true", "false"}))
	}
	if secGitReflog.HasKey("EXPIRATION") {
		deprecatedSetting(rootCfg, "git.reflog", "EXPIRATION", "git.config", "core.reflogExpire", "1.21")
		GitConfig.SetOption("gc.reflogExpire", secGitReflog.Key("EXPIRATION").String())
	}

	for _, key := range secGitConfig.Keys() {
		GitConfig.SetOption(key.Name(), key.String())
	}

	Git.HomePath = sec.Key("HOME_PATH").MustString("home")
	if !filepath.IsAbs(Git.HomePath) {
		Git.HomePath = filepath.Join(AppDataPath, Git.HomePath)
	} else {
		Git.HomePath = filepath.Clean(Git.HomePath)
	}

	Git.CredentialHelperPath = sec.Key("CREDENTIAL_HELPER_PATH").MustString("tmp/git-credentials-helper")
	if !filepath.IsAbs(Git.CredentialHelperPath) {
		Git.CredentialHelperPath = filepath.Join(AppDataPath, Git.CredentialHelperPath)
	} else {
		Git.CredentialHelperPath = filepath.Clean(Git.CredentialHelperPath)
	}
	if err := selfTestGitCredentialHelperPath(Git.CredentialHelperPath); err != nil {
		log.Fatal("Self-test for [git].CREDENTIAL_HELPER_PATH failed: %v", err)
	}
}

func selfTestGitCredentialHelperPath(credentialHelperPath string) error {
	// Check the path exists and is a directory.
	if stat, err := os.Stat(credentialHelperPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(credentialHelperPath, 0o700); err != nil {
				return fmt.Errorf("could not create a directory at %q: %w", credentialHelperPath, err)
			}
		} else {
			return fmt.Errorf("could not determine if %q is a directory: %w", credentialHelperPath, err)
		}
	} else if !stat.IsDir() {
		return fmt.Errorf("the path %q is not a directory", credentialHelperPath)
	}

	// Context: forgejo/forgejo#9733
	// Check that Forgejo is able to execute a script in this directory, we do
	// this by trying to execute a shell script. If Forgejo is not able to
	// execute a script in this directory then very likely Git will also not be
	// able to execute a script there and will result in migration failing.
	path := filepath.Join(credentialHelperPath, "forgejo-self-test-"+util.CryptoRandomString(util.RandomStringLow))
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho 123\n"), 0o700); err != nil {
		return fmt.Errorf("could not write self-test program to %q: %w", path, err)
	}

	defer func() {
		if err := os.Remove(path); err != nil {
			log.Warn("Was not able to remove self-test program: %v", err)
		}
	}()

	if err := exec.Command(path).Run(); err != nil {
		return fmt.Errorf("to safely provide authorization to Git, Forgejo relies on temporary helper scripts that are "+
			"executed by Git. Forgejo is currently configured to store these scripts in %v.A script was written to that "+
			"directory but could not be executed It is possible that scripts are not allowed to be executed in this directory, "+
			"in which case you must change the directory path to a directory where this is allowed: %v", credentialHelperPath, err)
	}

	return nil
}
