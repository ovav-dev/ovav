package permissions

import (
	"fmt"
	"regexp"
	"strings"
)

// BashRule represents a bash command governance rule.
type BashRule struct {
	Name           string
	Pattern        string
	Action         string // "allow", "deny"
	Category       string
	Note           string
	F0Integrations []string
	RateLimited    bool
}

// BashDecision represents the result of bash command evaluation.
type BashDecision struct {
	Allowed           bool
	Command           string
	MatchedRule       string
	Reason            string
	RequiresRateLimit bool
	F0ChecksPassed    bool
}

// BashCommandGovernor governs bash command execution.
type BashCommandGovernor struct {
	rules       []BashRule
	decisionLog []map[string]interface{}
	// CEOActive, when true, causes DENY rules to return ALLOW (CEO session bypass).
	// This mirrors the cmd/ovav/main.go:177 gate logic for shell-level enforcement.
	CEOActive bool
}

// NewBashCommandGovernor creates a new governor with default rules.
// CEOActive defaults to false; set Governor.CEOActive = true for CEO sessions.
func NewBashCommandGovernor() *BashCommandGovernor {
	return &BashCommandGovernor{
		rules: []BashRule{
			// ALLOW rules (9)
			{Name: "git_read", Pattern: `^git\s+(status|log|show)(\s+.*)?$`, Action: "allow", Category: "source_control_read", Note: "Read-only git operations"},
			{Name: "git_diff", Pattern: `^git\s+diff(\s+.*)?$`, Action: "allow", Category: "source_control_read", Note: "Git diff operations"},
			{Name: "git_branch_list", Pattern: `^git\s+branch(\s+(-a|-r|--list|--all|--remote))*$`, Action: "allow", Category: "source_control_read", Note: "Listing branches"},
			{Name: "ovav_tools", Pattern: `^python3\s+(-m\s+)?tools/(validators|harnesses|permissions|runtime)/.*`, Action: "allow", Category: "ovav_internal", Note: "OVAV internal tool execution"},
			{Name: "file_basic", Pattern: `^(ls|cat|echo|pwd|wc|head|tail|sort|uniq)(\s+.*)?$`, Action: "allow", Category: "filesystem_read", Note: "Basic read-only filesystem commands"},
			{Name: "python_inline", Pattern: `^python3\s+-c\s+.*`, Action: "allow", Category: "interpreted_execution", Note: "Inline Python execution", RateLimited: true},
			{Name: "gh_issue_read", Pattern: `^gh\s+issue\s+(list|view|status)(\s+.*)?$`, Action: "allow", Category: "github_read", Note: "GitHub issue read operations", F0Integrations: []string{"f0.4_network_guard"}, RateLimited: true},
			{Name: "test_runners", Pattern: `^(python3\s+-m\s+pytest|make\s+test|python3\s+setup\.py\s+test)(\s+.*)?$`, Action: "allow", Category: "testing", Note: "Test execution"},
			{Name: "governed_git_push", Pattern: `^python3\s+tools/github/ovav_git_push_gate\.py(\s+--confirm)?$`, Action: "allow", Category: "governed_git", Note: "Git push through governed gate", F0Integrations: []string{"f0.5_bootstrap_chain"}},
			// DENY rules (7)
			{Name: "git_push_force", Pattern: `^git\s+push\s+(-f|--force|--force-with-lease)`, Action: "deny", Category: "source_control_mutate", Note: "Force push permanently blocked"},
			{Name: "git_branch_delete", Pattern: `^git\s+branch\s+(-d|-D|--delete)\s+.*`, Action: "deny", Category: "source_control_mutate", Note: "Branch deletion requires user action"},
			{Name: "git_checkout_new_branch", Pattern: `^git\s+(checkout|switch)\s+(-b|-c)\s+.*`, Action: "deny", Category: "source_control_mutate", Note: "Branch creation must use OVAV harness"},
			{Name: "sudo_root", Pattern: `^sudo\s+.*`, Action: "deny", Category: "privilege_escalation", Note: "Root/sudo execution permanently blocked"},
			{Name: "package_install", Pattern: `^(pip|pip3|npm|yarn|apt|apt-get|yum|dnf|pacman|brew)\s+(install|uninstall)\s+.*`, Action: "deny", Category: "package_management", Note: "Package install blocked"},
			{Name: "gh_auth_token", Pattern: `^gh\s+auth\s+(token|login|logout|status|setup-git|credential)(\s+.*)?$`, Action: "deny", Category: "auth_management", Note: "GitHub auth reconfiguration blocked"},
			{Name: "network_unbounded", Pattern: `^(curl|wget|httpie|nc|ncat|telnet|ssh\s+-|scp|rsync|ftp)(\s+.*)?$`, Action: "deny", Category: "network_external", Note: "Unbounded network commands blocked", F0Integrations: []string{"f0.4_network_guard"}},
		},
	}
}

