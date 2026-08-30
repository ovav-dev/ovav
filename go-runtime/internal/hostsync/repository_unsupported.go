//go:build !linux

package hostsync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func canonicalMainRepoRoot(repoRoot string) (string, error) {
	gitPath := filepath.Join(repoRoot, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New(".git must exist and must not be a symlink")
	}
	if info.IsDir() {
		return repoRoot, nil
	}
	if !info.Mode().IsRegular() {
		return "", errors.New(".git must be a regular file or directory")
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}
	content := strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r")
	const prefix = "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return "", errors.New("malformed worktree .git file")
	}
	gitDir := strings.TrimPrefix(content, prefix)
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoRoot, gitDir)
	}
	gitDir = filepath.Clean(gitDir)
	if filepath.Base(filepath.Dir(gitDir)) != "worktrees" || filepath.Base(filepath.Dir(filepath.Dir(gitDir))) != ".git" {
		return "", errors.New("malformed worktree gitdir path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(gitDir))), nil
}
