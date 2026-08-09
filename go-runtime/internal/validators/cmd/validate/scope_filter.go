package main

import (
	"context"
	"time"

	"github.com/ovav/ovav/internal/validators"
)

// scopeFilter wraps a validator and skips it if no relevant files changed.
// It uses the shared ScopeCheck from the validators package.
type scopeFilter struct {
	v      validators.Validator
	scopes []string // path prefixes this validator cares about
}

// ScopeFilter returns a validator that is scope-filtered.
// If any scope prefix matches a changed file, the validator runs normally.
// If no scope matches, the validator returns SKIP (not PASS/FAIL).
func ScopeFilter(v validators.Validator, scopes []string) validators.Validator {
	return &scopeFilter{v: v, scopes: scopes}
}

func (s *scopeFilter) ID() string   { return s.v.ID() }
func (s *scopeFilter) Name() string { return s.v.Name() }
func (s *scopeFilter) Description() string {
	return s.v.Description() + " [scope-filtered]"
}
func (s *scopeFilter) Weight() int { return s.v.Weight() }

func (s *scopeFilter) Validate(ctx context.Context, root string) validators.Result {
	start := time.Now()

	// Use shared scope check from validators package
	if shouldRun, skipMsg := validators.ScopeCheck(s.ID(), root); !shouldRun {
		return validators.Result{
			ID:          s.ID(),
			Name:        s.Name(),
			Status:      "skip",
			Message:     skipMsg,
			Weight:      s.Weight(),
			Duration:    time.Since(start),
			Description: s.Description(),
		}
	}

	// Run the wrapped validator
	return s.v.Validate(ctx, root)
}

// scopeFilteredRegistry wraps certain validators with scope filtering.
// This enables intelligent opt-in: validators only run if relevant files changed.
// Uses the shared ValidatorScope and ScopeCheck from the validators package.
//
// When changedFiles is non-empty, sets ChangedFilesScope in the validators package
// so ScopeCheck can filter validators without scope declarations.
func scopeFilteredRegistry(root string, changedFiles []string) *validators.Registry {
	// Configure the changed-files scope for the validators package.
	// ScopeCheck reads this to filter validators without scope declarations.
	if len(changedFiles) > 0 {
		validators.SetChangedFiles(changedFiles)
		defer validators.ClearChangedFiles()
	}

	// Base registry with all validators
	registry := validators.DefaultRegistry()

	// Wrap validators that have scope declarations in ValidatorScope
	allValidators := registry.All()
	var filteredVals []validators.Validator

	for _, v := range allValidators {
		if scopes, ok := validators.ValidatorScope[v.ID()]; ok {
			// Use shared scope check to determine if validator should run
			if shouldRun, _ := validators.ScopeCheck(v.ID(), root); !shouldRun {
				// No relevant files changed → wrap with scope filter (will skip at validate time)
				filteredVals = append(filteredVals, ScopeFilter(v, scopes))
			} else {
				// Scope path exists and has changes → run normally
				filteredVals = append(filteredVals, v)
			}
		} else {
			filteredVals = append(filteredVals, v)
		}
	}

	return validators.NewRegistry(filteredVals...)
}
