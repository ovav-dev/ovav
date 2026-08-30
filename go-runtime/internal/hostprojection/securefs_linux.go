//go:build linux

package hostprojection

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type fileIdentity struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

func (i fileIdentity) zero() bool { return i.Device == 0 && i.Inode == 0 }

type snapshot struct {
	data    []byte
	mode    uint32
	special uint32
	links   uint64
	owner   uint32
	id      fileIdentity
	hash    string
}

type durabilityTracker struct {
	level  DurabilityLevel
	detail string
}

func newDurability() durabilityTracker { return durabilityTracker{level: DurabilityFull} }

func (d *durabilityTracker) syncDir(dir *os.File, label string) error {
	if err := syscall.Fsync(int(dir.Fd())); err != nil {
		if isUnsupportedDirSync(err) {
			d.level = DurabilityDegraded
			if d.detail == "" {
				d.detail = label + ": directory fsync unsupported (common on DrvFs)"
			}
			return nil
		}
		return fmt.Errorf("fsync directory %s: %w", label, err)
	}
	return nil
}

func isUnsupportedDirSync(err error) bool {
	return errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}

func openDirAbsolute(path string) (*os.File, error) {
	dir, missing, err := openDirPrefix(path)
	if err != nil {
		return nil, err
	}
	if len(missing) != 0 {
		dir.Close()
		return nil, fmt.Errorf("directory does not exist: %s", path)
	}
	return dir, nil
}

func openDirPrefix(path string) (*os.File, []string, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return nil, nil, fmt.Errorf("directory path is not absolute: %s", path)
	}
	fd, err := syscall.Open(string(os.PathSeparator), syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open filesystem root: %w", err)
	}
	if clean == string(os.PathSeparator) {
		return os.NewFile(uintptr(fd), clean), nil, nil
	}
	components := strings.Split(strings.TrimPrefix(clean, string(os.PathSeparator)), string(os.PathSeparator))
	currentPath := string(os.PathSeparator)
	for index, component := range components {
		next, openErr := syscall.Openat(fd, component, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			return os.NewFile(uintptr(fd), currentPath), components[index:], nil
		}
		if openErr != nil {
			_ = syscall.Close(fd)
			return nil, nil, fmt.Errorf("open no-follow directory component %q in %s: %w", component, clean, openErr)
		}
		_ = syscall.Close(fd)
		fd = next
		currentPath = filepath.Join(currentPath, component)
	}
	return os.NewFile(uintptr(fd), clean), nil, nil
}

func openDirRelative(root *os.File, relative string) (*os.File, error) {
	clean, err := safeRelative(relative)
	if err != nil {
		return nil, err
	}
	fd, err := syscall.Dup(int(root.Fd()))
	if err != nil {
		return nil, fmt.Errorf("duplicate root descriptor: %w", err)
	}
	if clean == "." {
		return os.NewFile(uintptr(fd), root.Name()), nil
	}
	for _, component := range strings.Split(clean, string(os.PathSeparator)) {
		next, openErr := syscall.Openat(fd, component, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
		_ = syscall.Close(fd)
		if openErr != nil {
			return nil, fmt.Errorf("open no-follow relative component %q: %w", component, openErr)
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), filepath.Join(root.Name(), clean)), nil
}

func openParentAbsolute(path string) (*os.File, string, error) {
	dir, err := openDirAbsolute(filepath.Dir(path))
	if err != nil {
		return nil, "", err
	}
	return dir, filepath.Base(path), nil
}

func openParentRelative(root *os.File, relative string) (*os.File, string, error) {
	clean, err := safeRelative(relative)
	if err != nil {
		return nil, "", fmt.Errorf("invalid file path %q: %w", relative, err)
	}
	if clean == "." {
		return nil, "", fmt.Errorf("invalid file path %q", relative)
	}
	dir, err := openDirRelative(root, filepath.Dir(clean))
	if err != nil {
		return nil, "", err
	}
	return dir, filepath.Base(clean), nil
}

func safeRelative(path string) (string, error) {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes root: %s", path)
	}
	return clean, nil
}

func readRegularAt(parent *os.File, name string) (snapshot, error) {
	return readRegularAtBounded(parent, name, 0)
}

