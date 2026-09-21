//go:build !linux

package main

import (
	"fmt"
	"os"
)

func secureSessionReplace(string, []byte, os.FileMode) error {
	return fmt.Errorf("secure session replacement is unavailable on this platform")
}
