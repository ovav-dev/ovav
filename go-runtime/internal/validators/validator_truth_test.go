package validators

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentRuntimeEnforcementUsesGoNativeTruth(t *testing.T) {
	files := goRuntimeTruthFixture()
	files["go-runtime/internal/agents/observability.go"] = observabilityTruthFixture
	root := tempRepoWithFiles(t, files)

	result := NewAgentRuntimeEnforcement().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("expected complete Go runtime enforcement, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, "tools/agent_runtime")
}

func TestSquadNormalizationUsesGoRuntimeAndRegistries(t *testing.T) {
	files := goRuntimeTruthFixture()
	files["go-runtime/internal/agents/observability.go"] = observabilityTruthFixture
	for path, content := range map[string]string{
		".ovav/registry/squads.yaml":                        "systems_architecture_squad\nresearch_intelligence_squad\n",
		".ovav/registry/operators.yaml":                     "thavren\neidren\n",
		".ovav/registry/service_profiles.yaml":              "systems_architecture_squad\nresearch_intelligence_squad\n",
		".ovav/registry/delegation_rules.yaml":              "lead_only skill_only focused_squad full_squad critical_squad\ndo_not_delegate_when\nsquad_usage\nService Area Router\nDelegation Router\nContext Gateway\nTool Gateway\nDelivery Contract\nObservability Trace\n",
		".ovav/service_areas/shared/delegation_policy.yaml": "lead_only skill_only focused_squad full_squad critical_squad\nsquad_usage\n",
		".ovav/service_areas/shared/squad_roles.yaml":       "Service Area Router\nDelegation Router\nContext Gateway\nTool Gateway\nDelivery Contract\nObservability Trace\n",
	} {
		files[path] = content
	}
	root := tempRepoWithFiles(t, files)

	result := NewSquadNormalization().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("expected complete squad normalization, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, "tools/agent_runtime")
}

const observabilityTruthFixture = `package agents
type TraceID string
type TraceEvent struct{}
type TraceSink interface { Append() error }
func NewTraceEvent() {}
func (TraceEvent) Validate() error { return nil }
func (TraceEvent) MarshalJSON() ([]byte, error) { return nil, nil }
func NewFileTraceSink() {}
func NewMemoryTraceSink() {}
func RouteRequestWithTrace() {}
`

func TestRegoPoliciesUsesGoEngineAndRealPolicies(t *testing.T) {
	root := tempRepoWithFiles(t, map[string]string{
		"go-runtime/internal/permissions/rego_engine.go": "package permissions\ntype RegoEngine struct{}\nfunc NewRegoEngine(string) *RegoEngine { return &RegoEngine{} }\nfunc (e *RegoEngine) LoadPolicies() error { return nil }\nfunc (e *RegoEngine) Evaluate(string, map[string]interface{}) bool { return false }\nfunc (e *RegoEngine) TestPolicy([]map[string]interface{}) map[string]interface{} { return nil }\nfunc BuiltinTests() []map[string]interface{} { return nil }\n",
		"go-runtime/internal/permissions/simulate.go":    "package permissions\nfunc Simulate() error { return nil }\n",
		"go-runtime/internal/permissions/verify.go":      "package permissions\nfunc VerifyPermissionAuthority(string) bool { return true }\n",
		".ovav/registry/rego_policies/security.rego":     "package ovav.security\ndefault allow = false\ndeny_bash { true }\nallow { false }\n",
	})

	result := NewRegoPolicies().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("expected Go-native Rego validation to pass, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, "rego_engine.py")
}

func TestF1ArchitectureUsesBehavioralAPIs(t *testing.T) {
	root := tempRepoWithFiles(t, map[string]string{
		".ovav/policy/permission_authority.json": `{
			"schema_version":"ovav.permission_authority.v3",
			"authority":{"owner":"OVAV"},
			"governor":{"owner":"Thavren"},
			"materialized_targets":["opencode.json"]
		}`,
		".ovav/registry/rego_policies/a.rego": "package a\ndefault allow = false\nallow { true }\ndeny_bash { true }\n",
		".ovav/registry/rego_policies/b.rego": "package b\nallow_read { true }\n",
		".ovav/registry/rego_policies/c.rego": "package c\ndeny_force_push { true }\n",
		"docs/research/F1_EAL7_GUIDANCE.md":   "guidance\n",
	})

	result := NewF1Architecture().Validate(context.Background(), root)
	if result.Status != "warn" || !strings.Contains(result.Message, "INTENTIONALLY_GATED/PARTIAL") {
		t.Fatalf("expected F1 startup enforcement to be intentionally gated, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, ".py")
}

func TestSSHProfileUsesGoWorkstationInstallBoundary(t *testing.T) {
	root := tempRepoWithFiles(t, sshProfileTruthFixture())

	result := NewSSHProfile().Validate(context.Background(), root)
	if result.Status != "warn" {
		t.Fatalf("expected Go-native workstation projection to remain intentionally gated, got %s: %v", result.Status, result.Issues)
	}
	assertNoIssueContains(t, result.Issues, "ovav_workstation_access.py")
}

func TestBehavioralDirectivesValidateCanonicalSourceNotProjections(t *testing.T) {
	root := tempRepoWithFiles(t, map[string]string{
		"go-runtime/internal/convert/convert.go":            "package convert\nconst ovavIdentityGuardFmt = `OVAV_IDENTITY_GUARD`\nfunc WriteIdentityGuard() {}\n",
		"go-runtime/internal/convert/opencode.go":           "package convert\nfunc convert() { WriteIdentityGuard() }\n",
		".ovav/source/agents/areas/platform-engineering.md": "## Scope del área\nNo cubre: research → Research Intelligence\n",
		".ovav/source/agents/leads/thavren.md":              "## Limitaciones Explícitas\nRedirigir a Eidren\n🚫 HARD STOP — Fuera de mi área\n",
		"clients/opencode/agents/generated-without-seal.md": "generated projection intentionally ignored\n",
	})

	result := NewBehavioralDirectives().Validate(context.Background(), root)
	if result.Status != "pass" {
		t.Fatalf("expected canonical source validation to pass without duplicate directives, got %s: %v", result.Status, result.Issues)
	}
}

func TestHostConfigDriftGlobalOpenCodeSchemaBoundary(t *testing.T) {
	t.Run("schema only is benign", func(t *testing.T) {
		for _, name := range []string{"opencode.json", "opencode.jsonc"} {
			t.Run(name, func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				writeTestFile(t, home, ".config/opencode/"+name, `{"$schema":"https://opencode.ai/config.json"}`)
				result := NewHostConfigDrift().Validate(context.Background(), t.TempDir())
				if result.Status != "pass" {
					t.Fatalf("expected schema-only global config to pass, got %s: %v", result.Status, result.Issues)
				}
			})
		}
	})

	t.Run("global intelligence is intrusion", func(t *testing.T) {
		for _, key := range []string{"agent", "permission", "provider"} {
			t.Run(key, func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				writeTestFile(t, home, ".config/opencode/opencode.json", `{"$schema":"https://opencode.ai/config.json","`+key+`":{}}`)
				result := NewHostConfigDrift().Validate(context.Background(), t.TempDir())
				if result.Status != "fail" {
					t.Fatalf("expected global %s config to fail, got %s: %v", key, result.Status, result.Issues)
				}
			})
		}
	})
}

func goRuntimeTruthFixture() map[string]string {
	return map[string]string{
		"go-runtime/internal/agents/service_area.go":            "package agents\nfunc RouteRequest() {}\nfunc ServiceAreaForAgent() {}\nfunc InternalRepoAccessDeniedByDefault() {}\n",
		"go-runtime/internal/agents/context.go":                 "package agents\nfunc RequestContext() {}\nfunc ResearchNoRepoDefault() {}\nfunc DecideContext() {}\n",
		"go-runtime/internal/validators/context_firewall_v2.go": "package validators\ntype ContextFirewallV2 struct{}\nfunc (v *ContextFirewallV2) Validate() {}\nfunc containsSuspiciousPattern() {}\n",
		"go-runtime/internal/agents/tool_gateway.go":            "package agents\nfunc RequestTool() {}\nfunc RequiresPermission() {}\nfunc Decision() {}\n",
		"go-runtime/internal/agents/handoff.go":                 "package agents\nfunc CreateHandoff() {}\nfunc EvaluateHandoff() {}\nfunc DeniedContext() {}\n",
		"go-runtime/internal/agents/delegation.go":              "package agents\nfunc DecideDelegation() {}\nfunc DelegationModeForSquad() {}\nfunc CriticalSquad() {}\n",
		".ovav/service_areas/shared/observability_policy.yaml":  "observability_policy:\n  non_trivial_action_requires_trace: true\n  trace_fields:\n    - trace_id\n    - timestamp\n",
	}
}

func sshProfileTruthFixture() map[string]string {
	return map[string]string{
		"docs/workstation/OVAV_THAVREN_SSH_PROFILE.md":          "~/.ssh/config no autoriza cambios globales",
		"docs/workstation/OVAV_WORKSTATION_ACCESS_PROFILE.md":   "No es un almacén de secretos",
		"docs/workstation/OVAV_THAVREN_SSH_INSTALL_PLAN.md":     "No aplica cambios reales todavía",
		"config/ssh/ovav-thavren.ssh.config.example":            "Host github-ovav-thavren\nHostName github.com\nUser git\nIdentityFile ~/.ssh/ovav_thavren_ed25519\nIdentitiesOnly yes\nAddKeysToAgent yes",
		"config/fish/ovav-thavren-ssh-agent.fish.example":       "set -gx OVAV_SSH_HOST_ALIAS github-ovav-thavren\nset -gx OVAV_SSH_KEY_PATH x\nset -gx OVAV_SSH_KEY_COMMENT ovav-thavren-github\nset -gx OVAV_SSH_AGENT_LIFETIME 24h\nfunction ovav_ssh_unlock\nfunction ovav_ssh_status\nssh-add -t \"$OVAV_SSH_AGENT_LIFETIME\"\nIntentionally not auto-unlocking on shell startup",
		"config/workstation/ovav-thavren-ssh-profile.yaml":      "profile_id: ovav-thavren-ssh-profile\ntransport: ssh\npassphrase_required: true\nhost_alias: github-ovav-thavren\nremote_url_shape: git@github-ovav-thavren:ORG/REPO.git\nshell_profile_template: config/fish/ovav-thavren-ssh-agent.fish.example\nunlock_command: ovav_ssh_unlock\nexpected_prompt_behavior: ask_passphrase_once_per_24h_when_key_is_missing_or_expired\nmust_not_store: []\nblocked_until_explicit_install_approval: []\n",
		"config/workstation/ovav-thavren-ssh-install-plan.yaml": "plan_id: ovav-thavren-ssh-install-plan\nstatus: source_local_dry_run_only\nssh_config_fragment: ~/.ssh/config.d/ovav-thavren.conf\nfish_agent_helper: ~/.config/fish/conf.d/ovav-thavren-ssh-agent.fish\nrollback: restore_previous_git_remote_url\nboundary: real_install_remains_blocked_until_explicit_install_gate\n",
		"go-runtime/internal/install/install.go":                "package install\n// Pipeline: plan manifest safety boundaries backup apply verify report.\nvar PermanentlyBlockedSurfaces = []string{\"user_home_config\", \"opencode_global_config\"}\nconst ModeDryRun = \"dry-run\"\n",
		"go-runtime/internal/tailor/apply.go":                   "package tailor\nfunc ApplySelection() {}\nfunc InstallConfirmationRows() {}\n// preview backup verify\n",
	}
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertNoIssueContains(t *testing.T, issues []string, forbidden string) {
	t.Helper()
	for _, issue := range issues {
		if strings.Contains(issue, forbidden) {
			t.Fatalf("unexpected stale issue %q in %v", forbidden, issues)
		}
	}
}
