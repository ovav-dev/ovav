//go:build linux

package identity

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

// secureAtomicReplace publishes through one verified directory FD. published is
// true once renameat commits, even when the following directory fsync fails.
func secureAtomicReplace(path string, data, expected []byte, expectedInfo os.FileInfo, unconditional bool, syncFn func(string) error) (published bool, err error) {
	dirPath := filepath.Dir(path)
	dirFD, err := unix.Open(dirPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return false, err
	}
	defer unix.Close(dirFD)
	if err := validateSecureDirectoryFD(dirFD); err != nil {
		return false, err
	}

	name := filepath.Base(path)
	tmpName := ".identity-recovery-" + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	tmpFD, err := unix.Openat(dirFD, tmpName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return false, err
	}
	tmp := os.NewFile(uintptr(tmpFD), tmpName)
	tmpOpen := true
	defer func() {
		if tmpOpen {
			_ = tmp.Close()
		}
		_ = unix.Unlinkat(dirFD, tmpName, 0)
	}()
	if _, err := tmp.Write(data); err != nil {
		return false, err
	}
	if err := tmp.Sync(); err != nil {
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	tmpOpen = false

	if !unconditional {
		targetFD, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if err != nil {
			return false, err
		}
		if err := unix.Flock(targetFD, unix.LOCK_EX); err != nil {
			_ = unix.Close(targetFD)
			return false, err
		}
		target := os.NewFile(uintptr(targetFD), name)
		defer target.Close()
		current, err := io.ReadAll(target)
		if err != nil {
			return false, err
		}
		currentInfo, err := target.Stat()
		if err != nil {
			return false, err
		}
		if expectedInfo == nil || !os.SameFile(expectedInfo, currentInfo) || !bytes.Equal(current, expected) {
			return false, fmt.Errorf("registry changed during recovery")
		}
		var pathStat unix.Stat_t
		var fdStat unix.Stat_t
		if err := unix.Fstatat(dirFD, name, &pathStat, unix.AT_SYMLINK_NOFOLLOW); err != nil ||
			unix.Fstat(targetFD, &fdStat) != nil || pathStat.Ino != fdStat.Ino || pathStat.Dev != fdStat.Dev {
			return false, fmt.Errorf("registry changed during recovery")
		}
	}

	if err := unix.Renameat(dirFD, tmpName, dirFD, name); err != nil {
		return false, err
	}
	published = true
	if syncFn != nil {
		if err := syncFn(dirPath); err != nil {
			return true, err
		}
		return true, nil
	}
	if err := unix.Fsync(dirFD); err != nil {
		return true, err
	}
	return true, nil
}

func secureAtomicCreate(path string, data []byte, syncFn func(string) error) (bool, error) {
	if _, err := os.Lstat(path); err == nil {
		return false, fmt.Errorf("path appeared during atomic create")
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return secureAtomicReplace(path, data, nil, nil, true, syncFn)
}

func validateSecureDirectoryFD(fd int) error {
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFDIR || (stat.Uid != 0 && stat.Uid != uint32(os.Geteuid())) || stat.Mode&0o022 != 0 {
		return fmt.Errorf("directory FD is not trusted")
	}
	return nil
}
