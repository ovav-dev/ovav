package main

import (
	"context"
	"time"
)

// makeStdContextWithTimeout wraps context.WithTimeout for use in launch_cli.go
// without forcing a context import there.
func makeStdContextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
