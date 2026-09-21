//go:build !linux

package validators

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func readRegularFileNoFollow(path string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	current := string(filepath.Separator)
	parts := strings.Split(strings.TrimPrefix(filepath.Clean(abs), string(filepath.Separator)), string(filepath.Separator))
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("unsafe parent path")
		}
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	after, err := file.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fmt.Errorf("file changed while opening")
	}
	return io.ReadAll(file)
}

func openRegularFileAtNoFollow(_, _ string, _ os.FileInfo) (*anchoredRegularFile, error) {
	return nil, fmt.Errorf("anchored directory projection is unsupported on this platform")
}
