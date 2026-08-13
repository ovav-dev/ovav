//go:build !linux

package identity

import (
	"fmt"
	"os"
)

func secureAtomicReplace(string, []byte, []byte, os.FileInfo, bool, func(string) error) (bool, error) {
	return false, fmt.Errorf("secure atomic replacement is unavailable on this platform")
}

func secureAtomicCreate(string, []byte, func(string) error) (bool, error) {
	return false, fmt.Errorf("secure atomic creation is unavailable on this platform")
}
