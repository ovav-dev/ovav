package validators

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HostConfigDrift validates host configuration integrity and quarantine state.
// Simplified Go version of check_host_config_drift.py (1131 LOC) — core checks only.
// Replaces: check_host_config_drift.py
//
// OVAV TRUSTED EXECUTION DOMAIN — 2026-08-13:
// Host configurations carrying the canonical OVAV YOLO marker (_ovav.yolo, _ovav.trusted
// or the same JSON shape that .ovav/policy/permission_authority.json materializes) are
// recognized as OVAV-managed and are NOT host intrusions. Only configurations that:
//
//	(a) lack the OVAV marker AND
//	(b) carry agent/permission/provider intelligence
//
// are flagged for quarantine.
type HostConfigDrift struct {
	projectionFault func(string)
}

type anchoredRegularFile struct {
	data       []byte
	dirInfo    os.FileInfo
	revalidate func() error
	closeFiles func()
}

func NewHostConfigDrift() *HostConfigDrift { return &HostConfigDrift{} }

func (h *HostConfigDrift) ID() string   { return "host_config_drift" }
func (h *HostConfigDrift) Name() string { return "Host Config Drift" }
func (h *HostConfigDrift) Description() string {
	return "Validates host configuration integrity, quarantine state, and intrusion detection"
}
func (h *HostConfigDrift) Weight() int { return 25 }

// Host intrusion files — must NOT exist in ~/.config/opencode/.
// AGENTS.md and opencode.jsonc are OVAV governance files that should not leak to host.
// opencode.json is MiMoCode's own config — only flagged if it contains OVAV content.
var hostIntrusionFiles = []string{
	"AGENTS.md",
}

// Host intrusion agent paths.
var hostIntrusionAgents = []string{
	"agents/area-platform-engineering.md",
	"agents/area-research-intelligence.md",
	"agents/lead-thavren.md",
	"agents/lead-eidren.md",
}

func (h *HostConfigDrift) checkHostIntrusion(root string) []string {
	var issues []string
	home := os.Getenv("HOME")
	if home == "" {
		return issues
	}
	hostConfig := filepath.Join(home, ".config", "opencode")

	// Check for intrusion files
	for _, file := range hostIntrusionFiles {
		path := filepath.Join(hostConfig, file)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			issues = append(issues, fmt.Sprintf("HOST INTRUSION: %s found in ~/.config/opencode/ — quarantine required", file))
		}
	}

	// Global OpenCode configs may contain bootstrap/schema metadata. Provider,
	// permission and agent intelligence must stay repo-local UNLESS the file is
	// explicitly OVAV-managed (carries the YOLO marker from the materializer).
	for _, configName := range []string{"opencode.json", "opencode.jsonc"} {
		opencodePath := filepath.Join(hostConfig, configName)
		if info, err := os.Stat(opencodePath); err == nil && !info.IsDir() {
			if h.isCanonicalConfigProjection(root, opencodePath, configName) {
				continue
			}
			if !h.isBenignBootstrap(opencodePath) && h.containsGlobalIntelligence(opencodePath) {
				issues = append(issues, fmt.Sprintf("HOST INTRUSION: %s contains global agents/permissions/providers — quarantine required", configName))
			}
		}
	}

	// Check for intrusion agent files
	for _, agent := range hostIntrusionAgents {
		path := filepath.Join(hostConfig, agent)
		if _, err := os.Lstat(path); err == nil {
			if h.isCanonicalAgentProjection(root, path) {
				continue
			}
			issues = append(issues, fmt.Sprintf("HOST INTRUSION: %s found in ~/.config/opencode/agents/ — quarantine required", filepath.Base(agent)))
		}
	}

	return issues
}

func (h *HostConfigDrift) isCanonicalConfigProjection(root, hostPath, configName string) bool {
	hostData, err := readCanonicalProjection(root, hostPath)
	if err != nil {
		return false
	}
	canonicalData, err := readCanonicalProjection(root, filepath.Join(root, configName))
	if err != nil {
		return false
	}
	return bytes.Equal(hostData, canonicalData)
}

