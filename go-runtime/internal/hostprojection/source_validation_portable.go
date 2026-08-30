package hostprojection

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func validateSourceNoFollow(path string, validate SourceValidator) (string, error) {
	pathInfo, err := lstatNoSymlinkPath(path)
	if err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat opened source: %w", err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(pathInfo, opened) {
		return "", errors.New("source identity changed before open")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}
	after, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("restat opened source: %w", err)
	}
	pathAfter, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("restat source path: %w", err)
	}
	if pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.Mode().IsRegular() ||
		!os.SameFile(opened, after) || !os.SameFile(after, pathAfter) ||
		opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return "", fmt.Errorf("%w while reading source", ErrConcurrentChange)
	}
	if validate != nil {
		if err := validate(append([]byte(nil), data...)); err != nil {
			return "", fmt.Errorf("validate source content: %w", err)
		}
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func lstatNoSymlinkPath(path string) (os.FileInfo, error) {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return nil, errors.New("source path must be absolute")
	}
	volume := filepath.VolumeName(clean)
	remainder := strings.TrimPrefix(clean, volume)
	remainder = strings.TrimLeft(remainder, string(os.PathSeparator))
	components := strings.Split(remainder, string(os.PathSeparator))
	current := volume + string(os.PathSeparator)
	if volume == "" {
		current = string(os.PathSeparator)
	}
	var info os.FileInfo
	for index, component := range components {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		var err error
		info, err = os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("lstat source component %q: %w", component, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("source component %q is a symlink", component)
		}
		if index < len(components)-1 && !info.IsDir() {
			return nil, fmt.Errorf("source parent component %q is not a directory", component)
		}
	}
	if info == nil || !info.Mode().IsRegular() {
		return nil, errors.New("source is not a regular file")
	}
	return info, nil
}
