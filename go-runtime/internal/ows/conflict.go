package ows

import (
	"fmt"
	"os/exec"
	"strings"
)

// ── F3: Conflict Prediction Engine ───────────────────────────────────────────
//
// Refactored: replaces ~150 LOC of `git diff -U0` hunk parsing and heuristic
// line-range overlap detection with a single `git merge-tree` plumbing call
// (git 2.38+). merge-tree performs a real in-memory three-way merge and reports
// exact conflicts — no guesswork.

// ConflictPrediction represents the conflict analysis for a single file.
type ConflictPrediction struct {
	FilePath      string      `json:"file_path"`
	Status        string      `json:"status"` // "conflict", "safe", "source_only", "target_only"
	SourceRanges  []LineRange `json:"source_ranges,omitempty"`
	TargetRanges  []LineRange `json:"target_ranges,omitempty"`
	OverlapRanges []LineRange `json:"overlap_ranges,omitempty"`
	SourceAdds    int         `json:"source_additions"`
	TargetAdds    int         `json:"target_additions"`
	SourceDels    int         `json:"source_deletions"`
	TargetDels    int         `json:"target_deletions"`
	CanAutoMerge  bool        `json:"can_auto_merge"`
}

// LineRange represents a contiguous range of modified lines.
type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// ConflictMatrix is the complete conflict analysis between two refs.
type ConflictMatrix struct {
	BaseRef       string               `json:"base_ref"`
	SourceRef     string               `json:"source_ref"`
	TargetRef     string               `json:"target_ref"`
	TotalFiles    int                  `json:"total_files"`
	ConflictFiles int                  `json:"conflict_files"`
	SafeFiles     int                  `json:"safe_files"`
	Files         []ConflictPrediction `json:"files"`
}

// PredictConflicts analyzes potential conflicts between sourceRef and targetRef
// using `git merge-tree` — a single plumbing call that performs an in-memory
// three-way merge and reports exact conflicts.
//
// This replaces the previous ~150 LOC implementation that used `git diff -U0`
// with manual hunk parsing and heuristic line-range overlap detection.
//
// sourceRef: the branch being merged (e.g., "task/my-feature")
// targetRef: the target branch (e.g., "develop")
func PredictConflicts(repoRoot, sourceRef, targetRef string) (*ConflictMatrix, error) {
	// Detect common ancestor
	baseRef, err := runGitOutput(repoRoot, "merge-base", sourceRef, targetRef)
	if err != nil {
		return nil, fmt.Errorf("detect merge-base %s..%s: %w", sourceRef, targetRef, err)
	}

	matrix := &ConflictMatrix{
		BaseRef:   baseRef,
		SourceRef: sourceRef,
		TargetRef: targetRef,
	}

	// Phase 1: git merge-tree → exact conflicts (structural + textual)
	// Legacy 3-arg form: merge-tree <base-tree> <our-tree> <their-tree>.
	output, err := runMergeTree(repoRoot, baseRef, sourceRef, targetRef)
	if err != nil {
		return nil, fmt.Errorf("merge-tree %s %s %s: %w", baseRef, sourceRef, targetRef, err)
	}
	conflictPreds := parseMergeTreeOutput(output)

	// Build set of conflicting file paths for fast lookup
	conflictSet := make(map[string]*ConflictPrediction, len(conflictPreds))
	for i := range conflictPreds {
		conflictSet[conflictPreds[i].FilePath] = &conflictPreds[i]
	}

	// Phase 2: git diff --name-only → all changed files (source-side + target-side)
	// This catches files that merge-tree doesn't report (safe, one-sided additions).
	sourceFiles, _ := changedFiles(repoRoot, baseRef, sourceRef)
	targetFiles, _ := changedFiles(repoRoot, baseRef, targetRef)
	sourceSet := makeSet(sourceFiles)
	targetSet := makeSet(targetFiles)
	allFiles := union(sourceSet, targetSet)

	for filePath := range allFiles {
		if cp, isConflict := conflictSet[filePath]; isConflict {
			// Already classified by merge-tree
			matrix.Files = append(matrix.Files, *cp)
		} else {
			// Not in merge-tree output → safe to auto-merge
			inSource := sourceSet[filePath]
			inTarget := targetSet[filePath]
			status := "safe"
			if inSource && !inTarget {
				status = "source_only"
			} else if !inSource && inTarget {
				status = "target_only"
			}
			pred := ConflictPrediction{
				FilePath:     filePath,
				Status:       status,
				CanAutoMerge: true,
			}
			matrix.Files = append(matrix.Files, pred)
		}
		matrix.TotalFiles++
		if matrix.Files[len(matrix.Files)-1].Status == "conflict" {
			matrix.ConflictFiles++
		} else {
			matrix.SafeFiles++
		}
	}

	return matrix, nil
}

