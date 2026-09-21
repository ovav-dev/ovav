package validators

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// ContextFirewallV2 validates context firewall v2 — scans runtime files for
// suspicious injection patterns: external URLs, base64 blocks, hidden Unicode.
// Replaces: tools/validators/check_context_firewall_v2.py
type ContextFirewallV2 struct{}

func NewContextFirewallV2() *ContextFirewallV2 { return &ContextFirewallV2{} }

func (v *ContextFirewallV2) ID() string   { return "context_firewall_v2" }
func (v *ContextFirewallV2) Name() string { return "Context Firewall v2" }
func (v *ContextFirewallV2) Description() string {
	return "Scans runtime files for injection patterns: external URLs, base64 blocks, hidden Unicode"
}
func (v *ContextFirewallV2) Weight() int { return 8 }

// approvedDomains lists domains that are safe to reference externally.
var approvedDomains = map[string]bool{
	// Development platforms
	"github.com":            true,
	"githubusercontent.com": true, // raw.githubusercontent.com
	"github.io":             true, // GitHub Pages — legitimate documentation hosting (openai.github.io, etc.)
	"gitlab.com":            true,
	// Cloudflare
	"cloudflare.com":            true,
	"developers.cloudflare.com": true,
	// Go ecosystem
	"pkg.go.dev": true,
	"golang.org": true,
	// Python ecosystem
	"python.org": true,
	"pypi.org":   true,
	// Node/JS ecosystem
	"npmjs.com":  true,
	"npmjs.org":  true,
	"nodejs.org": true,
	// Badge/shield services
	"shields.io": true,
	// OVAV own domains
	"ovav.dev": true,
	"fly.dev":  true, // ovav-systems.fly.dev, d678beea.ovav.dev
	// OpenCode
	"opencode.ai": true,
	// Standards
	"opencontainers.org": true,
	"json-schema.org":    true,
	"rfc-editor.org":     true,
	"apache.org":         true,
	// Policy as code
	"rego.io":             true,
	"openpolicyagent.org": true,
	// Education & reference
	"en.wikipedia.org":  true,
	"arxiv.org":         true,
	"ieee.org":          true, // ieeexplore.ieee.org
	"acm.org":           true, // dl.acm.org
	"google.com":        true, // developers.google.com, toolbox.google.com
	"research.google":   true,
	"youtube.com":       true,
	"coursera.org":      true,
	"kaggle.com":        true,
	"leetcode.com":      true,
	"exercism.org":      true,
	"runestone.academy": true,
	"openintro.org":     true,
	// AI/ML research
	"openai.com":    true,
	"anthropic.com": true,
	"claude.com":    true, // Anthropic Claude platform (claude.ai, platform.claude.com)
	// Infrastructure & monitoring
	"betterstack.com":    true,
	"nextjs.org":         true,
	"opencollective.com": true,
	// Dev tools & platforms (legitimate development references)
	"vercel.com":     true, // Vercel platform
	"supabase.com":   true, // Supabase backend
	"linear.app":     true, // Linear project tracking
	"zed.dev":        true, // Zed editor
	"warp.dev":       true, // Warp terminal
	"aka.ms":         true, // Microsoft short URLs
	"fly.io":         true, // Fly.io infrastructure
	"midjourney.com": true, // Midjourney docs
	// AI/Agent protocols & standards
	"modelcontextprotocol.io": true, // MCP protocol spec
	"a2a-protocol.org":        true, // Agent-to-Agent protocol
	// AI dev tools & competitors (documentation references only)
	"cursor.com":   true, // Cursor IDE docs
	"tabnine.com":  true, // Tabnine AI completion
	"continue.dev": true, // Continue.dev
	"codeium.com":  true, // Codeium AI
	"aider.chat":   true, // Aider chat AI pairing
	// Microsoft ecosystem
	"learn.microsoft.com": true, // Microsoft Learn
	"microsoft.com":       true,
	"owasp.org":           true, // OWASP security
	// Nerd fonts & terminal tools
	"nerdfonts.com": true,
	"ohmyposh.dev":  true,
	// OVAV internal & related
	"mimo.xiaomi.com": true, // MiMo internal
	"mimocode.ai":     true, // MiMoCode platform
	"minimax.io":      true, // MiniMax API
	"api.minimax.io":  true, // MiniMax API endpoint
	// DEPRECATED - delete from DNS
	"api.ovav.dev":      true, // DEPRECATED - delete from DNS
	"d678beea.ovav.dev": true, // OVAV cPanel (canonical — non-indexable Fly.io machine URL)
	"docs.ovav.dev":     true, // OVAV documentation site
	// DEPRECATED - never implemented
	"cdn.ovav.dev": true, // DEPRECATED - never implemented
	// DEPRECATED - redirect to ovav.dev
	"get.ovav.dev":    true, // DEPRECATED - redirect to ovav.dev
	"status.ovav.dev": true, // OVAV status page
	// DEPRECATED - use d678beea.ovav.dev
	"ovav-cpanel.fly.dev":     true, // DEPRECATED - use d678beea.ovav.dev
	"api-bitel-agent.fly.dev": true, // Bitel agent API
	"forms.gle":               true, // Google Forms (legitimate)
	"discord.gg":              true, // Discord community invite (legitimate dev community)
	// Local
	"localhost": true,
	"127.0.0.1": true,
	"0.0.0.0":   true,
}