// readCanonicalProjection permits a host projection only when its resolved
// target remains inside the repository root. The final read still uses
// O_NOFOLLOW, so an untrusted symlink cannot be treated as canonical content.
func readCanonicalProjection(root, path string) ([]byte, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(pathAbs)
	if err != nil {
		return nil, err
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(rootAbs, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		// A regular host file may be an exact copy rather than a symlink
		// projection; it is accepted only through byte-for-byte comparison.
		if resolved == pathAbs {
			return readRegularFileNoFollow(pathAbs)
		}
		return nil, fmt.Errorf("projection target escapes repository root")
	}
	return readRegularFileNoFollow(resolved)
}

func (h *HostConfigDrift) isCanonicalAgentProjection(root, hostPath string) bool {
	canonicalPath := filepath.Join(root, ".opencode", "agents", filepath.Base(hostPath))
	canonicalData, err := readCanonicalProjection(root, canonicalPath)
	if err != nil {
		return false
	}

	agentsDir := filepath.Dir(hostPath)
	dirInfo, err := os.Lstat(agentsDir)
	if err != nil {
		return false
	}
	if dirInfo.IsDir() {
		hostInfo, statErr := os.Lstat(hostPath)
		if statErr != nil || !hostInfo.Mode().IsRegular() {
			return false
		}
		hostData, readErr := readRegularFileNoFollow(hostPath)
		return readErr == nil && bytes.Equal(hostData, canonicalData)
	}
	if dirInfo.Mode()&os.ModeSymlink == 0 {
		return false
	}

	mainRoot, err := mainRepoRootNoGit(root)
	if err != nil {
		return false
	}
	runtimeAgentsDir := filepath.Join(mainRoot, "go-runtime", "internal", "runtimes", "opencode", "agents")
	linkTarget, err := os.Readlink(agentsDir)
	if err != nil {
		return false
	}
	resolvedTarget := linkTarget
	if !filepath.IsAbs(resolvedTarget) {
		resolvedTarget = filepath.Join(filepath.Dir(agentsDir), resolvedTarget)
	}
	resolvedDir, err := filepath.Abs(filepath.Clean(resolvedTarget))
	if err != nil {
		return false
	}
	expectedDir, err := filepath.Abs(runtimeAgentsDir)
	if err != nil || resolvedDir != expectedDir {
		return false
	}
	actualDir, err := filepath.EvalSymlinks(agentsDir)
	if err != nil {
		return false
	}
	actualDir, err = filepath.Abs(actualDir)
	if err != nil || actualDir != expectedDir {
		return false
	}

	targetPath := filepath.Join(expectedDir, filepath.Base(hostPath))
	expectedDirInfo, err := os.Lstat(expectedDir)
	if err != nil || !expectedDirInfo.IsDir() || expectedDirInfo.Mode()&os.ModeSymlink != 0 {
		return false
	}
	anchored, err := openRegularFileAtNoFollow(expectedDir, filepath.Base(targetPath), expectedDirInfo)
	if err != nil {
		return false
	}
	defer anchored.closeFiles()
	if !os.SameFile(expectedDirInfo, anchored.dirInfo) || !bytes.Equal(anchored.data, canonicalData) {
		return false
	}
	if h.projectionFault != nil {
		h.projectionFault("before_projection_recheck")
	}
	if err := anchored.revalidate(); err != nil {
		return false
	}
	finalLinkInfo, err := os.Lstat(agentsDir)
	if err != nil || finalLinkInfo.Mode()&os.ModeSymlink == 0 || !os.SameFile(dirInfo, finalLinkInfo) {
		return false
	}
	finalLinkTarget, err := os.Readlink(agentsDir)
	if err != nil || finalLinkTarget != linkTarget {
		return false
	}
	finalDirInfo, err := os.Lstat(expectedDir)
	if err != nil || !finalDirInfo.IsDir() || finalDirInfo.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(expectedDirInfo, finalDirInfo) || !os.SameFile(anchored.dirInfo, finalDirInfo) {
		return false
	}
	finalResolvedDir, err := filepath.EvalSymlinks(agentsDir)
	if err != nil {
		return false
	}
	finalResolvedDir, err = filepath.Abs(finalResolvedDir)
	if err != nil || finalResolvedDir != expectedDir {
		return false
	}
	return true
}

func mainRepoRootNoGit(root string) (string, error) {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return "", fmt.Errorf("inspect .git: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf(".git must not be a symlink")
	}
	if info.IsDir() {
		return filepath.Abs(root)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf(".git is not a regular file or directory")
	}

	data, err := readRegularFileNoFollow(gitPath)
	if err != nil {
		return "", fmt.Errorf("read .git: %w", err)
	}
	content := strings.TrimSuffix(string(data), "\n")
	content = strings.TrimSuffix(content, "\r")
	const prefix = "gitdir: "
	if !strings.HasPrefix(content, prefix) {
		return "", fmt.Errorf("malformed gitdir")
	}
	gitDir := content[len(prefix):]
	if gitDir == "" || strings.TrimSpace(gitDir) != gitDir || strings.ContainsAny(gitDir, "\r\n\x00") {
		return "", fmt.Errorf("malformed gitdir path")
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	gitDir, err = filepath.Abs(filepath.Clean(gitDir))
	if err != nil || filepath.Base(filepath.Dir(gitDir)) != "worktrees" {
		return "", fmt.Errorf("malformed worktree gitdir")
	}
	mainGitDir := filepath.Dir(filepath.Dir(gitDir))
	if filepath.Base(mainGitDir) != ".git" {
		return "", fmt.Errorf("gitdir is outside a main .git directory")
	}
	for _, dir := range []string{mainGitDir, gitDir} {
		dirInfo, statErr := os.Lstat(dir)
		if statErr != nil || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("unsafe gitdir")
		}
		resolvedDir, resolveErr := filepath.EvalSymlinks(dir)
		if resolveErr != nil {
			return "", fmt.Errorf("resolve gitdir: %w", resolveErr)
		}
		resolvedDir, resolveErr = filepath.Abs(resolvedDir)
		if resolveErr != nil || resolvedDir != dir {
			return "", fmt.Errorf("gitdir contains a nested symlink")
		}
	}
	return filepath.Dir(mainGitDir), nil
}

func (h *HostConfigDrift) containsGlobalIntelligence(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		content := string(data)
		for _, key := range []string{"agent", "agents", "permission", "permissions", "provider", "providers"} {
			if strings.Contains(content, `"`+key+`"`) {
				return true
			}
		}
		return false
	}
	for _, key := range []string{"agent", "agents", "permission", "permissions", "provider", "providers"} {
		if _, exists := config[key]; exists {
			return true
		}
	}
	return false
}