// ── merge-tree Execution & Parsing ───────────────────────────────────────────

// runMergeTree executes `git merge-tree` and returns output even on non-zero exit.
// merge-tree may exit non-zero when conflicts exist — the output is still valid.
func runMergeTree(repoRoot, base, source, target string) (string, error) {
	cmd := exec.Command("git", "merge-tree", base, source, target)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if output == "" {
			return "", fmt.Errorf("git merge-tree %s %s %s: %w", base, source, target, err)
		}
		// Non-zero exit with output → conflicts exist, output is still valid
	}
	return output, nil
}

// parseMergeTreeOutput parses the output of `git merge-tree <base> <our> <their>`.
//
// Output format (legacy mode):
//
//	<tree-OID>
//
//	changed in both
//	  base   100644 <hash> <path>
//	  our    100644 <hash> <path>
//	  their  100644 <hash> <path>
//	@@ -old,count +new,count @@
//	...diff hunks, possibly with <<<<<<< / ======= / >>>>>>> markers...
//
//	added in remote
//	  base   100644 0{40} <path>
//	  our    100644 0{40} <path>
//	  their  100644 <hash> <path>
func parseMergeTreeOutput(output string) []ConflictPrediction {
	if output == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	var predictions []ConflictPrediction

	// Skip first line if it's a tree OID (40 hex chars, no conflicts case).
	// When conflicts exist, merge-tree outputs no tree OID — content starts directly.
	i := 0
	if len(lines) > 0 && isTreeHash(lines[0]) {
		i = 1
	}
	// Skip blank lines after optional tree OID
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	for i < len(lines) {
		sectionType := detectSectionType(lines[i])
		if sectionType == "" {
			i++
			continue
		}
		i++ // consume section header

		// Read 3 metadata lines: base, our, their
		var basePath, ourPath, theirPath string
		if i < len(lines) {
			basePath = extractPath(lines[i])
			i++
		}
		if i < len(lines) {
			ourPath = extractPath(lines[i])
			i++
		}
		if i < len(lines) {
			theirPath = extractPath(lines[i])
			i++
		}

		filePath := firstNonEmpty(basePath, ourPath, theirPath)
		if filePath == "" {
			continue
		}

		// Read diff hunks until next section header or EOF
		hasConflictMarkers := false
		inOurBlock := false // between <<<<<<< and =======
		sourceAdds, targetAdds := 0, 0
		sourceDels, targetDels := 0, 0
		var sourceRanges, targetRanges []LineRange

		for i < len(lines) {
			h := lines[i]

			// Next section header → stop consuming hunks
			if detectSectionType(h) != "" {
				break
			}

			// Track conflict markers (merge-tree diff lines start with + or - prefix)
			if strings.HasPrefix(h, "+<<<<<<<") || strings.HasPrefix(h, "<<<<<<<") {
				hasConflictMarkers = true
				inOurBlock = true
			} else if strings.HasPrefix(h, "+=======") || strings.HasPrefix(h, "=======") {
				inOurBlock = false
			} else if strings.HasPrefix(h, "+>>>>>>>") || strings.HasPrefix(h, ">>>>>>>") {
				inOurBlock = false
			}

			// Parse hunk headers for line ranges
			if strings.HasPrefix(h, "@@") {
				if rng := parseHunkRange(h); rng != nil {
					sourceRanges = append(sourceRanges, *rng)
					targetRanges = append(targetRanges, *rng)
				}
			}

			// Count additions/deletions (skip markers, file headers, context)
			if strings.HasPrefix(h, "+") && !strings.HasPrefix(h, "+++") {
				if inOurBlock {
					sourceAdds++
				} else if !isConflictMarkerLine(h) {
					sourceAdds++
					targetAdds++
				}
			} else if strings.HasPrefix(h, "-") && !strings.HasPrefix(h, "---") {
				sourceDels++
				targetDels++
			}

			i++
		}

		pred := ConflictPrediction{
			FilePath:     filePath,
			SourceRanges: sourceRanges,
			TargetRanges: targetRanges,
			SourceAdds:   sourceAdds,
			TargetAdds:   targetAdds,
			SourceDels:   sourceDels,
			TargetDels:   targetDels,
		}

		switch sectionType {
		case "both":
			if hasConflictMarkers {
				pred.Status = "conflict"
				pred.CanAutoMerge = false
				pred.OverlapRanges = sourceRanges
			} else {
				pred.Status = "safe"
				pred.CanAutoMerge = true
			}
		case "source_only":
			pred.Status = "source_only"
			pred.CanAutoMerge = true
		case "target_only":
			pred.Status = "target_only"
			pred.CanAutoMerge = true
		default: // "removed"
			pred.Status = "safe"
			pred.CanAutoMerge = true
		}

		predictions = append(predictions, pred)
	}

	return predictions
}

