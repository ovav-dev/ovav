package ows

import (
	"fmt"
	"strings"
)

// ── F8: AI Conflict Resolution ───────────────────────────────────────────

// ResolutionStrategy defines how an AI-assisted merge resolves a conflict.
type ResolutionStrategy string

const (
	ResolvePreferSource ResolutionStrategy = "prefer_source"
	ResolvePreferTarget ResolutionStrategy = "prefer_target"
	ResolveUnion        ResolutionStrategy = "union"
	ResolveInteractive  ResolutionStrategy = "interactive"
)

// ConflictResolution represents an AI-suggested resolution for a single file.
type ConflictResolution struct {
	FilePath        string             `json:"file_path"`
	Strategy        ResolutionStrategy `json:"strategy"`
	Confidence      float64            `json:"confidence"` // 0.0-1.0
	Reasoning       string             `json:"reasoning"`
	ResolvedContent string             `json:"resolved_content,omitempty"`
}

// ResolutionPlan is a set of AI-suggested resolutions for a conflict matrix.
type ResolutionPlan struct {
	Matrix      *ConflictMatrix      `json:"matrix"`
	Resolutions []ConflictResolution `json:"resolutions"`
	AutoApplied int                  `json:"auto_applied"`
	NeedsReview int                  `json:"needs_review"`
}

// PlanResolution analyzes a conflict matrix and proposes resolutions.
// Files with non-overlapping changes are auto-resolved (union).
// Files with overlapping changes are flagged for review.
func PlanResolution(matrix *ConflictMatrix) *ResolutionPlan {
	plan := &ResolutionPlan{
		Matrix:      matrix,
		Resolutions: make([]ConflictResolution, 0, len(matrix.Files)),
	}

	for _, file := range matrix.Files {
		resolution := ConflictResolution{
			FilePath: file.FilePath,
		}

		switch file.Status {
		case "safe":
			// Non-overlapping changes — auto-merge with union strategy
			resolution.Strategy = ResolveUnion
			resolution.Confidence = 0.95
			resolution.Reasoning = fmt.Sprintf(
				"Non-overlapping changes: source modified lines %v, target modified lines %v. Safe to auto-merge.",
				formatRanges(file.SourceRanges), formatRanges(file.TargetRanges),
			)
			plan.AutoApplied++

		case "conflict":
			// Overlapping changes — needs review
			if len(file.OverlapRanges) <= 2 {
				// Small conflict — suggest prefer_source with medium confidence
				resolution.Strategy = ResolvePreferSource
				resolution.Confidence = 0.6
				resolution.Reasoning = fmt.Sprintf(
					"Minor overlap at lines %v. Suggest accepting source changes. Manual review recommended.",
					formatRanges(file.OverlapRanges),
				)
			} else {
				// Large conflict — requires interactive resolution
				resolution.Strategy = ResolveInteractive
				resolution.Confidence = 0.3
				resolution.Reasoning = fmt.Sprintf(
					"Significant overlap across %d ranges — interactive resolution required.",
					len(file.OverlapRanges),
				)
			}
			plan.NeedsReview++

		case "source_only", "target_only":
			// Only one side modified — auto-accept
			resolution.Strategy = ResolveUnion
			resolution.Confidence = 1.0
			resolution.Reasoning = fmt.Sprintf(
				"File only modified on one side (%s). Safe to auto-apply.", file.Status,
			)
			plan.AutoApplied++
		}

		plan.Resolutions = append(plan.Resolutions, resolution)
	}

	return plan
}

// Summary returns a one-line summary of the resolution plan.
func (p *ResolutionPlan) Summary() string {
	total := len(p.Resolutions)
	if p.NeedsReview == 0 {
		return fmt.Sprintf("✓ %d files — all auto-resolved (0 conflicts)", total)
	}
	return fmt.Sprintf("⚠ %d files — %d auto-applied, %d need review",
		total, p.AutoApplied, p.NeedsReview)
}

func formatRanges(ranges []LineRange) string {
	if len(ranges) == 0 {
		return "[]"
	}
	parts := make([]string, len(ranges))
	for i, r := range ranges {
		if r.Start == r.End {
			parts[i] = fmt.Sprintf("%d", r.Start)
		} else {
			parts[i] = fmt.Sprintf("%d-%d", r.Start, r.End)
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}
