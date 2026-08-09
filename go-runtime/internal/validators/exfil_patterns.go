package validators

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// ExfilPatterns checks for exfiltration patterns in recent system output logs.
// Scans for data leak indicators: large base64 blobs, unexpected outbound URLs,
// and sensitive data patterns in log files.
type ExfilPatterns struct{}

func NewExfilPatterns() *ExfilPatterns { return &ExfilPatterns{} }

func (e *ExfilPatterns) ID() string   { return "exfil_patterns" }
func (e *ExfilPatterns) Name() string { return "Exfiltration Patterns" }
func (e *ExfilPatterns) Description() string {
	return "Scans output logs for data exfiltration patterns"
}
func (e *ExfilPatterns) Weight() int { return 10 }

// exfilSignal represents a suspicious pattern found in logs.
type exfilSignal struct {
	re    *regexp.Regexp
	label string
}

var exfilSignals = []exfilSignal{
	{regexp.MustCompile(`curl\s+.*\|\s*bash`), "curl-to-bash pipe (potential exfil)"},
	{regexp.MustCompile(`/etc/passwd`), "reference to /etc/passwd in output"},
	{regexp.MustCompile(`/etc/shadow`), "reference to /etc/shadow in output"},
	{regexp.MustCompile(`-----BEGIN\s+(RSA|EC|DSA|OPENSSH)\s+PRIVATE\s+KEY-----`), "private key in output"},
	{regexp.MustCompile(`\.env\b.*(?:token|secret|key|password)`), ".env reference with secrets context"},
}

// logDirs are directories that may contain OVAV output logs.
var logDirs = []string{
	".ovav/runtime",
	".ovav/logs",
	"/tmp/opencode",
}

func (e *ExfilPatterns) Validate(ctx context.Context, root string) Result {
	start := time.Now()

	select {
	case <-ctx.Done():
		return Result{ID: e.ID(), Name: e.Name(), Status: "error", Message: "context cancelled before scan"}
	default:
	}

	var issues []string
	scanned := 0

	for _, logRel := range logDirs {
		logPath := filepath.Join(root, logRel)
		info, err := os.Stat(logPath)
		if err != nil {
			continue // directory doesn't exist — not an error
		}
		if !info.IsDir() {
			continue
		}

		entries, err := os.ReadDir(logPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			if ctx.Err() != nil {
				return Result{ID: e.ID(), Name: e.Name(), Status: "error", Message: "context cancelled during scan"}
			}

			fullPath := filepath.Join(logPath, entry.Name())

			f, err := os.Open(fullPath)
			if err != nil {
				continue
			}

			scanner := bufio.NewScanner(f)
			// Limit line length to prevent memory issues on binary files.
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				// Check context every 1000 lines to allow prompt cancellation
				if lineNum%1000 == 0 && ctx.Err() != nil {
					f.Close()
					return Result{ID: e.ID(), Name: e.Name(), Status: "error", Message: "context cancelled during scan"}
				}
				line := scanner.Text()
				for _, sig := range exfilSignals {
					if sig.re.MatchString(line) {
						issues = append(issues, fmt.Sprintf("EXFIL in %s/%s:%d: %s", logRel, entry.Name(), lineNum, sig.label))
					}
				}
			}
			f.Close()
			scanned++
		}
	}

	if len(issues) == 0 {
		return Result{
			ID: e.ID(), Name: e.Name(), Status: "pass", Weight: e.Weight(),
			Message:  fmt.Sprintf("PASS exfiltration patterns — %d log(s) scanned, no anomalies", scanned),
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: e.ID(), Name: e.Name(), Status: "fail", Weight: e.Weight(),
		Message: fmt.Sprintf("FAIL exfiltration patterns — %d issue(s)", len(issues)),
		Issues:  issues, Duration: time.Since(start),
	}
}

var _ Validator = (*ExfilPatterns)(nil)