// Check evaluates a bash command against the rules.
func (g *BashCommandGovernor) Check(command, operator string) BashDecision {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(command)), " ")

	for _, rule := range g.rules {
		matched, _ := regexp.MatchString(rule.Pattern, normalized)
		if matched {
			allowed := rule.Action == "allow"
			f0Passed := true
			f0Failures := []string{}

			for _, integration := range rule.F0Integrations {
				if integration == "f0.4_network_guard" {
					if !g.checkNetworkGuard() {
						f0Passed = false
						f0Failures = append(f0Failures, "network_guard")
					}
				}
				if integration == "f0.5_bootstrap_chain" {
					if !g.checkBootstrapChain() {
						f0Passed = false
						f0Failures = append(f0Failures, "bootstrap_chain")
					}
				}
			}

			reason := rule.Note
			if len(f0Failures) > 0 {
				reason += fmt.Sprintf(" — F0 checks failed: %s", strings.Join(f0Failures, ", "))
				allowed = false
			}

			decision := BashDecision{
				Allowed:           allowed,
				Command:           normalized[:min(128, len(normalized))],
				MatchedRule:       rule.Name,
				Reason:            reason,
				RequiresRateLimit: rule.RateLimited && allowed,
				F0ChecksPassed:    f0Passed,
			}
			g.logDecision(command, decision)
			return decision
		}
	}

	// No rule matched — default deny
	decision := BashDecision{
		Allowed:     false,
		Command:     normalized[:min(128, len(normalized))],
		MatchedRule: "",
		Reason:      fmt.Sprintf("No allowlist rule matched for: %s", normalized[:min(80, len(normalized))]),
	}
	g.logDecision(command, decision)
	return decision
}

// CheckWithCEO evaluates a command like Check() but applies CEO session bypass.
// When g.CEOActive is true, any DENY rule matched is converted to ALLOW with a
// "CEO bypass active" note, mirroring the gate at cmd/ovav/main.go:177.
func (g *BashCommandGovernor) CheckWithCEO(command, operator string) BashDecision {
	decision := g.Check(command, operator)

	if g.CEOActive && !decision.Allowed && decision.MatchedRule != "" {
		// CEO bypass: convert DENY → ALLOW (only when a rule actually matched)
		decision.Allowed = true
		decision.Reason = "[CEO-BYPASS] " + decision.Reason
		g.logDecision(command, decision)
	}

	return decision
}

func (g *BashCommandGovernor) checkNetworkGuard() bool {
	// Placeholder: would check tools.security.network_guard
	return true
}

func (g *BashCommandGovernor) checkBootstrapChain() bool {
	// Placeholder: would check tools.security.bootstrap_verifier
	return false
}

func (g *BashCommandGovernor) logDecision(command string, decision BashDecision) {
	g.decisionLog = append(g.decisionLog, map[string]interface{}{
		"command":          command[:min(128, len(command))],
		"matched_rule":     decision.MatchedRule,
		"allowed":          decision.Allowed,
		"reason":           decision.Reason,
		"f0_checks_passed": decision.F0ChecksPassed,
	})
}