// knownPlaceholderDomains are IETF-reserved base domains that can never be
// real external URLs. They are skipped during scanning.
var knownPlaceholderDomains = map[string]bool{
	"example.com": true, // RFC 2606 - reserved for documentation
	"example.org": true,
	"example.net": true,
	"test.com":    true, // commonly used placeholder
}

// isKnownPlaceholder returns true for reserved example domains that are never real.
func isKnownPlaceholder(domain string) bool {
	if knownPlaceholderDomains[domain] {
		return true
	}
	return false
}

// urlPattern matches http:// and https:// URLs.
var urlPattern = regexp.MustCompile(`https?://([^\s/\\"'\)>]+)`)

// base64Pattern matches potential base64-encoded blocks (40+ chars of base64 alphabet).
var base64Pattern = regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)

// Suspicious Unicode characters to detect.
var suspiciousChars = []struct {
	name  string
	runes []rune
}{
	{"zero-width space", []rune{'\u200B'}},
	{"zero-width non-joiner", []rune{'\u200C'}},
	{"zero-width joiner", []rune{'\u200D'}},
	{"left-to-right mark", []rune{'\u200E'}},
	{"right-to-left mark", []rune{'\u200F'}},
	{"left-to-right embedding", []rune{'\u202A'}},
	{"right-to-left embedding", []rune{'\u202B'}},
	{"pop directional formatting", []rune{'\u202C'}},
	{"left-to-right override", []rune{'\u202D'}},
	{"right-to-left override", []rune{'\u202E'}},
	{"word joiner", []rune{'\u2060'}},
	{"invisible separator", []rune{'\u2063'}},
	{"invisible times", []rune{'\u2062'}},
	{"invisible plus", []rune{'\u2064'}},
	{"object replacement character", []rune{'\uFFFC'}},
	{"replacement character", []rune{'\uFFFD'}},
}

// Files to scan for injection patterns.
var firewallScanFiles = []string{
	"AGENTS.md",
	"README.md",
}

var firewallScanDirs = []string{
	".ovav",
	"runtimes",
	".opencode",
}

// firewallSkipDirs lists directories to skip during scan to prevent memory explosion.
// .ovav/projects/ contains old product artifacts (538MB) with thousands of npm URLs.
// .ovav/evidence/ contains large push report files.
// worktrees/ contains worktree-local docs that duplicate .ovav/ content.
var firewallSkipDirs = map[string]bool{
	"node_modules":      true,
	"__pycache__":       true,
	".git":              true,
	"projects":          true,
	"evidence":          true,
	"integrity_backups": true,
	"runtime":           true,
	"cache":             true,
	"worktrees":         true, // worktree artifacts — duplicate of .ovav/ content, not security-relevant
}

