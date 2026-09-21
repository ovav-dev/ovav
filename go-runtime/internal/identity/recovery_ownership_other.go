//go:build !unix

package identity

import "os"

func verifyCurrentUserOwnership(os.FileInfo) error { return nil }