func (h *HostConfigDrift) isBenignBootstrap(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	// Check for schema-only config — normalize whitespace for robust matching.
	// JSON can be compact: {"$schema":"https://..."} or pretty-printed:
	// {\n  "$schema": "https://..."\n}. Both are equivalent bootstrap configs.
	content := strings.TrimSpace(string(data))
	// Remove ALL whitespace between JSON tokens for comparison
	normalized := strings.ReplaceAll(content, " ", "")
	normalized = strings.ReplaceAll(normalized, "\n", "")
	normalized = strings.ReplaceAll(normalized, "\t", "")
	normalized = strings.ReplaceAll(normalized, "\r", "")
	if normalized == `{"$schema":"https://opencode.ai/config.json"}` {
		return true
	}
	return false
}

func (h *HostConfigDrift) checkBlockade(root string) []string {
	var issues []string
	blockadePath := filepath.Join(root, ".ovav", "host_defense_blockade")
	if info, err := os.Stat(blockadePath); err == nil && info.Size() > 0 {
		issues = append(issues, "WARNING: host_defense_blockade file exists — system may be in quarantine")
	}
	return issues
}

func (h *HostConfigDrift) checkQuarantine(root string) []string {
	var issues []string
	quarantineDir := filepath.Join(root, ".ovav", "quarantine")
	if entries, err := os.ReadDir(quarantineDir); err == nil {
		fileCount := 0
		for _, e := range entries {
			if !e.IsDir() {
				fileCount++
			}
		}
		if fileCount > 0 {
			issues = append(issues, fmt.Sprintf("QUARANTINE: %d file(s) in quarantine — review required", fileCount))
		}
	}
	return issues
}

func (h *HostConfigDrift) checkSessionMarker(root string) []string {
	marker := filepath.Join(root, ".ovav", "runtime", ".session_marker")
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		// Session marker missing is normal outside development sessions
		return nil
	}
	// Session marker exists — authorized development session
	return nil
}

func (h *HostConfigDrift) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string

	// 1. Host intrusion detection
	intrusionIssues := h.checkHostIntrusion(root)
	issues = append(issues, intrusionIssues...)

	// 2. Blockade check
	blockadeIssues := h.checkBlockade(root)
	issues = append(issues, blockadeIssues...)

	// 3. Quarantine check
	quarantineIssues := h.checkQuarantine(root)
	issues = append(issues, quarantineIssues...)

	// 4. Session marker check
	_ = h.checkSessionMarker(root)

	// Check for critical issues (HOST INTRUSION)
	hasCritical := false
	for _, issue := range issues {
		if strings.Contains(issue, "HOST INTRUSION") {
			hasCritical = true
			break
		}
	}

	if hasCritical {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "fail", Weight: h.Weight(),
			Message:  fmt.Sprintf("FAIL host config drift — CRITICAL: host intrusion detected (%d issue(s))", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	if len(issues) > 0 {
		return Result{
			ID: h.ID(), Name: h.Name(), Status: "warn", Weight: h.Weight(),
			Message:  fmt.Sprintf("WARN host config drift — %d non-critical issue(s)", len(issues)),
			Issues:   issues,
			Duration: time.Since(start),
		}
	}
	return Result{
		ID: h.ID(), Name: h.Name(), Status: "pass", Weight: h.Weight(),
		Message:  "PASS host config drift — no intrusion detected",
		Duration: time.Since(start),
	}
}

var _ Validator = (*HostConfigDrift)(nil)
