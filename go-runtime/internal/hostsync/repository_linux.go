//go:build linux

package hostsync

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const maximumGitFileBytes = 4096

func canonicalMainRepoRoot(repoRoot string) (string, error) {
	repo, err := openDirectoryNoFollow(repoRoot)
	if err != nil {
		return "", fmt.Errorf("open repository root: %w", err)
	}
	defer repo.Close()
	var gitStat unix.Stat_t
	if err := unix.Fstatat(int(repo.Fd()), ".git", &gitStat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return "", fmt.Errorf("inspect .git: %w", err)
	}
	switch gitStat.Mode & syscall.S_IFMT {
	case syscall.S_IFDIR:
		gitDir, err := openDirectoryAt(repo, ".git")
		if err != nil {
			return "", err
		}
		gitDir.Close()
		return repoRoot, nil
	case syscall.S_IFREG:
		data, err := readRegularAt(repo, ".git")
		if err != nil {
			return "", err
		}
		return mainRootFromGitFile(repoRoot, string(data))
	default:
		return "", errors.New(".git must be a no-follow regular file or directory")
	}
}

func mainRootFromGitFile(repoRoot, content string) (string, error) {
	content = strings.TrimSuffix(strings.TrimSuffix(content, "\n"), "\r")
	const prefix = "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return "", errors.New("malformed worktree .git file")
	}
	gitDir := strings.TrimPrefix(content, prefix)
	if gitDir == "" || strings.TrimSpace(gitDir) != gitDir || strings.ContainsAny(gitDir, "\r\n\x00") {
		return "", errors.New("malformed worktree gitdir path")
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoRoot, gitDir)
	}
	gitDir = filepath.Clean(gitDir)
	if filepath.Base(filepath.Dir(gitDir)) != "worktrees" {
		return "", errors.New("worktree gitdir is outside a worktrees directory")
	}
	mainGitDir := filepath.Dir(filepath.Dir(gitDir))
	if filepath.Base(mainGitDir) != ".git" {
		return "", errors.New("worktree gitdir is outside a main .git directory")
	}
	for _, path := range []string{mainGitDir, gitDir} {
		dir, err := openDirectoryNoFollow(path)
		if err != nil {
			return "", fmt.Errorf("validate worktree gitdir %s: %w", path, err)
		}
		dir.Close()
	}
	metadataDir, err := openDirectoryNoFollow(gitDir)
	if err != nil {
		return "", fmt.Errorf("open worktree metadata: %w", err)
	}
	backlinkData, err := readRegularAt(metadataDir, "gitdir")
	metadataDir.Close()
	if err != nil {
		return "", fmt.Errorf("read worktree metadata backlink: %w", err)
	}
	backlink := strings.TrimSuffix(strings.TrimSuffix(string(backlinkData), "\n"), "\r")
	if backlink == "" || strings.TrimSpace(backlink) != backlink || strings.ContainsAny(backlink, "\r\n\x00") {
		return "", errors.New("malformed worktree metadata backlink")
	}
	if !filepath.IsAbs(backlink) {
		backlink = filepath.Join(gitDir, backlink)
	}
	if filepath.Clean(backlink) != filepath.Join(repoRoot, ".git") {
		return "", errors.New("worktree metadata does not point back to the supplied repository root")
	}
	return filepath.Dir(mainGitDir), nil
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) || clean != path {
		return nil, errors.New("directory path must be absolute and traversal-free")
	}
	fd, err := unix.Open(string(os.PathSeparator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	current := string(os.PathSeparator)
	for _, component := range strings.Split(strings.TrimPrefix(clean, string(os.PathSeparator)), string(os.PathSeparator)) {
		if component == "" {
			continue
		}
		next, openErr := unix.Openat(fd, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(fd)
		if openErr != nil {
			return nil, openErr
		}
		fd = next
		current = filepath.Join(current, component)
	}
	return os.NewFile(uintptr(fd), current), nil
}

func openDirectoryAt(parent *os.File, name string) (*os.File, error) {
	fd, err := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open no-follow directory %s: %w", name, err)
	}
	return os.NewFile(uintptr(fd), name), nil
}

func readRegularAt(parent *os.File, name string) ([]byte, error) {
	fd, err := unix.Openat(int(parent.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open no-follow file %s: %w", name, err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	var before unix.Stat_t
	if err := unix.Fstat(fd, &before); err != nil || before.Mode&syscall.S_IFMT != syscall.S_IFREG || before.Nlink != 1 {
		return nil, fmt.Errorf("%s is not a singly-linked regular file", name)
	}
	data, err := io.ReadAll(io.LimitReader(file, maximumGitFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	if len(data) > maximumGitFileBytes {
		return nil, fmt.Errorf("%s exceeds maximum size", name)
	}
	var after unix.Stat_t
	if err := unix.Fstat(fd, &after); err != nil || before.Dev != after.Dev || before.Ino != after.Ino || before.Size != after.Size || before.Mtim != after.Mtim {
		return nil, errors.New("worktree .git file changed while reading")
	}
	var pathAfter unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &pathAfter, unix.AT_SYMLINK_NOFOLLOW); err != nil ||
		pathAfter.Mode&syscall.S_IFMT != syscall.S_IFREG || pathAfter.Dev != after.Dev || pathAfter.Ino != after.Ino {
		return nil, errors.New("worktree .git file identity changed while reading")
	}
	return data, nil
}