// ── Parser Helpers ───────────────────────────────────────────────────────────

// detectSectionType identifies merge-tree section headers.
// Returns: "both", "source_only", "target_only", "removed", or "" (not a header).
func detectSectionType(line string) string {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "changed in both" || trimmed == "added in both":
		return "both"
	case trimmed == "added in local" || trimmed == "changed in local":
		return "source_only"
	case trimmed == "added in remote" || trimmed == "changed in remote":
		return "target_only"
	case strings.HasPrefix(trimmed, "removed in"):
		return "removed"
	}
	return ""
}

// extractPath extracts the file path from a merge-tree metadata line.
// Format: "  base   100644 abc123def456... path/to/file.go"
func extractPath(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return ""
	}
	// fields: [label, mode, hash, path...]
	return strings.Join(fields[3:], " ")
}

// parseHunkRange extracts the new-file line range from a diff hunk header.
// Format: @@ -oldStart,oldCount +newStart,newCount @@
func parseHunkRange(header string) *LineRange {
	plusIdx := strings.Index(header, "+")
	if plusIdx < 0 {
		return nil
	}
	plusPart := header[plusIdx+1:]
	if spaceIdx := strings.Index(plusPart, " "); spaceIdx > 0 {
		plusPart = plusPart[:spaceIdx]
	}
	plusPart = strings.TrimSpace(plusPart)

	parts := strings.Split(plusPart, ",")
	if len(parts) == 0 || parts[0] == "" {
		return nil
	}

	var start, count int
	if _, err := fmt.Sscanf(parts[0], "%d", &start); err != nil {
		return nil
	}
	count = 1
	if len(parts) > 1 {
		if _, err := fmt.Sscanf(parts[1], "%d", &count); err != nil {
			count = 1
		}
	}

	return &LineRange{Start: start, End: start + count - 1}
}

// isConflictMarkerLine checks if a "+" prefixed line is actually a conflict marker
// embedded in the diff (e.g., "+<<<<<<< .our").
func isConflictMarkerLine(line string) bool {
	return strings.HasPrefix(line, "+<<<<<<<") ||
		strings.HasPrefix(line, "+=======") ||
		strings.HasPrefix(line, "+>>>>>>>")
}

// isTreeHash returns true if the line looks like a git tree OID (40 hex chars).
func isTreeHash(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) != 40 {
		return false
	}
	for _, c := range trimmed {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// ── General Helpers ──────────────────────────────────────────────────────────

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// changedFiles returns files changed between two refs using git diff --name-only.
func changedFiles(repoRoot, baseRef, targetRef string) ([]string, error) {
	out, err := runGitOutput(repoRoot, "diff", "--name-only", baseRef+"..."+targetRef)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// makeSet converts a string slice to a set (map[string]bool).
func makeSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			s[item] = true
		}
	}
	return s
}

// union returns the union of two string sets.
func union(a, b map[string]bool) map[string]bool {
	u := make(map[string]bool, len(a)+len(b))
	for k := range a {
		u[k] = true
	}
	for k := range b {
		u[k] = true
	}
	return u
}

// runGitOutput runs a git command and returns trimmed stdout.
func runGitOutput(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ── ConflictMatrix Methods ───────────────────────────────────────────────────

// Summary returns a human-readable one-line summary of the conflict matrix.
func (m *ConflictMatrix) Summary() string {
	if m.ConflictFiles == 0 {
		return fmt.Sprintf("✓ %d files — 0 conflicts, safe to merge", m.TotalFiles)
	}
	return fmt.Sprintf("⚠ %d files — %d conflict(s), %d safe", m.TotalFiles, m.ConflictFiles, m.SafeFiles)
}

// Conflicts returns only the files with predicted conflicts.
func (m *ConflictMatrix) Conflicts() []ConflictPrediction {
	var conflicts []ConflictPrediction
	for _, f := range m.Files {
		if f.Status == "conflict" {
			conflicts = append(conflicts, f)
		}
	}
	return conflicts
}
