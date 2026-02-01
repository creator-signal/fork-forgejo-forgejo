// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bufio"
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	forgery.WrapTestMain(m.Run)
}

//go:embed *.go
var testsFS embed.FS

func TestPreventLegacyImports(t *testing.T) {
	require.NoError(t, fs.WalkDir(testsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		t.Run(d.Name(), func(t *testing.T) {
			f, err := testsFS.Open(path)
			require.NoError(t, err)
			defer f.Close()
			scanner := bufio.NewScanner(f)
			line := 0
			for scanner.Scan() {
				line++
				buf := scanner.Bytes()
				start := bytes.IndexByte(buf, '"')
				if start == -1 {
					continue
				}
				end := bytes.LastIndexByte(buf, '"')
				if end == start {
					continue
				}
				s := string(buf[start+1 : end])
				switch s {
				case "forgejo.org/models/unittest", "forgejo.org/tests/integration", "forgejo.org/tests":
					fullpath := path
					if wd, err := os.Getwd(); err == nil {
						fullpath = filepath.Join(wd, path)
					}
					t.Errorf("import %q is not allowed in new integration tests (%s:%d)", s, fullpath, line)
					t.Log("please re-implement the needed functionality in tests/forgery (ideally with an improved API)")
				}
			}
			require.NoError(t, scanner.Err())
		})
		return nil
	}))
}
