package barerepositories

import (
	"fmt"
	"os"
	"path/filepath"

	"forgejo.org/modules/git"
)

// CopyTo copies the bare repositories and sets the git hooks.
// setting.AppPath and setting.CustomConf must be already set properly.
func CopyTo(src, dst string) error {
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		return fmt.Errorf("os.CopyFS(%s, os.DirFS(%s)): %w", dst, src, err)
	}

	// write the git hooks
	orgEntries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("os.ReadDir(%s): %w", src, err)
	}
	for _, orgDir := range orgEntries {
		if !orgDir.IsDir() {
			continue
		}
		orgPath := filepath.Join(src, orgDir.Name())
		repoEntries, err := os.ReadDir(orgPath)
		if err != nil {
			return fmt.Errorf("os.ReadDir(%s): %w", orgPath, err)
		}
		for _, repoDir := range repoEntries {
			if !repoDir.IsDir() {
				continue
			}
			repoPath := filepath.Join(dst, orgDir.Name(), repoDir.Name())

			_, err := os.Stat(filepath.Join(repoPath, "HEAD")) // ensure that we are in a bare git repo
			if err != nil {
				return fmt.Errorf("os.Stat(%s): %w", filepath.Join(repoPath, "HEAD"), err)
			}

			if err := git.CreatePrivateHooks(repoPath); err != nil {
				return fmt.Errorf("git.CreatePrivateHooks(%s): %w", repoPath, err)
			}
		}
	}
	return nil
}
