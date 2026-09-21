package validators

// ── Fix #1: RegistryDrift honors auto_triggers execution_scope ────────────
// Bug: The validator at checkRouterTriggerCrossValidation treated every
// trigger declared under any block (router, registry_only_router, etc.) as
// needing a wired definition. The actual canonical contract in
// .ovav/registry/auto_triggers.yaml declares:
//
//   execution_scope: git_hook_stages_only
//   note: non-git-stage declarations are preserved below as
//         registry-only/disconnected records
//
// That means entries under `registry_only_router:` are explicitly NOT wired
// to git hooks. The validator must skip them.
//
// Pin: with execution_scope = git_hook_stages_only, only entries under the
// `router:` block are validated; entries under `registry_only_router:` and
// any other block are not.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryDrift_AutoTriggers_ExecutionScopeGitHookStagesOnly(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	// Contract manifest + auto_triggers with execution_scope set
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)

	// auto_triggers with execution_scope = git_hook_stages_only and entries
	// only under registry_only_router: should NOT cause drift.
	atYAML := `execution_scope: git_hook_stages_only
note: non-git-stage declarations preserved as registry-only
router: {}
registry_only_router:
  before_apply:
    - validate_workspace_safety_gate
  before_close:
    - check_L6_security_zero_trust
    - some_undefined_trigger_for_test
`
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(atYAML), 0644)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(`surfaces: {}
lane_validators: {}
`), 0644)

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	for _, issue := range result.Issues {
		if strings.Contains(issue, "auto_triggers router") {
			t.Errorf("router drift should be 0 with execution_scope=git_hook_stages_only and empty router block, got: %s", issue)
		}
		if strings.Contains(issue, "registry_only_router") {
			t.Errorf("registry_only_router entries must be ignored when execution_scope=git_hook_stages_only, got: %s", issue)
		}
		if strings.Contains(issue, "some_undefined_trigger_for_test") {
			t.Errorf("undefined triggers in registry_only_router must be ignored, got: %s", issue)
		}
	}
}

func TestRegistryDrift_AutoTriggers_MixedRouterScopes(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)

	// router block has one defined trigger + one undefined.
	// registry_only_router has many entries (must be ignored).
	atYAML := `execution_scope: git_hook_stages_only
router:
  before_git_stage:
    - git_push
    - totally_undefined_trigger_X
registry_only_router:
  before_apply:
    - some_undefined_trigger_in_registry_only
  before_close:
    - another_undefined_trigger_in_registry_only
`
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(atYAML), 0644)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(`surfaces: {}
lane_validators: {}
`), 0644)

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	routerCount := 0
	registryOnlyCount := 0
	for _, issue := range result.Issues {
		if strings.Contains(issue, "auto_triggers router") && strings.Contains(issue, "totally_undefined_trigger_X") {
			routerCount++
		}
		if strings.Contains(issue, "some_undefined_trigger_in_registry_only") {
			registryOnlyCount++
		}
		if strings.Contains(issue, "another_undefined_trigger_in_registry_only") {
			registryOnlyCount++
		}
	}
	if routerCount != 1 {
		t.Errorf("expected exactly 1 router drift issue for 'totally_undefined_trigger_X', got %d. Issues: %v", routerCount, result.Issues)
	}
	if registryOnlyCount != 0 {
		t.Errorf("registry_only_router entries must NOT be flagged, got %d. Issues: %v", registryOnlyCount, result.Issues)
	}
}

func TestRegistryDrift_AutoTriggers_NoExecutionScopeLegacyMode(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)

	// No execution_scope → legacy mode: validate ALL entries.
	atYAML := `router:
  before_git_stage:
    - totally_undefined_legacy_X
registry_only_router:
  before_apply:
    - also_undefined_legacy_Y
`
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(atYAML), 0644)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(`surfaces: {}
lane_validators: {}
`), 0644)

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	foundX := false
	foundY := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "totally_undefined_legacy_X") {
			foundX = true
		}
		if strings.Contains(issue, "also_undefined_legacy_Y") {
			foundY = true
		}
	}
	if !foundX {
		t.Errorf("legacy mode: expected router drift for 'totally_undefined_legacy_X', got: %v", result.Issues)
	}
	if !foundY {
		t.Errorf("legacy mode: expected drift for 'also_undefined_legacy_Y' (no execution_scope = everything is validated), got: %v", result.Issues)
	}
}

