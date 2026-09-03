//go:build unix

package identity

import (
	"fmt"
	"os"
	"syscall"
)

func verifyCurrentUserOwnership(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || (stat.Uid != uint32(os.Geteuid()) && stat.Uid != 0) {
		return fmt.Errorf("path is not owned by current user or root")
	}
	return nil
}