// GetSummary returns a summary of bash governance rules.
func (g *BashCommandGovernor) GetSummary() map[string]interface{} {
	rulesInfo := map[string]map[string]interface{}{}
	for _, rule := range g.rules {
		rulesInfo[rule.Name] = map[string]interface{}{
			"action":          rule.Action,
			"category":        rule.Category,
			"note":            rule.Note,
			"rate_limited":    rule.RateLimited,
			"f0_integrations": rule.F0Integrations,
		}
	}
	allowed := 0
	denied := 0
	for _, rule := range g.rules {
		if rule.Action == "allow" {
			allowed++
		} else {
			denied++
		}
	}
	return map[string]interface{}{
		"total_rules":     len(g.rules),
		"allowed":         allowed,
		"denied":          denied,
		"deny_by_default": true,
		"rules":           rulesInfo,
	}
}

// GetProtectedDenies returns all deny patterns.
func (g *BashCommandGovernor) GetProtectedDenies() []string {
	patterns := []string{}
	for _, rule := range g.rules {
		if rule.Action == "deny" {
			patterns = append(patterns, rule.Pattern)
		}
	}
	return patterns
}

// ClaimsGovernor governs production and profile claims.
type ClaimsGovernor struct {
	activeClaims  map[string]map[string]interface{}
	evidenceCache map[string]bool
}

// NewClaimsGovernor creates a new claims governor.
func NewClaimsGovernor() *ClaimsGovernor {
	return &ClaimsGovernor{
		activeClaims:  make(map[string]map[string]interface{}),
		evidenceCache: make(map[string]bool),
	}
}

// ClaimDecision represents the result of claim evaluation.
type ClaimDecision struct {
	Allowed          bool
	ClaimType        string
	Reason           string
	RequiredEvidence []string
	MissingEvidence  []string
}

// EvaluateClaim evaluates whether a claim can be made.
func (g *ClaimsGovernor) EvaluateClaim(claimType, operator string) ClaimDecision {
	requiredEvidence := map[string][]string{
		"production_ready":   {"bootstrap_chain_verified", "all_f0_validators_pass", "all_f1_validators_pass", "all_f2_validators_pass", "formal_verification_pass", "supply_chain_integrity_pass", "no_secrets_in_plaintext", "no_exfil_anomalies"},
		"global_ready":       {"production_ready_claim_active", "smoke_test_evidence", "multi_platform_verification", "network_hardening_pass", "rate_limiting_configured"},
		"new_public_profile": {"bootstrap_chain_verified", "profile_contract_complete", "operator_assigned", "scopes_defined", "permissions_audited"},
	}

	if (claimType == "production_ready" || claimType == "global_ready") && operator != "thavren" {
		return ClaimDecision{
			Allowed:   false,
			ClaimType: claimType,
			Reason:    fmt.Sprintf("Only Thavren can make %s claims", claimType),
		}
	}

	required := requiredEvidence[claimType]
	missing := []string{}
	for _, evidence := range required {
		if !g.checkEvidence(evidence) {
			missing = append(missing, evidence)
		}
	}

	allowed := len(missing) == 0
	reason := fmt.Sprintf("All %d evidence items verified", len(required))
	if !allowed {
		reason = fmt.Sprintf("Missing %d/%d evidence items", len(missing), len(required))
	}

	if allowed {
		g.activeClaims[claimType] = map[string]interface{}{
			"activated_at": "2026-06-24T00:00:00Z",
			"operator":     operator,
			"status":       "active",
		}
	}

	return ClaimDecision{
		Allowed:          allowed,
		ClaimType:        claimType,
		Reason:           reason,
		RequiredEvidence: required,
		MissingEvidence:  missing,
	}
}

func (g *ClaimsGovernor) checkEvidence(evidence string) bool {
	if cached, ok := g.evidenceCache[evidence]; ok {
		return cached
	}
	// Placeholder: would check actual evidence
	return false
}

// ConfigGovernor governs OpenCode configuration.
type ConfigGovernor struct {
	Root string
}