// ── Fix #2: SurfaceValidatorMap Python→Go migration ───────────────────────
// Bug: surface_validator_map.yaml still references the old Python validator
// IDs (validate_model_policy, check_supply_chain, etc.) inherited from the
// Python era. Those have been migrated to Go-native validators with different
// IDs (model_policy, supply_chain, etc.). The validator must translate known
// legacy Python IDs to their Go equivalents before reporting drift.

// fixtureGoValidator creates a Go validator stub file under
// dir/go-runtime/internal/validators/ with the given filename and ID.
// The stub declares a minimal ID() method returning id so the
// disk-scan in checkSurfaceValidatorMap picks up both the filename and
// the ID. Use this to model validators where ID != filename
// (e.g. memory_policy.go with ID "validate_memory_policy").
func fixtureGoValidator(t *testing.T, dir, filename, id string) {
	t.Helper()
	goValDir := filepath.Join(dir, "go-runtime", "internal", "validators")
	if err := os.MkdirAll(goValDir, 0755); err != nil {
		t.Fatal(err)
	}
	stub := fmt.Sprintf(`package validators

// Stub for test fixture.
type Stub struct{}

func (s *Stub) ID() string { return %q }
`, id)
	if err := os.WriteFile(filepath.Join(goValDir, filename+".go"),
		[]byte(stub), 0644); err != nil {
		t.Fatal(err)
	}
}

// fixtureGoValidatorByIDOnly creates a Go validator stub whose filename
// matches the ID. Most Go validators follow this pattern (e.g.
// model_policy.go with ID "model_policy").
func fixtureGoValidatorByIDOnly(t *testing.T, dir, id string) {
	fixtureGoValidator(t, dir, id, id)
}

func TestRegistryDrift_SurfaceValidatorMap_PythonToGoMigration(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)

	// auto_triggers minimal so it doesn't pollute the output
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(`router: {}
`), 0644)

	// All entries are legacy Python IDs that map to Go validators via the
	// migration table. None should be flagged as drift.
	svmYAML := `surfaces:
  .opencode/:
    validators:
      - name: validate_model_policy
  .ovav/:
    validators:
      - name: check_supply_chain
  .ovav/policy/:
    validators:
      - name: validate_permission_policy_drift
  .ovav/registry/:
    validators:
      - name: validate_harnesses
      - name: validate_memory_policy
      - name: validate_phase_dag
      - name: validate_registries
      - name: validate_result_contracts
      - name: validate_service_profiles
      - name: validate_skills
      - name: check_network_hardening
lane_validators: {}
`
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(svmYAML), 0644)

	// Stub Go validator files matching the migration targets. Note that
	// several Go validators keep their historical "validate_" prefix in
	// the ID even though the filename drops it (memory_policy.go →
	// "validate_memory_policy").
	fixtureGoValidatorByIDOnly(t, dir, "model_policy")
	fixtureGoValidatorByIDOnly(t, dir, "supply_chain")
	fixtureGoValidatorByIDOnly(t, dir, "permission_drift")
	fixtureGoValidatorByIDOnly(t, dir, "harness_integrity")
	fixtureGoValidator(t, dir, "memory_policy", "validate_memory_policy")
	fixtureGoValidator(t, dir, "phase_dag", "validate_phase_dag")
	fixtureGoValidatorByIDOnly(t, dir, "registry_validator")
	fixtureGoValidatorByIDOnly(t, dir, "contract_enforcement")
	fixtureGoValidator(t, dir, "service_profiles", "validate_service_profiles")
	fixtureGoValidator(t, dir, "skills", "validate_skills")
	fixtureGoValidatorByIDOnly(t, dir, "network_hardening")

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	for _, issue := range result.Issues {
		if strings.Contains(issue, "surface_validator_map") {
			t.Errorf("legacy Python IDs should be translated to Go IDs and NOT flagged, got: %s", issue)
		}
	}
}

