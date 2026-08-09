package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TodoDebt scans .py and .go source files for TODO/FIXME/HACK markers
// and reports count. Informational only — never fails.
// Replaces: check_todo_debt.py
type TodoDebt struct{}

func NewTodoDebt() *TodoDebt { return &TodoDebt{} }

func (t *TodoDebt) ID() string   { return "todo_debt" }
func (t *TodoDebt) Name() string { return "TODO Debt Tracker" }
func (t *TodoDebt) Description() string {
	return "Scans source code for TODO/FIXME/HACK markers (informational)"
}
func (t *TodoDebt) Weight() int { return 3 }

// Directories to scan for TODO markers.
var todoScanDirs = []string{"tools", ".opencode", "go-runtime"}

// Extensions to scan.
var todoScanExts = map[string]bool{".py": true, ".go": true}

// Files excluded from scanning.
var todoExcludeFiles = map[string]bool{
	"tools/validators/check_todo_debt.py": true,
}

// Skip directories matching these names.
var todoSkipDirs = map[string]bool{
	"__pycache__": true,
	".git":        true,
}

// Regex for TODO/FIXME/HACK markers. English-only, anchored to comment starts.
var todoPattern = regexp.MustCompile(`(?i)#\s*(?:TODO|FIXME|HACK)\b|//\s*(?:TODO|FIXME|HACK)\b`)

type todoCount struct {
	path  string
	count int
}

func (t *TodoDebt) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	counts := []todoCount{}
	totalTodos := 0

	for _, scanDir := range todoScanDirs {
		dirPath := filepath.Join(root, scanDir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if todoSkipDirs[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			ext := filepath.Ext(path)
			if !todoScanExts[ext] {
				return nil
			}

			rel, _ := filepath.Rel(root, path)
			if todoExcludeFiles[rel] {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			count := 0
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				// Skip doc-only comments (NOTE:, DOC:, EXAMPLE, REFERENCE)
				if strings.HasPrefix(trimmed, "#") {
					upper := strings.ToUpper(trimmed)
					if strings.Contains(upper, "NOTE:") ||
						strings.Contains(upper, "DOC:") ||
						strings.Contains(upper, "EXAMPLE") ||
						strings.Contains(upper, "REFERENCE") {
						continue
					}
				}
				if todoPattern.MatchString(trimmed) {
					count++
				}
			}

			if count > 0 {
				counts = append(counts, todoCount{rel, count})
				totalTodos += count
			}
			return nil
		})
	}

	// Report results (informational — always pass)
	filesWithTodos := len(counts)
	issues = append(issues, fmt.Sprintf(
		"INFO: %d TODO/FIXME/HACK markers in %d files",
		totalTodos, filesWithTodos,
	))

	// Per-directory breakdown
	dirCounts := map[string]int{}
	for _, c := range counts {
		topDir := strings.SplitN(c.path, string(filepath.Separator), 2)[0]
		dirCounts[topDir] += c.count
	}
	for dir, count := range dirCounts {
		issues = append(issues, fmt.Sprintf("  %s/: %d markers", dir, count))
	}

	// Threshold alerts
	const warnThreshold = 50
	const criticalThreshold = 100
	if totalTodos > criticalThreshold {
		issues = append(issues, fmt.Sprintf(
			"WARN: %d TODO markers exceeds critical threshold (%d). Technical debt accumulating.",
			totalTodos, criticalThreshold,
		))
	} else if totalTodos > warnThreshold {
		issues = append(issues, fmt.Sprintf(
			"WARN: %d TODO markers exceeds warning threshold (%d). Review recommended.",
			totalTodos, warnThreshold,
		))
	}

	// Top offenders (max 5)
	if len(counts) > 0 {
		// Sort by count descending (simple bubble for small N)
		for i := 0; i < len(counts)-1; i++ {
			for j := i + 1; j < len(counts); j++ {
				if counts[j].count > counts[i].count {
					counts[i], counts[j] = counts[j], counts[i]
				}
			}
		}
		limit := 5
		if len(counts) < limit {
			limit = len(counts)
		}
		issues = append(issues, "Top offenders:")
		for i := 0; i < limit; i++ {
			issues = append(issues, fmt.Sprintf("  %s: %d markers", counts[i].path, counts[i].count))
		}
	}

	// Informational — always pass
	return Result{
		ID: t.ID(), Name: t.Name(), Status: "pass", Weight: t.Weight(),
		Message:  fmt.Sprintf("INFO — %d TODO/FIXME/HACK markers across %d files", totalTodos, filesWithTodos),
		Issues:   issues,
		Duration: time.Since(start),
	}
}

var _ Validator = (*TodoDebt)(nil)
