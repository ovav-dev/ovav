package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PermissionAuthority manages canonical permission policy and materialization.
type PermissionAuthority struct {
	Root          string
	PolicyPath    string
	OpencodePath  string
	PlatformAgent string
	ThavrenAlias  string
	ToolPolicy    string
}

// PermissionBlock is the host projection derived from the canonical authority.
type PermissionBlock struct {
	Edit              string
	Bash              map[string]string
	ExternalDirectory map[string]string
	Extra             map[string]string // OVAV TRUSTED DOMAIN: wildcard + per-tool allows
}

// MaterializePermissionBlock returns the canonical host permission projection.
//
// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
// Returns a fully YOLO permission block: every tool category gets "allow"
// via the "*" wildcard, and bash preserves CriticalDenies for catastrophic
// operations. This makes the host fully autonomous — no --auto flag, no
// re-ask, no permission prompts.
func (a *PermissionAuthority) MaterializePermissionBlock() (PermissionBlock, error) {
	bash, err := a.expectedBashPermissions()
	if err != nil {
		return PermissionBlock{}, err
	}
	// Build a map with global wildcard + per-tool explicit allow.
	// opencode treats "*" as the default for any tool not explicitly listed.
	perms := map[string]string{}
	perms["*"] = "allow"
	perms["edit"] = "allow"
	perms["write"] = "allow"
	perms["read"] = "allow"
	perms["glob"] = "allow"
	perms["grep"] = "allow"
	perms["list"] = "allow"
	perms["patch"] = "allow"
	perms["task"] = "allow"
	perms["skill"] = "allow"
	perms["webfetch"] = "allow"
	perms["websearch"] = "allow"
	perms["doom_loop"] = "allow"
	// Merge bash (with critical denies) into the flat map
	for k, v := range bash {
		perms[k] = v
	}
	perms["external_directory"] = "*" // marker — handled as map by writer
	return PermissionBlock{
		Edit:              "allow",
		Bash:              bash,
		ExternalDirectory: ExpectedExternalDirectory(""),
		Extra:             perms,
	}, nil
}

// NewPermissionAuthority creates a new authority with default paths.
func NewPermissionAuthority(root string) *PermissionAuthority {
	return &PermissionAuthority{
		Root:          root,
		PolicyPath:    filepath.Join(root, ".ovav/policy/permission_authority.json"),
		OpencodePath:  filepath.Join(root, "opencode.json"),
		PlatformAgent: filepath.Join(root, ".opencode/agents/area-platform-engineering.md"),
		ThavrenAlias:  filepath.Join(root, ".opencode/agents/lead-thavren.md"),
		ToolPolicy:    filepath.Join(root, ".ovav/service_areas/shared/tool_access_policy.yaml"),
	}
}

// CriticalDenies returns the critical deny patterns.
//
// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
// YOLO is default-allow, not safety-free. Catastrophic host operations and raw
// git propagation remain permanently denied at the host and governor layers.
func CriticalDenies() map[string]string {
	return map[string]string{
		"git push*":             "deny",
		"sudo *":                "deny",
		"rm -rf /*":             "deny",
		"mkfs*":                 "deny",
		"dd *of=/dev/*":         "deny",
		"git branch -D *":       "deny",
		"git branch --delete *": "deny",
		"apt install *":         "deny",
		"npm install *":         "deny",
		"pip install *":         "deny",
	}
}

// RequiredAllows returns the required allow patterns.
func RequiredAllows() map[string]string {
	return map[string]string{
		"go run -C go-runtime ./cmd/ovav validate*":                "allow",
		"go run -C go-runtime ./internal/validators/cmd/validate*": "allow",
		"git status*":               "allow",
		"git diff*":                 "allow",
		"git log*":                  "allow",
		"git rev-parse*":            "allow",
		"git remote -v":             "allow",
		"git ls-remote *":           "allow",
		"git branch --show-current": "allow",
		"git add *":                 "allow",
		"git commit*":               "allow",
		"gh auth status*":           "allow",
		"gh repo view*":             "allow",
		"gh issue list*":            "allow",
		"gh issue view*":            "allow",
		"gh pr view*":               "allow",
		"gh pr status*":             "allow",
		"gh pr list*":               "allow",
		"gh pr create*":             "allow",
		"pytest*":                   "allow",
		"python3 -m pytest*":        "allow",
		"npm test*":                 "allow",
		"npm run test*":             "allow",
		"npm run lint*":             "allow",
		"npm run typecheck*":        "allow",
		"npm run build*":            "allow",
	}
}

