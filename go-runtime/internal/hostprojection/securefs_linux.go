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

	"golang.org/x/sys/unix"
)

const (
	v9fsMagic                             = 0x01021997
	destinationModeEnforcementUnsupported = "destination mode enforcement unsupported on v9fs"
)

type createdFilePurpose uint8

const (
	privateArtifact createdFilePurpose = iota
	destinationArtifact
)

type fileIdentity struct {
	Device uint64 `json:"device"`
	Inode  uint64 `json:"inode"`
}

func (i fileIdentity) zero() bool { return i.Device == 0 && i.Inode == 0 }

type snapshot struct {
	data     []byte
	kind     DestinationKind
	linkText string
	mode     uint32
	special  uint32
	links    uint64
	owner    uint32
	id       fileIdentity
	hash     string
}

type durabilityTracker struct {
	level  DurabilityLevel
	detail string
}

func newDurability() durabilityTracker { return durabilityTracker{level: DurabilityFull} }

func (d *durabilityTracker) degrade(detail string) {
	d.level = DurabilityDegraded
	if d.detail == "" || detail == destinationModeEnforcementUnsupported {
		d.detail = detail
	}
}

func (d *durabilityTracker) noteDestinationFilesystem(filesystemType int64) {
	if isV9FS(filesystemType) {
		d.degrade(destinationModeEnforcementUnsupported)
	}
}

func (d *durabilityTracker) syncDir(dir *os.File, label string) error {
	if err := syscall.Fsync(int(dir.Fd())); err != nil {
		if isUnsupportedDirSync(err) {
			d.degrade(label + ": directory fsync unsupported (common on DrvFs)")
			return nil
		}
		return fmt.Errorf("fsync directory %s: %w", label, err)
	}
	return nil
}

func isUnsupportedDirSync(err error) bool {
	return errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.EOPNOTSUPP)
}

func isV9FS(filesystemType int64) bool {
	return filesystemType == v9fsMagic
}

func descriptorFilesystemType(file *os.File) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Fstatfs(int(file.Fd()), &stat); err != nil {
		return 0, fmt.Errorf("fstatfs %s: %w", file.Name(), err)
	}
	return int64(stat.Type), nil
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
		data: data, kind: DestinationRegular, mode: before.Mode & 0o777,
		special: before.Mode & (syscall.S_ISUID | syscall.S_ISGID | syscall.S_ISVTX),
		links:   uint64(before.Nlink), owner: before.Uid, id: beforeID, hash: digest(data),
	}, nil
}

func readDestinationOptionalAt(parent *os.File, name, expectedTarget string) (snapshot, bool, error) {
	var before unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &before, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, syscall.ENOENT) {
			return snapshot{kind: DestinationAbsent}, false, nil
		}
		return snapshot{}, false, fmt.Errorf("lstat destination %s: %w", name, err)
	}
	switch before.Mode & syscall.S_IFMT {
	case syscall.S_IFREG:
		regular, err := readRegularAt(parent, name)
		return regular, err == nil, err
	case syscall.S_IFLNK:
		if expectedTarget == "" {
			return snapshot{}, false, fmt.Errorf("%s is a symlink; exact symlink migration is not enabled", name)
		}
		link, err := readExactSymlinkAt(parent, name, expectedTarget)
		return link, err == nil, err
	default:
		return snapshot{}, false, fmt.Errorf("%s is not a regular file or permitted direct symlink", name)
	}
}

func readExactSymlinkAt(parent *os.File, name, expectedTarget string) (snapshot, error) {
	if !filepath.IsAbs(expectedTarget) || filepath.Clean(expectedTarget) != expectedTarget {
		return snapshot{}, errors.New("expected symlink target must be absolute and traversal-free")
	}
	var before unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &before, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return snapshot{}, fmt.Errorf("lstat symlink %s: %w", name, err)
	}
	if before.Mode&syscall.S_IFMT != syscall.S_IFLNK {
		return snapshot{}, fmt.Errorf("%s is not a direct symlink", name)
	}
	buffer := make([]byte, 64*1024)
	length, err := unix.Readlinkat(int(parent.Fd()), name, buffer)
	if err != nil {
		return snapshot{}, fmt.Errorf("readlinkat %s: %w", name, err)
	}
	if length == len(buffer) {
		return snapshot{}, fmt.Errorf("symlink %s exceeds maximum link-text size", name)
	}
	linkText := string(buffer[:length])
	if !filepath.IsAbs(linkText) || filepath.Clean(linkText) != linkText {
		return snapshot{}, fmt.Errorf("symlink %s must contain an absolute traversal-free target", name)
	}
	if linkText != expectedTarget {
		return snapshot{}, fmt.Errorf("symlink %s target mismatch", name)
	}
	if err := validateRegularTargetNoFollow(expectedTarget); err != nil {
		return snapshot{}, fmt.Errorf("validate symlink target: %w", err)
	}
	var after unix.Stat_t
	if err := unix.Fstatat(int(parent.Fd()), name, &after, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return snapshot{}, fmt.Errorf("re-lstat symlink %s: %w", name, err)
	}
	beforeID := fileIdentity{Device: uint64(before.Dev), Inode: before.Ino}
	afterID := fileIdentity{Device: uint64(after.Dev), Inode: after.Ino}
	if beforeID != afterID || before.Size != after.Size || before.Mtim != after.Mtim {
		return snapshot{}, fmt.Errorf("%w while reading symlink %s", ErrConcurrentChange, name)
	}
	return snapshot{kind: DestinationSymlink, linkText: linkText, id: beforeID}, nil
}