// NewConfigGovernor creates a new config governor.
func NewConfigGovernor(root string) *ConfigGovernor {
	return &ConfigGovernor{Root: root}
}

// ConfigCheck represents the result of config drift detection.
type ConfigCheck struct {
	Path         string
	Status       string // "clean", "drifted", "missing", "restored"
	ActionTaken  string
	DriftDetails []string
}

// CheckDrift checks all governed config files for drift.
func (g *ConfigGovernor) CheckDrift() []ConfigCheck {
	results := []ConfigCheck{}
	// Placeholder: would check actual config files
	return results
}

// NewStatesGovernor governs advanced permission states.
type NewStatesGovernor struct {
	rules       map[string]StateRule
	decisionLog []map[string]interface{}
}

// StateRule represents a new state governance rule.
type StateRule struct {
	Name            string
	Action          string
	Category        string
	Note            string
	F0Integrations  []string
	RequiresF0Green bool
}

// StateDecision represents the result of state evaluation.
type StateDecision struct {
	Allowed        bool
	State          string
	Reason         string
	F0ChecksPassed bool
	F0Failures     []string
}

// NewNewStatesGovernor creates a new states governor.
func NewNewStatesGovernor() *NewStatesGovernor {
	return &NewStatesGovernor{
		rules: map[string]StateRule{
			"delegated":          {Name: "delegated", Action: "allow", Category: "trust_chain", Note: "Chain of trust between roles", F0Integrations: []string{"f0.5_bootstrap_chain"}, RequiresF0Green: true},
			"adaptive":           {Name: "adaptive", Action: "allow", Category: "real_time_risk", Note: "Real-time alert for temporary elevation", F0Integrations: []string{"f0.3_runtime_integrity"}, RequiresF0Green: true},
			"consensus_required": {Name: "consensus_required", Action: "allow", Category: "authority_escalation", Note: "Invasive operations escalate to Thavren", F0Integrations: []string{"f0.5_bootstrap_chain"}, RequiresF0Green: true},
			"provenance_gated":   {Name: "provenance_gated", Action: "allow", Category: "supply_chain", Note: "Verify code origin before execution", F0Integrations: []string{"f0.1_supply_chain"}, RequiresF0Green: true},
			"rate_limited":       {Name: "rate_limited", Action: "allow", Category: "resource_protection", Note: "N operations per time window", F0Integrations: []string{"f0.4_network_guard"}, RequiresF0Green: true},
			"geofenced":          {Name: "geofenced", Action: "allow", Category: "compliance", Note: "Jurisdiction-based restriction", F0Integrations: []string{"f0.4_network_guard"}, RequiresF0Green: true},
			"revocable":          {Name: "revocable", Action: "allow", Category: "safety", Note: "Permission revocable at any moment", RequiresF0Green: true},
			"step_up_required":   {Name: "step_up_required", Action: "allow", Category: "escalated_auth", Note: "Additional verification for sensitive ops", F0Integrations: []string{"f0.5_bootstrap_chain"}, RequiresF0Green: true},
			"circuit_breaker":    {Name: "circuit_breaker", Action: "allow", Category: "fault_tolerance", Note: "Auto-block on cascading failures", F0Integrations: []string{"f0.3_runtime_integrity"}, RequiresF0Green: true},
			"idempotency_gated":  {Name: "idempotency_gated", Action: "allow", Category: "safety", Note: "Repeated operation executes once", RequiresF0Green: true},
			"cost_gated":         {Name: "cost_gated", Action: "allow", Category: "access_control", Note: "Tiered access by subscription", F0Integrations: []string{"f0.2_secrets_vault"}, RequiresF0Green: true},
			"emergent":           {Name: "emergent", Action: "allow", Category: "ai_governance", Note: "System-created rules with weekly validation", F0Integrations: []string{"f0.1_supply_chain", "f0.5_bootstrap_chain"}, RequiresF0Green: true},
			"inherited":          {Name: "inherited", Action: "deny", Category: "independence", Note: "Each role has independent permissions", RequiresF0Green: true},
			"canary_gated":       {Name: "canary_gated", Action: "deny", Category: "deployment", Note: "No gradual rollouts", RequiresF0Green: true},
		},
	}
}

