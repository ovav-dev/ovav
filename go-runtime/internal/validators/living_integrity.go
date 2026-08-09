package validators

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LivingIntegrity is the F0 integrity mesh orchestrator.
// It runs all core validators and computes a weighted integrity score (0-100).
// Status: healthy (≥95%), degraded (≥1 failure), critical (≥3 failures).
type LivingIntegrity struct {
	validators []Validator
}

// NewLivingIntegrity creates the living integrity orchestrator with all F0 validators.
func NewLivingIntegrity() *LivingIntegrity {
	return &LivingIntegrity{
		validators: []Validator{
			NewSupplyChain(),
			NewSecretsHygiene(),
			NewRuntimeIntegrity(),
			NewExfilPatterns(),
			NewProtectedBranch(),
			NewWorkspaceSafety(),
			NewGitPush(),
			NewPermissionDrift(),
			NewContractFreshness(),
			NewInstallVerification(),
			NewSecurityPolicy(),
			NewConfigIntegrity(),
			NewAgentGovernance(),
			NewPluginSecurity(),
		},
	}
}

func (l *LivingIntegrity) ID() string   { return "living_integrity" }
func (l *LivingIntegrity) Name() string { return "Living Integrity Mesh" }
func (l *LivingIntegrity) Description() string {
	return "Orchestrates all F0 validators and computes integrity score"
}
func (l *LivingIntegrity) Weight() int { return 100 }

// MeshResult is the aggregated result from running all F0 validators.
type MeshResult struct {
	Schema         string   `json:"schema"`
	CheckedAt      string   `json:"checked_at"`
	Overall        string   `json:"overall"` // "healthy", "degraded", "critical"
	Score          float64  `json:"integrity_score"`
	Total          int      `json:"total_validators"`
	Passed         int      `json:"passed"`
	Failed         int      `json:"failed"`
	Errors         int      `json:"errors"`
	Results        []Result `json:"validators"`
	Recommendation string   `json:"recommendation"`
}

func (l *LivingIntegrity) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var allResults []Result
	passed := 0
	failed := 0
	errors := 0
	totalWeight := 0
	weightedScore := 0.0

	for _, v := range l.validators {
		// Scope-aware skip: skip validators whose scope files weren't changed
		if shouldRun, skipMsg := ScopeCheck(v.ID(), root); !shouldRun {
			result := Result{
				ID:       v.ID(),
				Name:     v.Name(),
				Status:   "skip",
				Message:  skipMsg,
				Weight:   v.Weight(),
				Duration: 0,
			}
			allResults = append(allResults, result)
			continue
		}

		result := v.Validate(ctx, root)

		switch result.Status {
		case "pass":
			passed++
			totalWeight += result.Weight
			weightedScore += float64(result.Weight)
		case "fail":
			failed++
			totalWeight += result.Weight
		case "error":
			errors++
			totalWeight += result.Weight
		}
		allResults = append(allResults, result)
	}

	// Compute score
	var score float64
	if totalWeight > 0 {
		score = (weightedScore / float64(totalWeight)) * 100
	} else if passed > 0 {
		score = 100.0
	}

	// Determine overall status
	overall := "healthy"
	if errors > 1 || failed >= 3 {
		overall = "critical"
	} else if failed >= 1 || errors == 1 {
		overall = "degraded"
	} else if score < 95 {
		overall = "degraded"
	}

	rec := recommendation(overall, score, failed, errors)

	// Persist integrity status
	persistIntegrityStatus(root, overall, score, passed, failed, errors)

	meshResult := MeshResult{
		Schema:         "ovav_living_integrity_v1",
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		Overall:        overall,
		Score:          score,
		Total:          len(l.validators),
		Passed:         passed,
		Failed:         failed,
		Errors:         errors,
		Results:        allResults,
		Recommendation: rec,
	}
	_ = meshResult

	if overall == "healthy" {
		return Result{
			ID: l.ID(), Name: l.Name(), Status: "pass", Weight: l.Weight(),
			Message:  fmt.Sprintf("PASS living integrity — %.0f%% (%d/%d passed)", score, passed, len(l.validators)),
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: l.ID(), Name: l.Name(), Status: "fail", Weight: l.Weight(),
		Message:  fmt.Sprintf("FAIL living integrity — %.0f%% (%d pass, %d fail, %d errors)", score, passed, failed, errors),
		Duration: time.Since(start),
	}
}

func recommendation(overall string, score float64, failed, errors int) string {
	switch overall {
	case "healthy":
		return "Integrity Mesh VERDE. Todos los validadores F0 pasan. Sistema seguro para operar."
	case "degraded":
		return fmt.Sprintf("Integrity Mesh DEGRADADO (%.1f%%). %d validador(es) fallaron, %d error(es). Revisar y reparar antes de continuar.", score, failed, errors)
	default:
		return fmt.Sprintf("Integrity Mesh CRÍTICO (%.1f%%). %d fallos, %d errores. Lockdown recomendado.", score, failed, errors)
	}
}

// persistIntegrityStatus writes the integrity mesh result to .ovav/runtime/integrity_status.json
func persistIntegrityStatus(root string, overall string, score float64, passed, failed, errors int) {
	runtimeDir := filepath.Join(root, ".ovav", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return
	}
	statusFile := filepath.Join(runtimeDir, "integrity_status.json")
	icon := "✅"
	if overall == "degraded" {
		icon = "⚠️"
	} else if overall == "critical" {
		icon = "🔴"
	}
	content := fmt.Sprintf(`{"updated_at":"%s","status":"%s","score":%.1f,"passed":%d,"failed":%d,"errors":%d,"label":"Integrity: %.0f%%","icon":"%s"}`,
		time.Now().UTC().Format(time.RFC3339), overall, score, passed, failed, errors, score, icon)
	os.WriteFile(statusFile, []byte(content), 0644)
}

var _ Validator = (*LivingIntegrity)(nil)