func readRegularAtBounded(parent *os.File, name string, maximumBytes int64) (snapshot, error) {
	fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return snapshot{}, fmt.Errorf("open no-follow file %s: %w", name, err)
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	var before syscall.Stat_t
	if err := syscall.Fstat(fd, &before); err != nil {
		return snapshot{}, fmt.Errorf("fstat %s: %w", name, err)
	}
	if before.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return snapshot{}, fmt.Errorf("%s is not a regular file", name)
	}
	if maximumBytes > 0 && (before.Size < 0 || before.Size > maximumBytes) {
		return snapshot{}, fmt.Errorf("%s exceeds maximum size %d", name, maximumBytes)
	}
	reader := io.Reader(file)
	if maximumBytes > 0 {
		reader = io.LimitReader(file, maximumBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return snapshot{}, fmt.Errorf("read %s: %w", name, err)
	}
	if maximumBytes > 0 && int64(len(data)) > maximumBytes {
		return snapshot{}, fmt.Errorf("%s exceeds maximum size %d", name, maximumBytes)
	}
	var after syscall.Stat_t
	if err := syscall.Fstat(fd, &after); err != nil {
		return snapshot{}, fmt.Errorf("re-fstat %s: %w", name, err)
	}
	beforeID, afterID := identityOf(before), identityOf(after)
	if beforeID != afterID || before.Size != after.Size || before.Mtim != after.Mtim {
		return snapshot{}, fmt.Errorf("%w while reading %s", ErrConcurrentChange, name)
	}
	return snapshot{
		data: data, mode: before.Mode & 0o777,
		special: before.Mode & (syscall.S_ISUID | syscall.S_ISGID | syscall.S_ISVTX),
		links:   uint64(before.Nlink), owner: before.Uid, id: beforeID, hash: digest(data),
	}, nil
}

func readOptionalAt(parent *os.File, name string) (snapshot, bool, error) {
	s, err := readRegularAt(parent, name)
	if errors.Is(err, syscall.ENOENT) {
		return snapshot{}, false, nil
	}
	return s, err == nil, err
}

func verifyAt(parent *os.File, name string, expected snapshot) (snapshot, error) {
	current, err := readRegularAt(parent, name)
	if err != nil {
		return snapshot{}, err
	}
	if current.id != expected.id || current.hash != expected.hash || !equalBytes(current.data, expected.data) {
		return snapshot{}, fmt.Errorf("%w at %s", ErrConcurrentChange, name)
	}
	return current, nil
}

func createFileAt(parent *os.File, prefix string, data []byte, mode uint32) (*os.File, string, fileIdentity, error) {
	for range 16 {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, "", fileIdentity{}, fmt.Errorf("generate temporary name: %w", err)
		}
		name := prefix + hex.EncodeToString(random[:])
		fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDWR|syscall.O_CREAT|syscall.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, mode)
		if errors.Is(err, syscall.EEXIST) {
			continue
		}
		if err != nil {
			return nil, "", fileIdentity{}, fmt.Errorf("create no-follow file %s: %w", name, err)
		}
		file := os.NewFile(uintptr(fd), name)
		if err := file.Chmod(os.FileMode(mode)); err != nil {
			file.Close()
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return nil, "", fileIdentity{}, fmt.Errorf("chmod %s: %w", name, err)
		}
		if _, err := file.Write(data); err != nil {
			file.Close()
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return nil, "", fileIdentity{}, fmt.Errorf("write %s: %w", name, err)
		}
		if err := file.Sync(); err != nil {
			file.Close()
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return nil, "", fileIdentity{}, fmt.Errorf("fsync %s: %w", name, err)
		}
		var stat syscall.Stat_t
		if err := syscall.Fstat(fd, &stat); err != nil {
			file.Close()
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return nil, "", fileIdentity{}, fmt.Errorf("fstat %s: %w", name, err)
		}
		if stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Mode&0o777 != mode || stat.Nlink != 1 || stat.Uid != uint32(os.Geteuid()) {
			file.Close()
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return nil, "", fileIdentity{}, fmt.Errorf("created file %s failed mode, link, or effective-user ownership validation", name)
		}
		return file, name, identityOf(stat), nil
	}
	return nil, "", fileIdentity{}, errors.New("create temporary file: name collision limit reached")
}

func openLockAt(parent *os.File, name string) (*os.File, error) {
	fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDWR|syscall.O_CREAT|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open transaction lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), name)
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		file.Close()
		return nil, fmt.Errorf("fstat transaction lock: %w", err)
	}
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG || stat.Nlink != 1 {
		file.Close()
		return nil, errors.New("transaction lock is not a private regular file")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		file.Close()
		return nil, fmt.Errorf("transaction lock owner is uid %d; require effective uid %d", stat.Uid, os.Geteuid())
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return nil, fmt.Errorf("chmod transaction lock: %w", err)
	}
	if err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrLocked
		}
		return nil, fmt.Errorf("flock transaction lock: %w", err)
	}
	return file, nil
}

func closeLock(file *os.File) {
	if file != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}
}

func identityOf(stat syscall.Stat_t) fileIdentity {
	return fileIdentity{Device: uint64(stat.Dev), Inode: stat.Ino}
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func equalBytes(a, b []byte) bool {
	return len(a) == len(b) && sha256.Sum256(a) == sha256.Sum256(b)
}