// Check evaluates a new state against the rules.
func (g *NewStatesGovernor) Check(state string) StateDecision {
	rule, ok := g.rules[strings.ToLower(strings.ReplaceAll(state, " ", "_"))]
	if !ok {
		return StateDecision{
			Allowed: false,
			State:   state,
			Reason:  fmt.Sprintf("Unknown state — default deny: %s", state),
		}
	}

	f0Failures := []string{}
	if rule.RequiresF0Green {
		f0Failures = g.checkF0Baseline()
	}

	for _, integration := range rule.F0Integrations {
		if !g.checkIntegration(integration) {
			f0Failures = append(f0Failures, integration)
		}
	}

	allowed := rule.Action == "allow"
	if len(f0Failures) > 0 {
		allowed = false
	}

	reason := rule.Note
	if len(f0Failures) > 0 {
		reason += fmt.Sprintf(" — BLOCKED: F0 checks failed: %s", strings.Join(f0Failures, ", "))
	}

	return StateDecision{
		Allowed:        allowed,
		State:          state,
		Reason:         reason,
		F0ChecksPassed: len(f0Failures) == 0,
		F0Failures:     f0Failures,
	}
}

func (g *NewStatesGovernor) checkF0Baseline() []string {
	// Placeholder: would check F0 validators
	return []string{}
}

func (g *NewStatesGovernor) checkIntegration(integration string) bool {
	// Placeholder: would check specific F0 integration
	return true
}

// PluginGovernor governs plugin installation.
type PluginGovernor struct {
	pluginAuthority map[string]bool
	requiredGates   []string
	installLog      []map[string]interface{}
}

// NewPluginGovernor creates a new plugin governor.
func NewPluginGovernor() *PluginGovernor {
	return &PluginGovernor{
		pluginAuthority: map[string]bool{"thavren": true},
		requiredGates:   []string{"f0.1_supply_chain_verification", "f0.2_secrets_vault_check", "f0.5_bootstrap_verification"},
	}
}

// PluginDecision represents the result of plugin authorization.
type PluginDecision struct {
	Allowed       bool
	PluginType    string
	PluginName    string
	Reason        string
	RequiresGates []string
}

// AuthorizePlugin authorizes a plugin installation.
func (g *PluginGovernor) AuthorizePlugin(operator, pluginName, pluginType string) PluginDecision {
	if !g.pluginAuthority[operator] {
		return PluginDecision{
			Allowed:    false,
			PluginType: pluginType,
			PluginName: pluginName,
			Reason:     "Plugin management restricted to authorized operators",
		}
	}

	if pluginType == "mcp_server" {
		return PluginDecision{
			Allowed:    false,
			PluginType: pluginType,
			PluginName: pluginName,
			Reason:     "External MCP/A2A servers permanently blocked",
		}
	}

	return PluginDecision{
		Allowed:       true,
		PluginType:    pluginType,
		PluginName:    pluginName,
		Reason:        "Authorized — pending supply chain verification",
		RequiresGates: g.requiredGates,
	}
}

// SandboxGovernor governs sandbox operations.
type SandboxGovernor struct {
	operations   map[string]map[string]interface{}
	operationLog []map[string]interface{}
}

