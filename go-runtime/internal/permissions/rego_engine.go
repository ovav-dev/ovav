package permissions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PolicyRule represents a parsed policy rule.
type PolicyRule struct {
	Name       string
	Package    string
	RuleType   string // "allow", "deny", "default"
	Conditions []map[string]interface{}
	Raw        string
}

// PolicyDecision represents the result of policy evaluation.
type PolicyDecision struct {
	Allowed      bool
	Reason       string
	MatchedRules []string
	DeniedBy     []string
	Warnings     []string
	Metadata     map[string]interface{}
}

// RegoEngine is an OPA/Rego-style policy evaluation engine.
type RegoEngine struct {
	policiesDir  string
	packages     map[string][]PolicyRule
	denyRules    []PolicyRule
	allowRules   []PolicyRule
	defaultAllow bool
}

// NewRegoEngine creates a new RegoEngine with the given policies directory.
func NewRegoEngine(policiesDir string) *RegoEngine {
	return &RegoEngine{
		policiesDir:  policiesDir,
		packages:     make(map[string][]PolicyRule),
		denyRules:    []PolicyRule{},
		allowRules:   []PolicyRule{},
		defaultAllow: false,
	}
}

// LoadPolicies loads all .rego policy files from the policies directory.
func (e *RegoEngine) LoadPolicies() error {
	e.packages = make(map[string][]PolicyRule)
	e.denyRules = []PolicyRule{}
	e.allowRules = []PolicyRule{}
	seenRules := make(map[string]bool)

	files, err := filepath.Glob(filepath.Join(e.policiesDir, "*.rego"))
	if err != nil {
		return fmt.Errorf("failed to glob policies: %w", err)
	}

	for _, file := range files {
		if err := e.parsePolicyFile(file, seenRules); err != nil {
			return fmt.Errorf("failed to parse %s: %w", file, err)
		}
	}
	return nil
}

func (e *RegoEngine) parsePolicyFile(filepath string, seenRules map[string]bool) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	content := string(data)
	currentPackage := "default"
	var rules []PolicyRule

	for _, line := range strings.Split(content, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			continue
		}

		// Package declaration
		if strings.HasPrefix(stripped, "package ") {
			currentPackage = strings.TrimPrefix(stripped, "package ")
			continue
		}

		// Default allow/deny
		if strings.Contains(stripped, "default allow") {
			if strings.Contains(stripped, "= false") {
				e.defaultAllow = false
			} else if strings.Contains(stripped, "= true") {
				e.defaultAllow = true
			}
			continue
		}

		// Rule detection
		if e.isRuleStart(stripped) {
			name := e.extractRuleName(stripped)
			ruleType := e.classifyRule(name, stripped)
			if ruleType == "info" {
				continue
			}
			ruleKey := fmt.Sprintf("%s.%s", currentPackage, name)
			if seenRules[ruleKey] {
				continue
			}
			seenRules[ruleKey] = true
			rule := PolicyRule{
				Name:     name,
				Package:  currentPackage,
				RuleType: ruleType,
				Raw:      stripped,
			}
			rules = append(rules, rule)
			if ruleType == "deny" {
				e.denyRules = append(e.denyRules, rule)
			} else if ruleType == "allow" {
				e.allowRules = append(e.allowRules, rule)
			}
		}
	}

	if _, ok := e.packages[currentPackage]; !ok {
		e.packages[currentPackage] = []PolicyRule{}
	}
	e.packages[currentPackage] = append(e.packages[currentPackage], rules...)
	return nil
}

var ruleStartPattern = regexp.MustCompile(`^(deny_|denied\b|allow\b|allow_|default\s+allow|require_|max_|context_budget|operators\s*:=|resources\s*:=|protected_branch\s*:=)`)
var ruleNamePattern = regexp.MustCompile(`^(\w+)`)

func (e *RegoEngine) isRuleStart(line string) bool {
	return ruleStartPattern.MatchString(line)
}

