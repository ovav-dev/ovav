// Package audit provides structured audit logging for go-runtime.
// All API operations are recorded with full context: operation, actor,
// resource, timestamp, duration, and result. Logs survive restarts via
// Badger (SSD-backed embedded KV store).
package audit

import "fmt"

// OpLevel represents the sensitivity level of an operation.
type OpLevel int

const (
	OpRead  OpLevel = iota // 0 — read-only operations
	OpWrite                // 1 — write operations
	OpAdmin                // 2 — administrative operations
)

func (l OpLevel) String() string {
	switch l {
	case OpRead:
		return "READ"
	case OpWrite:
		return "WRITE"
	case OpAdmin:
		return "ADMIN"
	default:
		return fmt.Sprintf("OPLEVEL(%d)", l)
	}
}

// IsValid returns true if the OpLevel is a recognized value.
func (l OpLevel) IsValid() bool {
	return l >= OpRead && l <= OpAdmin
}