func TestRegistryDrift_SurfaceValidatorMap_UnknownIDStillFlagged(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(`router: {}
`), 0644)

	// Mix known legacy ID (no drift) + unknown ID (must drift).
	svmYAML := `surfaces:
  .ovav/registry/:
    validators:
      - name: validate_harnesses
      - name: validate_total_nonsense_unknown_xyz
lane_validators: {}
`
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(svmYAML), 0644)

	// Stub for the known migration target.
	fixtureGoValidatorByIDOnly(t, dir, "harness_integrity")

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	foundUnknown := false
	for _, issue := range result.Issues {
		if strings.Contains(issue, "validate_total_nonsense_unknown_xyz") {
			foundUnknown = true
		}
		if strings.Contains(issue, "validate_harnesses") && strings.Contains(issue, "not found") {
			t.Errorf("legacy known ID 'validate_harnesses' should be migrated to harness_integrity, got: %s", issue)
		}
	}
	if !foundUnknown {
		t.Errorf("expected drift for unknown validator ID, got: %v", result.Issues)
	}
}

// ── Fix #3: ArtifactRegistry no longer hardcodes sample paths ────────────
// Bug: checkArtifactRegistry used a hardcoded samplePaths slice that
// referenced '.ovav/context/CURRENT_HANDOFF.md' — but no entry in
// artifacts.yaml actually points to that path. The hardcoded sample is
// wrong and produces a false positive drift.
//
// Fix: remove the hardcoded sample. The artifacts.yaml registry is a
// historical ledger of build-phase artifacts (S13, S22, S45, …) — most
// declared paths are not expected to exist on disk for the current
// build. artifact_registry is informational, not a gate. The check
// becomes a no-op.

func TestRegistryDrift_ArtifactRegistry_ValidDeclaredPaths(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(`router: {}
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(`surfaces: {}
lane_validators: {}
`), 0644)

	// Declare artifacts that exist on disk. The check is a no-op so
	// declared-but-missing historical entries are NOT flagged.
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "artifacts.yaml"), []byte(`artifacts:
  my_summary:
    path: .ovav/artifacts/init/MY_SUMMARY.md
    phase: init
  my_report:
    path: .ovav/artifacts/init/evidence/MY_REPORT.json
    phase: verify
  historical_artifact:
    path: .ovav/artifacts/S99/NEVER_EXISTED.md
    phase: archive
`), 0644)

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	for _, issue := range result.Issues {
		if strings.Contains(issue, "artifact_registry") {
			t.Errorf("artifact_registry check is a no-op (historical ledger), got: %s", issue)
		}
	}
}

func TestRegistryDrift_ArtifactRegistry_NoDriftEvenWithMissingPaths(t *testing.T) {
	// artifact_registry is informational, not a gate. Even if a declared
	// path is missing, no drift must be reported.
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(`router: {}
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(`surfaces: {}
lane_validators: {}
`), 0644)

	// Declare an artifact that does NOT exist on disk.
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "artifacts.yaml"), []byte(`artifacts:
  missing_artifact:
    path: .ovav/artifacts/init/MISSING.md
    phase: init
`), 0644)

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	for _, issue := range result.Issues {
		if strings.Contains(issue, "artifact_registry") {
			t.Errorf("artifact_registry is no-op, must not flag missing path, got: %s", issue)
		}
	}
}

func TestRegistryDrift_ArtifactRegistry_DoesNotHardcodeCurrentHandoff(t *testing.T) {
	// The legacy hardcoded sample path was '.ovav/context/CURRENT_HANDOFF.md'.
	// That hardcoded sample is gone — the file is gitignored / runtime-
	// generated and was never declared in any registry. The check is a
	// no-op; no drift must be reported.
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ovav", "registry"), 0755)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "contract_manifest.yaml"), []byte(`contracts: {}
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "auto_triggers.yaml"), []byte(`router: {}
`), 0644)
	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "surface_validator_map.yaml"), []byte(`surfaces: {}
lane_validators: {}
`), 0644)

	os.WriteFile(filepath.Join(dir, ".ovav", "registry", "artifacts.yaml"), []byte(`artifacts: {}
`), 0644)

	v := NewRegistryDrift()
	result := v.Validate(context.Background(), dir)

	for _, issue := range result.Issues {
		if strings.Contains(issue, "CURRENT_HANDOFF.md") {
			t.Errorf("hardcoded CURRENT_HANDOFF.md sample must be removed, got: %s", issue)
		}
	}
}
