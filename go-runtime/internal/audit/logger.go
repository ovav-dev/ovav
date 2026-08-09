package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogEntry represents a single audit log record.
type LogEntry struct {
	TS         time.Time              `json:"ts"`
	Op         string                 `json:"op"`
	Actor      string                 `json:"actor"`
	Resource   string                 `json:"resource"`
	DurationMs int64                  `json:"duration_ms"`
	Result     string                 `json:"result"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// AuditLogger records structured audit events to an append-only JSON file.
type AuditLogger struct {
	dir        string
	file       string
	maxLogSize int
	logger     *slog.Logger
	mu         sync.RWMutex
	closed     bool
}

// Option configures an AuditLogger.
type Option func(*AuditLogger)

// Dir sets the directory where the audit log file is stored.
// If not set, defaults to ".audit" in the OS temp dir.
func Dir(path string) Option {
	return func(l *AuditLogger) {
		l.dir = path
	}
}

// MaxLogSize sets the maximum log entry size in bytes.
// If not set, defaults to 64KB.
func MaxLogSize(n int) Option {
	return func(l *AuditLogger) {
		l.maxLogSize = n
	}
}

// New creates a new AuditLogger. The log file is append-only.
func New(opts ...Option) (*AuditLogger, error) {
	l := &AuditLogger{
		dir:        filepath.Join(os.TempDir(), ".audit"),
		maxLogSize: 64 * 1024,
		logger:     slog.Default(),
	}

	for _, o := range opts {
		o(l)
	}

	if err := os.MkdirAll(l.dir, 0755); err != nil {
		return nil, fmt.Errorf("audit.New: mkdir %s: %w", l.dir, err)
	}

	l.file = filepath.Join(l.dir, "audit.log")

	return l, nil
}

// Log records an audit event. It extracts actor/resource/op from ctx
// and measures execution time of the provided function.
func (l *AuditLogger) Log(ctx context.Context, op OpLevel, resource, result string, details map[string]interface{}, fn func() error) error {
	if l.isClosed() {
		return fmt.Errorf("audit.Log: logger is closed")
	}

	start := time.Now()
	var err error
	if fn != nil {
		err = fn()
	}
	duration := time.Since(start)

	entry := LogEntry{
		TS:         start.UTC(),
		Op:         op.String(),
		Actor:      ActorFrom(ctx),
		Resource:   resource,
		DurationMs: duration.Milliseconds(),
		Result:     result,
		Details:    details,
	}

	if err != nil {
		entry.Result = fmt.Sprintf("error: %v", err)
	}

	return l.appendEntry(entry)
}

// LogImmediate records an audit event without timing a function.
func (l *AuditLogger) LogImmediate(ctx context.Context, op OpLevel, resource, result string, details map[string]interface{}) error {
	if l.isClosed() {
		return fmt.Errorf("audit.LogImmediate: logger is closed")
	}
	entry := LogEntry{
		TS:         time.Now().UTC(),
		Op:         op.String(),
		Actor:      ActorFrom(ctx),
		Resource:   resource,
		DurationMs: 0,
		Result:     result,
		Details:    details,
	}
	return l.appendEntry(entry)
}

func (l *AuditLogger) appendEntry(entry LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit.appendEntry: json.Marshal: %w", err)
	}
	if len(data) > l.maxLogSize {
		data = data[:l.maxLogSize]
	}

	// Ensure JSON line ends with newline.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	f, err := os.OpenFile(l.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("audit.appendEntry: OpenFile: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("audit.appendEntry: Write: %w", err)
	}
	return nil
}

// Query returns a QueryBuilder for reading log entries.
func (l *AuditLogger) Query() *QueryBuilder {
	return &QueryBuilder{file: l.file}
}

// Close is a no-op for the file-based logger (log persists).
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	return nil
}

func (l *AuditLogger) isClosed() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.closed
}
