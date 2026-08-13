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

// SecureAtomicReplace is the cross-platform stub for the secure atomic
// replacement protocol. Returns an error on non-linux platforms; callers
// must have a portable fallback or be platform-gated.
func SecureAtomicReplace(string, []byte) error {
	return fmt.Errorf("secure atomic replacement is unavailable on this platform")
}