// ExpectedBashPermissions returns the expected bash permission map.
func ExpectedBashPermissions() map[string]string {
	perms := make(map[string]string)
	for k, v := range CriticalDenies() {
		perms[k] = v
	}
	for k, v := range RequiredAllows() {
		perms[k] = v
	}
	perms["*"] = "allow"
	return perms
}

func (a *PermissionAuthority) protectedBashDenies() ([]string, error) {
	data, err := os.ReadFile(a.PolicyPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read protected bash denies: %w", err)
	}
	var policy struct {
		ProtectedDenies struct {
			Bash []string `json:"bash"`
		} `json:"protected_denies"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parse protected bash denies: %w", err)
	}
	return policy.ProtectedDenies.Bash, nil
}

func (a *PermissionAuthority) expectedBashPermissions() (map[string]string, error) {
	permissions := ExpectedBashPermissions()
	denies, err := a.protectedBashDenies()
	if err != nil {
		return nil, err
	}
	for _, pattern := range denies {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || pattern == "*" {
			return nil, fmt.Errorf("invalid protected bash deny %q", pattern)
		}
		permissions[pattern] = "deny"
	}
	return permissions, nil
}

// ExpectedExternalDirectory returns the expected external directory permissions.
//
// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
// The upper trust layer is OVAV governor itself. The host runtime must not
// re-ask for permission on operations OVAV has already decided. CriticalDenies
// remain the only safety baseline for catastrophic host-level operations.
//
// YOLO: External directories are ALLOWED by default. The OVAV governor decides
// intent, routing, policy, and validation. The host (OpenCode / ACP / TUI / shell)
// executes without redundant confirmation.
func ExpectedExternalDirectory(agentName string) map[string]string {
	return map[string]string{
		"*": "allow",
	}
}

// ExpectedOpencodePermission returns the expected opencode.json permission block.
//
// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
// The "*" key is the global default for any tool/category not explicitly
// configured. Setting it to "allow" makes YOLO fully autonomous — opencode
// will not prompt for any tool whose name is not in the config.
//
// Per-tool entries override "*" where they exist (e.g., bash has its own
// map with critical denies). The combination guarantees:
//
//  1. Known tools (bash, edit, write, read, glob, etc.) → allow or per-tool map
//  2. Unknown / future tools → "*": "allow" (autonomous default)
//  3. Catastrophic bash operations → explicit "deny" (CriticalDenies)
//  4. external_directory → "*": "allow" (YOLO trust domain)
//
// Result: NO host re-ask. NO --auto flag required. OVAV decides, host runs.
func ExpectedOpencodePermission() map[string]interface{} {
	perms := make(map[string]interface{})
	perms["*"] = "allow" // Global wildcard: any unmatched tool → allow
	perms["edit"] = "allow"
	perms["write"] = "allow"
	perms["read"] = "allow"
	perms["glob"] = "allow"
	perms["grep"] = "allow"
	perms["list"] = "allow"
	perms["patch"] = "allow"
	perms["task"] = "allow"
	perms["skill"] = "allow"
	perms["webfetch"] = "allow"
	perms["websearch"] = "allow"
	perms["doom_loop"] = "allow"
	perms["bash"] = ExpectedBashPermissions()
	perms["external_directory"] = ExpectedExternalDirectory("")
	return perms
}

func (a *PermissionAuthority) expectedOpencodePermission() (map[string]interface{}, error) {
	bash, err := a.expectedBashPermissions()
	if err != nil {
		return nil, err
	}
	perms := make(map[string]interface{})
	perms["*"] = "allow" // Global wildcard for unmatched tools
	perms["edit"] = "allow"
	perms["write"] = "allow"
	perms["read"] = "allow"
	perms["glob"] = "allow"
	perms["grep"] = "allow"
	perms["list"] = "allow"
	perms["patch"] = "allow"
	perms["task"] = "allow"
	perms["skill"] = "allow"
	perms["webfetch"] = "allow"
	perms["websearch"] = "allow"
	perms["doom_loop"] = "allow"
	perms["bash"] = bash
	perms["external_directory"] = ExpectedExternalDirectory("")
	return perms, nil
}

// MaterializeAll materializes all permission projections.
func (a *PermissionAuthority) MaterializeAll(write bool) (map[string]interface{}, error) {
	if err := a.assertPolicySafe(); err != nil {
		return nil, err
	}

	projections := map[string]string{}
	opencodeProj, err := a.buildOpencodeProjection()
	if err != nil {
		return nil, fmt.Errorf("failed to build opencode projection: %w", err)
	}
	projections[a.OpencodePath] = opencodeProj

	platformProj, err := a.buildAgentProjection(a.PlatformAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to build platform agent projection: %w", err)
	}
	projections[a.PlatformAgent] = platformProj

	thavrenProj, err := a.buildAgentProjection(a.ThavrenAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to build thavren alias projection: %w", err)
	}
	projections[a.ThavrenAlias] = thavrenProj

	targets := []map[string]interface{}{}
	changedCount := 0
	for path, projectedText := range projections {
		currentText, err := os.ReadFile(path)
		if err != nil {
			currentText = []byte{}
		}
		changed := string(currentText) != projectedText
		target := map[string]interface{}{
			"path":    strings.TrimPrefix(path, a.Root+"/"),
			"changed": changed,
			"action":  "none",
		}
		if write && changed {
			target["action"] = "write"
			if err := os.WriteFile(path, []byte(projectedText), 0644); err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", path, err)
			}
			changedCount++
		} else if changed {
			changedCount++
		}
		targets = append(targets, target)
	}

	status := "clean"
	if changedCount > 0 {
		status = "changed"
	}
	mode := "check"
	if write {
		mode = "write"
	}

	return map[string]interface{}{
		"status":        status,
		"authority":     strings.TrimPrefix(a.PolicyPath, a.Root+"/"),
		"mode":          mode,
		"changed_count": changedCount,
		"targets":       targets,
	}, nil
}

func (a *PermissionAuthority) assertPolicySafe() error {
	data, err := os.ReadFile(a.PolicyPath)
	if err != nil {
		return fmt.Errorf("cannot read policy: %w", err)
	}
	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		return fmt.Errorf("invalid policy JSON: %w", err)
	}
	schemaVersion, _ := policy["schema_version"].(string)
	if schemaVersion != "ovav.permission_authority.v1" && schemaVersion != "ovav.permission_authority.v2" && schemaVersion != "ovav.permission_authority.v3" {
		return fmt.Errorf("permission authority schema mismatch: %s", schemaVersion)
	}
	expectedTargets := map[string]bool{
		"opencode.json": true,
		".opencode/agents/area-platform-engineering.md": true,
		".opencode/agents/lead-thavren.md":              true,
	}
	declaredTargets, _ := policy["materialized_targets"].([]interface{})
	for _, t := range declaredTargets {
		ts, _ := t.(string)
		delete(expectedTargets, ts)
	}
	if len(expectedTargets) > 0 {
		missing := []string{}
		for k := range expectedTargets {
			missing = append(missing, k)
		}
		return fmt.Errorf("permission authority missing materialized targets: %v", missing)
	}
	return nil
}

func (a *PermissionAuthority) buildOpencodeProjection() (string, error) {
	data, err := os.ReadFile(a.OpencodePath)
	if err != nil {
		return "", err
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}
	permission, err := a.expectedOpencodePermission()
	if err != nil {
		return "", err
	}
	config["permission"] = permission
	result, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(result) + "\n", nil
}

func (a *PermissionAuthority) buildAgentProjection(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return a.ProjectAgentDocument(string(data), strings.TrimSuffix(filepath.Base(path), ".md"))
}

// ProjectAgentDocument replaces one OpenCode agent's managed permission block
// with the canonical authority projection.
func (a *PermissionAuthority) ProjectAgentDocument(text, agentName string) (string, error) {
	if !strings.HasPrefix(text, "---") {
		return "", fmt.Errorf("agent frontmatter missing")
	}
	parts := strings.SplitN(text, "---", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("agent frontmatter malformed")
	}
	frontmatter := parts[1]
	body := parts[2]

	// Strip managed permissions
	kept := []string{}
	skipping := false
	for _, line := range strings.Split(frontmatter, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "# OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json" ||
			stripped == "OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json" {
			continue
		}
		if !skipping && strings.HasPrefix(line, "permission:") {
			skipping = true
			continue
		}
		if skipping {
			if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#") {
				skipping = false
			} else {
				continue
			}
		}
		if !skipping {
			kept = append(kept, strings.TrimRight(line, " \t"))
		}
	}
	// Trim trailing empty lines
	for len(kept) > 0 && strings.TrimSpace(kept[len(kept)-1]) == "" {
		kept = kept[:len(kept)-1]
	}
	for len(kept) > 0 && strings.TrimSpace(kept[0]) == "" {
		kept = kept[1:]
	}

	bash, err := a.expectedBashPermissions()
	if err != nil {
		return "", err
	}
	newFrontmatter := append(kept, "# OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json")
	newFrontmatter = append(newFrontmatter, agentPermissionYAML(agentName, bash)...)

	if !strings.HasPrefix(body, "\n") {
		body = "\n" + body
	}
	return "---\n" + strings.Join(newFrontmatter, "\n") + "\n---" + body, nil
}

func expectedAgentPermissionYAML(agentName string) []string {
	return agentPermissionYAML(agentName, ExpectedBashPermissions())
}

func agentPermissionYAML(agentName string, bash map[string]string) []string {
	lines := []string{"permission:", "  edit: allow", "  bash:"}
	for _, pattern := range orderedPermissionPatterns(bash) {
		decision := bash[pattern]
		lines = append(lines, fmt.Sprintf("    %s: %s", strconv.Quote(pattern), decision))
	}
	lines = append(lines, "  external_directory:")
	for _, pattern := range orderedPermissionPatterns(ExpectedExternalDirectory(agentName)) {
		decision := ExpectedExternalDirectory(agentName)[pattern]
		lines = append(lines, fmt.Sprintf("    %s: %s", strconv.Quote(pattern), decision))
	}
	return lines
}

func orderedPermissionPatterns(permissions map[string]string) []string {
	patterns := make([]string, 0, len(permissions))
	for pattern := range permissions {
		if pattern != "*" {
			patterns = append(patterns, pattern)
		}
	}
	sort.Strings(patterns)
	return append([]string{"*"}, patterns...)
}

// CheckAll checks all permission surfaces for drift.
func (a *PermissionAuthority) CheckAll(writeLog bool) ([]map[string]interface{}, error) {
	drift := []map[string]interface{}{}

	// Check opencode.json
	opencodeDrift, err := a.checkOpencode()
	if err != nil {
		return nil, err
	}
	drift = append(drift, opencodeDrift...)

	// Check platform agent
	platformDrift, err := a.checkAgent(a.PlatformAgent, ".opencode/agents/area-platform-engineering.md")
	if err != nil {
		return nil, err
	}
	drift = append(drift, platformDrift...)

	// Check thavren alias
	thavrenDrift, err := a.checkAgent(a.ThavrenAlias, ".opencode/agents/lead-thavren.md")
	if err != nil {
		return nil, err
	}
	drift = append(drift, thavrenDrift...)

	// Check runtime policy surfaces
	runtimeDrift, err := a.checkRuntimePolicySurfaces()
	if err != nil {
		return nil, err
	}
	drift = append(drift, runtimeDrift...)

	if writeLog && len(drift) > 0 {
		a.appendLog(drift)
	}

	return drift, nil
}

func (a *PermissionAuthority) checkOpencode() ([]map[string]interface{}, error) {
	drift := []map[string]interface{}{}
	data, err := os.ReadFile(a.OpencodePath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	permission, _ := config["permission"].(map[string]interface{})
	expected, err := a.expectedOpencodePermission()
	if err != nil {
		return nil, err
	}

	for _, field := range []string{"edit", "external_directory"} {
		if fmt.Sprintf("%v", permission[field]) != fmt.Sprintf("%v", expected[field]) {
			drift = append(drift, map[string]interface{}{
				"surface":  "opencode.json",
				"field":    "permission." + field,
				"expected": expected[field],
				"actual":   permission[field],
			})
		}
	}

	bash, _ := permission["bash"].(map[string]interface{})
	expectedBash, _ := expected["bash"].(map[string]string)
	for pattern, decision := range expectedBash {
		if fmt.Sprintf("%v", bash[pattern]) != decision {
			drift = append(drift, map[string]interface{}{
				"surface":  "opencode.json",
				"field":    "permission.bash." + pattern,
				"expected": decision,
				"actual":   bash[pattern],
			})
		}
	}
	if fmt.Sprintf("%v", bash["*"]) != "allow" {
		drift = append(drift, map[string]interface{}{
			"surface":  "opencode.json",
			"field":    "permission.bash.*",
			"expected": "allow",
			"actual":   bash["*"],
		})
	}

	return drift, nil
}

func (a *PermissionAuthority) checkAgent(path, name string) ([]map[string]interface{}, error) {
	drift := []map[string]interface{}{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(data)
	frontmatter := ""
	if strings.HasPrefix(text, "---") {
		parts := strings.SplitN(text, "---", 3)
		if len(parts) >= 3 {
			frontmatter = parts[1]
		}
	}

	authorityMarker := "OVAV_PERMISSION_AUTHORITY: .ovav/policy/permission_authority.json"
	if !strings.Contains(text, authorityMarker) {
		drift = append(drift, map[string]interface{}{
			"surface":  name,
			"field":    "authority_marker",
			"expected": authorityMarker,
		})
	}

	requiredLines := []string{
		"edit: allow",
		"\"git push*\": deny",
		"\"git push --force *\": deny",
		"\"gh auth token*\": deny",
		"\"sudo *\": deny",
		"\"go run -C go-runtime ./cmd/ovav validate*\": allow",
		"\"go run -C go-runtime ./internal/validators/cmd/validate*\": allow",
		"\"git commit*\": allow",
		"\"git ls-remote *\": allow",
		"\"gh pr create*\": allow",
		"\"*\": allow",
	}
	if strings.Contains(strings.ToLower(name), "thavren") {
		requiredLines = append(requiredLines, "\"*\": allow")
	} else {
		requiredLines = append(requiredLines, "\"/tmp/opencode/*\": allow", "\"*\": deny")
	}
	for _, line := range requiredLines {
		if !strings.Contains(frontmatter, line) {
			drift = append(drift, map[string]interface{}{
				"surface": name,
				"field":   "frontmatter",
				"missing": line,
			})
		}
	}

	forbiddenLines := []string{"edit: ask", "bash: ask", "\"*\": ask"}
	for _, line := range forbiddenLines {
		if strings.Contains(frontmatter, line) {
			drift = append(drift, map[string]interface{}{
				"surface":   name,
				"field":     "frontmatter",
				"forbidden": line,
			})
		}
	}

	return drift, nil
}

func (a *PermissionAuthority) checkRuntimePolicySurfaces() ([]map[string]interface{}, error) {
	drift := []map[string]interface{}{}

	toolPolicy, err := os.ReadFile(a.ToolPolicy)
	if err == nil {
		expected := []string{
			"canonical_source: .ovav/policy/permission_authority.json",
			"drift_response: log_and_restore_ovav_policy",
			"decision: allow_when_user_requested_and_workspace_safety_gate_passes",
			"decision: deny_raw_use_ovav_git_push_gate_with_user_confirmation",
		}
		for _, exp := range expected {
			if !strings.Contains(string(toolPolicy), exp) {
				drift = append(drift, map[string]interface{}{
					"surface": strings.TrimPrefix(a.ToolPolicy, a.Root+"/"),
					"missing": exp,
				})
			}
		}
	}

	return drift, nil
}

func (a *PermissionAuthority) appendLog(drift []map[string]interface{}) {
	data, err := os.ReadFile(a.PolicyPath)
	if err != nil {
		return
	}
	var policy map[string]interface{}
	if err := json.Unmarshal(data, &policy); err != nil {
		return
	}
	logPath, _ := policy["log_path"].(string)
	if logPath == "" {
		logPath = ".ovav/runtime/logs/permission_drift.jsonl"
	}
	fullLogPath := filepath.Join(a.Root, logPath)
	os.MkdirAll(filepath.Dir(fullLogPath), 0755)

	event := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"event":     "permission_policy_drift_detected",
		"authority": strings.TrimPrefix(a.PolicyPath, a.Root+"/"),
		"drift":     drift,
	}
	eventJSON, _ := json.Marshal(event)
	f, err := os.OpenFile(fullLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(eventJSON, '\n'))
}

// ReconcileAll detects drift, materializes, and re-checks.
func (a *PermissionAuthority) ReconcileAll(write, writeLog bool) (map[string]interface{}, error) {
	before, err := a.CheckAll(writeLog)
	if err != nil {
		return nil, err
	}

	materialized, err := a.MaterializeAll(write)
	if err != nil {
		return nil, err
	}

	after, err := a.CheckAll(false)
	if err != nil {
		return nil, err
	}

	status := "pass"
	if len(after) > 0 {
		status = "fail"
	}

	return map[string]interface{}{
		"status":             status,
		"authority":          strings.TrimPrefix(a.PolicyPath, a.Root+"/"),
		"mode":               map[bool]string{true: "write", false: "check"}[write],
		"drift_before_count": len(before),
		"drift_after_count":  len(after),
		"drift_before":       before,
		"drift_after":        after,
		"materialized":       materialized,
	}, nil
}