func (v *ContextFirewallV2) Validate(ctx context.Context, root string) Result {
	start := time.Now()
	var issues []string
	var suggestions []string
	scanned := 0

	// Initialize domain tracker — persists to .ovav/security/domain_registry.json
	registryPath := filepath.Join(root, ".ovav", "security", "domain_registry.json")
	tracker := NewDomainTracker(registryPath)

	// Collect all files to scan
	var filePaths []string

	// Add explicit files
	for _, f := range firewallScanFiles {
		fp := filepath.Join(root, f)
		if _, err := os.Stat(fp); err == nil {
			filePaths = append(filePaths, fp)
		}
	}

	// Add files from scan directories (skip vendor/dependency dirs)
	for _, dir := range firewallScanDirs {
		scanPath := filepath.Join(root, dir)
		filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip files with read errors
			}
			if info.IsDir() {
				name := info.Name()
				if firewallSkipDirs[name] {
					return filepath.SkipDir
				}
				return nil
			}
			// Skip lock files (vendor-managed, not OVAV content)
			base := filepath.Base(path)
			if base == "package-lock.json" || base == "pnpm-lock.yaml" || base == "yarn.lock" {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".md" || ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".toml" {
				filePaths = append(filePaths, path)
			}
			return nil
		})
	}

	// Scan each file
	const maxIssuesPerFile = 50
	const maxTotalIssues = 500
	for _, fp := range filePaths {
		rel, _ := filepath.Rel(root, fp)
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		scanned++
		content := string(data)
		fileIssues := 0

		// ── 1. Check for external URLs — domain tracker with novelty detection ──
		matches := urlPattern.FindAllStringSubmatch(content, -1)
		reportedDomains := make(map[string]bool)
		for _, match := range matches {
			if fileIssues >= maxIssuesPerFile || len(issues) >= maxTotalIssues {
				break
			}
			if len(match) < 2 {
				continue
			}
			domain := match[1]
			if idx := strings.Index(domain, ":"); idx > 0 {
				domain = domain[:idx]
			}
			domain = strings.ToLower(domain)
			domain = strings.TrimRight(domain, "`") // strip trailing backtick from inline code URLs
			domain = strings.TrimRight(domain, ".") // strip trailing dot (URLs may have trailing period in plain text)
			// Skip known placeholder domains — they are never real URLs
			if isKnownPlaceholder(domain) {
				continue
			}

			check := tracker.Check(domain, rel)
			if !check.Approved {
				if check.Suggestion && !reportedDomains[domain] {
					// Medium-confidence new domain — emit as suggestion, not failure
					suggestions = append(suggestions, fmt.Sprintf("domain=%s confidence=%.0f%% source=%s", domain, check.Confidence*100, rel))
					reportedDomains[domain] = true
				} else if !reportedDomains[domain] {
					// Low-confidence or genuinely suspicious — fail
					scheme := "https"
					if strings.HasPrefix(match[0], "http://") {
						scheme = "http"
					}
					issues = append(issues, fmt.Sprintf("FIREWALL: %s — external URL to unapproved domain: %s://%s", rel, scheme, domain))
					fileIssues++
					reportedDomains[domain] = true
				}
			}
		}
		if len(issues) >= maxTotalIssues {
			issues = append(issues, fmt.Sprintf("TRUNCATED: stopped at %d total issues (limit %d)", maxTotalIssues, maxTotalIssues))
			break
		}

		// ── 2. Check for base64-encoded blocks ────────────────────────────
		b64Matches := base64Pattern.FindAllString(content, -1)
		for _, b64 := range b64Matches {
			// Try to decode and check if it looks like encoded code/commands
			decoded, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				// Try URL-safe base64
				decoded, err = base64.URLEncoding.DecodeString(b64)
				if err != nil {
					continue
				}
			}
			// Check for suspicious decoded content
			decStr := string(decoded)
			if containsSuspiciousPattern(decStr) {
				issues = append(issues, fmt.Sprintf("FIREWALL: %s — suspicious base64 block decodes to: %q", rel, truncate(decStr, 80)))
			}
		}

		// ── 3. Check for hidden Unicode / control characters ─────────────
		for _, sc := range suspiciousChars {
			for _, r := range sc.runes {
				if strings.ContainsRune(content, r) {
					// Find context around the character
					idx := strings.IndexRune(content, r)
					ctx := safeContext(content, idx, 30)
					issues = append(issues, fmt.Sprintf("FIREWALL: %s — hidden char %s (U+%04X) near: %q", rel, sc.name, r, ctx))
					break // One report per character type per file
				}
			}
		}

		// Also check for control characters (excluding standard whitespace)
		for i, r := range content {
			if r < 0x20 && r != '\n' && r != '\r' && r != '\t' {
				ctx := safeContext(content, i, 20)
				issues = append(issues, fmt.Sprintf("FIREWALL: %s — control character U+%04X near: %q", rel, r, ctx))
				break // One report per file
			}
		}
	}

	// Persist tracker state
	tracker.Save()

	if len(issues) > 0 {
		return Result{
			ID: v.ID(), Name: v.Name(), Status: "fail", Weight: v.Weight(),
			Message:     fmt.Sprintf("FAIL context firewall v2 — %d issue(s) in %d files scanned", len(issues), scanned),
			Issues:      issues,
			Suggestions: suggestions,
			Duration:    time.Since(start),
		}
	}
	return Result{
		ID: v.ID(), Name: v.Name(), Status: "pass", Weight: v.Weight(),
		Message:     fmt.Sprintf("PASS context firewall v2 — %d files scanned, no injection patterns detected", scanned),
		Suggestions: suggestions,
		Duration:    time.Since(start),
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// isApprovedDomain checks if a domain is in the approved list.
// Also allows subdomains of approved domains (e.g., docs.github.com → github.com approved).
func isApprovedDomain(domain string) bool {
	if approvedDomains[domain] {
		return true
	}
	// Check parent domains
	parts := strings.Split(domain, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[i:], ".")
		if approvedDomains[parent] {
			return true
		}
	}
	return false
}

// containsSuspiciousPattern checks decoded base64 content for suspicious patterns.
func containsSuspiciousPattern(s string) bool {
	suspicious := []string{
		"ignore all previous instructions",
		"disable security",
		"bypass",
		"rm -rf",
		"curl",
		"wget",
		"eval",
		"exec(",
		"system(",
		"__import__",
		"os.system",
		"subprocess",
		"cat /etc/",
		"/bin/bash",
		"reverse shell",
		"base64 -d",
		"powershell",
		"cmd.exe",
	}
	lower := strings.ToLower(s)
	for _, pattern := range suspicious {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// truncated returns s truncated to maxLen characters.
func truncate(s string, maxLen int) string {
	// Remove non-printable chars for display
	var clean strings.Builder
	for _, r := range s {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			clean.WriteRune(r)
		}
	}
	s = clean.String()
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// safeContext returns a safe substring context around position idx.
func safeContext(s string, idx, radius int) string {
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + radius
	if end > len(s) {
		end = len(s)
	}
	return truncate(s[start:end], radius*2+10)
}

var _ Validator = (*ContextFirewallV2)(nil)