// NewSandboxGovernor creates a new sandbox governor.
func NewSandboxGovernor() *SandboxGovernor {
	return &SandboxGovernor{
		operations: map[string]map[string]interface{}{
			"live_probe":         {"allowed": true, "sandbox_required": true, "note": "Live probing operations"},
			"sandbox_runner":     {"allowed": false, "sandbox_required": true, "note": "DENY: test operations exclusively in sandbox", "permanent_deny": true},
			"write_gateway":      {"allowed": true, "sandbox_required": true, "note": "Write gateway operations"},
			"read_probe":         {"allowed": true, "sandbox_required": true, "note": "Read probe operations"},
			"redaction_policy":   {"allowed": true, "sandbox_required": false, "note": "Auto-detect + auto-delete secrets"},
			"privacy_classifier": {"allowed": true, "sandbox_required": false, "note": "Privacy classification"},
			"recall_filter":      {"allowed": true, "sandbox_required": false, "note": "Memory recall filter"},
			"continuity_legacy":  {"allowed": true, "sandbox_required": false, "note": "Legacy continuity operations"},
			"signal_simulator":   {"allowed": true, "sandbox_required": true, "note": "Signal simulation"},
			"candidate_preview":  {"allowed": true, "sandbox_required": false, "note": "Candidate preview"},
			"gateway_proof":      {"allowed": true, "sandbox_required": true, "note": "Proof + validation + conditional execution", "requires_proof": true},
		},
	}
}

// SandboxDecision represents the result of sandbox operation evaluation.
type SandboxDecision struct {
	Allowed           bool
	Operation         string
	Reason            string
	MustRunInSandbox  bool
	RedactionRequired bool
}

// CheckOperation evaluates a sandbox operation.
func (g *SandboxGovernor) CheckOperation(operation string, inSandbox bool) SandboxDecision {
	policy, ok := g.operations[operation]
	if !ok {
		return SandboxDecision{
			Allowed:   false,
			Operation: operation,
			Reason:    fmt.Sprintf("Unknown sandbox operation: %s", operation),
		}
	}

	if permanentDeny, _ := policy["permanent_deny"].(bool); permanentDeny {
		return SandboxDecision{
			Allowed:   false,
			Operation: operation,
			Reason:    "Permanently denied — tests must run exclusively in sandbox",
		}
	}

	if sandboxRequired, _ := policy["sandbox_required"].(bool); sandboxRequired && !inSandbox {
		return SandboxDecision{
			Allowed:   false,
			Operation: operation,
			Reason:    fmt.Sprintf("Operation '%s' must run in sandbox context", operation),
		}
	}

	allowed, _ := policy["allowed"].(bool)
	note, _ := policy["note"].(string)
	sandboxReq, _ := policy["sandbox_required"].(bool)

	return SandboxDecision{
		Allowed:          allowed,
		Operation:        operation,
		Reason:           note,
		MustRunInSandbox: sandboxReq,
	}
}

// SystemPathGovernor governs system path access.
type SystemPathGovernor struct {
	home string
}

// NewSystemPathGovernor creates a new path governor.
func NewSystemPathGovernor(home string) *SystemPathGovernor {
	return &SystemPathGovernor{home: home}
}

// PathDecision represents the result of path access evaluation.
type PathDecision struct {
	Allowed      bool
	Path         string
	Operation    string
	Reason       string
	RequiresGate string
}

// Check evaluates path access.
func (g *SystemPathGovernor) Check(targetPath, operation string) PathDecision {
	systemPaths := map[string]map[string]string{
		"/etc":  {"read": "allow", "write": "requires_security_gate", "execute": "deny"},
		"/boot": {"read": "allow", "write": "requires_atomic_backup", "execute": "deny"},
		"/sys":  {"read": "allow", "write": "requires_capability_grant", "execute": "deny"},
		"/proc": {"read": "allow", "write": "deny", "execute": "deny"},
		"/dev":  {"read": "allow", "write": "requires_explicit_auth", "execute": "deny"},
	}

	for sysPath, policy := range systemPaths {
		if strings.HasPrefix(targetPath, sysPath) {
			allowedOp := policy[operation]
			if allowedOp == "allow" {
				return PathDecision{Allowed: true, Path: targetPath, Operation: operation, Reason: "Allowed by system path policy"}
			}
			if allowedOp == "deny" {
				return PathDecision{Allowed: false, Path: targetPath, Operation: operation, Reason: "Denied by system path policy"}
			}
			return PathDecision{Allowed: false, Path: targetPath, Operation: operation, Reason: "Operation requires gate", RequiresGate: allowedOp}
		}
	}

	return PathDecision{Allowed: false, Path: targetPath, Operation: operation, Reason: "Path outside governed scopes"}
}