func (e *RegoEngine) extractRuleName(line string) string {
	matches := ruleNamePattern.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

func (e *RegoEngine) classifyRule(name, line string) string {
	if strings.HasPrefix(name, "deny_") || name == "denied" {
		return "deny"
	}
	if name == "allow" || strings.HasPrefix(name, "allow_") {
		return "allow"
	}
	if strings.Contains(line, "default allow") && strings.Contains(line, "false") {
		return "default_deny"
	}
	if strings.Contains(line, "default allow") && strings.Contains(line, "true") {
		return "default_allow"
	}
	return "info"
}

// Evaluate evaluates an action against loaded policies.
func (e *RegoEngine) Evaluate(action string, facts map[string]interface{}) PolicyDecision {
	if len(e.packages) == 0 {
		_ = e.LoadPolicies()
	}

	// Set defaults
	if _, ok := facts["action"]; !ok {
		facts["action"] = action
	}
	if _, ok := facts["operator"]; !ok {
		facts["operator"] = "unknown"
	}
	if _, ok := facts["scope"]; !ok {
		facts["scope"] = "unknown"
	}
	if _, ok := facts["bootstrap_valid"]; !ok {
		facts["bootstrap_valid"] = false
	}

	matchedAllows := []string{}
	deniedBy := []string{}
	warnings := []string{}

	// 1. Check deny rules first
	for _, rule := range e.denyRules {
		if e.evaluateDenyRule(rule, facts) {
			deniedBy = append(deniedBy, fmt.Sprintf("%s.%s", rule.Package, rule.Name))
		}
	}

	// 2. Check allow rules
	for _, rule := range e.allowRules {
		if e.evaluateAllowRule(rule, facts) {
			matchedAllows = append(matchedAllows, fmt.Sprintf("%s.%s", rule.Package, rule.Name))
		}
	}

	// 3. Composite decision
	var allowed bool
	var reason string
	if len(deniedBy) > 0 {
		allowed = false
		reason = fmt.Sprintf("Denied by: %s", strings.Join(deniedBy[:min(3, len(deniedBy))], ", "))
	} else if len(matchedAllows) > 0 {
		allowed = true
		reason = fmt.Sprintf("Allowed by: %s", strings.Join(matchedAllows[:min(3, len(matchedAllows))], ", "))
	} else if explicitGrant, ok := facts["explicit_grant"].(bool); ok && explicitGrant {
		allowed = true
		reason = "Explicit grant"
	} else {
		allowed = e.defaultAllow
		if allowed {
			reason = "Default policy"
		} else {
			reason = "No matching allow rule — default deny"
		}
	}

	// 4. Check bootstrap integrity
	if bootstrapValid, ok := facts["bootstrap_valid"].(bool); !ok || !bootstrapValid {
		if action != "bootstrap_check" && action != "read" {
			allowed = false
			deniedBy = append(deniedBy, "ovav.security.deny_without_bootstrap")
			reason = "Bootstrap chain not verified"
		}
	}

	return PolicyDecision{
		Allowed:      allowed,
		Reason:       reason,
		MatchedRules: matchedAllows,
		DeniedBy:     deniedBy,
		Warnings:     warnings,
		Metadata: map[string]interface{}{
			"action":          action,
			"operator":        facts["operator"],
			"packages_loaded": len(e.packages),
			"rules_evaluated": len(e.denyRules) + len(e.allowRules),
		},
	}
}

func (e *RegoEngine) evaluateDenyRule(rule PolicyRule, facts map[string]interface{}) bool {
	name := rule.Name

	// Bash deny rules
	if name == "deny_bash" {
		dangerous := []string{"sudo", "pip install", "npm install", "apt install", "gh auth token", "gh auth login", "gh release"}
		cmd, _ := facts["command"].(string)
		for _, d := range dangerous {
			if strings.Contains(cmd, d) {
				return true
			}
		}
		return false
	}

	if name == "deny_path_traversal" {
		path, _ := facts["path"].(string)
		return strings.Contains(path, "..")
	}

	if name == "deny_system_path_write" {
		path, _ := facts["path"].(string)
		ws, _ := facts["workspace_root"].(string)
		if path == "" {
			return false
		}
		return !strings.HasPrefix(path, ws) && !strings.HasPrefix(path, "/tmp/opencode/")
	}

	if name == "deny_force_push" {
		flags, _ := facts["flags"].(string)
		return strings.Contains(flags, "--force") || strings.Contains(flags, "-f")
	}

	if name == "deny_force_delete_branch" {
		flag, _ := facts["flag"].(string)
		return flag == "-D"
	}

	if name == "deny_protected_branch_push" {
		protectedBranches := []string{"main", "master", "develop", "production"}
		branch, _ := facts["branch"].(string)
		for _, pb := range protectedBranches {
			if branch == pb {
				return true
			}
		}
		return false
	}

	if name == "deny_plugin_install" {
		action, _ := facts["action"].(string)
		operator, _ := facts["operator"].(string)
		return action == "install_plugin" && operator != "thavren"
	}

	if name == "deny_extension_install" {
		action, _ := facts["action"].(string)
		operator, _ := facts["operator"].(string)
		return action == "install_extension" && operator != "thavren"
	}

	if name == "deny_external_mcp" {
		action, _ := facts["action"].(string)
		source, _ := facts["source"].(string)
		return action == "register_mcp_server" && source != "ovav_internal"
	}

	if name == "deny_external_network" {
		action, _ := facts["action"].(string)
		if action != "external_request" {
			return false
		}
		url, _ := facts["url"].(string)
		allowed := map[string]bool{
			"github.com": true, "api.github.com": true, "pypi.org": true,
			"files.pythonhosted.org": true, "docs.python.org": true,
			"arxiv.org": true, "scholar.google.com": true, "ovav.dev": true,
		}
		domain := e.extractDomain(url)
		if domain == "" {
			return true
		}
		if allowed[domain] {
			return false
		}
		for d := range allowed {
			if strings.HasSuffix(domain, "."+d) {
				return false
			}
		}
		return true
	}

	if name == "deny_secret_in_output" {
		secrets := []string{"ghp_", "github_pat_", "gho_", "AKIA", "sk-", "sk-ant-", "-----BEGIN " + "RSA PRIVATE KEY-----"}
		content, _ := facts["content"].(string)
		for _, s := range secrets {
			if strings.Contains(content, s) {
				return true
			}
		}
		return false
	}

	if name == "deny_without_bootstrap" {
		bootstrapValid, _ := facts["bootstrap_valid"].(bool)
		return !bootstrapValid
	}

	if name == "deny_rate_limit" {
		requestCount, _ := facts["request_count"].(int)
		rateLimit, _ := facts["rate_limit"].(int)
		if rateLimit == 0 {
			rateLimit = 100
		}
		return requestCount > rateLimit
	}

	return false
}

func (e *RegoEngine) evaluateAllowRule(rule PolicyRule, facts map[string]interface{}) bool {
	name := rule.Name

	if name == "allow" {
		op, _ := facts["operator"].(string)
		scope, _ := facts["scope"].(string)
		action, _ := facts["action"].(string)
		if op == "thavren" && scope == "repo_local" {
			return true
		}
		if op == "thavren" && action == "external_request" {
			return true
		}
		if op == "eidren" && action == "research" {
			return true
		}
		if explicitGrant, ok := facts["explicit_grant"].(bool); ok && explicitGrant {
			return true
		}
		return false
	}

	if name == "allow_operator_action" {
		op, _ := facts["operator"].(string)
		scope, _ := facts["scope"].(string)
		action, _ := facts["action"].(string)
		if op == "thavren" && scope == "repo_local" {
			return true
		}
		if op == "eidren" && action == "research" {
			return true
		}
		return false
	}

	if name == "allow_operator_scope" {
		op, _ := facts["operator"].(string)
		scope, _ := facts["scope"].(string)
		allowedScopes := map[string]map[string]bool{
			"thavren": {"repo_local": true, "global_diagnostic": true, "install_sandbox": true},
			"eidren":  {"repo_local": true, "research_external": true},
		}
		if scopes, ok := allowedScopes[op]; ok {
			return scopes[scope]
		}
		return false
	}

	return false
}

func (e *RegoEngine) extractDomain(url string) string {
	if strings.Contains(url, "://") {
		parts := strings.SplitN(url, "://", 2)
		if len(parts) > 1 {
			url = parts[1]
		}
	}
	parts := strings.SplitN(url, "/", 2)
	if len(parts) > 0 {
		url = parts[0]
	}
	parts = strings.SplitN(url, ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

// TestPolicy runs a suite of test cases against loaded policies.
func (e *RegoEngine) TestPolicy(testCases []map[string]interface{}) map[string]interface{} {
	results := []map[string]interface{}{}
	passed := 0
	failed := 0

	for _, tc := range testCases {
		action, _ := tc["action"].(string)
		facts, _ := tc["facts"].(map[string]interface{})
		if facts == nil {
			facts = map[string]interface{}{}
		}
		expect, _ := tc["expect"].(bool)

		decision := e.Evaluate(action, facts)
		ok := decision.Allowed == expect
		if ok {
			passed++
		} else {
			failed++
		}
		results = append(results, map[string]interface{}{
			"name":      tc["name"],
			"passed":    ok,
			"expected":  expect,
			"actual":    decision.Allowed,
			"reason":    decision.Reason,
			"denied_by": decision.DeniedBy,
		})
	}

	return map[string]interface{}{
		"total":   len(testCases),
		"passed":  passed,
		"failed":  failed,
		"results": results,
	}
}

// GetPolicySummary returns a summary of loaded policies.
func (e *RegoEngine) GetPolicySummary() map[string]interface{} {
	packageNames := []string{}
	for k := range e.packages {
		packageNames = append(packageNames, k)
	}
	denyRuleNames := []string{}
	for _, r := range e.denyRules {
		denyRuleNames = append(denyRuleNames, r.Name)
	}
	allowRuleNames := []string{}
	for _, r := range e.allowRules {
		allowRuleNames = append(allowRuleNames, r.Name)
	}

	return map[string]interface{}{
		"packages":         packageNames,
		"total_rules":      len(e.denyRules) + len(e.allowRules),
		"deny_rules":       len(e.denyRules),
		"allow_rules":      len(e.allowRules),
		"default_allow":    e.defaultAllow,
		"deny_rule_names":  denyRuleNames,
		"allow_rule_names": allowRuleNames,
	}
}

// BuiltinTests returns the built-in test suite.
func BuiltinTests() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "sudo blocked", "action": "bash",
			"facts":  map[string]interface{}{"command": "sudo rm -rf /", "operator": "thavren", "scope": "repo_local", "bootstrap_valid": true},
			"expect": false},
		{"name": "pip install blocked", "action": "bash",
			"facts":  map[string]interface{}{"command": "pip install malware", "operator": "thavren", "scope": "repo_local", "bootstrap_valid": true},
			"expect": false},
		{"name": "safe bash allowed", "action": "bash",
			"facts":  map[string]interface{}{"command": "python3 tools/ovav_runtime.py validate", "operator": "thavren", "scope": "repo_local", "bootstrap_valid": true},
			"expect": true},
		{"name": "path traversal blocked", "action": "file_write",
			"facts":  map[string]interface{}{"path": "../../../etc/passwd", "operator": "thavren", "bootstrap_valid": true},
			"expect": false},
		{"name": "force push blocked", "action": "git_push",
			"facts":  map[string]interface{}{"flags": "--force", "branch": "feature/x", "operator": "thavren", "bootstrap_valid": true},
			"expect": false},
		{"name": "plugin install blocked for eidren", "action": "install_plugin",
			"facts":  map[string]interface{}{"operator": "eidren", "bootstrap_valid": true},
			"expect": false},
		{"name": "plugin install allowed for thavren", "action": "install_plugin",
			"facts":  map[string]interface{}{"operator": "thavren", "scope": "repo_local", "bootstrap_valid": true},
			"expect": true},
		{"name": "github.com allowed", "action": "external_request",
			"facts":  map[string]interface{}{"url": "https://api.github.com/repos/x", "operator": "thavren", "bootstrap_valid": true},
			"expect": true},
		{"name": "evil domain blocked", "action": "external_request",
			"facts":  map[string]interface{}{"url": "https://evil.example.com/steal", "operator": "thavren", "bootstrap_valid": true},
			"expect": false},
		{"name": "secret in output blocked", "action": "output",
			"facts":  map[string]interface{}{"content": "TEST_TOKEN_PLACEHOLDER_NOT_REAL", "operator": "thavren", "bootstrap_valid": true},
			"expect": false},
		{"name": "without bootstrap blocked", "action": "bash",
			"facts":  map[string]interface{}{"command": "ls", "operator": "thavren", "scope": "repo_local", "bootstrap_valid": false},
			"expect": false},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
