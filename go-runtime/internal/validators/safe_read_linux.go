//go:build linux

package validators

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

func readRegularFileNoFollow(path string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(abs), string(filepath.Separator)), string(filepath.Separator))
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return nil, fmt.Errorf("invalid file path")
	}

	dirFD, err := unix.Open(string(filepath.Separator), unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	defer func() { _ = unix.Close(dirFD) }()
	for _, part := range parts[:len(parts)-1] {
		nextFD, openErr := unix.Openat(dirFD, part, unix.O_PATH|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if openErr != nil {
			return nil, openErr
		}
		_ = unix.Close(dirFD)
		dirFD = nextFD
	}

	fd, err := unix.Openat(dirFD, parts[len(parts)-1], unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), abs)
	if file == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("open regular file")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	return io.ReadAll(file)
}

func openRegularFileAtNoFollow(dirPath, name string, expectedDirInfo os.FileInfo) (*anchoredRegularFile, error) {
	if name == "" || name == "." || filepath.Base(name) != name {
		return nil, fmt.Errorf("invalid file name")
	}
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, err
	}
	dirFD, err := openDirectoryNoFollow(absDir)
	if err != nil {
		return nil, err
	}
	dir := os.NewFile(uintptr(dirFD), absDir)
	if dir == nil {
		_ = unix.Close(dirFD)
		return nil, fmt.Errorf("open anchored directory")
	}
	dirInfo, err := dir.Stat()
	if err != nil || !dirInfo.IsDir() || !os.SameFile(expectedDirInfo, dirInfo) {
		_ = dir.Close()
		return nil, fmt.Errorf("anchored directory identity mismatch")
	}

	fileFD, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		_ = dir.Close()
		return nil, err
	}
	file := os.NewFile(uintptr(fileFD), filepath.Join(absDir, name))
	if file == nil {
		_ = unix.Close(fileFD)
		_ = dir.Close()
		return nil, fmt.Errorf("open anchored regular file")
	}
	closeFiles := func() {
		_ = file.Close()
		_ = dir.Close()
	}

	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		closeFiles()
		return nil, fmt.Errorf("not a regular file")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		closeFiles()
		return nil, err
	}
	after, err := file.Stat()
	if err != nil || !stableFileIdentity(before, after) {
		closeFiles()
		return nil, fmt.Errorf("file changed while reading")
	}

	return &anchoredRegularFile{
		data:    data,
		dirInfo: dirInfo,
		revalidate: func() error {
			currentDirInfo, statErr := dir.Stat()
			if statErr != nil || !os.SameFile(dirInfo, currentDirInfo) {
				return fmt.Errorf("anchored directory changed")
			}
			currentFileInfo, statErr := file.Stat()
			if statErr != nil || !stableFileIdentity(after, currentFileInfo) {
				return fmt.Errorf("anchored file changed")
			}
			var descriptorStat unix.Stat_t
			if statErr := unix.Fstat(fileFD, &descriptorStat); statErr != nil {
				return statErr
			}
			var pathStat unix.Stat_t
			if statErr := unix.Fstatat(dirFD, name, &pathStat, unix.AT_SYMLINK_NOFOLLOW); statErr != nil {
				return statErr
			}
			if pathStat.Mode&unix.S_IFMT != unix.S_IFREG ||
				descriptorStat.Dev != pathStat.Dev || descriptorStat.Ino != pathStat.Ino {
				return fmt.Errorf("anchored file path changed")
			}
			return nil
		},
		closeFiles: closeFiles,
	}, nil
}

func openDirectoryNoFollow(path string) (int, error) {
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator)), string(filepath.Separator))
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return -1, fmt.Errorf("invalid directory path")
	}
	dirFD, err := unix.Open(string(filepath.Separator), unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	for _, part := range parts {
		nextFD, openErr := unix.Openat(dirFD, part, unix.O_PATH|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		_ = unix.Close(dirFD)
		if openErr != nil {
			return -1, openErr
		}
		dirFD = nextFD
	}
	return dirFD, nil
}

func stableFileIdentity(before, after os.FileInfo) bool {
	return before.Mode() == after.Mode() && before.Size() == after.Size() &&
		before.ModTime().Equal(after.ModTime()) && os.SameFile(before, after)
}
