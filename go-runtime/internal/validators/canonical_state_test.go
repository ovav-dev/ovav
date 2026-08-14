package validators

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCanonicalCapsReflectsRuntimeEvidence(t *testing.T) {
	root, err := findOVAVRoot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".ovav", "plan", "caps.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var caps struct {
		GitHead     string `yaml:"git_head"`
		Status      string `yaml:"status"`
		NextPhase   string `yaml:"next_phase"`
		LayerStatus string `yaml:"layer_status"`
		Product     struct {
			Posture string `yaml:"posture"`
		} `yaml:"product"`
		Governance struct {
			GateMode   string `yaml:"gate_mode"`
			Validators struct {
				Registered int `yaml:"registered"`
				Passed     int `yaml:"passed"`
				Warned     int `yaml:"warned"`
				Failed     int `yaml:"failed"`
			} `yaml:"validators"`
		} `yaml:"governance"`
		Subsystems map[string]struct {
			State string `yaml:"state"`
		} `yaml:"subsystem_matrix"`
	}
	if err := yaml.Unmarshal(data, &caps); err != nil {
		t.Fatal(err)
	}
	if caps.GitHead != "git HEAD is the canonical temporal authority; resolve dynamically" {
		t.Fatalf("git_head must express temporal authority without a self-staling hash: %q", caps.GitHead)
	}
	if caps.Status != "launch_verification_blocked" || caps.Governance.GateMode != "blocked" {
		t.Fatalf("unexpected candidate state: status=%q gate=%q", caps.Status, caps.Governance.GateMode)
	}
	if caps.Governance.Validators.Registered != len(DefaultRegistry().All()) {
		t.Fatalf("caps validator count=%d, registry=%d", caps.Governance.Validators.Registered, len(DefaultRegistry().All()))
	}
	if got := len(DefaultRegistry().All()); got != 73 {
		t.Fatalf("default validator registry contains %d validators, want 73", got)
	}
	if caps.Governance.Validators.Passed != 63 || caps.Governance.Validators.Warned != 8 || caps.Governance.Validators.Failed != 0 {
		t.Fatalf("unexpected latest developer validation: %+v", caps.Governance.Validators)
	}
	if caps.NextPhase != "Governed candidate commit, then generate baseline, commit baseline attestation, and run gate validation" {
		t.Fatalf("unexpected dependency-first next work: %q", caps.NextPhase)
	}
	for subsystem, state := range map[string]string{
		"session_capsule":           "removed_obsolete",
		"knowledge_compiler_python": "disconnected",
		"snv_advanced_learning":     "registry_only_disconnected",
		"evaluation_pipeline":       "absent",
		"trigger_engine":            "partial",
		"runtime_integrity":         "pending",
	} {
		if got := caps.Subsystems[subsystem].State; got != state {
			t.Errorf("subsystem %s state=%q, want %q", subsystem, got, state)
		}
	}
	current := strings.ToLower(caps.Status + " " + caps.LayerStatus + " " + caps.Product.Posture)
	for _, stale := range []string{"all_11_complete", "fully integrated", "production —", "global ready"} {
		if strings.Contains(current, stale) {
			t.Errorf("active authority contains stale claim %q", stale)
		}
	}
}

func TestCapabilityAndTriggerRegistriesReflectDisconnectedState(t *testing.T) {
	root, err := findOVAVRoot()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".ovav", "registry", "capability_registry.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var registry struct {
		Capabilities map[string]struct {
			State    string `yaml:"state"`
			Evidence string `yaml:"evidence"`
		} `yaml:"capabilities"`
	}
	if err := yaml.Unmarshal(data, &registry); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"snv", "nerve_bus", "knowledge_graph", "hebbian_weights", "temporal_cortex", "pattern_learner"} {
		capability := registry.Capabilities[id]
		if capability.State != "registry_only_disconnected" || capability.Evidence == "" {
			t.Errorf("capability %s not reconciled: %+v", id, capability)
		}
	}
	if capability := registry.Capabilities["knowledge_compiler"]; capability.State != "disconnected" || capability.Evidence == "" {
		t.Errorf("knowledge_compiler not reconciled: %+v", capability)
	}

	scoreData, err := os.ReadFile(filepath.Join(root, ".ovav", "registry", "capability_scores.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var scores struct {
		CurrentStatus             string `yaml:"current_status"`
		RuntimeEvaluationPipeline string `yaml:"runtime_evaluation_pipeline"`
	}
	if err := yaml.Unmarshal(scoreData, &scores); err != nil {
		t.Fatal(err)
	}
	if scores.CurrentStatus != "stale_historical_snapshot" || scores.RuntimeEvaluationPipeline != "absent" {
		t.Fatalf("capability scores must not imply a live evaluation pipeline: %+v", scores)
	}

	triggerData, err := os.ReadFile(filepath.Join(root, ".ovav", "registry", "auto_triggers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var triggers struct {
		Status         string              `yaml:"status"`
		ExecutionScope string              `yaml:"execution_scope"`
		Router         map[string][]string `yaml:"router"`
	}
	if err := yaml.Unmarshal(triggerData, &triggers); err != nil {
		t.Fatal(err)
	}
	if triggers.Status != "partial" || triggers.ExecutionScope != "git_hook_stages_only" {
		t.Fatalf("trigger state is not partial/git-stage-only: %+v", triggers)
	}
	if len(triggers.Router) != 2 || len(triggers.Router["before_git_stage"]) == 0 || len(triggers.Router["after_git_stage"]) == 0 {
		t.Fatalf("connected trigger router must contain only git stages: %+v", triggers.Router)
	}
}