func validateRegularTargetNoFollow(path string) error {
	parent, name, err := openParentAbsolute(path)
	if err != nil {
		return err
	}
	defer parent.Close()
	fd, err := syscall.Openat(int(parent.Fd()), name, syscall.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open no-follow target %s: %w", name, err)
	}
	defer syscall.Close(fd)
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		return fmt.Errorf("fstat target %s: %w", name, err)
	}
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return fmt.Errorf("target %s is not a regular file", name)
	}
	return nil
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
	return createFileAtForPurpose(parent, prefix, data, mode, privateArtifact, 0, nil)
}

func createDestinationFileAt(parent *os.File, prefix string, data []byte, mode uint32, durability *durabilityTracker) (*os.File, string, fileIdentity, error) {
	filesystemType, err := descriptorFilesystemType(parent)
	if err != nil {
		return nil, "", fileIdentity{}, fmt.Errorf("verify destination parent filesystem: %w", err)
	}
	return createFileAtForPurpose(parent, prefix, data, mode, destinationArtifact, filesystemType, durability)
}

func createSymlinkAt(parent *os.File, prefix, target string) (string, fileIdentity, error) {
	for range 16 {
		name, err := randomEntryName(prefix)
		if err != nil {
			return "", fileIdentity{}, err
		}
		if err := unix.Symlinkat(target, int(parent.Fd()), name); errors.Is(err, syscall.EEXIST) {
			continue
		} else if err != nil {
			return "", fileIdentity{}, fmt.Errorf("symlinkat %s: %w", name, err)
		}
		created, err := readExactSymlinkAt(parent, name, target)
		if err != nil {
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return "", fileIdentity{}, fmt.Errorf("validate temporary symlink: %w", err)
		}
		return name, created.id, nil
	}
	return "", fileIdentity{}, errors.New("create temporary symlink: name collision limit reached")
}

func randomEntryName(prefix string) (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate temporary name: %w", err)
	}
	return prefix + hex.EncodeToString(random[:]), nil
}

func createFileAtForPurpose(
	parent *os.File,
	prefix string,
	data []byte,
	mode uint32,
	purpose createdFilePurpose,
	filesystemType int64,
	durability *durabilityTracker,
) (*os.File, string, fileIdentity, error) {
	for range 16 {
		name, err := randomEntryName(prefix)
		if err != nil {
			return nil, "", fileIdentity{}, err
		}
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
		degraded, validationErr := validateCreatedFile(stat, mode, purpose, filesystemType)
		if validationErr != nil {
			file.Close()
			_ = syscall.Unlinkat(int(parent.Fd()), name)
			return nil, "", fileIdentity{}, fmt.Errorf("created file %s: %w", name, validationErr)
		}
		if degraded && durability != nil {
			durability.degrade(destinationModeEnforcementUnsupported)
		}
		return file, name, identityOf(stat), nil
	}
	return nil, "", fileIdentity{}, errors.New("create temporary file: name collision limit reached")
}

func validateCreatedFile(stat syscall.Stat_t, requestedMode uint32, purpose createdFilePurpose, filesystemType int64) (bool, error) {
	if stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return false, errors.New("created file is not regular")
	}
	if stat.Mode&(syscall.S_ISUID|syscall.S_ISGID|syscall.S_ISVTX) != 0 {
		return false, errors.New("created file has unsafe special mode bits")
	}
	if stat.Nlink != 1 {
		return false, errors.New("created file must have exactly one hard link")
	}
	if stat.Uid != uint32(os.Geteuid()) {
		return false, fmt.Errorf("created file owner is uid %d; require effective uid %d", stat.Uid, os.Geteuid())
	}
	actualMode := stat.Mode & 0o777
	if actualMode == requestedMode {
		return false, nil
	}
	if purpose == destinationArtifact && isV9FS(filesystemType) {
		return true, nil
	}
	return false, fmt.Errorf("created file mode is %04o; require %04o", actualMode, requestedMode)
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
	if err := syscall.Fstat(fd, &stat); err != nil {
		file.Close()
		return nil, fmt.Errorf("re-fstat transaction lock: %w", err)
	}
	if _, err := validateCreatedFile(stat, 0o600, privateArtifact, 0); err != nil {
		file.Close()
		return nil, fmt.Errorf("transaction lock is not private: %w", err)
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
