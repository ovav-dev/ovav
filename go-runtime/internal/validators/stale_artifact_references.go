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

// StaleArtifactReferences detects stale S* segment references and pre-B18 BUILD references.
// Replaces: check_stale_artifact_references.py
type StaleArtifactReferences struct{}

func NewStaleArtifactReferences() *StaleArtifactReferences { return &StaleArtifactReferences{} }

func (s *StaleArtifactReferences) ID() string   { return "stale_artifact_references" }
func (s *StaleArtifactReferences) Name() string { return "Stale Artifact References Validator" }
func (s *StaleArtifactReferences) Description() string {
	return "Detects stale S* segment references and pre-B18 BUILD references in active files"
}
func (s *StaleArtifactReferences) Weight() int { return 7 }

var (
	staleSegmentPattern = regexp.MustCompile(`\bS\d{2,3}\b`)
	staleBuildPattern   = regexp.MustCompile(`BUILD\s+1[0-7]\b`)
	staleBuildPattern2  = regexp.MustCompile(`build\s*1[0-7]`)
)

var criticalPaths = []string{
	".ovav/plan/caps.yaml",
	"AGENTS.md",
}

var staleScanDirs = []string{
	"docs", ".opencode", ".ovav/topology", ".ovav/service_areas", "tools", "bin",
}

var staleExcludePatterns = []string{
	".ovav/artifacts/", ".ovav/runtime/sessions/", ".git/", "__pycache__", ".pytest_cache", "node_modules",
}

var allowedExtensions = map[string]bool{
	".md": true, ".yaml": true, ".py": true, ".json": true, ".sh": true, ".txt": true, ".lua": true,
}

func (s *StaleArtifactReferences) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var criticalIssues []string
	var warnIssues []string
	scanned := 0
	clean := 0

	shouldExclude := func(path string) bool {
		for _, pat := range staleExcludePatterns {
			if strings.Contains(path, pat) {
				return true
			}
		}
		return false
	}

	isCritical := func(path string) bool {
		rel, _ := filepath.Rel(root, path)
		for _, cp := range criticalPaths {
			if rel == cp || strings.HasSuffix(rel, "/"+cp) {
				return true
			}
		}
		return false
	}

	scanFile := func(fpath string) []string {
		data, err := os.ReadFile(fpath)
		if err != nil {
			return nil
		}
		content := string(data)
		var issues []string
		if matches := staleSegmentPattern.FindAllString(content, -1); len(matches) > 0 {
			unique := uniqueStrings(matches)
			if len(unique) > 5 {
				unique = unique[:5]
			}
			issues = append(issues, fmt.Sprintf("S* segment reference: %v", unique))
		}
		if matches := staleBuildPattern.FindAllString(content, -1); len(matches) > 0 {
			unique := uniqueStrings(matches)
			if len(unique) > 5 {
				unique = unique[:5]
			}
			issues = append(issues, fmt.Sprintf("BUILD 10-17 reference: %v", unique))
		}
		return issues
	}

	for _, scanDir := range staleScanDirs {
		fullDir := filepath.Join(root, scanDir)
		if info, err := os.Stat(fullDir); err != nil || !info.IsDir() {
			continue
		}
		filepath.Walk(fullDir, func(fpath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if shouldExclude(fpath) {
				return nil
			}
			ext := filepath.Ext(fpath)
			if !allowedExtensions[ext] {
				return nil
			}
			scanned++
			issues := scanFile(fpath)
			if len(issues) == 0 {
				clean++
				return nil
			}
			rel, _ := filepath.Rel(root, fpath)
			for _, issue := range issues {
				entry := fmt.Sprintf("%s: %s", rel, issue)
				if isCritical(fpath) {
					criticalIssues = append(criticalIssues, entry)
				} else {
					warnIssues = append(warnIssues, entry)
				}
			}
			return nil
		})
	}

	allIssues := append(criticalIssues, warnIssues...)
	if len(criticalIssues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "fail", Weight: s.Weight(),
			Message:  fmt.Sprintf("FAIL — %d critical, %d warnings in %d files scanned (%d clean)", len(criticalIssues), len(warnIssues), scanned, clean),
			Issues:   allIssues,
			Duration: time.Since(start)}
	}
	if len(warnIssues) > 0 {
		return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
			Message:  fmt.Sprintf("PASS (warnings) — %d warnings in %d files scanned (%d clean)", len(warnIssues), scanned, clean),
			Issues:   warnIssues,
			Duration: time.Since(start)}
	}
	return Result{ID: s.ID(), Name: s.Name(), Status: "pass", Weight: s.Weight(),
		Message:  fmt.Sprintf("PASS — no stale references in %d files scanned", scanned),
		Duration: time.Since(start)}
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

var _ Validator = (*StaleArtifactReferences)(nil)
