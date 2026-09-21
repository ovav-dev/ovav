//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

func secureSessionReplace(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	dirFD, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)
	name := filepath.Base(path)
	tmpName := ".session-" + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	fd, err := unix.Openat(dirFD, tmpName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, uint32(mode.Perm()))
	if err != nil {
		return err
	}
	tmp := os.NewFile(uintptr(fd), tmpName)
	defer unix.Unlinkat(dirFD, tmpName, 0)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := unix.Renameat(dirFD, tmpName, dirFD, name); err != nil {
		return err
	}
	return unix.Fsync(dirFD)
}
